package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"hanfe/internal/autostart/focus"
	"hanfe/internal/ttybridge"
)

type mode int

const (
	modeHanfe mode = iota
	modeTTY
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hanfe-autostart: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	hanfePath, err := exec.LookPath("hanfe")
	if err != nil {
		return fmt.Errorf("cannot find hanfe binary: %w", err)
	}

	ttyPath, err := exec.LookPath("hanfe-tty")
	if err != nil {
		return fmt.Errorf("cannot find hanfe-tty binary: %w", err)
	}

	detector, err := focus.NewDetector()
	if err != nil {
		return err
	}
	defer detector.Close()

	ttyDevice, err := detectTTYDevice()
	if err != nil {
		fmt.Fprintf(os.Stderr, "hanfe-autostart: warning: %v\n", err)
	}
	var ttyFile *os.File
	if ttyDevice != "" {
		ttyFile, err = os.OpenFile(ttyDevice, os.O_RDWR, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hanfe-autostart: warning: open %s: %v\n", ttyDevice, err)
		} else {
			defer ttyFile.Close()
		}
	}

	hanfeCmd := exec.CommandContext(ctx, hanfePath)
	hanfeCmd.Stdout = os.Stdout
	hanfeCmd.Stderr = os.Stderr
	hanfeCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	ttyCmd := exec.CommandContext(ctx, ttyPath)
	ttyCmd.Stdout = chooseFile(ttyFile, os.Stdout)
	ttyCmd.Stderr = os.Stderr
	ttyCmd.Stdin = chooseFile(ttyFile, os.Stdin)
	ttyCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := hanfeCmd.Start(); err != nil {
		return fmt.Errorf("failed to start hanfe: %w", err)
	}
	if err := ttyCmd.Start(); err != nil {
		hanfeCmd.Process.Signal(syscall.SIGTERM)
		_ = hanfeCmd.Wait()
		return fmt.Errorf("failed to start hanfe-tty: %w", err)
	}

	_ = suspend(ttyCmd.Process)
	current := modeHanfe
	modeCh := detector.WaitForChange(300*time.Millisecond, ctx.Done())

	applyMode := func(next mode) {
		if next == current {
			return
		}
		current = next
		switch next {
		case modeTTY:
			_ = resume(ttyCmd.Process)
			_ = suspend(hanfeCmd.Process)
		case modeHanfe:
			_ = resume(hanfeCmd.Process)
			_ = suspend(ttyCmd.Process)
		default:
			_ = resume(hanfeCmd.Process)
			_ = suspend(ttyCmd.Process)
		}
	}

	errs := make(chan error, 2)
	go func() {
		errs <- hanfeCmd.Wait()
	}()
	go func() {
		errs <- ttyCmd.Wait()
	}()

	for {
		select {
		case <-ctx.Done():
			terminate(hanfeCmd.Process)
			terminate(ttyCmd.Process)
			<-errs
			<-errs
			return ctx.Err()
		case st, ok := <-modeCh:
			if !ok {
				modeCh = nil
				continue
			}
			if st.Terminal {
				applyMode(modeTTY)
			} else {
				applyMode(modeHanfe)
			}
		case err := <-errs:
			terminate(hanfeCmd.Process)
			terminate(ttyCmd.Process)
			<-errs
			return err
		}
	}
}

func chooseFile(primary, fallback *os.File) *os.File {
	if primary != nil {
		return primary
	}
	return fallback
}

func suspend(proc *os.Process) error {
	if proc == nil {
		return nil
	}
	return proc.Signal(syscall.SIGSTOP)
}

func resume(proc *os.Process) error {
	if proc == nil {
		return nil
	}
	return proc.Signal(syscall.SIGCONT)
}

func terminate(proc *os.Process) {
	if proc == nil {
		return
	}
	_ = proc.Signal(syscall.SIGTERM)
}

func detectTTYDevice() (string, error) {
	path, err := ttybridge.DetectTTYPath()
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(path, "/dev/pts/") || strings.HasPrefix(path, "/dev/tty") {
		return path, nil
	}
	return "", fmt.Errorf("unsupported tty path %s", filepath.Base(path))
}
