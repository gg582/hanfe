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
  (`sebeolsik-390`) layouts with full consonant/vowel composition logic, plus a
  built-in Katakana layout (`kana-86`) and a Latin helper map for other
  backends.
- **Configurable toggle keys** – Choose one or more toggle chords (e.g.
  `alt_r`, `hangul`, `ctrl+space`, `alt+space`). Each chord flips between Hangul
  and Latin modes when pressed.
- **Multilingual profiles** – Define ordered input profiles that cycle with the
  toggle key so sequences like `default → dubeolsik → kana-86` are one shortcut
  away. Profiles support three or more layouts and reuse the same toggle key.
- **Custom key pairs** – Override individual key outputs per profile so
  favourites, macros, or extra glyphs can be bound without recompiling.
- **Database-driven backends** – Attach dictionary files (e.g. Pinyin → Hanzi)
  as IME backends so Latin keystrokes can commit characters from lookup tables.
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
- `--layout NAME` – Keyboard layout (`dubeolsik`, `sebeolsik-390`, `kana-86`, or
  `latin`).
- `--toggle-config PATH` – Path to a toggle configuration file (defaults to
  `./toggle.ini` when present).
- `--profile-config PATH` – Path to a multilingual profile file (defaults to
  `./profiles.ini` when present).
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

Launch both daemons and automatically prefer the TTY helper whenever a terminal
window has focus:

```bash
hanfe-autostart
```

The wrapper resolves `hanfe` and `hanfe-tty` on `PATH`, starts them, and keeps
both under supervision. It watches the active X11 window (falling back to the
current console session) and sends `SIGSTOP`/`SIGCONT` to ensure only the
appropriate process is active:

- **Terminal focus** – `hanfe-tty` is resumed, `hanfe` is paused, and the helper
  reads/writes directly against the controlling TTY so input is altered instead
  of mirrored through uinput.
- **Non-terminal focus** – `hanfe` resumes normal uinput handling while
  `hanfe-tty` is kept suspended.

The monitor recognises common terminal emulators such as GNOME Terminal, KGX,
Konsole, Alacritty, Kitty, WezTerm, Ghostty, Foot, Tilix, Terminator, Yakuake,
Guake, and other well-known variants by class, title, or owning process name.
If no X11 session is available the wrapper assumes a console is active and keeps
`hanfe-tty` enabled by default.

## Configuration

### Toggle keys

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

`default_mode` chooses the initial input mode (`hangul`, `latin`, `kana`, or
`database`). When the file is missing or malformed the daemon falls back to the
internal defaults of `alt_r` and `hangul` toggles with Hangul mode enabled.

### Profiles

`profiles.ini` defines the ordered list of input profiles that cycle when the
toggle chord is pressed. Each profile specifies the mode, layout, optional
dictionary backend, and any custom key overrides. Example `profiles.ini`:

```ini
[profiles]
order = default, dubeolsik, kana86

[profile default]
mode = latin
layout = latin

[profile dubeolsik]
mode = hangul
layout = dubeolsik

[profile kana86]
mode = kana
layout = kana-86
pair = shift+key_a:text:ァ

[profile pinyin]
mode = database
layout = latin
database = third_party/db/pinyin.tsv
pair = key_semicolon:text:；
```

- `order` lists the profile names in the cycle order. Pressing the toggle key
  repeatedly walks this list and wraps at the end.
- Each `[profile NAME]` block sets `mode`, `layout`, and optional `database`
  values. `mode` accepts `latin`, `hangul`, `kana`, or `database`.
- `pair` lines override specific keys. Prefix with `shift+` to target the shifted
  variant. Payloads can be `text:<string>` or `jamo:<glyph>` for Hangul jamo.

The `--layout` CLI flag overrides the layout for every Hangul profile, keeping
legacy behaviour intact. Database profiles expect a simple tab-separated file
mapping search strings to committed text; `third_party/db/pinyin.tsv` contains a
starter Pinyin dictionary.

## Testing

```bash
go test ./...
```
