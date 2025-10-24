# hanfe

`hanfe` is a Linux-wide Hangul IME daemon written in Go. The process grabs a
physical keyboard through evdev, composes Hangul syllables on the fly, and
re-injects finished text through a fallback `uinput` device so GUI applications
without proper input-context support still receive composed characters.

## Features

- **Global interception** – Grabs a keyboard evdev node and forwards events
  through a virtual device, making Hangul available to any application that can
  read key events.
- **Hangul composition** – Includes 두벌식 (`dubeolsik`) and 세벌식 390
  (`sebeolsik-390`) layouts with full consonant/vowel composition logic.
- **Configurable toggle keys** – Choose one or more toggle chords (e.g.
  `alt_r`, `hangul`, `ctrl+space`, `alt+space`). Each chord flips between Hangul
  and Latin modes when pressed.
- **Daemon friendly** – Runs as a background service by default while keeping a
  `--no-daemon` flag for foreground debugging. `--tty` mirroring locks to the
  active terminal automatically, with an optional `--pty` override for custom
  sessions.
- **Dedicated TTY composer** – A new `hanfe-tty` binary offers a raw terminal
  translator using fcitx/libhangul-style automata so shells receive composed
  Hangul without relying on `TIOCSTI` mirroring.
- **Autostart wrapper** – `hanfe-autostart` launches both daemons so the IME and
  the TTY helper start together with a single command.

## Building

Go 1.22 or newer is recommended.

```bash
go build ./...
```

## Running

Root (or proper udev rules/capabilities) is typically required to access
`/dev/input/event*` and `/dev/uinput`.

```bash
sudo ./hanfe --no-daemon
```

Useful command-line options for `hanfe`:

- `--device PATH` – Explicit evdev keyboard path (auto-detected when omitted).
- `--layout NAME` – Keyboard layout (`dubeolsik` or `sebeolsik-390`).
- `--toggle-config PATH` – Path to a toggle configuration file (defaults to
  `./toggle.ini` when present).
- `--tty PATH` – Mirror committed text into a TTY using `TIOCSTI` via a helper
  daemon (the controlling TTY is detected automatically when omitted and the
  daemon exits if no terminal is available).
- `--pty PATH` – Mirror committed text into a PTY without exposing the Unicode hex sequence.
- `--no-hex` – Skip Unicode hex injection and rely on the TTY/PTY helper for
  direct Hangul output. This mode is enabled automatically when no `DISPLAY`
  or `WAYLAND_DISPLAY` is present.
- `--daemon` / `--no-daemon` – Control background execution (daemon mode is the
  default).
- `--list-layouts` – Print available layouts and exit.
- `-h`, `--help` – Show usage information.

### `hanfe-tty`

`hanfe-tty` focuses on direct terminal composition. It keeps STDIN in raw mode,
translates Latin key strokes into Hangul syllables with the same state machine
fcitx/libhangul uses, and writes the composed text straight to STDOUT. This is
handy for remote shells or applications that cannot receive fallback `uinput`
events.

```bash
./hanfe-tty
```

Press `Ctrl+C` to terminate; the composer flushes any pending syllable before
exiting.

### `hanfe-autostart`

To launch both the IME daemon and the TTY helper together, run:

```bash
hanfe-autostart
```

The wrapper looks up `hanfe` and `hanfe-tty` on `PATH`, starts them, and shuts
down the remaining process if either one exits or a termination signal arrives.

## Configuration

Toggle behavior is controlled through a minimal INI file. Example `toggle.ini`:

```ini
[toggle]
keys = alt_r, hangul, ctrl+space
default_mode = hangul
```

Each entry under `keys` is a comma-separated chord. A chord can be a single key
(`hangul`, `alt_r`) or a modifier plus trigger (`ctrl+space`, `alt+space`).
Recognised modifiers are `alt`, `alt_l`, `alt_r`, `ctrl`, `ctrl_l`, `ctrl_r`,
`shift`, and `meta`. The last token in a chord must resolve to a single key.

`default_mode` chooses the initial input mode (`hangul` or `latin`). When the
file is missing or malformed the daemon falls back to the internal defaults of
`alt_r` and `hangul` toggles with Hangul mode enabled.

## Testing

```bash
go test ./...
```
