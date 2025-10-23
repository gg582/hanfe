package emitter

import (
	"fmt"
	"syscall"
	"unicode/utf8"
	"unsafe"

	"hanfe/internal/linux"
	"hanfe/internal/util"
)

type FallbackEmitter struct {
	uinputFD    int
	ttyFD       int
	closed      bool
	hexKeycodes [16]int
}

const (
	absCnt = 0x3f + 1
)

type inputID struct {
	Bustype uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

type uinputUserDev struct {
	Name         [linux.UinputMaxNameSize]byte
	ID           inputID
	FFEffectsMax int32
	Absmax       [absCnt]int32
	Absmin       [absCnt]int32
	Absfuzz      [absCnt]int32
	Absflat      [absCnt]int32
}

func Open(hexMap map[rune]uint16, ttyPath string) (*FallbackEmitter, error) {
	emitter := &FallbackEmitter{uinputFD: -1, ttyFD: -1}
	for i := range emitter.hexKeycodes {
		emitter.hexKeycodes[i] = -1
	}
	for ch, code := range hexMap {
		idx := hexIndex(ch)
		if idx >= 0 && idx < len(emitter.hexKeycodes) {
			emitter.hexKeycodes[idx] = int(code)
		}
	}

	fd, err := syscall.Open("/dev/uinput", syscall.O_WRONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/uinput: %w", err)
	}
	emitter.uinputFD = fd

	if err := configureUinput(fd); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	if ttyPath != "" {
		ttyFD, err := syscall.Open(ttyPath, syscall.O_WRONLY|syscall.O_CLOEXEC, 0)
		if err != nil {
			emitter.Close()
			return nil, fmt.Errorf("open tty %s: %w", ttyPath, err)
		}
		emitter.ttyFD = ttyFD
	}

	return emitter, nil
}

func configureUinput(fd int) error {
	if err := linux.IoctlSetInt(fd, linux.UISetEvbit, linux.EvSyn); err != nil {
		return fmt.Errorf("UI_SET_EVBIT(EV_SYN): %w", err)
	}
	if err := linux.IoctlSetInt(fd, linux.UISetEvbit, linux.EvKey); err != nil {
		return fmt.Errorf("UI_SET_EVBIT(EV_KEY): %w", err)
	}
	for code := 0; code <= linux.KeyMax; code++ {
		_ = linux.IoctlSetInt(fd, linux.UISetKeybit, code)
	}

	var setup uinputUserDev
	copy(setup.Name[:], []byte("hanfe-fallback"))
	setup.ID.Bustype = linux.BusUSB
	setup.ID.Vendor = 0x1
	setup.ID.Product = 0x1
	setup.ID.Version = 1

	size := unsafe.Sizeof(setup)
	buf := linux.UnsafeSlice((*byte)(unsafe.Pointer(&setup)), int(size))
	if _, err := syscall.Write(fd, buf); err != nil {
		return fmt.Errorf("write uinput setup: %w", err)
	}

	if err := linux.IoctlSetInt(fd, linux.UIDevCreate, 0); err != nil {
		return fmt.Errorf("UI_DEV_CREATE: %w", err)
	}
	return nil
}

func (e *FallbackEmitter) Close() error {
	if e.closed {
		return nil
	}
	e.closed = true
	if e.uinputFD >= 0 {
		_ = linux.IoctlSetInt(e.uinputFD, linux.UIDevDestroy, 0)
		syscall.Close(e.uinputFD)
		e.uinputFD = -1
	}
	if e.ttyFD >= 0 {
		syscall.Close(e.ttyFD)
		e.ttyFD = -1
	}
	return nil
}

func (e *FallbackEmitter) ForwardEvent(ev *util.InputEvent) error {
	if e.uinputFD < 0 || ev == nil {
		return nil
	}
	if _, err := syscall.Write(e.uinputFD, ev.Bytes()); err != nil {
		return err
	}
	return e.emitSync()
}

func (e *FallbackEmitter) SendKeyState(code uint16, pressed bool) error {
	value := int32(0)
	if pressed {
		value = 1
	}
	event := util.InputEvent{Type: linux.EvKey, Code: code, Value: value}
	if _, err := syscall.Write(e.uinputFD, event.Bytes()); err != nil {
		return err
	}
	return e.emitSync()
}

func (e *FallbackEmitter) TapKey(code uint16) error {
	if err := e.SendKeyState(code, true); err != nil {
		return err
	}
	return e.SendKeyState(code, false)
}

func (e *FallbackEmitter) emitSync() error {
	if e.uinputFD < 0 {
		return nil
	}
	syn := util.InputEvent{Type: linux.EvSyn, Code: linux.SynReport, Value: 0}
	_, err := syscall.Write(e.uinputFD, syn.Bytes())
	return err
}

func (e *FallbackEmitter) SendBackspace(count int) error {
	for i := 0; i < count; i++ {
		if err := e.TapKey(uint16(linux.KeyBackspace)); err != nil {
			return err
		}
		if err := e.ttyPushByte('\b'); err != nil {
			return err
		}
	}
	return nil
}

func (e *FallbackEmitter) SendText(text string) error {
	if text == "" {
		return nil
	}
	for len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		if r == utf8.RuneError && size == 1 {
			return fmt.Errorf("invalid utf-8 sequence")
		}
		segment := text[:size]
		if err := e.ttyWriteBytes(segment); err != nil {
			return err
		}
		if err := e.typeUnicode(r); err != nil {
			return err
		}
		text = text[size:]
	}
	return nil
}

