package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"hanfe/internal/common"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hanfe-autostart: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	defaultSocket := common.DefaultSocketPath()

	layoutName := flag.String("layout", common.DefaultLayoutName, "layout to use for both hanfe and hanfe-tty")
	socketPath := flag.String("socket", defaultSocket, "unix socket shared between hanfe and hanfe-tty")
	noTTY := flag.Bool("no-tty", false, "do not launch hanfe-tty after starting the daemon")
	flag.Parse()

	_, canonical, err := common.ResolveLayout(*layoutName)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	if err := launchHanfeDaemon(ctx, canonical, *socketPath); err != nil {
		return err
	}

	if *noTTY {
		return nil
	}

	ttyArgs := []string{"--layout", canonical, "--socket", *socketPath}
	ttyArgs = append(ttyArgs, flag.Args()...)
	ttyPath, err := exec.LookPath("hanfe-tty")
	if err != nil {
		return fmt.Errorf("cannot find hanfe-tty binary: %w", err)
	}

	ttyCmd := exec.CommandContext(ctx, ttyPath, ttyArgs...)
	ttyCmd.Stdout = os.Stdout
	ttyCmd.Stderr = os.Stderr
	ttyCmd.Stdin = os.Stdin

	if err := ttyCmd.Start(); err != nil {
		return fmt.Errorf("failed to start hanfe-tty: %w", err)
	}

	return ttyCmd.Wait()
}

func launchHanfeDaemon(ctx context.Context, layout, socketPath string) error {
	hanfePath, err := exec.LookPath("hanfe")
	if err != nil {
		return fmt.Errorf("cannot find hanfe binary: %w", err)
	}

	args := []string{"--daemonize", "--layout", layout, "--socket", socketPath}
	hanfeCmd := exec.CommandContext(ctx, hanfePath, args...)
	hanfeCmd.Stdout = os.Stdout
	hanfeCmd.Stderr = os.Stderr

	if err := hanfeCmd.Start(); err != nil {
		return fmt.Errorf("failed to start hanfe daemon: %w", err)
	}

	if err := hanfeCmd.Wait(); err != nil {
		return fmt.Errorf("hanfe daemon exited with error: %w", err)
	}

	return waitForSocket(socketPath, 5*time.Second)
}

func waitForSocket(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		if time.Now().After(deadline) {
			if lastErr != nil {
				return fmt.Errorf("timeout waiting for socket %s: %w", path, lastErr)
			}
			return fmt.Errorf("timeout waiting for socket %s", path)
		}

		conn, err := net.DialTimeout("unix", path, 250*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
}
