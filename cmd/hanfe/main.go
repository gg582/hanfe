package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/snowmerak/hangul-logotype/hangul"

	"hanfe/internal/backend"
	"hanfe/internal/cli"
	"hanfe/internal/common"
	"hanfe/internal/config"
	"hanfe/internal/device"
	"hanfe/internal/emitter"
	"hanfe/internal/engine"
	"hanfe/internal/layout"
	"hanfe/internal/ttybridge"
	"hanfe/internal/types"
)

const daemonEnv = "HANFE_DAEMONIZED"

func main() {
	if ttybridge.InHelperMode() {
		if err := ttybridge.RunHelper(); err != nil {
			fmt.Fprintf(os.Stderr, "hanfe helper: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "hanfe: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, err := cli.Parse(args)
	if err != nil {
		return err
	}
	if opts.ShowHelp {
		fmt.Println(cli.Usage())
		return nil
	}
	if opts.ListLayouts {
		listLayouts()
		return nil
	}

	if opts.Daemonize {
		spawned, derr := daemonizeIfNeeded()
		if derr != nil {
			return derr
		}
		if spawned {
			return nil
		}
	}

	translatorLayout, canonical, err := resolveTranslatorLayout(opts.LayoutName)
	if err != nil {
		return err
	}
	engineLayout, hangulName, err := resolveEngineLayout(canonical, opts.LayoutName, opts.KeypairPath)
	if err != nil {
		return err
	}

	toggleCfg, err := config.ResolveToggleConfig(opts.ToggleConfigPath)
	if err != nil {
		return err
	}
	applyModeOrder(&toggleCfg, opts.ModeOrder, hangulName, engineLayout != nil)

	socketPath := opts.SocketPath
	if socketPath == "" {
		socketPath = common.DefaultSocketPath()
	}

	var database backend.Database
	if opts.PinyinDBPath != "" {
		database, err = backend.LoadDatabase(opts.PinyinDBPath)
		if err != nil {
			return err
		}
	}

	devicePath := opts.DevicePath
	if devicePath == "" {
		detected, derr := device.DetectKeyboardDevice()
		if derr != nil {
			return derr
		}
		devicePath = detected.Path
		fmt.Fprintf(os.Stderr, "hanfe: using keyboard %s (%s)\n", detected.Path, detected.Name)
	}

	deviceFD, err := syscall.Open(devicePath, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", devicePath, err)
	}
	closeDevice := func() {
		if deviceFD >= 0 {
			syscall.Close(deviceFD)
			deviceFD = -1
		}
	}
	defer closeDevice()

	ttyPath := opts.TTYPath
	if ttyPath == "" {
		if detectedTTY, derr := ttybridge.DetectTTYPath(); derr == nil {
			ttyPath = detectedTTY
		}
	}

	directCommit := opts.SuppressHex || !hasDisplay() || opts.TTYPath != "" || opts.PTYPath != ""
	if directCommit && ttyPath == "" && opts.PTYPath == "" {
		directCommit = false
	}

	var ttyClient *ttybridge.Client
	if directCommit && ttyPath != "" {
		if err := ttybridge.SpawnHelper(ttyPath); err != nil {
			return err
		}
		ttyClient, err = ttybridge.Attach()
		if err != nil {
			return err
		}
	}

	ptyPath := ""
	if directCommit {
		ptyPath = opts.PTYPath
	}

	fallback, err := emitter.Open(layout.UnicodeHexKeycodes(), ttyClient, ptyPath, directCommit)
	if err != nil {
		return err
	}

	modes, err := buildModes(toggleCfg.ModeCycle, engineLayout, hangulName, database)
	if err != nil {
		fallback.Close()
		return err
	}

	eng, err := engine.NewEngine(deviceFD, modes, toggleCfg, fallback)
	if err != nil {
		fallback.Close()
		return err
	}

	server, err := startTranslationServer(socketPath, translatorLayout)
	if err != nil {
		return err
	}
	if server != nil {
		defer server.Close()
	}

	engineErrCh := make(chan error, 1)
	go func() {
		engineErrCh <- eng.Run()
	}()

	var serverErrCh <-chan error
	if server != nil {
		serverErrCh = server.Err()
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(sigs)

	for {
		select {
		case err := <-engineErrCh:
			if err != nil && !errors.Is(err, syscall.EBADF) && !errors.Is(err, syscall.ENODEV) {
				return err
			}
			return nil
		case err, ok := <-serverErrCh:
			if !ok {
				serverErrCh = nil
				continue
			}
			if err != nil {
				return fmt.Errorf("translation server: %w", err)
			}
			serverErrCh = nil
		case <-sigs:
			closeDevice()
		}
	}
}

func listLayouts() {
	for _, name := range layout.AvailableLayouts() {
		fmt.Println(name)
	}
	fmt.Println("none")
}

func resolveTranslatorLayout(name string) (hangul.KeyboardLayout, string, error) {
	layout, canonical, err := common.ResolveLayout(name)
	if err != nil {
		trimmed := strings.ToLower(strings.TrimSpace(name))
		if trimmed == "" {
			return nil, "", err
		}
		return nil, trimmed, nil
	}
	return layout, canonical, nil
}

func resolveEngineLayout(canonical, raw string, keypairPath string) (*layout.Layout, string, error) {
	normalized := strings.ToLower(strings.TrimSpace(canonical))
	if normalized == "" {
		normalized = strings.ToLower(strings.TrimSpace(raw))
	}
	switch normalized {
	case "", "dubeolsik", "hangul", "korean":
		normalized = "dubeolsik"
	case "sebulshik-final", "sebulshik", "sebulsik", "sebeolsik-final":
		normalized = "sebeolsik-390"
	case "latin", "raw", "none":
		return nil, "", nil
	}
	loaded, err := layout.Load(normalized)
	if err != nil {
		if canonical != "" && canonical != normalized {
			loaded, err = layout.Load(strings.ToLower(strings.TrimSpace(canonical)))
			if err != nil {
				return nil, "", fmt.Errorf("load layout %s: %w", canonical, err)
			}
		} else {
			return nil, "", fmt.Errorf("load layout %s: %w", normalized, err)
		}
	}
	if keypairPath != "" {
		pairs, err := layout.LoadCustomPairs(keypairPath)
		if err != nil {
			return nil, "", err
		}
		loaded, err = layout.ApplyCustomPairs(loaded, pairs)
		if err != nil {
			return nil, "", err
		}
	}
	copy := loaded
	return &copy, copy.Name(), nil
}

func applyModeOrder(cfg *config.ToggleConfig, override []string, hangulName string, haveHangul bool) {
	if override != nil {
		cfg.ModeCycle = override
	}
	cfg.DefaultMode = normalizeModeName(cfg.DefaultMode, hangulName, haveHangul)
	cfg.ModeCycle = normalizeModeCycle(cfg.ModeCycle, hangulName, haveHangul)
	if cfg.DefaultMode == "" {
		if haveHangul && hangulName != "" {
			cfg.DefaultMode = hangulName
		} else {
			cfg.DefaultMode = "latin"
		}
	}
	if !containsString(cfg.ModeCycle, cfg.DefaultMode) {
		cfg.ModeCycle = append([]string{cfg.DefaultMode}, cfg.ModeCycle...)
		cfg.ModeCycle = uniqueStrings(cfg.ModeCycle)
	}
}

func normalizeModeCycle(cycle []string, hangulName string, haveHangul bool) []string {
	normalized := make([]string, 0, len(cycle))
	seen := make(map[string]struct{})
	for _, entry := range cycle {
		name := normalizeModeName(entry, hangulName, haveHangul)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	if haveHangul && hangulName != "" && !containsString(normalized, hangulName) {
		normalized = append([]string{hangulName}, normalized...)
	}
	if !containsString(normalized, "latin") {
		normalized = append(normalized, "latin")
	}
	normalized = uniqueStrings(normalized)
	if len(normalized) == 0 {
		normalized = append(normalized, "latin")
	}
	return normalized
}

func normalizeModeName(value, hangulName string, haveHangul bool) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "hangul", "korean":
		if haveHangul && hangulName != "" {
			return hangulName
		}
		return "latin"
	case "latin", "english", "default":
		return "latin"
	case "kana", "kana86", "hiragana", "katakana", "japanese":
		return "kana86"
	case "pinyin", "zhuyin", "ime", "database":
		return "pinyin"
	case "none", "off":
		return ""
	default:
		return normalized
	}
}

func containsString(list []string, target string) bool {
	for _, entry := range list {
		if entry == target {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func buildModes(cycle []string, hangulLayout *layout.Layout, hangulName string, database backend.Database) ([]engine.ModeSpec, error) {
	available := make(map[string]engine.ModeSpec)
	available["latin"] = engine.ModeSpec{Name: "latin", Kind: types.ModeLatin}

	if hangulLayout != nil {
		layoutCopy := *hangulLayout
		kind := types.ModeHangul
		if layoutCopy.Category() == layout.CategoryKana {
			kind = types.ModeKana
		}
		spec := engine.ModeSpec{Name: hangulName, Kind: kind, Layout: &layoutCopy}
		if hangulName != "" {
			available[strings.ToLower(hangulName)] = spec
		}
	}

	for _, entry := range cycle {
		if entry == "kana86" {
			if _, ok := available["kana86"]; !ok {
				kanaLayout, err := layout.Load("kana86")
				if err != nil {
					return nil, fmt.Errorf("load kana layout: %w", err)
				}
				layoutCopy := kanaLayout
				available["kana86"] = engine.ModeSpec{Name: "kana86", Kind: types.ModeKana, Layout: &layoutCopy}
			}
		}
	}

	if database.Available() {
		available["pinyin"] = engine.ModeSpec{Name: "pinyin", Kind: types.ModeDatabase, Database: database}
	}

	modes := make([]engine.ModeSpec, 0, len(cycle))
	added := make(map[string]struct{})
	for _, entry := range cycle {
		key := strings.ToLower(entry)
		spec, ok := available[key]
		if !ok {
			continue
		}
		if _, seen := added[spec.Name]; seen {
			continue
		}
		modes = append(modes, spec)
		added[spec.Name] = struct{}{}
	}
	if len(modes) == 0 {
		return nil, fmt.Errorf("no input modes available after applying toggle configuration")
	}
	return modes, nil
}

func hasDisplay() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

type translationServer struct {
	listener net.Listener
	socket   string
	errCh    chan error
}

func startTranslationServer(path string, layout hangul.KeyboardLayout) (*translationServer, error) {
	if path == "" {
		return nil, nil
	}
	if err := common.EnsureSocketDir(path); err != nil {
		return nil, fmt.Errorf("create socket dir: %w", err)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o660); err != nil && !errors.Is(err, os.ErrNotExist) {
		listener.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("chmod socket: %w", err)
	}
	srv := &translationServer{listener: listener, socket: path, errCh: make(chan error, 1)}
	go func() {
		srv.errCh <- serveTranslations(listener, layout)
		close(srv.errCh)
	}()
	return srv, nil
}

func (s *translationServer) Close() {
	if s == nil {
		return
	}
	s.listener.Close()
	for range s.errCh {
		// drain channel to ensure goroutine exits
	}
	_ = os.Remove(s.socket)
}

func (s *translationServer) Err() <-chan error {
	if s == nil {
		return nil
	}
	return s.errCh
}

func serveTranslations(listener net.Listener, layout hangul.KeyboardLayout) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
			return err
		}
		go func(c net.Conn) {
			defer c.Close()
			if err := handleTranslationConnection(c, layout); err != nil {
				fmt.Fprintf(os.Stderr, "hanfe: translation error: %v\n", err)
			}
		}(conn)
	}
}

func handleTranslationConnection(conn net.Conn, layout hangul.KeyboardLayout) error {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	writer := bufio.NewWriter(conn)
	for scanner.Scan() {
		text := scanner.Text()
		response := translate(layout, text)
		if _, err := writer.WriteString(response); err != nil {
			return err
		}
		if err := writer.WriteByte('\n'); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, net.ErrClosed) {
			return nil
		}
		return err
	}
	return nil
}

func translate(layout hangul.KeyboardLayout, text string) string {
	if layout == nil {
		return text
	}
	typer := hangul.NewLogoTyper().WithLayout(layout)
	typer.WriteString(text)
	return string(typer.Result())
}

func daemonizeIfNeeded() (bool, error) {
	if os.Getenv(daemonEnv) == "1" {
		return false, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return false, err
	}

	env := append([]string{}, os.Environ()...)
	env = setEnv(env, daemonEnv, "1")

	files := []*os.File{os.Stdin, os.Stdout, os.Stderr}
	if fd, envName, ok := ttybridge.BridgeFDForFork(); ok {
		fdIndex := len(files)
		files = append(files, fd)
		env = setEnv(env, envName, fmt.Sprint(fdIndex))
	}

	attrs := &os.ProcAttr{
		Files: files,
		Env:   env,
		Sys:   &syscall.SysProcAttr{Setsid: true},
	}

	proc, err := os.StartProcess(exe, os.Args, attrs)
	if err != nil {
		return false, err
	}
	return true, proc.Release()
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
