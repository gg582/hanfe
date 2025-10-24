package engine

import (
	"testing"

	"hanfe/internal/config"
	"hanfe/internal/layout"
	"hanfe/internal/linux"
	"hanfe/internal/util"
)

type fakeEmitter struct {
	buffer []rune
}

func (f *fakeEmitter) Close() error { return nil }

func (f *fakeEmitter) ForwardEvent(ev *util.InputEvent) error { return nil }

func (f *fakeEmitter) SendKeyState(code uint16, pressed bool) error { return nil }

func (f *fakeEmitter) TapKey(code uint16) error { return nil }

func (f *fakeEmitter) SendBackspace(count int) error {
	if count <= 0 {
		return nil
	}
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
	f.buffer = append(f.buffer, []rune(text)...)
	return nil
}

func (f *fakeEmitter) String() string { return string(f.buffer) }

func newTestEngine(t *testing.T) (*Engine, *fakeEmitter) {
	t.Helper()

	keyLayout, err := layout.Load("dubeolsik")
	if err != nil {
		t.Fatalf("load layout: %v", err)
	}
	emitter := &fakeEmitter{}
	eng := NewEngine(0, keyLayout, config.DefaultToggleConfig(), emitter)
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
		t.Fatalf("expected buffer to contain '보자', got %q", got)
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
		t.Fatalf("expected buffer to contain '봦', got %q", got)
	}
	if eng.preedit != "봦" {
		t.Fatalf("expected preedit '봦', got %q", eng.preedit)
	}
}
