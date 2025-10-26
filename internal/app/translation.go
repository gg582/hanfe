package app

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/gg582/hanfe/internal/common"
	"github.com/gg582/hangul-logotype/hangul"
)

type TranslationServer struct {
	listener net.Listener
	socket   string
	errCh    chan error
}

func StartTranslationServer(path string, layout hangul.KeyboardLayout) (*TranslationServer, error) {
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
	srv := &TranslationServer{listener: listener, socket: path, errCh: make(chan error, 1)}
	go func() {
		srv.errCh <- serveTranslations(listener, layout)
		close(srv.errCh)
	}()
	return srv, nil
}

func (s *TranslationServer) Close() {
	if s == nil {
		return
	}
	s.listener.Close()
	for range s.errCh {
	}
	_ = os.Remove(s.socket)
}

func (s *TranslationServer) Err() <-chan error {
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