func (e *FallbackEmitter) ttyWriteBytes(data string) error {
	if e.ttyFD < 0 {
		return nil
	}
	for i := 0; i < len(data); i++ {
		if err := e.ttyPushByte(data[i]); err != nil {
			return err
		}
	}
	return nil
}

func (e *FallbackEmitter) ttyPushByte(b byte) error {
	if e.ttyFD < 0 {
		return nil
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(e.ttyFD), uintptr(syscall.TIOCSTI), uintptr(unsafe.Pointer(&b)))
	if errno != 0 {
		if _, err := syscall.Write(e.ttyFD, []byte{b}); err != nil {
			return err
		}
	}
	return nil
}

func (e *FallbackEmitter) typeUnicode(r rune) error {
	if e.uinputFD < 0 {
		return nil
	}
	if err := e.SendKeyState(uint16(linux.KeyLeftCtrl), true); err != nil {
		return err
	}
	if err := e.SendKeyState(uint16(linux.KeyLeftShift), true); err != nil {
		return err
	}
	if err := e.TapKey(uint16(linux.KeyU)); err != nil {
		return err
	}
	if err := e.SendKeyState(uint16(linux.KeyLeftShift), false); err != nil {
		return err
	}
	if err := e.SendKeyState(uint16(linux.KeyLeftCtrl), false); err != nil {
		return err
	}

	hexDigits := fmt.Sprintf("%x", r)
	for _, ch := range hexDigits {
		idx := hexIndex(ch)
		if idx < 0 || idx >= len(e.hexKeycodes) {
			continue
		}
		key := e.hexKeycodes[idx]
		if key < 0 {
			continue
		}
		if err := e.TapKey(uint16(key)); err != nil {
			return err
		}
	}

	if err := e.SendKeyState(uint16(linux.KeyLeftCtrl), true); err != nil {
		return err
	}
	if err := e.SendKeyState(uint16(linux.KeyLeftShift), true); err != nil {
		return err
	}
	if err := e.TapKey(uint16(linux.KeyEnter)); err != nil {
		return err
	}
	if err := e.SendKeyState(uint16(linux.KeyLeftShift), false); err != nil {
		return err
	}
	if err := e.SendKeyState(uint16(linux.KeyLeftCtrl), false); err != nil {
		return err
	}
	return nil
}

func hexIndex(ch rune) int {
	switch {
	case ch >= '0' && ch <= '9':
		return int(ch - '0')
	case ch >= 'a' && ch <= 'f':
		return 10 + int(ch-'a')
	case ch >= 'A' && ch <= 'F':
		return 10 + int(ch-'A')
	default:
		return -1
	}
}
