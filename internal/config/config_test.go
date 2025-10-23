package config

import (
	"os"
	"path/filepath"
	"testing"

	"hanfe/internal/linux"
	"hanfe/internal/types"
)

func TestDefaultToggleConfig(t *testing.T) {
	cfg := DefaultToggleConfig()
	if cfg.DefaultMode != types.ModeHangul {
		t.Fatalf("expected default mode hangul, got %v", cfg.DefaultMode)
	}
	if len(cfg.Chords) != 2 {
		t.Fatalf("expected two default toggle chords, got %d", len(cfg.Chords))
	}

	if cfg.Chords[0].Key != uint16(linux.KeyRightAlt) {
		t.Fatalf("expected first chord to toggle on RightAlt, got %d", cfg.Chords[0].Key)
	}
	if len(cfg.Chords[0].ModifierGroups) != 0 {
		t.Fatalf("expected RightAlt chord to have no modifiers")
	}

	if cfg.Chords[1].Key != uint16(linux.KeyHangeul) {
		t.Fatalf("expected second chord to toggle on Hangul, got %d", cfg.Chords[1].Key)
	}
}

func TestLoadToggleConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "toggle.ini")
	contents := "[toggle]\nkeys = alt_r, hangul, ctrl+space, alt+space\ndefault_mode = latin\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := LoadToggleConfig(path)
	if err != nil {
		t.Fatalf("LoadToggleConfig returned error: %v", err)
	}

	if cfg.DefaultMode != types.ModeLatin {
		t.Fatalf("expected default mode latin, got %v", cfg.DefaultMode)
	}

	if len(cfg.Chords) != 4 {
		t.Fatalf("expected 4 toggle chords, got %d", len(cfg.Chords))
	}

	checkChord := func(idx int, key uint16, modifiers ...uint16) {
		chord := cfg.Chords[idx]
		if chord.Key != key {
			t.Fatalf("unexpected key at index %d: got %d want %d", idx, chord.Key, key)
		}
		if len(modifiers) == 0 {
			if len(chord.ModifierGroups) != 0 {
				t.Fatalf("expected no modifiers for index %d", idx)
			}
			return
		}
		if len(chord.ModifierGroups) != 1 {
			t.Fatalf("expected one modifier group for index %d, got %d", idx, len(chord.ModifierGroups))
		}
		group := chord.ModifierGroups[0]
		for _, mod := range modifiers {
			if !containsCode(group, mod) {
				t.Fatalf("modifier %d missing from chord %d", mod, idx)
			}
		}
	}

	checkChord(0, uint16(linux.KeyRightAlt))
	checkChord(1, uint16(linux.KeyHangeul))
	checkChord(2, uint16(linux.KeySpace), uint16(linux.KeyLeftCtrl), uint16(linux.KeyRightCtrl))
	checkChord(3, uint16(linux.KeySpace), uint16(linux.KeyLeftAlt), uint16(linux.KeyRightAlt))
}

func containsCode(list []uint16, value uint16) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
