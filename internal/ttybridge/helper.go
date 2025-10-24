package ttybridge

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

// RunHelper serves the dedicated TTY helper loop. The helper listens for
// commands on the inherited UNIX domain socket and injects characters back into
// the controlling terminal using TIOCSTI (falling back to direct writes when
// necessary).
func RunHelper() error {
	fdStr := os.Getenv(childFDEnv)
	if fdStr == "" {
		fdStr = "3"
	}
	fd, err := strconv.Atoi(fdStr)
	if err != nil {
		return fmt.Errorf("parse %s: %w", childFDEnv, err)
	}

	conn := os.NewFile(uintptr(fd), "hanfe-tty-conn")
	defer conn.Close()

	ttyPath := os.Getenv(pathEnv)
	if ttyPath == "" {
		return fmt.Errorf("%s not provided", pathEnv)
	}

	ttyFD, err := syscall.Open(ttyPath, syscall.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open tty %s: %w", ttyPath, err)
	}
	defer syscall.Close(ttyFD)

	reader := bufio.NewReader(conn)
	for {
		cmd, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch cmd {
		case 'T':
			if err := handleText(reader, ttyFD); err != nil {
				return err
			}
		case 'B':
			if err := handleBackspace(reader, ttyFD); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown helper command %q", cmd)
		}
	}
}

func handleText(r *bufio.Reader, ttyFD int) error {
	length, err := readLength(r)
	if err != nil {
		return err
	}
	if length == 0 {
		return nil
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	for _, b := range buf {
		if err := ttyPushByte(ttyFD, b); err != nil {
			return err
		}
	}
	return nil
}

func handleBackspace(r *bufio.Reader, ttyFD int) error {
	length, err := readLength(r)
	if err != nil {
		return err
	}
	for i := uint32(0); i < length; i++ {
		if err := ttyPushByte(ttyFD, '\b'); err != nil {
			return err
		}
	}
	return nil
}

func readLength(r *bufio.Reader) (uint32, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(header[:]), nil
}

func ttyPushByte(fd int, b byte) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TIOCSTI), uintptr(unsafe.Pointer(&b)))
	if errno != 0 {
		if _, err := syscall.Write(fd, []byte{b}); err != nil {
			return err
		}
	}
	return nil
}
