package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"hanfe/internal/cli"
	"hanfe/internal/config"
	"hanfe/internal/device"
	"hanfe/internal/emitter"
	"hanfe/internal/engine"
	"hanfe/internal/layout"
)

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	opts, err := cli.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, cli.Usage())
		return 1
	}

	if opts.ShowHelp {
		fmt.Println(cli.Usage())
		fmt.Println()
		fmt.Println("Available layouts:")
		for _, name := range layout.AvailableLayouts() {
			fmt.Printf("  %s\n", name)
		}
		return 0
	}

	if opts.ListLayouts {
		for _, name := range layout.AvailableLayouts() {
			fmt.Println(name)
		}
		return 0
	}

	lay, err := layout.Load(opts.LayoutName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	toggleCfg, err := config.ResolveToggleConfig(opts.ToggleConfigPath)
	if err != nil {
		var cfgErr config.ConfigError
		if errors.As(err, &cfgErr) {
			fmt.Fprintf(os.Stderr, "Configuration error: %s\n", cfgErr.Error())
			return 2
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	devicePath := opts.DevicePath
	if devicePath == "" {
		detected, err := device.DetectKeyboardDevice()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to auto-detect a keyboard device")
			if err.Error() != "" {
				fmt.Fprintf(os.Stderr, ": %v", err)
			}
			fmt.Fprintln(os.Stderr, "\nProvide --device /dev/input/eventX explicitly.")
			return 1
		}
		devicePath = detected.Path
		fmt.Printf("Auto-detected keyboard device: %s", detected.Path)
		if detected.Name != "" {
			fmt.Printf(" [%s]", detected.Name)
		}
		fmt.Println()
	}

	fd, err := syscall.Open(devicePath, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open device '%s': %v\n", devicePath, err)
		return 1
	}
	defer syscall.Close(fd)

	emitterInstance, err := emitter.Open(layout.UnicodeHexKeycodes(), opts.TTYPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create fallback emitter: %v\n", err)
		return 1
	}
	defer emitterInstance.Close()

	eng := engine.NewEngine(fd, lay, toggleCfg, emitterInstance)
	if err := eng.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
