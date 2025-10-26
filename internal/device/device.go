package device

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unsafe"

	"github.com/gg582/hanfe/internal/linux"
)

type DetectedDevice struct {
	Path string
	Name string
}

type DetectionError struct {
	Message string
}

func (e DetectionError) Error() string { return e.Message }

func bitsToBytes(bits int) int {
	return (bits + 7) / 8
}

func ioctlRead(fd int, request uintptr, buffer []byte) error {
	if len(buffer) == 0 {
		return nil
	}
	return linux.Ioctl(fd, request, uintptr(unsafe.Pointer(&buffer[0])))
}

func isKeyboardFD(fd int) bool {
	evBits := make([]byte, bitsToBytes(linux.EvMax+1))
	if err := ioctlRead(fd, linux.EVIOCGBIT(0, len(evBits)), evBits); err != nil {
		return false
	}
	if !testBit(evBits, linux.EvKey) {
		return false
	}

	keyBits := make([]byte, bitsToBytes(linux.KeyMax+1))
	if err := ioctlRead(fd, linux.EVIOCGBIT(int(linux.EvKey), len(keyBits)), keyBits); err != nil {
		return false
	}

	required := []int{linux.KeyA, linux.KeyZ, linux.KeySpace, linux.KeyEnter, linux.KeyLeftShift}
	for _, code := range required {
		if !testBit(keyBits, code) {
			return false
		}
	}
	return true
}

func testBit(bits []byte, bit int) bool {
	idx := bit / 8
	off := bit % 8
	if idx < 0 || idx >= len(bits) {
		return false
	}
	return (bits[idx] & (1 << uint(off))) != 0
}

func readDeviceName(fd int) string {
	buf := make([]byte, 256)
	if err := ioctlRead(fd, linux.EVIOCGNAME(len(buf)), buf); err != nil {
		return ""
	}
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}

func collectKeyboardSymlinks(dir string) []string {
	entries := make([]string, 0)
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		lower := strings.ToLower(name)
		if strings.Contains(lower, "kbd") || strings.Contains(lower, "keyboard") {
			entries = append(entries, path)
		}
		return nil
	})
	sort.Strings(entries)
	entries = unique(entries)
	return entries
}

func collectEventNodes() []string {
	entries := make([]string, 0)
	dirEntries, err := os.ReadDir("/dev/input")
	if err != nil {
		return entries
	}
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "event") {
			entries = append(entries, filepath.Join("/dev/input", name))
		}
	}
	sort.Strings(entries)
	return unique(entries)
}

func unique(items []string) []string {
	if len(items) == 0 {
		return items
	}
	out := make([]string, 0, len(items))
	var last string
	for i, item := range items {
		if i == 0 || item != last {
			out = append(out, item)
			last = item
		}
	}
	return out
}

func gatherCandidates() []string {
	seen := make(map[string]struct{})
	appendUnique := func(paths []string) {
		for _, p := range paths {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
		}
	}

	appendUnique(collectKeyboardSymlinks("/dev/input/by-id"))
	appendUnique(collectKeyboardSymlinks("/dev/input/by-path"))
	appendUnique(collectEventNodes())

	candidates := make([]string, 0, len(seen))
	for path := range seen {
		candidates = append(candidates, path)
	}
	sort.Strings(candidates)
	return candidates
}

func ListKeyboardDevices() ([]DetectedDevice, error) {
	candidates := gatherCandidates()
	devices := make([]DetectedDevice, 0)
	permissionDenied := false
	var lastErr error

	for _, path := range candidates {
		fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
		if err != nil {
			if errors.Is(err, os.ErrPermission) || err == syscall.EACCES || err == syscall.EPERM {
				permissionDenied = true
			}
			lastErr = fmt.Errorf("%s: %w", path, err)
			continue
		}
		name := readDeviceName(fd)
		if isKeyboardFD(fd) {
			devices = append(devices, DetectedDevice{Path: path, Name: name})
		}
		syscall.Close(fd)
	}

	if len(devices) == 0 {
		switch {
		case permissionDenied:
			return nil, DetectionError{Message: "Permission denied while probing input devices. Try running as root or adjusting udev permissions."}
		case len(candidates) == 0:
			return nil, DetectionError{Message: "No evdev devices found under /dev/input."}
		case lastErr != nil:
			return nil, DetectionError{Message: fmt.Sprintf("No keyboard-like device found. Last error: %v", lastErr)}
		default:
			return nil, DetectionError{Message: "No keyboard-like device found."}
		}
	}

	sort.SliceStable(devices, func(i, j int) bool { return devices[i].Path < devices[j].Path })
	return devices, nil
}

func DetectKeyboardDevice() (DetectedDevice, error) {
	devices, err := ListKeyboardDevices()
	if err != nil {
		return DetectedDevice{}, err
	}
	if len(devices) == 0 {
		return DetectedDevice{}, DetectionError{Message: "no keyboard-like device found"}
	}
	return devices[0], nil
}
