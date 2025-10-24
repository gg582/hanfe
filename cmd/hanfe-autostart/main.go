package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
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

	cmds := []*exec.Cmd{
		exec.CommandContext(ctx, hanfePath),
		exec.CommandContext(ctx, ttyPath),
	}

	// share stdout/stderr but keep stdin for the TTY helper only
	if len(cmds) > 0 {
		cmds[0].Stdout = os.Stdout
		cmds[0].Stderr = os.Stderr
	}
	if len(cmds) > 1 {
		cmds[1].Stdout = os.Stdout
		cmds[1].Stderr = os.Stderr
		cmds[1].Stdin = os.Stdin
	}

	errCh := make(chan error, len(cmds))
	started := 0
	var startErr error

	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			startErr = fmt.Errorf("failed to start %s: %w", cmd.Path, err)
			stop()
			break
		}
		started++
		go func(c *exec.Cmd) {
			errCh <- c.Wait()
		}(cmd)
	}

	if startErr != nil {
		for i := 0; i < started; i++ {
			<-errCh
		}
		return startErr
	}

	var firstErr error
	for i := 0; i < started; i++ {
		waitErr := <-errCh
		if waitErr != nil && firstErr == nil {
			firstErr = waitErr
		}
		stop()
	}

	return firstErr
}
