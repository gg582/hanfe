package engine

import (
	"os"
	"path/filepath"
	"testing"

	"hanfe/internal/config"
	"hanfe/internal/dbinput"
	"hanfe/internal/layout"
	"hanfe/internal/linux"
	"hanfe/internal/types"
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
	profiles := []ProfileSpec{
		{Name: "default", Mode: types.ModeLatin},
		{Name: "dubeolsik", Mode: types.ModeHangul, Layout: keyLayout},
	}
	eng := NewEngine(0, profiles, config.DefaultToggleConfig(), emitter)
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

func TestEngineCyclesProfiles(t *testing.T) {
	hangulLayout, err := layout.Load("dubeolsik")
	if err != nil {
		t.Fatalf("load hangul layout: %v", err)
	}
	kanaLayout, err := layout.Load("kana-86")
	if err != nil {
		t.Fatalf("load kana layout: %v", err)
	}
	profiles := []ProfileSpec{
		{Name: "default", Mode: types.ModeLatin},
		{Name: "hangul", Mode: types.ModeHangul, Layout: hangulLayout},
		{Name: "kana", Mode: types.ModeKana, Layout: kanaLayout},
	}
	emitter := &fakeEmitter{}
	cfg := config.DefaultToggleConfig()
	eng := NewEngine(0, profiles, cfg, emitter)

	if eng.mode != types.ModeHangul {
		t.Fatalf("expected initial mode hangul, got %s", eng.mode)
	}

	pressKey(t, eng, uint16(linux.KeyRightAlt))
	if eng.mode != types.ModeKana {
		t.Fatalf("expected mode kana after first toggle, got %s", eng.mode)
	}

	pressKey(t, eng, uint16(linux.KeyRightAlt))
	if eng.mode != types.ModeLatin {
		t.Fatalf("expected mode latin after second toggle, got %s", eng.mode)
	}

	pressKey(t, eng, uint16(linux.KeyRightAlt))
	if eng.mode != types.ModeHangul {
		t.Fatalf("expected mode hangul after third toggle, got %s", eng.mode)
	}
}

func TestEngineDatabaseCommit(t *testing.T) {
	dir := t.TempDir()
	dictPath := filepath.Join(dir, "pinyin.tsv")
	if err := os.WriteFile(dictPath, []byte("ni\t你\n"), 0o644); err != nil {
		t.Fatalf("write dict: %v", err)
	}

	dict, err := dbinput.LoadDictionary(dictPath)
	if err != nil {
		t.Fatalf("load dict: %v", err)
	}

	latinLayout, err := layout.Load("latin")
	if err != nil {
		t.Fatalf("load latin layout: %v", err)
	}

	profiles := []ProfileSpec{
		{Name: "default", Mode: types.ModeLatin},
		{Name: "pinyin", Mode: types.ModeDatabase, Layout: latinLayout, Dictionary: dict},
	}
	emitter := &fakeEmitter{}
	cfg := config.DefaultToggleConfig()
	cfg.DefaultMode = types.ModeDatabase
	eng := NewEngine(0, profiles, cfg, emitter)

	if eng.mode != types.ModeDatabase {
		t.Fatalf("expected initial mode database, got %s", eng.mode)
	}
	if eng.activeProfile().session == nil {
		t.Fatalf("expected database session to be initialised")
	}

	pressKey(t, eng, uint16(linux.KeyN))
	if eng.preedit != "n" {
		t.Fatalf("expected preedit n after first key, got %q", eng.preedit)
	}
	pressKey(t, eng, uint16(linux.KeyI))

	if eng.preedit != "ni" {
		t.Fatalf("expected preedit ni, got %q", eng.preedit)
	}

	pressKey(t, eng, uint16(linux.KeySpace))
	if got := emitter.String(); got != "你" {
		t.Fatalf("expected commit 你, got %q", got)
	}
	if eng.preedit != "" {
		t.Fatalf("expected cleared preedit, got %q", eng.preedit)
	}
}
