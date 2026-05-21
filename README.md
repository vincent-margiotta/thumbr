# Thumbr

**Thumbr** is a terminal card box emulator for Luhmann-style zettelkasten notes. It walks a directory of `.txt` files, renders them as index cards in the terminal, and pulls the active card into an overlay for reading (content shown as plain text — no rendering).

## Influence

Notes are files. Thumbr keeps them that way — plain text on disk, no database, readable by any tool long after the app is gone. What it adds is the feel of a physical Zettelkasten: cards in a stack you thumb through with momentum, an overlay for reading, and Luhmann-style alphanumeric addressing for branching and continuing ideas. Navigation favors serendipity over search — random jumps, no hierarchy, no backlinks graph — because landing on an unexpected old card is often where connections form. When a note calls for a direct reply, the split-pane editor keeps source and response in view at once.

## Features
- **TUI Interface:** Built with Bubble Tea for a responsive terminal experience.
- **Luhmann Addressing:** Filename IS the card address. Continue (`c`), branch (`C`), and create the next integer root (`N`) using Luhmann alphanumeric addressing. Custom shell commands can override the derivation.
- **Focus Mode:** Pull cards into an overlay to read long content without distraction.
- **In-App Editor:** Browse, create, and edit notes without leaving Thumbr. Built-in vim-style editing (normal/insert/command modes) with `:w`/`ctrl+s` to save and `:q`/`:wq` to exit. When `c`/`C` creates a linked card, both notes open in a split pane (source top, new card bottom); `ctrl+w` switches focus between panes.
- **Organization:** Mark important cards (`m`) and toggle "marked-only" filters (`t`) — the digital equivalent of pulling slips or orienting cards sideways.
- **Fast Loads:** On an i7-4770HQ with SSD and warm cache, loading/sorting ~90k notes benchmarks at ~0.23s (`make bench`).

## Requirements
- **Go 1.24+**

## Quick Start

### Run from Source
The fastest way to try Thumbr is using the included sample notes:

