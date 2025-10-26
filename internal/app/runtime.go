package app

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gg582/hanfe/internal/backend"
	"github.com/gg582/hanfe/internal/cli"
	"github.com/gg582/hanfe/internal/common"
	"github.com/gg582/hanfe/internal/config"
	"github.com/gg582/hanfe/internal/device"
	"github.com/gg582/hanfe/internal/emitter"
	"github.com/gg582/hanfe/internal/engine"
	"github.com/gg582/hanfe/internal/layout"
	"github.com/gg582/hanfe/internal/ttybridge"
	"github.com/gg582/hangul-logotype/hangul"
)

type Runtime struct {
	opts             cli.Options
	translatorLayout hangul.KeyboardLayout
	engineLayout     *layout.Layout
	hangulName       string
	toggle           config.ToggleConfig
	database         backend.Database
	modes            []engine.ModeSpec
	fallback         emitter.Output
	ttyClient        *ttybridge.Client
	directCommit     bool
	deviceFD         int
	cleanups         []func()
}

func NewRuntime(opts cli.Options) *Runtime {
	return &Runtime{opts: opts, deviceFD: -1}
}

func (rt *Runtime) Run() error {
	defer rt.cleanup()

	if err := rt.prepareLayouts(); err != nil {
		return err
	}
	if err := rt.prepareToggle(); err != nil {
		return err
	}
	if err := rt.prepareDatabase(); err != nil {
		return err
	}
	if err := rt.prepareTTY(); err != nil {
		return err
	}
	if err := rt.openDevice(); err != nil {
		return err
	}
	if err := rt.buildEmitter(); err != nil {
		return err
	}
	if err := rt.buildModes(); err != nil {
		return err
	}

	eng, err := engine.NewEngine(rt.deviceFD, rt.modes, rt.toggle, rt.fallback)
	if err != nil {
		return err
	}

	if rt.opts.SocketPath == "" {
		rt.opts.SocketPath = common.DefaultSocketPath()
	}

	server, err := StartTranslationServer(rt.opts.SocketPath, rt.translatorLayout)
	if err != nil {
		return err
	}
	if server != nil {
		rt.registerCleanup(server.Close)
	}

	return rt.runEventLoop(eng, server)
}

func (rt *Runtime) prepareLayouts() error {
	translator, canonical, err := ResolveTranslatorLayout(rt.opts.LayoutName)
	if err != nil {
		return err
	}
	rt.translatorLayout = translator

	engineLayout, hangulName, err := ResolveEngineLayout(canonical, rt.opts.LayoutName, rt.opts.KeypairPath)
	if err != nil {
		return err
	}
	rt.engineLayout = engineLayout
	rt.hangulName = hangulName
	return nil
}

func (rt *Runtime) prepareToggle() error {
	cfg, err := config.ResolveToggleConfig(rt.opts.ToggleConfigPath)
	if err != nil {
		return err
	}
	ApplyModeOrder(&cfg, rt.opts.ModeOrder, rt.hangulName, rt.engineLayout != nil)
	rt.toggle = cfg
	return nil
}

func (rt *Runtime) prepareDatabase() error {
	if rt.opts.PinyinDBPath == "" {
		rt.database = backend.Database{}
		return nil
	}
	db, err := backend.LoadDatabase(rt.opts.PinyinDBPath)
	if err != nil {
		return err
	}
	rt.database = db
	return nil
}

func (rt *Runtime) prepareTTY() error {
	ttyPath := strings.TrimSpace(rt.opts.TTYPath)
	ptyPath := strings.TrimSpace(rt.opts.PTYPath)

	if ttyPath == "" {
		ttyPath = ttybridge.TTYPathHint()
		if ttyPath == "" {
			if detected, err := ttybridge.DetectTTYPath(); err == nil {
				ttyPath = detected
			}
		}
	}
	if ttyPath != "" {
		ttybridge.RememberTTYPath(ttyPath)
	}

	directCommit := ptyPath != "" || ttyPath != "" || rt.opts.SuppressHex
	if directCommit && ttyPath == "" && ptyPath == "" {
		directCommit = false
	}

	if directCommit && ttyPath != "" {
		if err := ttybridge.SpawnHelper(ttyPath); err != nil {
			return err
		}
		client, err := ttybridge.Attach()
		if err != nil {
			return err
		}
		rt.ttyClient = client
		rt.registerCleanup(func() { _ = client.Close() })
	}

	rt.directCommit = directCommit
	return nil
}

func (rt *Runtime) openDevice() error {
	devicePath := strings.TrimSpace(rt.opts.DevicePath)
	if devicePath == "" {
		detected, err := device.DetectKeyboardDevice()
		if err != nil {
			return err
		}
		devicePath = detected.Path
		fmt.Fprintf(os.Stderr, "hanfe: using keyboard %s (%s)\n", detected.Path, detected.Name)
	}

	fd, err := syscall.Open(devicePath, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", devicePath, err)
	}
	rt.deviceFD = fd
	rt.registerCleanup(func() {
		if rt.deviceFD >= 0 {
			syscall.Close(rt.deviceFD)
			rt.deviceFD = -1
		}
	})
	return nil
}

func (rt *Runtime) buildEmitter() error {
	hexCodes := layout.UnicodeHexKeycodes()
	fallback, err := emitter.Open(hexCodes, rt.ttyClient, rt.opts.PTYPath, rt.directCommit)
	if err != nil {
		return err
	}
	rt.fallback = fallback
	rt.registerCleanup(func() { _ = fallback.Close() })
	return nil
}

func (rt *Runtime) buildModes() error {
	modes, err := BuildModes(rt.toggle.ModeCycle, rt.engineLayout, rt.hangulName, rt.database)
	if err != nil {
		return err
	}
	rt.modes = modes
	return nil
}

func (rt *Runtime) runEventLoop(eng *engine.Engine, server *TranslationServer) error {
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
			rt.releaseDevice()
		}
	}
}

func (rt *Runtime) releaseDevice() {
	if rt.deviceFD >= 0 {
		syscall.Close(rt.deviceFD)
		rt.deviceFD = -1
	}
}

func (rt *Runtime) registerCleanup(fn func()) {
	if fn == nil {
		return
	}
	rt.cleanups = append([]func(){fn}, rt.cleanups...)
}

func (rt *Runtime) cleanup() {
	for _, fn := range rt.cleanups {
		fn()
	}
	rt.cleanups = nil
}
