package engine

import (
	"fmt"
	"syscall"

	"hanfe/internal/config"
	"hanfe/internal/emitter"
	"hanfe/internal/hangul"
	"hanfe/internal/layout"
	"hanfe/internal/linux"
	"hanfe/internal/types"
	"hanfe/internal/util"
)

type Engine struct {
	deviceFD           int
	layout             layout.Layout
	toggle             config.ToggleConfig
	emitter            *emitter.FallbackEmitter
	composer           *hangul.HangulComposer
	mode               types.InputMode
	toggleKeys         map[uint16]struct{}
	modifierState      map[uint16]bool
	forwardedModifiers map[uint16]bool
	forwardedKeys      map[uint16]struct{}
	preedit            string
}

var (
	shiftKeys = []uint16{uint16(linux.KeyLeftShift), uint16(linux.KeyRightShift)}
	ctrlKeys  = []uint16{uint16(linux.KeyLeftCtrl), uint16(linux.KeyRightCtrl)}
	altKeys   = []uint16{uint16(linux.KeyLeftAlt), uint16(linux.KeyRightAlt)}
	metaKeys  = []uint16{uint16(linux.KeyLeftMeta), uint16(linux.KeyRightMeta)}
)

func combine(a, b []uint16) []uint16 {
	result := make([]uint16, 0, len(a)+len(b))
	result = append(result, a...)
	result = append(result, b...)
	return result
}

var (
	modifierKeys  = combine(combine(combine(shiftKeys, ctrlKeys), altKeys), metaKeys)
	alwaysForward = combine(combine(ctrlKeys, altKeys), metaKeys)
)

func NewEngine(deviceFD int, layout layout.Layout, toggle config.ToggleConfig, emitter *emitter.FallbackEmitter) *Engine {
	eng := &Engine{
		deviceFD:           deviceFD,
		layout:             layout,
		toggle:             toggle,
		emitter:            emitter,
		composer:           hangul.NewHangulComposer(),
		mode:               toggle.DefaultMode,
		toggleKeys:         make(map[uint16]struct{}),
		modifierState:      make(map[uint16]bool),
		forwardedModifiers: make(map[uint16]bool),
		forwardedKeys:      make(map[uint16]struct{}),
	}
	for _, code := range toggle.ToggleKeys {
		eng.toggleKeys[code] = struct{}{}
	}
	for _, code := range modifierKeys {
		eng.modifierState[code] = false
		eng.forwardedModifiers[code] = false
	}
	return eng
}

func (e *Engine) Run() error {
	if err := linux.IoctlSetInt(e.deviceFD, linux.EVIOCGRAB, 1); err != nil {
		return fmt.Errorf("grab device: %w", err)
	}
	defer linux.IoctlSetInt(e.deviceFD, linux.EVIOCGRAB, 0)
	defer e.emitter.Close()

	size := util.InputEventSize()
	for {
		var ev util.InputEvent
		buf := ev.Bytes()
		n, err := syscall.Read(e.deviceFD, buf)
		if err != nil {
			if err == syscall.EAGAIN || err == syscall.EINTR {
				continue
			}
			return fmt.Errorf("read input event: %w", err)
		}
		if n == 0 {
			return nil
		}
		if n != size {
			continue
		}
		if err := e.processEvent(&ev); err != nil {
			return err
		}
	}
}

func (e *Engine) processEvent(event *util.InputEvent) error {
	if event.Type != linux.EvKey {
		if e.mode == types.ModeLatin {
			return e.forwardKeyEvent(event)
		}
		return nil
	}

	code := event.Code
	if _, ok := e.toggleKeys[code]; ok {
		if isKeyPress(event) {
			return e.toggleMode()
		}
		return nil
	}

	if contains(modifierKeys, code) {
		return e.handleModifier(event)
	}

	if e.mode == types.ModeLatin {
		return e.forwardKeyEvent(event)
	}

	if code == uint16(linux.KeyBackspace) {
		return e.handleBackspace(event)
	}

	if isKeyRelease(event) {
		return e.handleKeyRelease(event)
	}

	return e.handleKeyPress(event)
}

func (e *Engine) handleModifier(event *util.InputEvent) error {
	code := event.Code
	press := isKeyPress(event)
	release := isKeyRelease(event)

	if press {
		e.modifierState[code] = true
	} else if release {
		e.modifierState[code] = false
	}

	if e.mode == types.ModeLatin || contains(alwaysForward, code) {
		if err := e.forwardKeyEvent(event); err != nil {
			return err
		}
		e.forwardedModifiers[code] = press && !release
		return nil
	}

	if release && e.forwardedModifiers[code] {
		e.setForwardedModifier(code, false)
	}
	return nil
}

func (e *Engine) handleBackspace(event *util.InputEvent) error {
	if e.mode == types.ModeLatin {
		return e.forwardKeyEvent(event)
	}
	if isKeyRelease(event) {
		if _, ok := e.forwardedKeys[event.Code]; ok {
			return e.forwardKeyEvent(event)
		}
		return nil
	}
	if newPreedit, ok := e.composer.Backspace(); ok {
		return e.replacePreedit(newPreedit)
	}
	if err := e.commitPreedit(); err != nil {
		return err
	}
	return e.forwardKeyEvent(event)
}

