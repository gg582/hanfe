package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	ini "github.com/go-ini/ini"
)

type Config struct {
	ToggleKey string
	Layout    string
}

const (
	defaultToggle = "ctrl+space"
	defaultLayout = "dubeolsik"
)

func Load(path string) (Config, error) {
	cfg := Config{ToggleKey: defaultToggle, Layout: defaultLayout}

	if path == "" {
		return cfg, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("config: %w", err)
	}
	if info.IsDir() {
		return cfg, fmt.Errorf("config: %s is a directory", path)
	}

	file, err := ini.Load(filepath.Clean(path))
	if err != nil {
		return cfg, fmt.Errorf("config: %w", err)
	}

	toggleKey := file.Section("toggle").Key("key").MustString(cfg.ToggleKey)
	layout := file.Section("layout").Key("name").MustString(cfg.Layout)

	cfg.ToggleKey = toggleKey
	cfg.Layout = layout
	return cfg, nil
}
