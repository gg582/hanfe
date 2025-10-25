package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/snowmerak/hangul-logotype/hangul"
)

const (
	DefaultLayoutName = "dubeolsik"
	socketEnv         = "HANFE_SOCKET"
)

var availableLayouts = []string{
	"dubeolsik",
	"sebulshik-final",
	"none",
}

// ResolveLayout converts a user-provided layout name into a hangul keyboard layout.
// When "none" (or an empty string) is requested, a nil layout is returned which keeps
// the original characters untouched.
func ResolveLayout(name string) (hangul.KeyboardLayout, string, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch normalized {
	case "", "default", "dubeolsik", "2beolsik", "두벌식":
		return hangul.DubeolsikLayout, "dubeolsik", nil
	case "sebulshik", "sebulshik-final", "3beolsik-final", "sebulsik", "세벌식", "sebul":
		return hangul.SebulshikFinalLayout, "sebulshik-final", nil
	case "none", "latin", "raw":
		return nil, "none", nil
	default:
		return nil, "", fmt.Errorf("unknown layout %q (available: %s)", name, strings.Join(availableLayouts, ", "))
	}
}

// DefaultSocketPath returns the default unix domain socket path used by the hanfe daemon.
func DefaultSocketPath() string {
	if env := os.Getenv(socketEnv); env != "" {
		return env
	}
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, "hanfe.sock")
	}
	if stateDir := os.Getenv("XDG_STATE_HOME"); stateDir != "" {
		return filepath.Join(stateDir, "hanfe", "hanfe.sock")
	}
	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		return filepath.Join(configDir, "hanfe", "hanfe.sock")
	}
	return filepath.Join(os.TempDir(), "hanfe.sock")
}

// EnsureSocketDir ensures that the directory containing the unix socket exists.
func EnsureSocketDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" || dir == string(filepath.Separator) {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// AvailableLayouts returns the list of layout names understood by ResolveLayout.
func AvailableLayouts() []string {
	copyOf := make([]string, len(availableLayouts))
	copy(copyOf, availableLayouts)
	return copyOf
}
