package app

import (
	"fmt"
	"strings"

	"github.com/gg582/hanfe/internal/common"
	"github.com/gg582/hanfe/internal/layout"
	"github.com/gg582/hangul-logotype/hangul"
)

func ResolveTranslatorLayout(name string) (hangul.KeyboardLayout, string, error) {
	layout, canonical, err := common.ResolveLayout(name)
	if err != nil {
		trimmed := strings.ToLower(strings.TrimSpace(name))
		if trimmed == "" {
			return nil, "", err
		}
		return nil, trimmed, nil
	}
	return layout, canonical, nil
}

func ResolveEngineLayout(canonical, raw, keypairPath string) (*layout.Layout, string, error) {
	normalized := strings.ToLower(strings.TrimSpace(canonical))
	if normalized == "" {
		normalized = strings.ToLower(strings.TrimSpace(raw))
	}
	switch normalized {
	case "", "dubeolsik", "hangul", "korean":
		normalized = "dubeolsik"
	case "sebulshik-final", "sebulshik", "sebulsik", "sebeolsik-final":
		normalized = "sebeolsik-390"
	case "latin", "raw", "none":
		return nil, "", nil
	}
	loaded, err := layout.Load(normalized)
	if err != nil {
		if canonical != "" && canonical != normalized {
			loaded, err = layout.Load(strings.ToLower(strings.TrimSpace(canonical)))
			if err != nil {
				return nil, "", fmt.Errorf("load layout %s: %w", canonical, err)
			}
		} else {
			return nil, "", fmt.Errorf("load layout %s: %w", normalized, err)
		}
	}
	if keypairPath != "" {
		pairs, err := layout.LoadCustomPairs(keypairPath)
		if err != nil {
			return nil, "", err
		}
		loaded, err = layout.ApplyCustomPairs(loaded, pairs)
		if err != nil {
			return nil, "", err
		}
	}
	copy := loaded
	return &copy, copy.Name(), nil
}
