package engine

import (
	"fmt"
	"syscall"

	"hanfe/internal/config"
	"hanfe/internal/dbinput"
	"hanfe/internal/emitter"
	"hanfe/internal/hangul"
	"hanfe/internal/layout"
	"hanfe/internal/linux"
	"hanfe/internal/types"
	"hanfe/internal/util"
)

type Engine struct {
	deviceFD           int
	toggleChords       []config.ToggleChord
	emitter            emitter.Output
	profiles           []*profileState
	currentProfile     int
	mode               types.InputMode
	modifierState      map[uint16]bool
	forwardedModifiers map[uint16]bool
	forwardedKeys      map[uint16]struct{}
	preedit            string
}

type ProfileSpec struct {
	Name       string
	Mode       types.InputMode
	Layout     *layout.Layout
	Dictionary *dbinput.Dictionary
}

type profileState struct {
	spec     ProfileSpec
	composer *hangul.HangulComposer
	session  *dbinput.Session
}

func (e *Engine) activeProfile() *profileState {
	if len(e.profiles) == 0 {
		return nil
	}
	if e.currentProfile < 0 || e.currentProfile >= len(e.profiles) {
		e.currentProfile = 0
	}
	return e.profiles[e.currentProfile]
}

func (e *Engine) resolveDefaultProfile(mode types.InputMode) int {
	for i, state := range e.profiles {
		if state.spec.Mode == mode {
			return i
		}
	}
	return 0
}

func (e *Engine) resetProfileState(state *profileState) {
	if state == nil {
		return
	}
	if state.composer != nil {
		state.composer.Flush()
	}
	if state.session != nil {
		state.session.Reset()
	}
	_ = e.replacePreedit("")
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

func NewEngine(deviceFD int, profiles []ProfileSpec, toggle config.ToggleConfig, emitter emitter.Output) *Engine {
	if len(profiles) == 0 {
		profiles = append(profiles, ProfileSpec{Name: "default", Mode: types.ModeLatin})
	}

	states := make([]*profileState, len(profiles))
	for i, spec := range profiles {
		state := &profileState{spec: spec}
		if spec.Mode == types.ModeHangul {
			state.composer = hangul.NewHangulComposer()
		}
		if spec.Mode == types.ModeDatabase && spec.Dictionary != nil {
			state.session = spec.Dictionary.NewSession()
		}
		states[i] = state
	}

	eng := &Engine{
		deviceFD:           deviceFD,
		toggleChords:       toggle.Chords,
		emitter:            emitter,
		profiles:           states,
		modifierState:      make(map[uint16]bool),
		forwardedModifiers: make(map[uint16]bool),
		forwardedKeys:      make(map[uint16]struct{}),
	}
	for _, code := range modifierKeys {
		eng.modifierState[code] = false
		eng.forwardedModifiers[code] = false
	}
	for _, chord := range toggle.Chords {
		for _, group := range chord.ModifierGroups {
			for _, code := range group {
				if _, ok := eng.modifierState[code]; !ok {
					eng.modifierState[code] = false
					eng.forwardedModifiers[code] = false
				}
			}
		}
	}
	eng.currentProfile = eng.resolveDefaultProfile(toggle.DefaultMode)
	eng.mode = eng.activeProfile().spec.Mode
	eng.resetProfileState(eng.activeProfile())
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
		if e.mode == types.ModeLatin || e.mode == types.ModeKana || e.mode == types.ModeDatabase {
			return e.forwardKeyEvent(event)
		}
		return nil
	}

	if e.shouldToggle(event) {
		if isKeyPress(event) {
			return e.toggleMode()
		}
		return nil
	}

	code := event.Code
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
	active := e.activeProfile()
	if active == nil {
		return e.forwardKeyEvent(event)
	}

	switch active.spec.Mode {
	case types.ModeHangul:
		if isKeyRelease(event) {
			if _, ok := e.forwardedKeys[event.Code]; ok {
				return e.forwardKeyEvent(event)
			}
			return nil
		}
		if newPreedit, ok := active.composer.Backspace(); ok {
			return e.replacePreedit(newPreedit)
		}
		if err := e.commitPreedit(); err != nil {
			return err
		}
		return e.forwardKeyEvent(event)
	case types.ModeDatabase:
		if isKeyRelease(event) {
			if _, ok := e.forwardedKeys[event.Code]; ok {
				return e.forwardKeyEvent(event)
			}
			return nil
		}
		if active.session != nil {
			if newPreedit, ok := active.session.Backspace(); ok {
				return e.replacePreedit(newPreedit)
			}
		}
		if err := e.commitPreedit(); err != nil {
			return err
		}
		return e.forwardKeyEvent(event)
	default:
		return e.forwardKeyEvent(event)
	}
}

