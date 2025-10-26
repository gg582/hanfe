package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/gg582/hangul-logotype/hangul"

	"hanfe/internal/common"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hanfe-tty: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	defaultSocket := common.DefaultSocketPath()

	layoutName := flag.String("layout", common.DefaultLayoutName, fmt.Sprintf("keyboard layout (%s)", strings.Join(common.AvailableLayouts(), ", ")))
	socketPath := flag.String("socket", defaultSocket, "unix socket used to talk with the hanfe daemon")
	localOnly := flag.Bool("local", true, "convert locally without contacting the hanfe daemon")
	remote := flag.Bool("remote", false, "force use of the hanfe daemon for conversion")
	flag.Parse()

	if *remote {
		*localOnly = false
	}

	layout, _, err := common.ResolveLayout(*layoutName)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	warned := false

	for scanner.Scan() {
		line := scanner.Text()
		var converted string
		if !*localOnly {
			converted, err = translateViaSocket(*socketPath, line)
			if err != nil {
				if !warned {
					fmt.Fprintf(os.Stderr, "hanfe-tty: falling back to local conversion: %v\n", err)
					warned = true
				}
				*localOnly = true
			}
		}
		if *localOnly {
			converted = translate(layout, line)
		}
		if _, err := writer.WriteString(converted); err != nil {
			return err
		}
		if err := writer.WriteByte('\n'); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func translateViaSocket(socketPath, text string) (string, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if _, err := fmt.Fprintln(conn, text); err != nil {
		return "", err
	}

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(response, "\n"), nil
}

func translate(layout hangul.KeyboardLayout, text string) string {
	typer := hangul.NewLogoTyper().WithLayout(layout)
	typer.WriteString(text)
	return string(typer.Result())
}
