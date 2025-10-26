package engine

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/gg582/hanfe/internal/backend"
	"github.com/gg582/hanfe/internal/config"
	"github.com/gg582/hanfe/internal/emitter"
	"github.com/gg582/hanfe/internal/hangul"
	"github.com/gg582/hanfe/internal/layout"
	"github.com/gg582/hanfe/internal/linux"
	"github.com/gg582/hanfe/internal/types"
	"github.com/gg582/hanfe/internal/util"
	"golang.org/x/sys/unix"
)

type ModeSpec struct {
	Name     string
	Kind     types.InputMode
	Layout   *layout.Layout
	Database backend.Database
}

type Engine struct {
	deviceFD           int
	modes              []ModeSpec
	modeIndex          int
	toggleChords       []config.ToggleChord
	emitter            emitter.Output
	hangulComposers    map[int]*hangul.HangulComposer
	modifierState      map[uint16]bool
	forwardedModifiers map[uint16]bool
	forwardedKeys      map[uint16]struct{}
	preedit            string
	pinyinBuffer       string
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

func NewEngine(deviceFD int, modes []ModeSpec, toggle config.ToggleConfig, emitter emitter.Output) (*Engine, error) {
	if len(modes) == 0 {
		return nil, fmt.Errorf("no input modes configured")
	}

	eng := &Engine{
		deviceFD:           deviceFD,
		modes:              modes,
		toggleChords:       toggle.Chords,
		emitter:            emitter,
		hangulComposers:    make(map[int]*hangul.HangulComposer),
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
	defaultIndex := 0
	if toggle.DefaultMode != "" {
		for idx, mode := range modes {
			if strings.EqualFold(mode.Name, toggle.DefaultMode) {
				defaultIndex = idx
				break
			}
		}
	}
	eng.modeIndex = defaultIndex

	for idx, mode := range modes {
		if mode.Kind == types.ModeHangul {
			eng.hangulComposers[idx] = hangul.NewHangulComposer()
		}
	}
	return eng, nil
}

func (e *Engine) Run() error {
	if err := linux.IoctlSetInt(e.deviceFD, linux.EVIOCGRAB, 1); err != nil {
		return fmt.Errorf("grab device: %w", err)
	}
	defer linux.IoctlSetInt(e.deviceFD, linux.EVIOCGRAB, 0)
	defer e.emitter.Close()

	size := util.InputEventSize()
	pollFDs := []unix.PollFd{{Fd: int32(e.deviceFD), Events: unix.POLLIN}}
	for {
		var ev util.InputEvent
		buf := ev.Bytes()
		n, err := syscall.Read(e.deviceFD, buf)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			if err == syscall.EAGAIN {
				if pollErr := waitForReadable(pollFDs); pollErr != nil {
					return fmt.Errorf("poll input device: %w", pollErr)
				}
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

func waitForReadable(pollFDs []unix.PollFd) error {
	for {
		for i := range pollFDs {
			pollFDs[i].Revents = 0
		}
		n, err := unix.Poll(pollFDs, -1)
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		ready := false
		for _, fd := range pollFDs {
			if fd.Revents&(unix.POLLERR|unix.POLLHUP|unix.POLLNVAL) != 0 {
				return syscall.EIO
			}
			if fd.Revents&(unix.POLLIN|unix.POLLPRI) != 0 {
				ready = true
			}
		}
		if ready {
			return nil
		}
	}
}

func (e *Engine) processEvent(event *util.InputEvent) error {
	if event.Type != linux.EvKey {
		if e.currentModeKind() == types.ModeLatin {
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

	if e.currentModeKind() == types.ModeLatin {
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

	if e.currentModeKind() == types.ModeLatin || contains(alwaysForward, code) {
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
	switch e.currentModeKind() {
	case types.ModeLatin, types.ModeKana:
		return e.forwardKeyEvent(event)
	case types.ModeDatabase:
		return e.handleDatabaseBackspace(event)
	}
	if isKeyRelease(event) {
		if _, ok := e.forwardedKeys[event.Code]; ok {
			return e.forwardKeyEvent(event)
		}
		return nil
	}
	if composer := e.currentComposerIfHangul(); composer != nil {
		if newPreedit, ok := composer.Backspace(); ok {
			return e.replacePreedit(newPreedit)
		}
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
	mode := e.currentMode()
	if mode.Kind == types.ModeDatabase {
		return e.handleDatabaseKeyPress(event)
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

	if mode.Layout == nil {
		if err := e.commitPreedit(); err != nil {
			return err
		}
		return e.forwardKeyEvent(event)
	}

	symbol := mode.Layout.Translate(event.Code, e.shiftActive())
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
		composer := e.currentComposer()
		result := composer.Feed(symbol.Jamo, symbol.Role)
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

func (e *Engine) currentMode() ModeSpec {
	return e.modes[e.modeIndex]
}

func (e *Engine) currentModeKind() types.InputMode {
	return e.modes[e.modeIndex].Kind
}

func (e *Engine) currentComposer() *hangul.HangulComposer {
	composer, ok := e.hangulComposers[e.modeIndex]
	if !ok || composer == nil {
		composer = hangul.NewHangulComposer()
		e.hangulComposers[e.modeIndex] = composer
	}
	return composer
}

func (e *Engine) currentComposerIfHangul() *hangul.HangulComposer {
	if e.currentModeKind() != types.ModeHangul {
		return nil
	}
	return e.currentComposer()
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
	if len(e.modes) == 0 {
		return nil
	}
	e.modeIndex = (e.modeIndex + 1) % len(e.modes)
	e.pinyinBuffer = ""
	return e.replacePreedit("")
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
	mode := e.currentMode()
	switch mode.Kind {
	case types.ModeHangul:
		composer := e.currentComposer()
		commit := composer.Flush()
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
	case types.ModeDatabase:
		return e.commitPinyinBuffer()
	default:
		if e.preedit == "" {
			return nil
		}
		return e.replacePreedit("")
	}
}

func (e *Engine) handleDatabaseKeyPress(event *util.InputEvent) error {
	if e.modifiersActive(alwaysForward) {
		if err := e.commitPreedit(); err != nil {
			return err
		}
		if err := e.ensureShiftForwarded(); err != nil {
			return err
		}
		return e.forwardKeyEvent(event)
	}

	code := event.Code
	if code >= uint16(linux.KeyA) && code <= uint16(linux.KeyZ) {
		ch := rune('a' + int(code-uint16(linux.KeyA)))
		e.pinyinBuffer += string(ch)
		return e.replacePreedit(e.pinyinBuffer)
	}

	if code >= uint16(linux.Key1) && code <= uint16(linux.Key5) {
		ch := rune('1' + int(code-uint16(linux.Key1)))
		e.pinyinBuffer += string(ch)
		return e.replacePreedit(e.pinyinBuffer)
	}

	if code == uint16(linux.KeyApostrophe) {
		e.pinyinBuffer += "'"
		return e.replacePreedit(e.pinyinBuffer)
	}

	switch code {
	case uint16(linux.KeySpace), uint16(linux.KeyEnter), uint16(linux.KeyTab):
		if err := e.commitPinyinBuffer(); err != nil {
			return err
		}
		return e.forwardKeyEvent(event)
	default:
		if err := e.commitPinyinBuffer(); err != nil {
			return err
		}
		if err := e.ensureShiftForwarded(); err != nil {
			return err
		}
		return e.forwardKeyEvent(event)
	}
}

func (e *Engine) handleDatabaseBackspace(event *util.InputEvent) error {
	if isKeyRelease(event) {
		return nil
	}
	if e.pinyinBuffer == "" {
		return e.forwardKeyEvent(event)
	}
	runes := []rune(e.pinyinBuffer)
	if len(runes) == 0 {
		return e.replacePreedit("")
	}
	e.pinyinBuffer = string(runes[:len(runes)-1])
	return e.replacePreedit(e.pinyinBuffer)
}

func (e *Engine) commitPinyinBuffer() error {
	if e.pinyinBuffer == "" {
		if e.preedit != "" {
			return e.replacePreedit("")
		}
		return nil
	}
	mode := e.currentMode()
	text := e.pinyinBuffer
	if mode.Database.Available() {
		if candidate, ok := mode.Database.Lookup(text); ok {
			text = candidate
		}
	}
	if err := e.replacePreedit(""); err != nil {
		return err
	}
	if err := e.sendText(text); err != nil {
		return err
	}
	e.pinyinBuffer = ""
	return nil
}

func (e *Engine) replacePreedit(newText string) error {
	if newText == e.preedit {
		return nil
	}
	if !e.emitter.SupportsPreedit() {
		e.preedit = newText
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
