package util

import (
	"syscall"
	"unsafe"
)

type InputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

func InputEventSize() int {
	return int(unsafe.Sizeof(InputEvent{}))
}

func (ev *InputEvent) Bytes() []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(ev)), InputEventSize())
}
