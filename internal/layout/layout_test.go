package layout

import (
	"testing"

	"github.com/gg582/hanfe/internal/hangul"
	"github.com/gg582/hanfe/internal/linux"
)

func TestAvailableLayouts(t *testing.T) {
	names := AvailableLayouts()

	expected := []string{"dubeolsik", "kana86", "sebeolsik-390"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d layouts, got %d", len(expected), len(names))
	}

	for i, name := range expected {
		if names[i] != name {
			t.Fatalf("expected layout %d to be %q, got %q", i, name, names[i])
		}
	}
}

func TestLoadDubeolsik(t *testing.T) {
	layout, err := Load("dubeolsik")
	if err != nil {
		t.Fatalf("unexpected error loading dubeolsik: %v", err)
	}

	symbol := layout.Translate(uint16(linux.KeyQ), false)
	if symbol == nil {
		t.Fatalf("expected symbol for KeyQ")
	}
	if symbol.Kind != SymbolJamo || symbol.Jamo != 'ㅂ' || symbol.Role != hangul.RoleAuto {
		t.Fatalf("unexpected symbol for KeyQ: %#v", symbol)
	}

	shifted := layout.Translate(uint16(linux.KeyQ), true)
	if shifted == nil || shifted.Jamo != 'ㅃ' {
		t.Fatalf("expected shifted symbol 'ㅃ', got %#v", shifted)
	}

	missing := layout.Translate(uint16(0xffff), false)
	if missing != nil {
		t.Fatalf("expected no mapping for unknown key")
	}
}

func TestLoadSebeolsikTrailingRole(t *testing.T) {
	layout, err := Load("sebeolsik-390")
	if err != nil {
		t.Fatalf("unexpected error loading sebeolsik-390: %v", err)
	}

	symbol := layout.Translate(uint16(linux.KeyY), true)
	if symbol == nil {
		t.Fatalf("expected shifted symbol for KeyY")
	}
	if symbol.Kind != SymbolJamo || symbol.Role != hangul.RoleTrailing || symbol.Jamo != 'ㅅ' {
		t.Fatalf("expected trailing role symbol for shifted KeyY, got %#v", symbol)
	}
}

func TestLoadKana86(t *testing.T) {
	layout, err := Load("kana86")
	if err != nil {
		t.Fatalf("unexpected error loading kana86: %v", err)
	}

	symbol := layout.Translate(uint16(linux.KeyQ), false)
	if symbol == nil || symbol.Kind != SymbolText || symbol.Text != "た" {
		t.Fatalf("expected hiragana for KeyQ, got %#v", symbol)
	}

	shifted := layout.Translate(uint16(linux.KeyQ), true)
	if shifted == nil || shifted.Kind != SymbolText || shifted.Text != "タ" {
		t.Fatalf("expected katakana for shifted KeyQ, got %#v", shifted)
	}
}

func TestLoadUnknownLayout(t *testing.T) {
	if _, err := Load("does-not-exist"); err == nil {
		t.Fatalf("expected error for unknown layout")
	}
}
