package ttybridge

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
)

const (
	helperEnv   = "HANFE_TTY_HELPER"
	bridgeFDEnv = "HANFE_TTY_BRIDGE_FD"
	childFDEnv  = "HANFE_TTY_CHILD_FD"
	pathEnv     = "HANFE_TTY_PATH"
)

var retainedParent *os.File

// RememberTTYPath stores the resolved TTY path in the process environment so it
// can be reused after daemonization or across exec boundaries.
func RememberTTYPath(path string) {
	if path == "" {
		return
	}
	os.Setenv(pathEnv, path)
}

// TTYPathHint retrieves a previously remembered TTY path from the environment.
func TTYPathHint() string {
	return os.Getenv(pathEnv)
}

// BridgeFDForFork exposes the retained parent-side file descriptor for the
// helper bridge so callers can preserve it across fork/exec boundaries. The
// returned boolean indicates whether a descriptor should be forwarded and the
// environment variable name that must be updated with the inherited descriptor
// number.
func BridgeFDForFork() (*os.File, string, bool) {
	if retainedParent == nil {
		return nil, "", false
	}
	if os.Getenv(bridgeFDEnv) == "" {
		return nil, "", false
	}
	return retainedParent, bridgeFDEnv, true
}

// InHelperMode reports whether the current process should act as the TTY helper
// daemon instead of running the main engine.
func InHelperMode() bool {
	return os.Getenv(helperEnv) == "1"
}

// SpawnHelper starts the dedicated TTY helper process when a TTY path is
// supplied. The helper retains control of the user's terminal so characters can
// be injected back into STDIN even after the main daemon detaches from the
// controlling TTY.
func SpawnHelper(ttyPath string) error {
	if ttyPath == "" {
		return nil
	}
	if os.Getenv(bridgeFDEnv) != "" {
		// Helper already spawned in this environment.
		return nil
	}

	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return fmt.Errorf("socketpair: %w", err)
	}

	parentFD := fds[0]
	childFD := fds[1]
	if err := clearCloseOnExec(parentFD); err != nil {
		syscall.Close(parentFD)
		syscall.Close(childFD)
		return err
	}
	if err := clearCloseOnExec(childFD); err != nil {
		syscall.Close(parentFD)
		syscall.Close(childFD)
		return err
	}

	childFile := os.NewFile(uintptr(childFD), "hanfe-tty-child")
	exe, err := os.Executable()
	if err != nil {
		syscall.Close(parentFD)
		childFile.Close()
		return fmt.Errorf("resolve executable: %w", err)
	}

	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), helperEnv+"=1", pathEnv+"="+ttyPath, childFDEnv+"=3")
	cmd.ExtraFiles = []*os.File{childFile}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		syscall.Close(parentFD)
		childFile.Close()
		return fmt.Errorf("start tty helper: %w", err)
	}

	childFile.Close()
	retainedParent = os.NewFile(uintptr(parentFD), "hanfe-tty-parent")
	os.Setenv(bridgeFDEnv, strconv.Itoa(parentFD))
	return nil
}

// Attach creates a client connected to the helper process. When no helper is
// available the returned client will be nil.
func Attach() (*Client, error) {
	fdStr := os.Getenv(bridgeFDEnv)
	if fdStr == "" {
		return nil, nil
	}
	fd, err := strconv.Atoi(fdStr)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", bridgeFDEnv, err)
	}

	var file *os.File
	if retainedParent != nil && int(retainedParent.Fd()) == fd {
		file = retainedParent
		retainedParent = nil
	} else {
		file = os.NewFile(uintptr(fd), "hanfe-tty-parent")
	}
	return &Client{file: file}, nil
}

type Client struct {
	file *os.File
	mu   sync.Mutex
}

// Close shuts down the connection to the helper daemon.
func (c *Client) Close() error {
	if c == nil || c.file == nil {
		return nil
	}
	err := c.file.Close()
	c.file = nil
	return err
}

// WriteString forwards committed text to the helper.
func (c *Client) WriteString(text string) error {
	if c == nil || c.file == nil || text == "" {
		return nil
	}
	payload := []byte(text)
	header := make([]byte, 5)
	header[0] = 'T'
	binary.BigEndian.PutUint32(header[1:], uint32(len(payload)))

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.file.Write(header); err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}
	_, err := c.file.Write(payload)
	return err
}

// SendBackspace requests that the helper synthesise a number of backspace
// keypresses on the controlling TTY.
func (c *Client) SendBackspace(count int) error {
	if c == nil || c.file == nil || count <= 0 {
		return nil
	}
	header := make([]byte, 5)
	header[0] = 'B'
	binary.BigEndian.PutUint32(header[1:], uint32(count))

	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.file.Write(header)
	return err
}

func clearCloseOnExec(fd int) error {
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), uintptr(syscall.F_SETFD), 0)
	if errno != 0 {
		return errno
	}
	return nil
}
