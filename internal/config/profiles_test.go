package config

import (
	"os"
	"path/filepath"
	"testing"

	"hanfe/internal/types"
)

func TestDefaultProfilesConfig(t *testing.T) {
	cfg := DefaultProfilesConfig()
	if len(cfg.Profiles) != 3 {
		t.Fatalf("expected 3 default profiles, got %d", len(cfg.Profiles))
	}
	if cfg.Profiles[0].Mode != types.ModeLatin {
		t.Fatalf("expected first profile to be latin")
	}
	if cfg.Profiles[1].Mode != types.ModeHangul {
		t.Fatalf("expected second profile to be hangul")
	}
}

func TestLoadProfilesConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.ini")
	content := `
[profiles]
order = default, hangul, kana

[profile default]
mode = latin
layout = latin

[profile hangul]
mode = hangul
layout = dubeolsik
pair = key_q:jamo:ㅂ

[profile kana]
mode = kana
layout = kana-86
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write profiles: %v", err)
	}

	cfg, err := LoadProfilesConfig(path)
	if err != nil {
		t.Fatalf("load profiles: %v", err)
	}
	if len(cfg.Profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(cfg.Profiles))
	}
	hangul := cfg.Profiles[1]
	if len(hangul.Overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(hangul.Overrides))
	}
	if hangul.Overrides[0].Kind != PairJamo {
		t.Fatalf("expected jamo override")
	}
}
