package linux

import "unsafe"

const (
	EvSyn = 0x00
	EvKey = 0x01

	SynReport = 0

	KeyEsc        = 1
	Key1          = 2
	Key2          = 3
	Key3          = 4
	Key4          = 5
	Key5          = 6
	Key6          = 7
	Key7          = 8
	Key8          = 9
	Key9          = 10
	Key0          = 11
	KeyMinus      = 12
	KeyEqual      = 13
	KeyBackspace  = 14
	KeyTab        = 15
	KeyQ          = 16
	KeyW          = 17
	KeyE          = 18
	KeyR          = 19
	KeyT          = 20
	KeyY          = 21
	KeyU          = 22
	KeyI          = 23
	KeyO          = 24
	KeyP          = 25
	KeyLeftBrace  = 26
	KeyRightBrace = 27
	KeyEnter      = 28
	KeyLeftCtrl   = 29
	KeyA          = 30
	KeyS          = 31
	KeyD          = 32
	KeyF          = 33
	KeyG          = 34
	KeyH          = 35
	KeyJ          = 36
	KeyK          = 37
	KeyL          = 38
	KeySemicolon  = 39
	KeyApostrophe = 40
	KeyGrave      = 41
	KeyLeftShift  = 42
	KeyBackslash  = 43
	KeyZ          = 44
	KeyX          = 45
	KeyC          = 46
	KeyV          = 47
	KeyB          = 48
	KeyN          = 49
	KeyM          = 50
	KeyComma      = 51
	KeyDot        = 52
	KeySlash      = 53
	KeyRightShift = 54
	KeyLeftAlt    = 56
	KeySpace      = 57
	KeyCapsLock   = 58
	KeyF1         = 59
	KeyF2         = 60
	KeyF3         = 61
	KeyF4         = 62
	KeyF5         = 63
	KeyF6         = 64
	KeyF7         = 65
	KeyF8         = 66
	KeyF9         = 67
	KeyF10        = 68
	KeyRightCtrl  = 97
	KeyRightAlt   = 100
	KeyF11        = 87
	KeyF12        = 88
	KeyLeftMeta   = 125
	KeyRightMeta  = 126
	KeyHangeul    = 122
	KeyHanja      = 123
)

const (
	KeyHangul = KeyHangeul
)

const (
	KeyMax = 0x2ff
	EvMax  = 0x1f
)

const (
	BusUSB            = 0x03
	UinputMaxNameSize = 80
)

const (
	intSize = 4
)

const (
	iocNrBits   = 8
	iocTypeBits = 8
	iocSizeBits = 14
	iocDirBits  = 2

	iocNrShift   = 0
	iocTypeShift = iocNrShift + iocNrBits
	iocSizeShift = iocTypeShift + iocTypeBits
	iocDirShift  = iocSizeShift + iocSizeBits

	iocNone  = 0
	iocWrite = 1
	iocRead  = 2
)

func ioc(dir, typ, nr, size uintptr) uintptr {
	return (dir << iocDirShift) | (typ << iocTypeShift) | (nr << iocNrShift) | (size << iocSizeShift)
}

func IOW(typ byte, nr uintptr, size uintptr) uintptr {
	return ioc(iocWrite, uintptr(typ), nr, size)
}

func IOR(typ byte, nr uintptr, size uintptr) uintptr {
	return ioc(iocRead, uintptr(typ), nr, size)
}

func IO(typ byte, nr uintptr) uintptr {
	return ioc(iocNone, uintptr(typ), nr, 0)
}

func EVIOCGBIT(ev int, length int) uintptr {
	return ioc(iocRead, uintptr('E'), uintptr(0x20+ev), uintptr(length))
}

func EVIOCGNAME(length int) uintptr {
	return ioc(iocRead, uintptr('E'), 0x06, uintptr(length))
}

var (
	EVIOCGRAB    = IOW('E', 0x90, intSize)
	UISetEvbit   = IOW('U', 100, intSize)
	UISetKeybit  = IOW('U', 101, intSize)
	UIDevCreate  = IO('U', 1)
	UIDevDestroy = IO('U', 2)
)

func UnsafeSlice(p *byte, length int) []byte {
	return unsafe.Slice(p, length)
}
