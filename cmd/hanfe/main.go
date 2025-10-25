package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/snowmerak/hangul-logotype/hangul"

	"hanfe/internal/common"
)

const daemonEnv = "HANFE_DAEMONIZED"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	defaultSocket := common.DefaultSocketPath()

	layoutName := flag.String("layout", common.DefaultLayoutName, fmt.Sprintf("keyboard layout (%s)", strings.Join(common.AvailableLayouts(), ", ")))
	daemonize := flag.Bool("daemonize", false, "run as a background daemon that serves translations over a unix socket")
	socketPath := flag.String("socket", defaultSocket, "unix socket path for daemon mode")
	flag.Parse()

	layout, canonical, err := common.ResolveLayout(*layoutName)
	if err != nil {
		return err
	}

	if *daemonize {
		spawned, derr := daemonizeIfNeeded()
		if derr != nil {
			return derr
		}
		if spawned {
			return nil
		}
		return runDaemon(layout, canonical, *socketPath)
	}

	if flag.NArg() == 0 {
		info, err := os.Stdin.Stat()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeCharDevice != 0 {
			return errors.New("provide text through arguments or stdin")
		}
		return runStdin(layout)
	}

	return runArgs(layout, flag.Args())
}

func runArgs(layout hangul.KeyboardLayout, args []string) error {
	typer := hangul.NewLogoTyper().WithLayout(layout)
	typer.WriteString(strings.Join(args, " "))
	if _, err := os.Stdout.Write(typer.Result()); err != nil {
		return err
	}
	if len(args) > 0 {
		if _, err := os.Stdout.WriteString("\n"); err != nil {
			return err
		}
	}
	return nil
}

func runStdin(layout hangul.KeyboardLayout) error {
	typer := hangul.NewLogoTyper().WithLayout(layout)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	first := true
	for scanner.Scan() {
		if !first {
			typer.WriteRune('\n')
		}
		first = false
		typer.WriteString(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if _, err := os.Stdout.Write(typer.Result()); err != nil {
		return err
	}
	if !first {
		if _, err := os.Stdout.WriteString("\n"); err != nil {
			return err
		}
	}
	return nil
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

	attrs := &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Env:   env,
		Sys:   &syscall.SysProcAttr{Setsid: true},
	}

	proc, err := os.StartProcess(exe, os.Args, attrs)
	if err != nil {
		return false, err
	}
	return true, proc.Release()
}

func runDaemon(layout hangul.KeyboardLayout, layoutName, socketPath string) error {
	if err := common.EnsureSocketDir(socketPath); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}
	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", socketPath, err)
	}
	defer func() {
		listener.Close()
		_ = os.Remove(socketPath)
	}()
	if err := os.Chmod(socketPath, 0o660); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("chmod socket: %w", err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(sigs)

	errCh := make(chan error, 1)
	go func() {
		errCh <- serve(listener, layout)
	}()

	fmt.Fprintf(os.Stderr, "hanfe daemon running with layout %s on %s\n", layoutName, socketPath)

	select {
	case <-sigs:
		listener.Close()
		return nil
	case err := <-errCh:
		return err
	}
}

func serve(listener net.Listener, layout hangul.KeyboardLayout) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
			return err
		}

		go func(c net.Conn) {
			defer c.Close()
			if err := handleConnection(c, layout); err != nil {
				fmt.Fprintf(os.Stderr, "hanfe daemon: %v\n", err)
			}
		}(conn)
	}
}

func handleConnection(conn net.Conn, layout hangul.KeyboardLayout) error {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	writer := bufio.NewWriter(conn)
	for scanner.Scan() {
		text := scanner.Text()
		response := translate(layout, text)
		if _, err := writer.WriteString(response); err != nil {
			return err
		}
		if err := writer.WriteByte('\n'); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, net.ErrClosed) {
			return nil
		}
		return err
	}
	return nil
}

func translate(layout hangul.KeyboardLayout, text string) string {
	typer := hangul.NewLogoTyper().WithLayout(layout)
	typer.WriteString(text)
	return string(typer.Result())
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
