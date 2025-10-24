package ttybridge

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// DetectTTYPath attempts to resolve the active controlling TTY for the current
// process. The function probes stdin, stdout, and stderr (in that order) and
// returns the first terminal path that can be read via /proc/self/fd. When no
// terminal is attached an empty string and an error are returned.
func DetectTTYPath() (string, error) {
	fds := []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}
	for _, fd := range fds {
		if !isTTY(fd) {
			continue
		}
		path := fmt.Sprintf("/proc/self/fd/%d", fd)
		target, err := os.Readlink(path)
		if err != nil || target == "" {
			continue
		}
		return target, nil
	}
	return "", fmt.Errorf("no controlling tty detected")
}

func isTTY(fd uintptr) bool {
	var termios syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&termios)))
	return errno == 0
}
