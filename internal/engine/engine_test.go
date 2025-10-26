package engine

import (
	"testing"

	"github.com/gg582/hanfe/internal/config"
	"github.com/gg582/hanfe/internal/layout"
	"github.com/gg582/hanfe/internal/linux"
	"github.com/gg582/hanfe/internal/types"
	"github.com/gg582/hanfe/internal/util"
)

type fakeEmitter struct {
	buffer          []rune
	supportsPreedit bool
	texts           []string
	backspaces      []int
}

func (f *fakeEmitter) Close() error { return nil }

func (f *fakeEmitter) ForwardEvent(ev *util.InputEvent) error { return nil }

func (f *fakeEmitter) SendKeyState(code uint16, pressed bool) error { return nil }

func (f *fakeEmitter) TapKey(code uint16) error { return nil }

func (f *fakeEmitter) SendBackspace(count int) error {
	if count <= 0 {
		return nil
	}
	f.backspaces = append(f.backspaces, count)
	if count > len(f.buffer) {
		f.buffer = nil
		return nil
	}
	f.buffer = f.buffer[:len(f.buffer)-count]
	return nil
}

func (f *fakeEmitter) SendText(text string) error {
	if text == "" {
		return nil
	}
	f.texts = append(f.texts, text)
	f.buffer = append(f.buffer, []rune(text)...)
	return nil
}

func (f *fakeEmitter) String() string { return string(f.buffer) }

func (f *fakeEmitter) SupportsPreedit() bool { return f.supportsPreedit }

func newTestEngine(t *testing.T) (*Engine, *fakeEmitter) {
	t.Helper()

	keyLayout, err := layout.Load("dubeolsik")
	if err != nil {
		t.Fatalf("load layout: %v", err)
	}
	emitter := &fakeEmitter{supportsPreedit: true}
	layoutCopy := keyLayout
	toggle := config.DefaultToggleConfig()
	toggle.ModeCycle = []string{"dubeolsik", "latin"}
	toggle.DefaultMode = "dubeolsik"
	modes := []ModeSpec{
		{Name: "dubeolsik", Kind: types.ModeHangul, Layout: &layoutCopy},
		{Name: "latin", Kind: types.ModeLatin},
	}
	eng, err := NewEngine(0, modes, toggle, emitter)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	return eng, emitter
}

func pressKey(t *testing.T, eng *Engine, code uint16) {
	t.Helper()
	press := util.InputEvent{Type: linux.EvKey, Code: code, Value: 1}
	if err := eng.processEvent(&press); err != nil {
		t.Fatalf("process press %d: %v", code, err)
	}
	release := util.InputEvent{Type: linux.EvKey, Code: code, Value: 0}
	if err := eng.processEvent(&release); err != nil {
		t.Fatalf("process release %d: %v", code, err)
	}
}

func TestEngineSeparatesBojaFromBwaj(t *testing.T) {
	eng, out := newTestEngine(t)

	pressKey(t, eng, uint16(linux.KeyQ))
	pressKey(t, eng, uint16(linux.KeyH))
	pressKey(t, eng, uint16(linux.KeyW))
	pressKey(t, eng, uint16(linux.KeyK))

	if got := out.String(); got != "보자" {
		t.Fatalf("expected buffer to contain committed '보' followed by preedit '자', got %q", got)
	}
	if eng.preedit != "자" {
		t.Fatalf("expected preedit '자', got %q", eng.preedit)
	}
}

func TestEngineRetainsDoubleMedialForBwaj(t *testing.T) {
	eng, out := newTestEngine(t)

	pressKey(t, eng, uint16(linux.KeyQ))
	pressKey(t, eng, uint16(linux.KeyH))
	pressKey(t, eng, uint16(linux.KeyK))
	pressKey(t, eng, uint16(linux.KeyW))

	if got := out.String(); got != "봦" {
		t.Fatalf("expected buffer to reflect preedit '봦', got %q", got)
	}
	if eng.preedit != "봦" {
		t.Fatalf("expected preedit '봦', got %q", eng.preedit)
	}
}

func TestEngineSkipsPreeditWhenUnsupported(t *testing.T) {
	eng, out := newTestEngine(t)
	out.supportsPreedit = false

	if err := eng.replacePreedit("난"); err != nil {
		t.Fatalf("replace preedit: %v", err)
	}
	if len(out.texts) != 0 {
		t.Fatalf("expected no text emissions, got %v", out.texts)
	}
	if len(out.backspaces) != 0 {
		t.Fatalf("expected no backspace emissions, got %v", out.backspaces)
	}
	if eng.preedit != "난" {
		t.Fatalf("expected stored preedit '난', got %q", eng.preedit)
	}

	if err := eng.replacePreedit(""); err != nil {
		t.Fatalf("clear preedit: %v", err)
	}
	if len(out.texts) != 0 {
		t.Fatalf("expected no text emissions after clearing, got %v", out.texts)
	}
	if len(out.backspaces) != 0 {
		t.Fatalf("expected no backspace emissions after clearing, got %v", out.backspaces)
	}
	if eng.preedit != "" {
		t.Fatalf("expected empty preedit, got %q", eng.preedit)
	}

	if err := eng.sendText("난"); err != nil {
		t.Fatalf("send text: %v", err)
	}
	if got := out.String(); got != "난" {
		t.Fatalf("expected committed text '난', got %q", got)
	}
	if len(out.texts) != 1 || out.texts[0] != "난" {
		t.Fatalf("expected single committed text '난', got %v", out.texts)
	}
}
