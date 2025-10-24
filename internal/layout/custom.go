package layout

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"hanfe/internal/hangul"
	"hanfe/internal/linux"
)

type CustomPair struct {
	Key          string `json:"key"`
	Kind         string `json:"kind"`
	Normal       string `json:"normal"`
	Shifted      string `json:"shifted"`
	Role         string `json:"role"`
	CommitBefore bool   `json:"commit_before"`
}

func LoadCustomPairs(path string) ([]CustomPair, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open custom keypair file: %w", err)
	}
	defer file.Close()

	var pairs []CustomPair
	if err := json.NewDecoder(file).Decode(&pairs); err != nil {
		return nil, fmt.Errorf("parse custom keypair file: %w", err)
	}
	return pairs, nil
}

func ApplyCustomPairs(l Layout, pairs []CustomPair) (Layout, error) {
	for _, pair := range pairs {
		code, err := resolveKeyCode(pair.Key)
		if err != nil {
			return Layout{}, err
		}
		entry := l.mapping[code]

		switch strings.ToLower(pair.Kind) {
		case "passthrough":
			symbol := makePassthroughSymbol(pair.CommitBefore)
			entry.Normal = symbol
			entry.Shifted = symbol
		case "text":
			normal := makeTextSymbol(pair.Normal, pair.CommitBefore)
			entry.Normal = normal
			if pair.Shifted != "" {
				entry.Shifted = makeTextSymbol(pair.Shifted, pair.CommitBefore)
			} else if entry.Shifted != nil {
				entry.Shifted = makeTextSymbol(pair.Normal, pair.CommitBefore)
			}
		case "jamo":
			normal, shifted, err := makeJamoPair(pair.Normal, pair.Shifted, pair.Role)
			if err != nil {
				return Layout{}, err
			}
			entry.Normal = normal
			entry.Shifted = shifted
		default:
			return Layout{}, fmt.Errorf("unsupported custom keypair kind '%s'", pair.Kind)
		}

		l.mapping[code] = entry
	}
	return l, nil
}

func makeJamoPair(normal, shifted, role string) (*LayoutSymbol, *LayoutSymbol, error) {
	makeSymbol := func(value string) (*LayoutSymbol, error) {
		if value == "" {
			return nil, nil
		}
		r := []rune(value)
		if len(r) != 1 {
			return nil, fmt.Errorf("jamo value must be a single rune, got %q", value)
		}
		return makeJamoSymbol(r[0], parseRole(role)), nil
	}

	normalSymbol, err := makeSymbol(normal)
	if err != nil {
		return nil, nil, err
	}
	shiftedSymbol, err := makeSymbol(shifted)
	if err != nil {
		return nil, nil, err
	}
	return normalSymbol, shiftedSymbol, nil
}

func parseRole(role string) hangul.JamoRole {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "leading":
		return hangul.RoleLeading
	case "trailing":
		return hangul.RoleTrailing
	default:
		return hangul.RoleAuto
	}
}

func resolveKeyCode(name string) (uint16, error) {
	normalized := strings.ToUpper(strings.TrimSpace(name))
	if normalized == "" {
		return 0, fmt.Errorf("empty key name")
	}

	aliases := map[string]int{
		"KEY_A":          linux.KeyA,
		"KEY_B":          linux.KeyB,
		"KEY_C":          linux.KeyC,
		"KEY_D":          linux.KeyD,
		"KEY_E":          linux.KeyE,
		"KEY_F":          linux.KeyF,
		"KEY_G":          linux.KeyG,
		"KEY_H":          linux.KeyH,
		"KEY_I":          linux.KeyI,
		"KEY_J":          linux.KeyJ,
		"KEY_K":          linux.KeyK,
		"KEY_L":          linux.KeyL,
		"KEY_M":          linux.KeyM,
		"KEY_N":          linux.KeyN,
		"KEY_O":          linux.KeyO,
		"KEY_P":          linux.KeyP,
		"KEY_Q":          linux.KeyQ,
		"KEY_R":          linux.KeyR,
		"KEY_S":          linux.KeyS,
		"KEY_T":          linux.KeyT,
		"KEY_U":          linux.KeyU,
		"KEY_V":          linux.KeyV,
		"KEY_W":          linux.KeyW,
		"KEY_X":          linux.KeyX,
		"KEY_Y":          linux.KeyY,
		"KEY_Z":          linux.KeyZ,
		"KEY_1":          linux.Key1,
		"KEY_2":          linux.Key2,
		"KEY_3":          linux.Key3,
		"KEY_4":          linux.Key4,
		"KEY_5":          linux.Key5,
		"KEY_6":          linux.Key6,
		"KEY_7":          linux.Key7,
		"KEY_8":          linux.Key8,
		"KEY_9":          linux.Key9,
		"KEY_0":          linux.Key0,
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
	}

	if !strings.HasPrefix(normalized, "KEY_") {
		if len(normalized) == 1 {
			ch := normalized[0]
			if ch >= 'A' && ch <= 'Z' {
				normalized = fmt.Sprintf("KEY_%c", ch)
			} else if ch >= '0' && ch <= '9' {
				normalized = fmt.Sprintf("KEY_%c", ch)
			}
		}
	}

	if code, ok := aliases[normalized]; ok {
		return uint16(code), nil
	}

	return 0, fmt.Errorf("unknown key name '%s'", name)
}
