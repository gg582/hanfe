package linux

import "syscall"

func Ioctl(fd int, req uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, arg)
	if errno != 0 {
		return errno
	}
	return nil
}

func IoctlSetInt(fd int, req uintptr, value int) error {
	return Ioctl(fd, req, uintptr(value))
}