func (e *Engine) handleKeyRelease(event *util.InputEvent) error {
	if _, ok := e.forwardedKeys[event.Code]; ok {
		return e.forwardKeyEvent(event)
	}
	return nil
}

func (e *Engine) handleKeyPress(event *util.InputEvent) error {
	if e.modifiersActive(alwaysForward) {
		if err := e.commitPreedit(); err != nil {
			return err
		}
		if err := e.ensureShiftForwarded(); err != nil {
			return err
		}
		return e.forwardKeyEvent(event)
	}

	symbol := e.layout.Translate(event.Code, e.shiftActive())
	if symbol == nil {
		if err := e.commitPreedit(); err != nil {
			return err
		}
		if err := e.ensureShiftForwarded(); err != nil {
			return err
		}
		return e.forwardKeyEvent(event)
	}

	switch symbol.Kind {
	case layout.SymbolPassthrough:
		if symbol.CommitBefore {
			if err := e.commitPreedit(); err != nil {
				return err
			}
		}
		if err := e.ensureShiftForwarded(); err != nil {
			return err
		}
		return e.forwardKeyEvent(event)
	case layout.SymbolText:
		if symbol.CommitBefore {
			if err := e.commitPreedit(); err != nil {
				return err
			}
		}
		return e.sendText(symbol.Text)
	case layout.SymbolJamo:
		result := e.composer.Feed(symbol.Jamo, symbol.Role)
		if result.Commit != "" {
			if err := e.commitText(result.Commit); err != nil {
				return err
			}
		}
		if result.Preedit != e.preedit {
			if err := e.replacePreedit(result.Preedit); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

func (e *Engine) forwardKeyEvent(event *util.InputEvent) error {
	if err := e.emitter.ForwardEvent(event); err != nil {
		return err
	}
	if isKeyPress(event) {
		e.forwardedKeys[event.Code] = struct{}{}
	} else if isKeyRelease(event) {
		delete(e.forwardedKeys, event.Code)
	}
	return nil
}

func (e *Engine) modifiersActive(codes []uint16) bool {
	for _, code := range codes {
		if e.modifierState[code] {
			return true
		}
	}
	return false
}

func (e *Engine) shiftActive() bool {
	return e.modifiersActive(shiftKeys)
}

func (e *Engine) ensureShiftForwarded() error {
	for _, code := range shiftKeys {
		if e.modifierState[code] && !e.forwardedModifiers[code] {
			if err := e.setForwardedModifier(code, true); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Engine) setForwardedModifier(code uint16, pressed bool) error {
	if current := e.forwardedModifiers[code]; current == pressed {
		return nil
	}
	if err := e.emitter.SendKeyState(code, pressed); err != nil {
		return err
	}
	e.forwardedModifiers[code] = pressed
	return nil
}

func (e *Engine) suspendForwardedModifiers() ([]uint16, error) {
	var suspended []uint16
	for code, forwarded := range e.forwardedModifiers {
		if forwarded {
			if err := e.setForwardedModifier(code, false); err != nil {
				return suspended, err
			}
			suspended = append(suspended, code)
		}
	}
	return suspended, nil
}

func (e *Engine) restoreForwardedModifiers(codes []uint16) {
	for _, code := range codes {
		if e.modifierState[code] {
			_ = e.setForwardedModifier(code, true)
		}
	}
}

func (e *Engine) toggleMode() error {
	if err := e.commitPreedit(); err != nil {
		return err
	}
	if e.mode == types.ModeHangul {
		e.mode = types.ModeLatin
	} else {
		e.mode = types.ModeHangul
	}
	return nil
}

func (e *Engine) commitText(text string) error {
	if text == "" {
		return nil
	}
	if err := e.replacePreedit(""); err != nil {
		return err
	}
	return e.sendText(text)
}

func (e *Engine) commitPreedit() error {
	commit := e.composer.Flush()
	if commit == "" && e.preedit == "" {
		return nil
	}
	if err := e.replacePreedit(""); err != nil {
		return err
	}
	if commit != "" {
		if err := e.sendText(commit); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) replacePreedit(newText string) error {
	if newText == e.preedit {
		return nil
	}
	suspended, err := e.suspendForwardedModifiers()
	if err != nil {
		return err
	}
	if e.preedit != "" {
		count := countRunes(e.preedit)
		if count > 0 {
			if err := e.emitter.SendBackspace(count); err != nil {
				e.restoreForwardedModifiers(suspended)
				return err
			}
		}
	}
	if newText != "" {
		if err := e.emitter.SendText(newText); err != nil {
			e.restoreForwardedModifiers(suspended)
			return err
		}
	}
	e.preedit = newText
	e.restoreForwardedModifiers(suspended)
	return nil
}

func (e *Engine) sendText(text string) error {
	if text == "" {
		return nil
	}
	suspended, err := e.suspendForwardedModifiers()
	if err != nil {
		return err
	}
	err = e.emitter.SendText(text)
	e.restoreForwardedModifiers(suspended)
	return err
}

func countRunes(s string) int {
	count := 0
	for range s {
		count++
	}
	return count
}

func isKeyPress(ev *util.InputEvent) bool {
	return ev.Value == 1 || ev.Value == 2
}

func isKeyRelease(ev *util.InputEvent) bool {
	return ev.Value == 0
}

func contains(list []uint16, code uint16) bool {
	for _, c := range list {
		if c == code {
			return true
		}
	}
	return false
}
