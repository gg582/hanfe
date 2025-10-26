package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gg582/hanfe/internal/linux"
)

type ToggleChord struct {
	Key            uint16
	ModifierGroups [][]uint16
}

type ToggleConfig struct {
	Chords      []ToggleChord
	DefaultMode string
	ModeCycle   []string
}

type ConfigError struct {
	msg string
}

func (e ConfigError) Error() string { return e.msg }

func DefaultToggleConfig() ToggleConfig {
	return ToggleConfig{
		Chords: []ToggleChord{
			{Key: uint16(linux.KeyRightAlt)},
			{Key: uint16(linux.KeyHangeul)},
		},
		DefaultMode: "dubeolsik",
		ModeCycle:   []string{"dubeolsik", "latin"},
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
	var keyLine string
	var keysLine string
	var modeLine string
	var cycleLine string

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
		case "key":
			keyLine = value
		case "keys":
			keysLine = value
		case "default_mode":
			modeLine = value
		case "mode_cycle":
			cycleLine = value
		}
	}

	if err := scanner.Err(); err != nil {
		return ToggleConfig{}, ConfigError{msg: fmt.Sprintf("failed to read %s: %v", path, err)}
	}

	tokens := splitComma(keysLine)
	if trimmed := strings.TrimSpace(keyLine); trimmed != "" {
		tokens = append(tokens, trimmed)
	}

	if len(tokens) == 0 {
		return ToggleConfig{}, ConfigError{msg: fmt.Sprintf("no toggle keys defined in %s", path)}
	}

	chords := make([]ToggleChord, 0, len(tokens))
	for _, token := range tokens {
		chord, err := parseToggleExpression(token)
		if err != nil {
			return ToggleConfig{}, err
		}
		chords = append(chords, chord)
	}

	cfg := ToggleConfig{Chords: chords, DefaultMode: "dubeolsik"}
	if modeLine != "" {
		cfg.DefaultMode = normalizeModeName(modeLine)
	}

	cycle := parseModeCycle(cycleLine)
	if len(cycle) == 0 {
		if cfg.DefaultMode == "latin" {
			cycle = []string{"latin", "dubeolsik"}
		} else {
			cycle = []string{cfg.DefaultMode, "latin"}
		}
	}

	if !containsMode(cycle, cfg.DefaultMode) {
		cycle = append([]string{cfg.DefaultMode}, cycle...)
	}
	cfg.ModeCycle = uniqueModes(cycle)

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

func parseToggleExpression(expr string) (ToggleChord, error) {
	segments := strings.Split(expr, "+")
	if len(segments) == 0 {
		return ToggleChord{}, ConfigError{msg: fmt.Sprintf("invalid toggle expression '%s'", expr)}
	}

	chord := ToggleChord{}
	for i, segment := range segments {
		codes, err := parseKeyToken(segment)
		if err != nil {
			return ToggleChord{}, err
		}
		if len(codes) == 0 {
			return ToggleChord{}, ConfigError{msg: fmt.Sprintf("invalid key token '%s'", segment)}
		}
		if i == len(segments)-1 {
			if len(codes) != 1 {
				return ToggleChord{}, ConfigError{msg: fmt.Sprintf("toggle trigger '%s' must resolve to a single key", segment)}
			}
			chord.Key = codes[0]
		} else {
			chord.ModifierGroups = append(chord.ModifierGroups, codes)
		}
	}
	return chord, nil
}

func parseKeyToken(name string) ([]uint16, error) {
	normalized := strings.ToUpper(strings.TrimSpace(name))
	if normalized == "" {
		return nil, ConfigError{msg: "empty key name"}
	}

	if codes, ok := modifierAlias()[normalized]; ok {
		return codes, nil
	}

	aliases := map[string]string{
		"ALT_R":      "KEY_RIGHTALT",
		"ALT_L":      "KEY_LEFTALT",
		"RIGHTALT":   "KEY_RIGHTALT",
		"LEFTALT":    "KEY_LEFTALT",
		"CTRL_R":     "KEY_RIGHTCTRL",
		"CTRL_L":     "KEY_LEFTCTRL",
		"CONTROL_R":  "KEY_RIGHTCTRL",
		"CONTROL_L":  "KEY_LEFTCTRL",
		"RIGHTCTRL":  "KEY_RIGHTCTRL",
		"LEFTCTRL":   "KEY_LEFTCTRL",
		"SHIFT_R":    "KEY_RIGHTSHIFT",
		"SHIFT_L":    "KEY_LEFTSHIFT",
		"RIGHTSHIFT": "KEY_RIGHTSHIFT",
		"LEFTSHIFT":  "KEY_LEFTSHIFT",
		"META_R":     "KEY_RIGHTMETA",
		"META_L":     "KEY_LEFTMETA",
		"HANGUL":     "KEY_HANGUL",
		"HANGEUL":    "KEY_HANGEUL",
	}
	if alias, ok := aliases[normalized]; ok {
		normalized = alias
	}

	if !strings.HasPrefix(normalized, "KEY_") {
		normalized = "KEY_" + normalized
	}

	table := keycodeTable()
	code, ok := table[normalized]
	if !ok {
		return nil, ConfigError{msg: fmt.Sprintf("unknown key code '%s'", name)}
	}
	return []uint16{code}, nil
}

func modifierAlias() map[string][]uint16 {
	return map[string][]uint16{
		"ALT":     {uint16(linux.KeyLeftAlt), uint16(linux.KeyRightAlt)},
		"CTRL":    {uint16(linux.KeyLeftCtrl), uint16(linux.KeyRightCtrl)},
		"CONTROL": {uint16(linux.KeyLeftCtrl), uint16(linux.KeyRightCtrl)},
		"SHIFT":   {uint16(linux.KeyLeftShift), uint16(linux.KeyRightShift)},
		"META":    {uint16(linux.KeyLeftMeta), uint16(linux.KeyRightMeta)},
	}
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

func normalizeModeName(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "hangul":
		return "dubeolsik"
	case "latin", "english", "default":
		return "latin"
	default:
		return normalized
	}
}

func parseModeCycle(line string) []string {
	tokens := splitComma(line)
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		normalized := normalizeModeName(token)
		if normalized == "" {
			continue
		}
		out = append(out, normalized)
	}
	return out
}

func containsMode(modes []string, target string) bool {
	for _, mode := range modes {
		if mode == target {
			return true
		}
	}
	return false
}

func uniqueModes(modes []string) []string {
	seen := make(map[string]struct{}, len(modes))
	out := make([]string, 0, len(modes))
	for _, mode := range modes {
		if _, ok := seen[mode]; ok {
			continue
		}
		seen[mode] = struct{}{}
		out = append(out, mode)
	}
	return out
}
