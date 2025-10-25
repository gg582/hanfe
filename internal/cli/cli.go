package cli

import (
	"fmt"
	"strings"
)

type Options struct {
	ShowHelp         bool
	ListLayouts      bool
	DevicePath       string
	LayoutName       string
	ToggleConfigPath string
	SocketPath       string
	TTYPath          string
	PTYPath          string
	Daemonize        bool
	SuppressHex      bool
	ModeOrder        []string
	KeypairPath      string
	PinyinDBPath     string
}

func Parse(args []string) (Options, error) {
	opts := Options{Daemonize: true}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			opts.ShowHelp = true
		case arg == "--list-layouts":
			opts.ListLayouts = true
		case arg == "--daemon":
			opts.Daemonize = true
		case arg == "--no-daemon" || arg == "--foreground":
			opts.Daemonize = false
		case strings.HasPrefix(arg, "--socket"):
			value, next, err := extractValue(arg, i, args)
			if err != nil {
				return Options{}, err
			}
			opts.SocketPath = value
			i = next
		case strings.HasPrefix(arg, "--device"):
			value, next, err := extractValue(arg, i, args)
			if err != nil {
				return Options{}, err
			}
			opts.DevicePath = value
			i = next
		case strings.HasPrefix(arg, "--layout"):
			value, next, err := extractValue(arg, i, args)
			if err != nil {
				return Options{}, err
			}
			opts.LayoutName = value
			i = next
		case strings.HasPrefix(arg, "--mode-order"):
			value, next, err := extractValue(arg, i, args)
			if err != nil {
				return Options{}, err
			}
			opts.ModeOrder = splitList(value)
			i = next
		case strings.HasPrefix(arg, "--toggle-config"):
			value, next, err := extractValue(arg, i, args)
			if err != nil {
				return Options{}, err
			}
			opts.ToggleConfigPath = value
			i = next
		case strings.HasPrefix(arg, "--keypairs"):
			value, next, err := extractValue(arg, i, args)
			if err != nil {
				return Options{}, err
			}
			opts.KeypairPath = value
			i = next
		case strings.HasPrefix(arg, "--pinyin-db"):
			value, next, err := extractValue(arg, i, args)
			if err != nil {
				return Options{}, err
			}
			opts.PinyinDBPath = value
			i = next
		case strings.HasPrefix(arg, "--tty"):
			value, next, err := extractValue(arg, i, args)
			if err != nil {
				return Options{}, err
			}
			opts.TTYPath = value
			i = next
		case strings.HasPrefix(arg, "--pty"):
			value, next, err := extractValue(arg, i, args)
			if err != nil {
				return Options{}, err
			}
			opts.PTYPath = value
			i = next
		case arg == "--no-hex" || arg == "--direct-tty":
			opts.SuppressHex = true
		default:
			return Options{}, fmt.Errorf("unknown option: %s", arg)
		}
	}
	return opts, nil
}

func extractValue(current string, index int, args []string) (string, int, error) {
	if eq := strings.IndexRune(current, '='); eq >= 0 {
		return current[eq+1:], index, nil
	}
	if index+1 >= len(args) {
		return "", index, fmt.Errorf("option %s requires a value", current)
	}
	return args[index+1], index + 1, nil
}

func splitList(value string) []string {
	if value == "" {
		return nil
	}
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

func Usage() string {
	return `hanfe - Hangul IME interceptor
Usage: hanfe [--device /dev/input/eventX] [options]

Options:
  --device PATH           Path to the evdev keyboard device (auto-detected if omitted)
  --layout NAME           Keyboard layout (default: dubeolsik)
  --socket PATH           Path to the translation unix socket (default: $XDG_RUNTIME_DIR/hanfe.sock)
  --mode-order LIST       Comma-separated input mode cycle (overrides toggle.ini)
  --toggle-config PATH    Path to toggle.ini (default: ./toggle.ini if present)
  --keypairs PATH         JSON file describing custom keypairs to merge into the layout
  --pinyin-db PATH        JSON database for database-backed input (e.g. Pinyin)
  --tty PATH              TTY to mirror text output to (defaults to controlling TTY)
  --pty PATH              Optional PTY to mirror committed text without raw hex
  --no-hex                Skip Unicode hex injection and rely on direct TTY/PTY mirroring
  --daemon                Run in the background (default)
  --no-daemon             Stay in the foreground
  --list-layouts          List available layouts
  -h, --help              Show this help message`
}
