package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/gg582/hanfe/internal/app"
	"github.com/gg582/hanfe/internal/cli"
	"github.com/gg582/hanfe/internal/layout"
	"github.com/gg582/hanfe/internal/ttybridge"
)

const daemonEnv = "HANFE_DAEMONIZED"

func main() {
	if ttybridge.InHelperMode() {
		if err := ttybridge.RunHelper(); err != nil {
			fmt.Fprintf(os.Stderr, "hanfe helper: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, err := cli.Parse(args)
	if err != nil {
		return err
	}
	if opts.ShowHelp {
		fmt.Println(cli.Usage())
		return nil
	}
	if opts.ListLayouts {
		listLayouts()
		return nil
	}

	if opts.Daemonize {
		spawned, derr := daemonizeIfNeeded()
		if derr != nil {
			return derr
		}
		if spawned {
			return nil
		}
	}

	runtime := app.NewRuntime(opts)
	return runtime.Run()
}

func listLayouts() {
	for _, name := range layout.AvailableLayouts() {
		fmt.Println(name)
	}
	fmt.Println("none")
}

func daemonizeIfNeeded() (bool, error) {
	if os.Getenv(daemonEnv) == "1" {
		return false, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return false, err
	}

	env := append([]string{}, os.Environ()...)
	env = setEnv(env, daemonEnv, "1")

	files := []*os.File{os.Stdin, os.Stdout, os.Stderr}
	if fd, envName, ok := ttybridge.BridgeFDForFork(); ok {
		fdIndex := len(files)
		files = append(files, fd)
		env = setEnv(env, envName, fmt.Sprint(fdIndex))
	}

	attrs := &os.ProcAttr{
		Files: files,
		Env:   env,
		Sys:   &syscall.SysProcAttr{Setsid: true},
	}

	proc, err := os.StartProcess(exe, os.Args, attrs)
	if err != nil {
		return false, err
	}
	return true, proc.Release()
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
