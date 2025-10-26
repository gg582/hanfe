package app

import (
	"fmt"
	"strings"

	"github.com/gg582/hanfe/internal/backend"
	"github.com/gg582/hanfe/internal/config"
	"github.com/gg582/hanfe/internal/engine"
	"github.com/gg582/hanfe/internal/layout"
	"github.com/gg582/hanfe/internal/types"
)

func ApplyModeOrder(cfg *config.ToggleConfig, override []string, hangulName string, haveHangul bool) {
	if override != nil {
		cfg.ModeCycle = override
	}
	cfg.DefaultMode = normalizeModeName(cfg.DefaultMode, hangulName, haveHangul)
	cfg.ModeCycle = normalizeModeCycle(cfg.ModeCycle, hangulName, haveHangul)
	if cfg.DefaultMode == "" {
		if haveHangul && hangulName != "" {
			cfg.DefaultMode = hangulName
		} else {
			cfg.DefaultMode = "latin"
		}
	}
	if !containsString(cfg.ModeCycle, cfg.DefaultMode) {
		cfg.ModeCycle = append([]string{cfg.DefaultMode}, cfg.ModeCycle...)
		cfg.ModeCycle = uniqueStrings(cfg.ModeCycle)
	}
}

func normalizeModeCycle(cycle []string, hangulName string, haveHangul bool) []string {
	normalized := make([]string, 0, len(cycle))
	seen := make(map[string]struct{})
	for _, entry := range cycle {
		name := normalizeModeName(entry, hangulName, haveHangul)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	if haveHangul && hangulName != "" && !containsString(normalized, hangulName) {
		normalized = append([]string{hangulName}, normalized...)
	}
	if !containsString(normalized, "latin") {
		normalized = append(normalized, "latin")
	}
	normalized = uniqueStrings(normalized)
	if len(normalized) == 0 {
		normalized = append(normalized, "latin")
	}
	return normalized
}

func normalizeModeName(value, hangulName string, haveHangul bool) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "hangul", "korean":
		if haveHangul && hangulName != "" {
			return hangulName
		}
		return "latin"
	case "latin", "english", "default":
		return "latin"
	case "kana", "kana86", "hiragana", "katakana", "japanese":
		return "kana86"
	case "pinyin", "zhuyin", "ime", "database":
		return "pinyin"
	case "none", "off":
		return ""
	default:
		return normalized
	}
}

func containsString(list []string, target string) bool {
	for _, entry := range list {
		if entry == target {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func BuildModes(cycle []string, hangulLayout *layout.Layout, hangulName string, database backend.Database) ([]engine.ModeSpec, error) {
	available := make(map[string]engine.ModeSpec)
	available["latin"] = engine.ModeSpec{Name: "latin", Kind: types.ModeLatin}

	if hangulLayout != nil {
		layoutCopy := *hangulLayout
		kind := types.ModeHangul
		if layoutCopy.Category() == layout.CategoryKana {
			kind = types.ModeKana
		}
		spec := engine.ModeSpec{Name: hangulName, Kind: kind, Layout: &layoutCopy}
		if hangulName != "" {
			available[strings.ToLower(hangulName)] = spec
		}
	}

	for _, entry := range cycle {
		if entry == "kana86" {
			if _, ok := available["kana86"]; !ok {
				kanaLayout, err := layout.Load("kana86")
				if err != nil {
					return nil, fmt.Errorf("load kana layout: %w", err)
				}
				layoutCopy := kanaLayout
				available["kana86"] = engine.ModeSpec{Name: "kana86", Kind: types.ModeKana, Layout: &layoutCopy}
			}
		}
	}

	if database.Available() {
		available["pinyin"] = engine.ModeSpec{Name: "pinyin", Kind: types.ModeDatabase, Database: database}
	}

	modes := make([]engine.ModeSpec, 0, len(cycle))
	added := make(map[string]struct{})
	for _, entry := range cycle {
		key := strings.ToLower(entry)
		spec, ok := available[key]
		if !ok {
			continue
		}
		if _, seen := added[spec.Name]; seen {
			continue
		}
		modes = append(modes, spec)
		added[spec.Name] = struct{}{}
	}
	if len(modes) == 0 {
		return nil, fmt.Errorf("no input modes available after applying toggle configuration")
	}
	return modes, nil
}
