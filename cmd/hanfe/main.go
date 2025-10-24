package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"hanfe/internal/backend"
	"hanfe/internal/cli"
	"hanfe/internal/config"
	"hanfe/internal/device"
	"hanfe/internal/emitter"
	"hanfe/internal/engine"
	"hanfe/internal/layout"
	"hanfe/internal/ttybridge"
	"hanfe/internal/types"
)

const daemonEnv = "HANFE_DAEMONIZED"

func main() {
	if ttybridge.InHelperMode() {
		if err := ttybridge.RunHelper(); err != nil {
			fmt.Fprintf(os.Stderr, "hanfe tty helper: %v\n", err)
			os.Exit(1)
		}
		return
	}

	opts, err := cli.Parse(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}

	if opts.ShowHelp {
		fmt.Println(cli.Usage())
		return
	}

	if opts.ListLayouts {
		for _, name := range layout.AvailableLayouts() {
			fmt.Println(name)
		}
		return
	}

	if opts.TTYPath == "" {
		detected, err := ttybridge.DetectTTYPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
			os.Exit(1)
		}
		opts.TTYPath = detected
	}

	if err := ttybridge.SpawnHelper(opts.TTYPath); err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}

	spawned, err := daemonizeIfNeeded(opts.Daemonize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: failed to daemonize: %v\n", err)
		os.Exit(1)
	}
	if spawned {
		return
	}

	ttyClient, err := ttybridge.Attach()
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}
	if opts.TTYPath != "" && ttyClient == nil {
		fmt.Fprintf(os.Stderr, "hanfe: failed to connect to tty helper\n")
		os.Exit(1)
	}

	toggleCfg, err := config.ResolveToggleConfig(opts.ToggleConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}

	normalizeMode := func(name string) string {
		normalized := strings.ToLower(strings.TrimSpace(name))
		switch normalized {
		case "", "hangul":
			return "dubeolsik"
		case "default", "latin", "english":
			return "latin"
		default:
			return normalized
		}
	}

	modeOrder := toggleCfg.ModeCycle
	if len(opts.ModeOrder) > 0 {
		modeOrder = make([]string, 0, len(opts.ModeOrder))
		for _, name := range opts.ModeOrder {
			modeOrder = append(modeOrder, normalizeMode(name))
		}
	}
	if len(modeOrder) == 0 {
		modeOrder = []string{"dubeolsik", "latin"}
	}

	layoutOverride := normalizeMode(opts.LayoutName)
	if opts.LayoutName != "" {
		replaced := false
		for i, name := range modeOrder {
			if name == "dubeolsik" || strings.EqualFold(name, toggleCfg.DefaultMode) {
				modeOrder[i] = layoutOverride
				replaced = true
				break
			}
		}
		if !replaced {
			modeOrder = append([]string{layoutOverride}, modeOrder...)
		}
		toggleCfg.DefaultMode = layoutOverride
	}

	if toggleCfg.DefaultMode == "" && len(modeOrder) > 0 {
		toggleCfg.DefaultMode = modeOrder[0]
	}

	var customPairs []layout.CustomPair
	if opts.KeypairPath != "" {
		pairs, perr := layout.LoadCustomPairs(opts.KeypairPath)
		if perr != nil {
			fmt.Fprintf(os.Stderr, "hanfe: %v\n", perr)
			os.Exit(1)
		}
		customPairs = pairs
	}

	var pinyinDB backend.Database
	if opts.PinyinDBPath != "" {
		db, derr := backend.LoadDatabase(opts.PinyinDBPath)
		if derr != nil {
			fmt.Fprintf(os.Stderr, "hanfe: %v\n", derr)
			os.Exit(1)
		}
		pinyinDB = db
	}

	modeSpecs := make([]engine.ModeSpec, 0, len(modeOrder))
	for _, name := range modeOrder {
		normalized := normalizeMode(name)
		switch normalized {
		case "latin":
			modeSpecs = append(modeSpecs, engine.ModeSpec{Name: "latin", Kind: types.ModeLatin})
		case "pinyin":
			if !pinyinDB.Available() {
				fmt.Fprintf(os.Stderr, "hanfe: pinyin mode requires --pinyin-db database file\n")
				os.Exit(1)
			}
			modeSpecs = append(modeSpecs, engine.ModeSpec{Name: "pinyin", Kind: types.ModeDatabase, Database: pinyinDB})
		default:
			keyLayout, lerr := layout.Load(normalized)
			if lerr != nil {
				fmt.Fprintf(os.Stderr, "hanfe: %v\n", lerr)
				os.Exit(1)
			}
			if len(customPairs) > 0 {
				keyLayout, lerr = layout.ApplyCustomPairs(keyLayout, customPairs)
				if lerr != nil {
					fmt.Fprintf(os.Stderr, "hanfe: %v\n", lerr)
					os.Exit(1)
				}
			}
			layoutCopy := keyLayout
			kind := types.ModeHangul
			if keyLayout.Category() == layout.CategoryKana {
				kind = types.ModeKana
			}
			modeSpecs = append(modeSpecs, engine.ModeSpec{Name: normalized, Kind: kind, Layout: &layoutCopy})
		}
	}

	if len(modeSpecs) == 0 {
		fmt.Fprintf(os.Stderr, "hanfe: no input modes configured\n")
		os.Exit(1)
	}

	defaultFound := false
	for _, spec := range modeSpecs {
		if strings.EqualFold(spec.Name, toggleCfg.DefaultMode) {
			defaultFound = true
			break
		}
	}
	if !defaultFound {
		toggleCfg.DefaultMode = modeSpecs[0].Name
	}

	devicePath := opts.DevicePath
	if devicePath == "" {
		detected, derr := device.DetectKeyboardDevice()
		if derr != nil {
			var detectionErr device.DetectionError
			if errors.As(derr, &detectionErr) {
				fmt.Fprintf(os.Stderr, "hanfe: %s\n", detectionErr.Message)
			} else {
				fmt.Fprintf(os.Stderr, "hanfe: failed to detect keyboard: %v\n", derr)
			}
			os.Exit(1)
		}
		devicePath = detected.Path
	}

	fd, err := syscall.Open(devicePath, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: open %s: %v\n", devicePath, err)
		os.Exit(1)
	}
	defer syscall.Close(fd)

	directCommit := opts.SuppressHex
	if !directCommit && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		directCommit = true
	}
	if directCommit && ttyClient == nil && opts.PTYPath == "" {
		fmt.Fprintf(os.Stderr, "hanfe: cannot disable unicode hex without tty or pty mirror; falling back to hex\n")
		directCommit = false
	}

	fallback, err := emitter.Open(layout.UnicodeHexKeycodes(), ttyClient, opts.PTYPath, directCommit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}

	eng, err := engine.NewEngine(fd, modeSpecs, toggleCfg, fallback)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}
	if err := eng.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}
}

func daemonizeIfNeeded(enabled bool) (bool, error) {
	if !enabled {
		return false, nil
	}
	if os.Getenv(daemonEnv) == "1" {
		return false, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return false, err
	}

	devNull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if err != nil {
		return false, err
	}
	defer devNull.Close()

	attrs := &os.ProcAttr{
		Files: []*os.File{devNull, devNull, devNull},
		Env:   append(os.Environ(), daemonEnv+"=1"),
		Sys:   &syscall.SysProcAttr{Setsid: true},
	}

	proc, err := os.StartProcess(exe, os.Args, attrs)
	if err != nil {
		return false, err
	}
	if err := proc.Release(); err != nil {
		return false, err
	}
	return true, nil
}
