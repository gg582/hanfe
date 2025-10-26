package emitter

import (
	"fmt"
	"os"
	"sync"
	"unicode/utf8"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgb/xtest"
)

const (
	keysymReturn = 0xff0d
	keysymTab    = 0xff09
)

type x11Injector struct {
	conn     *xgb.Conn
	keycode  byte
	width    int
	original []xproto.Keysym
	mu       sync.Mutex
}

func newX11Injector() (*x11Injector, error) {
	display := os.Getenv("DISPLAY")
	if display == "" {
		return nil, fmt.Errorf("DISPLAY not set")
	}
	conn, err := xgb.NewConnDisplay(display)
	if err != nil {
		return nil, err
	}
	if err := xtest.Init(conn); err != nil {
		conn.Close()
		return nil, err
	}
	setup := xproto.Setup(conn)
	min := byte(setup.MinKeycode)
	max := byte(setup.MaxKeycode)
	count := int(max - min + 1)
	reply, err := xproto.GetKeyboardMapping(conn, xproto.Keycode(min), byte(count)).Reply()
	if err != nil {
		conn.Close()
		return nil, err
	}
	width := int(reply.KeysymsPerKeycode)
	if width <= 0 {
		conn.Close()
		return nil, fmt.Errorf("invalid keysyms width")
	}
	var chosen byte
	keysyms := reply.Keysyms
	for i := 0; i < count; i++ {
		kc := byte(int(min) + i)
		slice := keysyms[i*width : (i+1)*width]
		empty := true
		for _, sym := range slice {
			if sym != 0 {
				empty = false
				break
			}
		}
		if empty {
			chosen = kc
			break
		}
	}
	if chosen == 0 {
		chosen = max
	}
	idx := int(int(chosen)-int(min)) * width
	original := append([]xproto.Keysym(nil), keysyms[idx:idx+width]...)
	inj := &x11Injector{conn: conn, keycode: chosen, width: width, original: original}
	if err := inj.updateMapping(0); err != nil {
		conn.Close()
		return nil, err
	}
	return inj, nil
}

func (x *x11Injector) updateMapping(sym xproto.Keysym) error {
	keysyms := make([]xproto.Keysym, x.width)
	if sym != 0 {
		keysyms[0] = sym
	}
	return xproto.ChangeKeyboardMappingChecked(x.conn, byte(x.width), xproto.Keycode(x.keycode), 1, keysyms).Check()
}

func (x *x11Injector) typeRune(r rune) error {
	if r == '\n' {
		return x.typeKeySym(xproto.Keysym(keysymReturn))
	}
	if r == '\r' {
		return nil
	}
	if r == '\t' {
		return x.typeKeySym(xproto.Keysym(keysymTab))
	}
	if r < 0 {
		return fmt.Errorf("invalid rune")
	}
	keysym := xproto.Keysym(0x01000000 | uint32(r))
	return x.typeKeySym(keysym)
}

func (x *x11Injector) typeKeySym(sym xproto.Keysym) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	if err := x.updateMapping(sym); err != nil {
		return err
	}
	if err := xtest.FakeInputChecked(x.conn, xproto.KeyPress, x.keycode, 0, xproto.Window(0), 0, 0, 0).Check(); err != nil {
		return err
	}
	if err := xtest.FakeInputChecked(x.conn, xproto.KeyRelease, x.keycode, 0, xproto.Window(0), 0, 0, 0).Check(); err != nil {
		return err
	}
	x.conn.Sync()
	return nil
}

func (x *x11Injector) TypeText(text string) error {
	for len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		if r == utf8.RuneError && size == 1 {
			return fmt.Errorf("invalid utf-8")
		}
		if err := x.typeRune(r); err != nil {
			return err
		}
		text = text[size:]
	}
	return nil
}

func (x *x11Injector) Close() error {
	x.mu.Lock()
	defer x.mu.Unlock()
	defer x.conn.Close()
	_ = xproto.ChangeKeyboardMappingChecked(x.conn, byte(x.width), xproto.Keycode(x.keycode), 1, x.original).Check()
	x.conn.Sync()
	return nil
}