```bash
# Run with default sample notes
make run

# Or build and run manually
make build
./thumbr samples/notes
````

### Install via Go

```bash
go install github.com/vincent-margiotta/thumbr/cmd/thumbr@latest
```

Or run directly from a cloned repo without installing:

```bash
go run ./cmd/thumbr .
```

## Usage

```
thumbr [options] [path]
```

- `path` is optional; defaults to the current working directory. Positional `path` beats the config file.
- Run `thumbr -h` to see all flags. `-v/--version` prints the embedded build version.

## Controls

Thumbr has three interaction modes: the **Stack** (browsing), the **Overlay** (reading), and the **Editor** (editing in-app).

### Stack

| Action | Keybinding | Notes |
| :--- | :--- | :--- |
| **Forward** | `k` / `Up` | Move deeper into the stack. Accelerates with rapid presses. |
| **Back** | `j` / `Down` | Move back toward the front. |
| **First card** | `g` `g` | Jump to the first card in the deck. |
| **Last card** | `G` | Jump to the last card. |
| **Jump to ~%** | `g` `1`–`9` | Jump to 10%–90% through the deck. |
| **Random** | `r` | Jump to a random card. |
| **Open overlay** | `Enter` | Pull the active card into the reading overlay. |
| **Edit (in-app)** | `e` | Open the active card in the built-in vim-style editor. If another editor is suspended, opens as a companion pane. Set `editMode: "external"` to redirect `e` to `$EDITOR` instead. |
| **Edit (external)** | `E` | Open the active card in `$EDITOR`. |
| **Continue** | `c` | Create a Luhmann continuation card (e.g. `16a` → `16a1`). Opens in a split pane by default (`autoSplitOnLink`). |
| **Branch** | `C` | Create a Luhmann sibling card (e.g. `16a` → `16b`). Opens in a split pane by default (`autoSplitOnLink`). |
| **Next root** | `N` | Create the next integer root card (e.g. `17` if `16` is highest). |
| **Mark card** | `m` | Toggle mark (`*`). |
| **Filter** | `t` | Toggle "Marked-Only" view. |
| **Reload** | `R` | Reload the card list from disk. |
| **Help** | `?` / `h` | Show current keybindings. |
| **Quit** | `q` / `Ctrl+c` | Quit. |

### Overlay

| Action | Keybinding | Notes |
| :--- | :--- | :--- |
| **Close** | `Enter` | Return to the stack. |
| **Scroll** | `j` / `k` | Scroll content line-by-line. |
| **Page** | `n` / `p` | Next / previous page (when content overflows). |

### Editor

The in-app editor uses vim-style modes.

#### Navigation

| Action | Keybinding | Notes |
| :--- | :--- | :--- |
| **Insert mode** | `i` / `a` | Enter insert mode before / after cursor. |
| **Append at EOL** | `A` | Enter insert mode at end of line. |
| **Open line below** | `o` | Insert new line below and enter insert mode. |
| **Open line above** | `O` | Insert new line above and enter insert mode. |
| **Normal mode** | `Esc` / `Ctrl+c` | Return to normal mode. |
| **Character** | `h` / `l` | Move left / right. |
| **Line** | `j` / `k` | Move down / up. |
| **Word** | `w` / `b` | Move to next / previous word start (crosses lines). |
| **Word end** | `e` | Move to end of current or next word (crosses lines). |
| **Line start / end** | `0` / `$` | Jump to start / end of current line. |
| **File top / bottom** | `gg` / `G` | Jump to first / last line of the file. |

#### Editing

| Action | Keybinding | Notes |
| :--- | :--- | :--- |
| **Delete char** | `x` | Delete character under cursor (saved to register). |
| **Replace char** | `r` | Replace character under cursor. |
| **Delete to EOL** | `D` | Delete from cursor to end of line (saved to register). |
| **Change to EOL** | `C` | Delete from cursor to end of line and enter insert mode. |
| **Delete line** | `dd` | Delete current line (saved to register). |
| **Delete word** | `dw` / `diw` | Delete to next word boundary / delete inner word (saved to register). |
| **Change word** | `cw` / `ciw` | Replace to next word boundary / replace inner word (enters insert mode). |
| **Yank line** | `yy` | Copy current line to register without deleting. |
| **Yank word** | `yw` / `yiw` | Copy to next word boundary / inner word to register. |
| **Paste** | `p` | Paste register after cursor (inline for word yanks, new line for line yanks). |
| **Undo** | `u` | Undo last change. |
| **Redo** | `Ctrl+r` | Redo last undone change. |
| **Dot repeat** | `.` | Repeat the last normal-mode change. |

#### Commands

| Action | Keybinding | Notes |
| :--- | :--- | :--- |
| **Save** | `:w` / `Ctrl+s` | Save without exiting. |
| **Quit** | `:q` | Close active pane (blocked if unsaved changes). Split → single; single → browse. |
| **Discard & quit** | `:q!` | Discard changes and close active pane. |
| **Save & quit** | `:wq` | Save and close active pane. |
| **Save & quit all** | `:wqa` | Save all open panes and exit the editor. |
| **Sort lines** | `:sort` | Sort all lines in the file alphabetically. |

#### Pane Management

| Action | Keybinding | Notes |
| :--- | :--- | :--- |
| **Switch pane** | `Ctrl+w` | Switch focus between top and bottom panes (split view only). |
| **Suspend editor** | `Ctrl+b` | Suspend editor and return to browse; press `e` to resume. |

> Keybindings can be customized via the configuration file. The debug overlay (`d`) is disabled by default; enable it with `enableDebugUI: true`.

## Configuration

Thumbr looks for a config file via the `--config` flag (supports JSON, YAML, TOML). Command line flags override config file values.

**Config search:** Thumbr looks for `config.json`, `config.yaml`, `config.yml`, or `config.toml` (in that order) in the current directory, then in `~/.thumbr/`, unless you pass `--config` to point somewhere else.

Precedence:

1. Positional path (if provided)
2. CLI flags
3. Config file values (via `--config` or auto-discovery above)
4. Built-in defaults

Tip: copy `config.example.json` to `config.json` and adjust only the fields you care about.
If you prefer inline documentation, use `config.annotated.toml` as a commented reference and copy settings into your own config (JSON/YAML/TOML all supported).

### Configuration (quick scan)

Common tweaks:
- **Layout/Styling:** `stackVisible`, `textWidth`, `colors`, border characters.
- **Bindings:** `bind*` keys to remap navigation, overlay, marks, etc.
- **Editor:** `editMode` (`"inapp"` / `"external"`) — redirect `e` to `$EDITOR`; `textWidth` for hard-wrap column; `autoSplitOnLink` for split-pane on `c`/`C`.

Full reference lives in `config.annotated.toml` (TOML with inline docs); JSON and YAML are also supported.

On panic, Thumbr writes a crash log to `~/.thumbr/crash.log` (override with `--crash-log`).


## Development

### Project Layout

  - `cmd/thumbr`: CLI entry point and Bubble Tea program initialization.
  - `internal/notes`: Card discovery, file parsing, and filtering logic.
  - `internal/ui`: The Bubble Tea model, input handling, rendering, and physics.
  - `samples/notes`: A test suite of notes (nested, hidden, long content) for smoke testing.

### Make Targets

Uses a local `.gocache` to keep your system clean.

  - `make install`: Install binary to `$GOBIN` / `$GOPATH/bin`.
  - `make run`: Build and run (defaults to `samples/notes`).
  - `make build`: Compile binary (embeds `git describe` version).
  - `make check`: Run lint (`go vet`) and tests.
  - `make test`: Run unit tests.
  - `make bench`: Run the 90k note load benchmark (override with `BENCH_NOTES_COUNT=...`; uses a local `.gocache` and auto-cleans its temp data).
  - `make fmt`: Format code.
  - `make clean`: Remove artifacts.

### Docker

Thumbr includes a multi-stage Dockerfile for development and release.

**1. Development Shell**
Mounts your local source code into a container with tools installed.

```bash
docker build -t thumbr-dev --target dev .
docker run --rm -it -v "$PWD":/app -w /app thumbr-dev
```

**2. Release Image**
Builds a lightweight image for running Thumbr.

```bash
# Build
docker build -t thumbr --build-arg VERSION=$(git describe --tags --dirty --always) .

# Run (Mount your notes to /notes)
docker run --rm -v "$PWD/samples/notes":/notes thumbr /notes
```

## Roadmap / Known Gaps

  - Release pipelines (GoReleaser/Homebrew) — on hold pending demand.
