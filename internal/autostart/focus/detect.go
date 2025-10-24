package focus

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

type State struct {
	Terminal bool
	Name     string
}

type Detector struct {
	conn  *xgb.Conn
	root  xproto.Window
	atoms map[string]xproto.Atom
	mu    sync.Mutex
	last  State
	haveX bool
}

func NewDetector() (*Detector, error) {
	d := &Detector{}

	display := os.Getenv("DISPLAY")
	if display == "" {
		return d, nil
	}

	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("connect x server: %w", err)
	}
	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	d.conn = conn
	d.root = screen.Root
	d.haveX = true
	d.atoms = make(map[string]xproto.Atom)
	if err := d.cacheAtoms([]string{"_NET_ACTIVE_WINDOW", "_NET_WM_PID", "WM_CLASS", "_NET_WM_NAME", "WM_NAME"}); err != nil {
		conn.Close()
		return nil, err
	}
	return d, nil
}

func (d *Detector) cacheAtoms(names []string) error {
	for _, name := range names {
		atom, err := xproto.InternAtom(d.conn, true, uint16(len(name)), name).Reply()
		if err != nil {
			return err
		}
		d.atoms[name] = atom.Atom
	}
	return nil
}

func (d *Detector) Close() error {
	if d.conn != nil {
		d.conn.Close()
		d.conn = nil
	}
	return nil
}

func (d *Detector) Poll() State {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.haveX {
		if st, err := d.pollX11(); err == nil {
			d.last = st
			return st
		}
	}

	// fallback for non-X11 environments: assume TTY session when DISPLAY empty
	if os.Getenv("DISPLAY") == "" {
		if isConsoleSession() {
			st := State{Terminal: true, Name: "tty"}
			d.last = st
			return st
		}
		st := State{Terminal: true, Name: "unknown-console"}
		d.last = st
		return st
	}

	if os.Getenv("XDG_SESSION_TYPE") == "tty" {
		st := State{Terminal: true, Name: "tty"}
		d.last = st
		return st
	}

	return d.last
}

func (d *Detector) pollX11() (State, error) {
	activeAtom := d.atoms["_NET_ACTIVE_WINDOW"]
	if activeAtom == 0 {
		return State{}, errors.New("missing _NET_ACTIVE_WINDOW atom")
	}

	reply, err := xproto.GetProperty(d.conn, false, d.root, activeAtom, xproto.AtomWindow, 0, 1).Reply()
	if err != nil {
		return State{}, err
	}
	if reply == nil || reply.ValueLen == 0 {
		return State{}, errors.New("no active window")
	}
	window := xproto.Window(xgbGet32(reply.Value))
	if window == 0 {
		return State{}, errors.New("invalid active window")
	}

	if terminal, name := d.inspectWindow(window); terminal {
		return State{Terminal: true, Name: name}, nil
	}
	return State{Terminal: false}, nil
}

func (d *Detector) inspectWindow(win xproto.Window) (bool, string) {
	names := d.windowClassNames(win)
	for _, name := range names {
		if isTerminalName(name) {
			return true, name
		}
	}
	if pid := d.windowPID(win); pid != 0 {
		if pname := processName(pid); pname != "" {
			if isTerminalName(pname) {
				return true, pname
			}
		}
	}
	if title := strings.ToLower(strings.Join(d.windowTitles(win), " ")); title != "" {
		if maybeTerminalTitle(title) {
			return true, title
		}
	}
	return false, ""
}

func (d *Detector) windowClassNames(win xproto.Window) []string {
	atom := d.atoms["WM_CLASS"]
	if atom == 0 {
		return nil
	}
	reply, err := xproto.GetProperty(d.conn, false, win, atom, xproto.AtomString, 0, 64).Reply()
	if err != nil || reply == nil {
		return nil
	}
	raw := reply.Value
	if len(raw) == 0 {
		return nil
	}
	parts := strings.Split(string(raw), "\x00")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, strings.ToLower(part))
	}
	return out
}