func (e *Engine) handleKeyRelease(event *util.InputEvent) error {
	if _, ok := e.forwardedKeys[event.Code]; ok {
		return e.forwardKeyEvent(event)
	}
	return nil
}

func (e *Engine) handleKeyPress(event *util.InputEvent) error {
	active := e.activeProfile()
	if active == nil {
		return e.forwardKeyEvent(event)
	}

	if e.modifiersActive(alwaysForward) {
		if err := e.commitPreedit(); err != nil {
			return err
		}
		if err := e.ensureShiftForwarded(); err != nil {
			return err
		}
		return e.forwardKeyEvent(event)
	}

	var symbol *layout.LayoutSymbol
	if active.spec.Layout != nil {
		symbol = active.spec.Layout.Translate(event.Code, e.shiftActive())
	}

	switch active.spec.Mode {
	case types.ModeHangul:
		return e.handleHangulPress(active, symbol, event)
	case types.ModeDatabase:
		return e.handleDatabasePress(active, symbol, event)
	case types.ModeKana:
		return e.handleTextualPress(symbol, event)
	case types.ModeLatin:
		if symbol == nil {
			if err := e.commitPreedit(); err != nil {
				return err
			}
			if err := e.ensureShiftForwarded(); err != nil {
				return err
			}
			return e.forwardKeyEvent(event)
		}
		return e.handleTextualPress(symbol, event)
	default:
		return e.forwardKeyEvent(event)
	}
}

func (e *Engine) handleHangulPress(state *profileState, symbol *layout.LayoutSymbol, event *util.InputEvent) error {
	if state == nil {
		return e.forwardKeyEvent(event)
	}
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
		if state.composer == nil {
			return e.forwardKeyEvent(event)
		}
		result := state.composer.Feed(symbol.Jamo, symbol.Role)
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

func (e *Engine) handleDatabasePress(state *profileState, symbol *layout.LayoutSymbol, event *util.InputEvent) error {
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
		if state.session == nil {
			if symbol.CommitBefore {
				if err := e.commitPreedit(); err != nil {
					return err
				}
			}
			return e.sendText(symbol.Text)
		}
		pre := state.session.Append(symbol.Text)
		if pre != e.preedit {
			if err := e.replacePreedit(pre); err != nil {
				return err
			}
		}
		return nil
	case layout.SymbolJamo:
		text := string(symbol.Jamo)
		if state.session == nil {
			if symbol.CommitBefore {
				if err := e.commitPreedit(); err != nil {
					return err
				}
			}
			return e.sendText(text)
		}
		pre := state.session.Append(text)
		if pre != e.preedit {
			if err := e.replacePreedit(pre); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

func (e *Engine) handleTextualPress(symbol *layout.LayoutSymbol, event *util.InputEvent) error {
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
		if symbol.CommitBefore {
			if err := e.commitPreedit(); err != nil {
				return err
			}
		}
		return e.sendText(string(symbol.Jamo))
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

func (e *Engine) modifierGroupsActive(groups [][]uint16) bool {
	for _, group := range groups {
		active := false
		for _, code := range group {
			if e.modifierState[code] {
				active = true
				break
			}
		}
		if !active {
			return false
		}
	}
	return true
}

func (e *Engine) shouldToggle(event *util.InputEvent) bool {
	if !isKeyPress(event) {
		return false
	}
	code := event.Code
	for _, chord := range e.toggleChords {
		if chord.Key != code {
			continue
		}
		if len(chord.ModifierGroups) == 0 || e.modifierGroupsActive(chord.ModifierGroups) {
			return true
		}
	}
	return false
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
	if len(e.profiles) <= 1 {
		return nil
	}
	if err := e.commitPreedit(); err != nil {
		return err
	}
	if current := e.activeProfile(); current != nil {
		if current.session != nil {
			current.session.Reset()
		}
		if current.composer != nil {
			current.composer.Flush()
		}
	}
	e.currentProfile = (e.currentProfile + 1) % len(e.profiles)
	next := e.activeProfile()
	if next != nil {
		e.mode = next.spec.Mode
		if next.session != nil {
			next.session.Reset()
		}
		if next.composer != nil {
			next.composer.Flush()
		}
	}
	if err := e.replacePreedit(""); err != nil {
		return err
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
	active := e.activeProfile()
	var commit string
	if active != nil {
		switch active.spec.Mode {
		case types.ModeHangul:
			if active.composer != nil {
				commit = active.composer.Flush()
			}
		case types.ModeDatabase:
			if active.session != nil {
				commit = active.session.Commit()
			}
		}
	}
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
