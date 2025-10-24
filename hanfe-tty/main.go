package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"

	"hanfe/hanfe-tty/internal/automata"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hanfe-tty: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	composer := automata.NewComposer()
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	fd := int(os.Stdin.Fd())
	var restore func() error
	if term.IsTerminal(fd) {
		state, err := term.MakeRaw(fd)
		if err != nil {
			return err
		}
		restore = func() error { return term.Restore(fd, state) }
		defer func() {
			if restore != nil {
				_ = restore()
			}
		}()
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)

	preedit := ""

	for {
		select {
		case <-sigs:
			if preedit != "" {
				erase(writer, preedit)
				preedit = ""
			}
			if commit := composer.Flush(); commit != "" {
				writer.WriteString(commit)
			}
			writer.Flush()
			return nil
		default:
		}

		r, _, err := reader.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if preedit != "" {
					erase(writer, preedit)
				}
				if commit := composer.Flush(); commit != "" {
					writer.WriteString(commit)
				}
				writer.Flush()
				return nil
			}
			return err
		}

		switch r {
		case 0x03: // Ctrl+C
			if preedit != "" {
				erase(writer, preedit)
				preedit = ""
			}
			if commit := composer.Flush(); commit != "" {
				writer.WriteString(commit)
			}
			writer.Flush()
			return nil
		case '\r':
			r = '\n'
		}

		if r == 0x7f || r == '\b' {
			if preedit != "" {
				erase(writer, preedit)
			}
			if updated, ok := composer.Backspace(); ok {
				preedit = updated
				if preedit != "" {
					writer.WriteString(preedit)
				}
				writer.Flush()
				continue
			}
			writer.WriteString("\b \b")
			writer.Flush()
			preedit = ""
			continue
		}

		if r == '\n' {
			if preedit != "" {
				erase(writer, preedit)
				preedit = ""
			}
			if commit := composer.Flush(); commit != "" {
				writer.WriteString(commit)
			}
			writer.WriteRune('\n')
			writer.Flush()
			continue
		}

		if preedit != "" {
			erase(writer, preedit)
			preedit = ""
		}

		commit, nextPreedit := composer.Type(r)
		if commit != "" {
			writer.WriteString(commit)
		}
		if nextPreedit != "" {
			writer.WriteString(nextPreedit)
			preedit = nextPreedit
		} else {
			preedit = ""
		}

		writer.Flush()
	}
}

func erase(w *bufio.Writer, text string) {
	width := runewidth.StringWidth(text)
	if width == 0 {
		return
	}
	for i := 0; i < width; i++ {
		w.WriteRune('\b')
	}
	for i := 0; i < width; i++ {
		w.WriteRune(' ')
	}
	for i := 0; i < width; i++ {
		w.WriteRune('\b')
	}
}
