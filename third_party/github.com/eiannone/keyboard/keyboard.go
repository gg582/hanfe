package keyboard

import (
	"errors"
	"io"
	"os"
	"sync"
	"syscall"
	"unicode/utf8"
	"unsafe"
)

type Key int

const (
	KeyUnknown Key = iota
	KeyRune
	KeySpace
	KeyEnter
	KeyTab
	KeyBackspace
	KeyEsc
	KeyCtrlC
	KeyCtrlSpace
)

var (
	stateMu     sync.Mutex
	original    *syscall.Termios
	isActivated bool
)

func Open() error {
	stateMu.Lock()
	defer stateMu.Unlock()

	if isActivated {
		return nil
	}

	fd := int(os.Stdin.Fd())
	termios, err := getTermios(fd)
	if err != nil {
		return err
	}

	updated := *termios
	updated.Iflag &^= syscall.ICRNL | syscall.INLCR
	updated.Lflag &^= syscall.ICANON | syscall.ECHO
	updated.Cc[syscall.VMIN] = 1
	updated.Cc[syscall.VTIME] = 0

	if err := setTermios(fd, &updated); err != nil {
		return err
	}

	original = termios
	isActivated = true
	return nil
}

func Close() error {
	stateMu.Lock()
	defer stateMu.Unlock()

	if !isActivated {
		return nil
	}

	fd := int(os.Stdin.Fd())
	if err := setTermios(fd, original); err != nil {
		return err
	}

	original = nil
	isActivated = false
	return nil
}

func GetSingleKey() (rune, Key, error) {
	stateMu.Lock()
	active := isActivated
	stateMu.Unlock()

	if !active {
		return 0, KeyUnknown, errors.New("keyboard: not initialized; call Open first")
	}

	var buf [8]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		if err == io.EOF {
			return 0, KeyUnknown, err
		}
		return 0, KeyUnknown, err
	}
	if n == 0 {
		return 0, KeyUnknown, nil
	}

	b := buf[0]
	switch b {
	case 0x00:
		return 0, KeyCtrlSpace, nil
	case 0x03:
		return 0, KeyCtrlC, nil
	case 0x09:
		return '\t', KeyTab, nil
	case 0x0a, 0x0d:
		return '\n', KeyEnter, nil
	case 0x1b:
		return 0, KeyEsc, nil
	case 0x20:
		return ' ', KeySpace, nil
	case 0x7f, 0x08:
		return 0, KeyBackspace, nil
	}

	r, size := utf8.DecodeRune(buf[:n])
	if r == utf8.RuneError && size == 1 {
		return 0, KeyUnknown, nil
	}
	return r, KeyRune, nil
}

func getTermios(fd int) (*syscall.Termios, error) {
	var term syscall.Termios
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&term)), 0, 0, 0); errno != 0 {
		return nil, errno
	}
	return &term, nil
}

func setTermios(fd int, term *syscall.Termios) error {
	if term == nil {
		return errors.New("keyboard: nil termios state")
	}
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(term)), 0, 0, 0); errno != 0 {
		return errno
	}
	return nil
}
