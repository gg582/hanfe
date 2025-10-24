package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"hanfe/internal/types"
)

type PairKind int

const (
	PairText PairKind = iota
	PairJamo
)

type PairOverride struct {
	Key   uint16
	Shift bool
	Kind  PairKind
	Value string
}

type ProfileSpec struct {
	Name         string
	Layout       string
	Mode         types.InputMode
	DatabasePath string
	Overrides    []PairOverride
}

type ProfilesConfig struct {
	Profiles []ProfileSpec
}

func DefaultProfilesConfig() ProfilesConfig {
	return ProfilesConfig{Profiles: []ProfileSpec{
		{Name: "default", Layout: "latin", Mode: types.ModeLatin},
		{Name: "dubeolsik", Layout: "dubeolsik", Mode: types.ModeHangul},
		{Name: "sebeolsik", Layout: "sebeolsik-390", Mode: types.ModeHangul},
	}}
}

func ResolveProfilesConfig(cliPath string) (ProfilesConfig, error) {
	if cliPath != "" {
		return LoadProfilesConfig(cliPath)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return DefaultProfilesConfig(), nil
	}
	defaultPath := filepath.Join(cwd, "profiles.ini")
	if _, err := os.Stat(defaultPath); err == nil {
		return LoadProfilesConfig(defaultPath)
	} else if errors.Is(err, os.ErrNotExist) {
		return DefaultProfilesConfig(), nil
	}
	return DefaultProfilesConfig(), nil
}

func LoadProfilesConfig(path string) (ProfilesConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return ProfilesConfig{}, ConfigError{msg: fmt.Sprintf("failed to open profiles config: %v", err)}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	order := []string{}
	specs := make(map[string]*ProfileSpec)
	current := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.TrimSpace(line[1 : len(line)-1])
			if strings.EqualFold(section, "profiles") {
				current = ""
			} else if strings.HasPrefix(strings.ToLower(section), "profile ") {
				name := strings.TrimSpace(section[len("profile "):])
				current = strings.ToLower(name)
				if _, ok := specs[current]; !ok {
					specs[current] = &ProfileSpec{Name: name}
				}
			} else {
				current = ""
			}
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return ProfilesConfig{}, ConfigError{msg: fmt.Sprintf("invalid line in %s: %s", path, line)}
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if current == "" {
			if strings.EqualFold(key, "order") {
				tokens := splitComma(value)
				if len(tokens) == 0 {
					return ProfilesConfig{}, ConfigError{msg: fmt.Sprintf("empty order in %s", path)}
				}
				order = tokens
			}
			continue
		}

		spec := specs[current]
		switch strings.ToLower(key) {
		case "layout":
			spec.Layout = value
		case "mode":
			mode, err := parseMode(value)
			if err != nil {
				return ProfilesConfig{}, err
			}
			spec.Mode = mode
		case "database":
			spec.DatabasePath = value
		case "pair":
			override, err := parsePairOverride(value)
			if err != nil {
				return ProfilesConfig{}, err
			}
			spec.Overrides = append(spec.Overrides, override)
		default:
			return ProfilesConfig{}, ConfigError{msg: fmt.Sprintf("unknown key '%s' in profile %s", key, spec.Name)}
		}
	}

	if err := scanner.Err(); err != nil {
		return ProfilesConfig{}, ConfigError{msg: fmt.Sprintf("failed to read %s: %v", path, err)}
	}

	if len(order) == 0 {
		order = make([]string, 0, len(specs))
		for name := range specs {
			order = append(order, name)
		}
	}

	profs := make([]ProfileSpec, 0, len(order))
	for _, name := range order {
		normalized := strings.ToLower(strings.TrimSpace(name))
		spec, ok := specs[normalized]
		if !ok {
			return ProfilesConfig{}, ConfigError{msg: fmt.Sprintf("profile '%s' referenced in order but not defined", name)}
		}
		finalized := *spec
		if finalized.Mode == 0 && strings.ToLower(finalized.Name) != "default" {
			finalized.Mode = types.ModeHangul
		}
		if finalized.Layout == "" {
			switch finalized.Mode {
			case types.ModeLatin:
				finalized.Layout = "latin"
			case types.ModeKana:
				finalized.Layout = "kana-86"
			default:
				finalized.Layout = "dubeolsik"
			}
		}
		profs = append(profs, finalized)
	}

	if len(profs) == 0 {
		return ProfilesConfig{}, ConfigError{msg: fmt.Sprintf("no profiles defined in %s", path)}
	}

	return ProfilesConfig{Profiles: profs}, nil
}

func parseMode(value string) (types.InputMode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "hangul":
		return types.ModeHangul, nil
	case "latin", "default":
		return types.ModeLatin, nil
	case "kana":
		return types.ModeKana, nil
	case "database":
		return types.ModeDatabase, nil
	default:
		return 0, ConfigError{msg: fmt.Sprintf("unknown input mode '%s'", value)}
	}
}

func parsePairOverride(value string) (PairOverride, error) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return PairOverride{}, ConfigError{msg: fmt.Sprintf("invalid pair override '%s'", value)}
	}
	lhs := strings.TrimSpace(parts[0])
	rhs := strings.TrimSpace(parts[1])
	if lhs == "" || rhs == "" {
		return PairOverride{}, ConfigError{msg: fmt.Sprintf("invalid pair override '%s'", value)}
	}

	tokens := strings.Split(lhs, "+")
	shift := false
	keyName := ""
	for _, token := range tokens {
		trimmed := strings.TrimSpace(token)
		if trimmed == "" {
			continue
		}
		if strings.EqualFold(trimmed, "shift") {
			shift = true
			continue
		}
		if keyName != "" {
			return PairOverride{}, ConfigError{msg: fmt.Sprintf("invalid pair lhs '%s'", lhs)}
		}
		keyName = trimmed
	}
	if keyName == "" {
		return PairOverride{}, ConfigError{msg: fmt.Sprintf("invalid pair lhs '%s'", lhs)}
	}

	code, err := ParseKeyName(keyName)
	if err != nil {
		return PairOverride{}, err
	}

	kind := PairText
	payload := rhs
	rhsParts := strings.SplitN(rhs, ":", 2)
	if len(rhsParts) == 2 {
		tag := strings.ToLower(strings.TrimSpace(rhsParts[0]))
		payload = strings.TrimSpace(rhsParts[1])
		switch tag {
		case "text":
			kind = PairText
		case "jamo":
			kind = PairJamo
		default:
			return PairOverride{}, ConfigError{msg: fmt.Sprintf("unknown pair kind '%s'", tag)}
		}
	}
	if payload == "" {
		return PairOverride{}, ConfigError{msg: fmt.Sprintf("empty payload in pair '%s'", value)}
	}

	return PairOverride{Key: code, Shift: shift, Kind: kind, Value: payload}, nil
}
