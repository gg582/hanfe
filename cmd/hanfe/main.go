package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"hanfe/internal/cli"
	"hanfe/internal/config"
	"hanfe/internal/device"
	"hanfe/internal/emitter"
	"hanfe/internal/engine"
	"hanfe/internal/layout"
)

const (
	daemonEnv = "HANFE_DAEMONIZED"
	ttyEnv    = "HANFE_TTY_PATH"
)

func main() {
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

	ttyPath := opts.TTYPath
	if ttyPath == "" {
		if envTTY := os.Getenv(ttyEnv); envTTY != "" {
			ttyPath = envTTY
		} else if inferred, ok := detectTTY(); ok {
			ttyPath = inferred
		}
	}

	spawned, err := daemonizeIfNeeded(opts.Daemonize, ttyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: failed to daemonize: %v\n", err)
		os.Exit(1)
	}
	if spawned {
		return
	}

	if opts.TTYPath == "" {
		opts.TTYPath = ttyPath
	}

	toggleCfg, err := config.ResolveToggleConfig(opts.ToggleConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}

	layoutName := opts.LayoutName
	if layoutName == "" {
		layoutName = "dubeolsik"
	}
	keyLayout, err := layout.Load(layoutName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
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

	fallback, err := emitter.Open(layout.UnicodeHexKeycodes(), opts.TTYPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}

	eng := engine.NewEngine(fd, keyLayout, toggleCfg, fallback)
	if err := eng.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}
}

func daemonizeIfNeeded(enabled bool, ttyPath string) (bool, error) {
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

	env := append(os.Environ(), daemonEnv+"=1")
	if ttyPath != "" {
		env = append(env, ttyEnv+"="+ttyPath)
	}

	attrs := &os.ProcAttr{
		Files: []*os.File{devNull, devNull, devNull},
		Env:   env,
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

func detectTTY() (string, bool) {
	for _, fd := range []int{0, 1, 2} {
		if path, ok := ttyPathFromFD(fd); ok {
			return path, true
		}
	}
	return "", false
}

func ttyPathFromFD(fd int) (string, bool) {
	link, err := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", fd))
	if err != nil || link == "" {
		return "", false
	}
	if !isTTYPath(link) {
		return "", false
	}
	info, err := os.Stat(link)
	if err != nil {
		return "", false
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return "", false
	}
	return link, true
}

func isTTYPath(path string) bool {
	if path == "" {
		return false
	}
	if strings.HasPrefix(path, "/dev/pts/") {
		return true
	}
	if strings.HasPrefix(path, "/dev/tty") && path != "/dev/tty" {
		return true
	}
	return false
}
