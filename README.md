# hanfe

`hanfe` is a terminal friendly Hangul composer written in Go. The tool relies on
ready-made Go packages for keyboard handling, Hangul syllable composition, and
INI configuration parsing so you can compose Korean text without wiring low level
Linux device code yourself.

The binary opens the current TTY in raw mode using
[`github.com/eiannone/keyboard`](https://github.com/eiannone/keyboard), maps
Dubeolsik key sequences through data shipped with
[`github.com/suapapa/go_hangul`](https://github.com/suapapa/go_hangul), and lets
you configure the toggle key via a simple INI file parsed by
[`github.com/go-ini/ini`](https://github.com/go-ini/ini).

## Features

- **Hangul composition** – Two-beolsik layout with combined vowels/finals handled
  by the embedded `go_hangul` tables.
- **Mode toggle** – Switch between English and Hangul modes with a configurable
  key (default: `Ctrl+Space`).
- **Terminal UI** – See the current buffer and mode status while typing in the
  terminal. Press `Ctrl+C` or `Esc` to exit.

## Building

Go 1.22 or newer is recommended.

```bash
go build ./...
```

The resulting binary lives at `./hanfe` when you build `cmd/hanfe`.

```bash
go build -o hanfe ./cmd/hanfe
```

## Running

By default the application looks for a `toggle.ini` file in the working
directory. Missing files simply fall back to internal defaults.

```bash
go run ./cmd/hanfe --layout dubeolsik
```

Command-line options:

- `--config PATH` – Path to an INI file that describes the toggle key and layout.
- `--layout NAME` – Override the layout declared in the config (currently only
  `dubeolsik` is available).

While running, use the configured toggle key to switch between Hangul and
English input. Hit `Enter` to print the current buffer on a new line.

## Configuration

The configuration file uses a very small INI subset. Example:

```ini
[toggle]
key = ctrl+space

[layout]
name = dubeolsik
```

The `[toggle]` section controls the key that flips between Hangul and English
modes. Supported values are `ctrl+space`, `tab`, `space`, `enter`, or a single
character (for example `` ` ``). The `[layout]` section chooses the keyboard
layout.

## Testing

```bash
go test ./...
```
