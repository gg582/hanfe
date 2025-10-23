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
	TTYPath          string
	Daemonize        bool
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
		case strings.HasPrefix(arg, "--toggle-config"):
			value, next, err := extractValue(arg, i, args)
			if err != nil {
				return Options{}, err
			}
			opts.ToggleConfigPath = value
			i = next
		case strings.HasPrefix(arg, "--tty"):
			value, next, err := extractValue(arg, i, args)
			if err != nil {
				return Options{}, err
			}
			opts.TTYPath = value
			i = next
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

func Usage() string {
	return `hanfe - Hangul IME interceptor
Usage: hanfe [--device /dev/input/eventX] [options]

Options:
  --device PATH           Path to the evdev keyboard device (auto-detected if omitted)
  --layout NAME           Keyboard layout (default: dubeolsik)
  --toggle-config PATH    Path to toggle.ini (default: ./toggle.ini if present)
  --tty PATH              Optional TTY to mirror text output to
  --daemon                Run in the background (default)
  --no-daemon             Stay in the foreground
  --list-layouts          List available layouts
  -h, --help              Show this help message`
}
