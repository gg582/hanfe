package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	input "github.com/eiannone/keyboard"
	hangulkeyboard "github.com/suapapa/go_hangul/keyboard"

	"hanfe/pkg/config"
	"hanfe/pkg/ime"
)

type toggleSpec struct {
	key  input.Key
	rune rune
}

func (t toggleSpec) matches(r rune, k input.Key) bool {
	if k != t.key {
		return false
	}
	if k == input.KeyRune && t.rune != 0 {
		return r == t.rune
	}
	return true
}

func parseToggleSpec(value string) toggleSpec {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "tab":
		return toggleSpec{key: input.KeyTab}
	case "space":
		return toggleSpec{key: input.KeySpace}
	case "ctrl+space":
		return toggleSpec{key: input.KeyCtrlSpace}
	case "enter":
		return toggleSpec{key: input.KeyEnter}
	}

	runes := []rune(v)
	if len(runes) == 1 {
		return toggleSpec{key: input.KeyRune, rune: runes[0]}
	}
	return toggleSpec{key: input.KeyCtrlSpace}
}

func main() {
	cfgPath := flag.String("config", "toggle.ini", "Path to configuration file")
	layoutName := flag.String("layout", "", "Keyboard layout to use")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if *layoutName != "" {
		cfg.Layout = *layoutName
	}

	layout, ok := hangulkeyboard.ByName(cfg.Layout)
	if !ok {
		names := make([]string, 0, len(hangulkeyboard.Available()))
		for _, l := range hangulkeyboard.Available() {
			names = append(names, l.Name)
		}
		fmt.Fprintf(os.Stderr, "unknown layout %q. available: %s\n", cfg.Layout, strings.Join(names, ", "))
		os.Exit(1)
	}

	toggle := parseToggleSpec(cfg.ToggleKey)

	if err := input.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to open keyboard: %v\n", err)
		os.Exit(1)
	}
	defer input.Close()

	composer := ime.NewComposer(layout)
	hangulMode := true

	fmt.Printf("hanfe ready (layout=%s, toggle=%s)\n", layout.Name, cfg.ToggleKey)
	fmt.Println("Press Ctrl+C or Esc to exit.")
	redraw := func() {
		status := "[ENG]"
		if hangulMode {
			status = "[KOR]"
		}
		text := composer.Text()
		fmt.Printf("\r%s %s", status, text)
		fmt.Printf("\x1b[K")
		os.Stdout.Sync()
	}

	redraw()

	for {
		r, key, err := input.GetSingleKey()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println()
				return
			}
			fmt.Fprintf(os.Stderr, "keyboard read error: %v\n", err)
			fmt.Println()
			return
		}

		if key == input.KeyCtrlC || key == input.KeyEsc {
			fmt.Println()
			return
		}

		if toggle.matches(r, key) {
			composer.FlushText()
			hangulMode = !hangulMode
			redraw()
			continue
		}

		switch key {
		case input.KeyBackspace:
			composer.Backspace()
		case input.KeyEnter:
			line := composer.FlushText()
			status := "[ENG]"
			if hangulMode {
				status = "[KOR]"
			}
			fmt.Printf("\r%s %s\n", status, line)
			composer.Reset()
		case input.KeySpace:
			composer.Space()
		case input.KeyTab:
			composer.AppendLiteral('\t')
		case input.KeyRune:
			if hangulMode {
				if !composer.TypeKey(r) {
					composer.AppendLiteral(r)
				}
			} else {
				composer.AppendLiteral(r)
			}
		case input.KeyUnknown:
			continue
		default:
			continue
		}

		redraw()
	}
}
