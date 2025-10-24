package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"hanfe/internal/cli"
	"hanfe/internal/config"
	"hanfe/internal/dbinput"
	"hanfe/internal/device"
	"hanfe/internal/emitter"
	"hanfe/internal/engine"
	"hanfe/internal/hangul"
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

	profilesCfg, err := config.ResolveProfilesConfig(opts.ProfileConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}

	engineProfiles := make([]engine.ProfileSpec, 0, len(profilesCfg.Profiles))
	for _, prof := range profilesCfg.Profiles {
		layoutName := prof.Layout
		if opts.LayoutName != "" && prof.Mode == types.ModeHangul {
			layoutName = opts.LayoutName
		}

		var keyLayout *layout.Layout
		effectiveName := layoutName
		if effectiveName == "" {
			effectiveName = prof.Name
		}
		if layoutName != "" {
			keyLayout, err = layout.Load(layoutName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
				os.Exit(1)
			}
		}

		if len(prof.Overrides) > 0 {
			if keyLayout == nil {
				keyLayout = layout.NewLayout(effectiveName)
			}
			for _, override := range prof.Overrides {
				var symbol *layout.LayoutSymbol
				switch override.Kind {
				case config.PairText:
					symbol = layout.NewTextSymbol(override.Value)
				case config.PairJamo:
					runes := []rune(override.Value)
					if len(runes) == 0 {
						fmt.Fprintf(os.Stderr, "hanfe: empty jamo override for %s\n", prof.Name)
						os.Exit(1)
					}
					symbol = layout.NewJamoSymbol(runes[0], hangul.RoleAuto)
				default:
					fmt.Fprintf(os.Stderr, "hanfe: unsupported pair override for %s\n", prof.Name)
					os.Exit(1)
				}
				keyLayout.ApplyOverride(override.Key, override.Shift, symbol)
			}
		}

		var dict *dbinput.Dictionary
		if prof.DatabasePath != "" {
			dict, err = dbinput.LoadDictionary(prof.DatabasePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
				os.Exit(1)
			}
		}

		engineProfiles = append(engineProfiles, engine.ProfileSpec{
			Name:       prof.Name,
			Mode:       prof.Mode,
			Layout:     keyLayout,
			Dictionary: dict,
		})
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

	eng := engine.NewEngine(fd, engineProfiles, toggleCfg, fallback)
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