func (d *Detector) windowPID(win xproto.Window) int {
	atom := d.atoms["_NET_WM_PID"]
	if atom == 0 {
		return 0
	}
	reply, err := xproto.GetProperty(d.conn, false, win, atom, xproto.AtomCardinal, 0, 1).Reply()
	if err != nil || reply == nil || len(reply.Value) < 4 {
		return 0
	}
	return int(xgbGet32(reply.Value))
}

func (d *Detector) windowTitles(win xproto.Window) []string {
	atoms := []string{"_NET_WM_NAME", "WM_NAME"}
	titles := make([]string, 0, len(atoms))
	for _, key := range atoms {
		atom := d.atoms[key]
		if atom == 0 {
			continue
		}
		reply, err := xproto.GetProperty(d.conn, false, win, atom, xproto.AtomAny, 0, 64).Reply()
		if err != nil || reply == nil || len(reply.Value) == 0 {
			continue
		}
		titles = append(titles, strings.ToLower(string(reply.Value)))
	}
	return titles
}

func xgbGet32(b []byte) uint32 {
	if len(b) < 4 {
		return 0
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

var terminalNames = map[string]struct{}{
	"alacritty":                {},
	"kitty":                    {},
	"wezterm":                  {},
	"wezterm-gui":              {},
	"ghostty":                  {},
	"gnome-terminal":           {},
	"gnome-terminal-server":    {},
	"kgx":                      {},
	"console":                  {},
	"konsole":                  {},
	"yakuake":                  {},
	"xfce4-terminal":           {},
	"terminator":               {},
	"tilix":                    {},
	"guake":                    {},
	"tilda":                    {},
	"xterm":                    {},
	"uxterm":                   {},
	"rxvt":                     {},
	"urxvt":                    {},
	"sakura":                   {},
	"hyper":                    {},
	"tabby":                    {},
	"rio":                      {},
	"foot":                     {},
	"footclient":               {},
	"cool-retro-term":          {},
	"contour":                  {},
	"st":                       {},
	"eterm":                    {},
	"aterm":                    {},
	"qterminal":                {},
	"lxterminal":               {},
	"mate-terminal":            {},
	"deepin-terminal":          {},
	"wezterm-mux-server":       {},
	"warp-terminal":            {},
	"termius-app":              {},
	"blackbox":                 {},
	"io.elementary.terminal":   {},
	"com.raggesilver.blackbox": {},
}

func isTerminalName(name string) bool {
	name = strings.ToLower(name)
	if name == "" {
		return false
	}
	if _, ok := terminalNames[name]; ok {
		return true
	}
	for _, part := range strings.Split(name, " ") {
		if _, ok := terminalNames[part]; ok {
			return true
		}
	}
	if strings.Contains(name, "terminal") || strings.Contains(name, "term") {
		return true
	}
	return false
}

func maybeTerminalTitle(title string) bool {
	keywords := []string{"terminal", "shell", "tty", "console"}
	for _, kw := range keywords {
		if strings.Contains(title, kw) {
			return true
		}
	}
	return false
}

func processName(pid int) string {
	if pid <= 0 {
		return ""
	}
	comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err == nil {
		return strings.TrimSpace(strings.ToLower(string(comm)))
	}
	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	fields := strings.Split(string(cmdline), "\x00")
	for _, field := range fields {
		if field == "" {
			continue
		}
		base := filepath.Base(field)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		base = strings.ToLower(base)
		if base != "" {
			return base
		}
	}
	return ""
}

func isConsoleSession() bool {
	ttyPath, err := os.ReadFile("/sys/class/tty/tty0/active")
	if err == nil && len(ttyPath) > 0 {
		return true
	}
	// fallback: any tty in /proc/self/fd pointing to /dev/tty*
	for fd := uintptr(0); fd < 3; fd++ {
		target, err := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", fd))
		if err != nil {
			continue
		}
		if strings.HasPrefix(target, "/dev/tty") {
			return true
		}
	}
	return false
}

func (d *Detector) WaitForChange(interval time.Duration, stop <-chan struct{}) <-chan State {
	ch := make(chan State, 1)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		last := d.Poll()
		ch <- last
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				next := d.Poll()
				if next != last {
					ch <- next
					last = next
				}
			}
		}
	}()
	return ch
}
