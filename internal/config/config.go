package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"hanfe/internal/linux"
	"hanfe/internal/types"
)

type ToggleConfig struct {
	ToggleKeys  []uint16
	DefaultMode types.InputMode
}

type ConfigError struct {
	msg string
}

func (e ConfigError) Error() string { return e.msg }

func DefaultToggleConfig() ToggleConfig {
	return ToggleConfig{
		ToggleKeys:  []uint16{uint16(linux.KeyRightAlt), uint16(linux.KeyHangeul)},
		DefaultMode: types.ModeHangul,
	}
}

func LoadToggleConfig(path string) (ToggleConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return ToggleConfig{}, ConfigError{msg: fmt.Sprintf("failed to open toggle config: %v", err)}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inToggle := false
	var keysLine string
	var modeLine string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.TrimSpace(line[1 : len(line)-1])
			inToggle = strings.EqualFold(section, "toggle")
			continue
		}
		if !inToggle {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return ToggleConfig{}, ConfigError{msg: fmt.Sprintf("invalid line in %s: %s", path, line)}
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "keys":
			keysLine = value
		case "default_mode":
			modeLine = value
		}
	}

	if err := scanner.Err(); err != nil {
		return ToggleConfig{}, ConfigError{msg: fmt.Sprintf("failed to read %s: %v", path, err)}
	}

	if strings.TrimSpace(keysLine) == "" {
		return ToggleConfig{}, ConfigError{msg: fmt.Sprintf("no toggle keys defined in %s", path)}
	}

	tokens := splitComma(keysLine)
	if len(tokens) == 0 {
		return ToggleConfig{}, ConfigError{msg: fmt.Sprintf("no toggle keys defined in %s", path)}
	}

	cfg := ToggleConfig{}
	for _, token := range tokens {
		code, err := parseKeycode(token)
		if err != nil {
			return ToggleConfig{}, err
		}
		cfg.ToggleKeys = append(cfg.ToggleKeys, code)
	}

	if modeLine != "" {
		switch strings.ToLower(strings.TrimSpace(modeLine)) {
		case "hangul":
			cfg.DefaultMode = types.ModeHangul
		case "latin":
			cfg.DefaultMode = types.ModeLatin
		default:
			return ToggleConfig{}, ConfigError{msg: fmt.Sprintf("invalid default_mode '%s' in %s", modeLine, path)}
		}
	} else {
		cfg.DefaultMode = types.ModeHangul
	}

	return cfg, nil
}

func splitComma(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseKeycode(name string) (uint16, error) {
	normalized := strings.ToUpper(strings.TrimSpace(name))
	if normalized == "" {
		return 0, ConfigError{msg: "empty key name"}
	}

	aliases := map[string]string{
		"ALT_R":   "KEY_RIGHTALT",
		"ALT_L":   "KEY_LEFTALT",
		"CTRL_L":  "KEY_LEFTCTRL",
		"CTRL_R":  "KEY_RIGHTCTRL",
		"SHIFT_L": "KEY_LEFTSHIFT",
		"SHIFT_R": "KEY_RIGHTSHIFT",
		"HANGUL":  "KEY_HANGUL",
		"HANGEUL": "KEY_HANGEUL",
	}
	if alias, ok := aliases[normalized]; ok {
		normalized = alias
	}

	if !strings.HasPrefix(normalized, "KEY_") {
		normalized = "KEY_" + normalized
	}

	code, ok := keycodeTable()[normalized]
	if !ok {
		return 0, ConfigError{msg: fmt.Sprintf("unknown key code '%s'", name)}
	}
	return code, nil
}

func keycodeTable() map[string]uint16 {
	table := map[string]uint16{}
	for ch := 'A'; ch <= 'Z'; ch++ {
		table[fmt.Sprintf("KEY_%c", ch)] = uint16(linux.KeyA + int(ch-'A'))
	}
	for ch := '0'; ch <= '9'; ch++ {
		table[fmt.Sprintf("KEY_%c", ch)] = uint16(linux.Key0 + int(ch-'0'))
	}

	additional := map[string]int{
		"KEY_MINUS":      linux.KeyMinus,
		"KEY_EQUAL":      linux.KeyEqual,
		"KEY_LEFTBRACE":  linux.KeyLeftBrace,
		"KEY_RIGHTBRACE": linux.KeyRightBrace,
		"KEY_BACKSLASH":  linux.KeyBackslash,
		"KEY_SEMICOLON":  linux.KeySemicolon,
		"KEY_APOSTROPHE": linux.KeyApostrophe,
		"KEY_GRAVE":      linux.KeyGrave,
		"KEY_COMMA":      linux.KeyComma,
		"KEY_DOT":        linux.KeyDot,
		"KEY_SLASH":      linux.KeySlash,
		"KEY_SPACE":      linux.KeySpace,
		"KEY_TAB":        linux.KeyTab,
		"KEY_ENTER":      linux.KeyEnter,
		"KEY_ESC":        linux.KeyEsc,
		"KEY_BACKSPACE":  linux.KeyBackspace,
		"KEY_LEFTSHIFT":  linux.KeyLeftShift,
		"KEY_RIGHTSHIFT": linux.KeyRightShift,
		"KEY_LEFTCTRL":   linux.KeyLeftCtrl,
		"KEY_RIGHTCTRL":  linux.KeyRightCtrl,
		"KEY_LEFTALT":    linux.KeyLeftAlt,
		"KEY_RIGHTALT":   linux.KeyRightAlt,
		"KEY_LEFTMETA":   linux.KeyLeftMeta,
		"KEY_RIGHTMETA":  linux.KeyRightMeta,
		"KEY_HANGUL":     linux.KeyHangul,
		"KEY_HANGEUL":    linux.KeyHangeul,
		"KEY_HANJA":      linux.KeyHanja,
		"KEY_CAPSLOCK":   linux.KeyCapsLock,
		"KEY_F1":         linux.KeyF1,
		"KEY_F2":         linux.KeyF2,
		"KEY_F3":         linux.KeyF3,
		"KEY_F4":         linux.KeyF4,
		"KEY_F5":         linux.KeyF5,
		"KEY_F6":         linux.KeyF6,
		"KEY_F7":         linux.KeyF7,
		"KEY_F8":         linux.KeyF8,
		"KEY_F9":         linux.KeyF9,
		"KEY_F10":        linux.KeyF10,
		"KEY_F11":        linux.KeyF11,
		"KEY_F12":        linux.KeyF12,
	}
	for name, code := range additional {
		table[name] = uint16(code)
	}
	return table
}

func ResolveToggleConfig(cliPath string) (ToggleConfig, error) {
	if cliPath != "" {
		return LoadToggleConfig(cliPath)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return DefaultToggleConfig(), nil
	}
	defaultPath := cwd + string(os.PathSeparator) + "toggle.ini"
	if _, statErr := os.Stat(defaultPath); statErr == nil {
		return LoadToggleConfig(defaultPath)
	} else if errors.Is(statErr, os.ErrNotExist) {
		return DefaultToggleConfig(), nil
	}
	return DefaultToggleConfig(), nil
}
