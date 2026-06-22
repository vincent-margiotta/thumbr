# Thumbr

**Thumbr** is a terminal card box emulator for Luhmann-style zettelkasten notes. It walks a directory of `.txt` files, renders them as index cards in the terminal, and pulls the active card into an overlay for reading (content shown as plain text — no rendering).

## Influence

Notes are files. Thumbr keeps them that way — plain text on disk, no database, readable by any tool long after the app is gone. What it adds is the feel of a physical Zettelkasten: cards in a stack you thumb through, an overlay for reading, and Luhmann-style alphanumeric addressing for branching and continuing ideas. Navigation favors serendipity over search — random jumps, bisect, no hierarchy, no backlinks graph — because landing on an unexpected old card is often where connections form. When a note calls for a direct reply, the split-pane editor keeps source and response in view at once.

## Features

- **TUI Interface:** Built with Bubble Tea for a responsive terminal experience.
- **Luhmann Addressing:** Filename IS the card address. Continue (`c`), branch (`C`), and create the next integer root (`N`) using Luhmann alphanumeric addressing. Custom shell commands can override the derivation.
- **Focus Mode:** Pull cards into an overlay to read long content without distraction.
- **Card Backs:** Cards can have a back side, separated by a `---back---` line. Flip between sides in the overlay with `f`; `e` opens whichever side is visible.
- **In-App Editor:** Browse, create, and edit notes without leaving Thumbr. Built-in vim-style editing (normal/insert/command modes) with `:w`/`ctrl+s` to save and `:q`/`:wq` to exit. When `c`/`C` creates a linked card, both notes open in a split pane (source top, new card bottom); `ctrl+w` switches focus between panes.
- **Multi-Box:** Open multiple note directories simultaneously with `--box`/`--free-box` and switch between them with `b`.
- **Organization:** Mark important cards (`m`) and toggle "marked-only" filters (`t`) — the digital equivalent of pulling slips or orienting cards sideways.
- **Physical Printing:** `scripts/print_cards.py` lays out cards on 8.5"×11" paper (two 4"×6" landscape cards per sheet) as a print-ready PDF. Supports card backs and duplex printing.
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
```

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

### Multiple Boxes

Open several note directories at once. Thumbr shows one box at a time; `b` opens a picker to switch between them.

```bash
thumbr --box ~/notes/zettel --box ~/notes/journal
thumbr --box ~/notes/zettel --free-box ~/notes/scratch
```

- `--box <dir>` — open a directory in standard (Luhmann) mode.
- `--free-box <dir>` — open a directory in free mode (disables `c`/`C`; `N` prompts for any filename).
- Flags can be interleaved in any order to control which box is active first.

## Controls

Thumbr has three interaction modes: the **Stack** (browsing), the **Overlay** (reading), and the **Editor** (editing in-app).

### Stack

| Action | Keybinding | Notes |
| :--- | :--- | :--- |
| **Forward** | `k` / `Up` | Move deeper into the stack. |
| **Back** | `j` / `Down` | Move back toward the front. |
| **Chunk forward** | `J` | Jump forward ~7 cards (configurable; varies slightly each press). |
| **Chunk back** | `K` | Jump backward ~7 cards. |
| **Bisect forward** | `]` | Jump to the midpoint between cursor and end. Repeat to converge. |
| **Bisect back** | `[` | Jump to the midpoint between cursor and start. |
| **First card** | `g` `g` | Jump to the first card in the deck. |
| **Last card** | `G` | Jump to the last card. |
| **Random** | `r` | Jump to a random card. |
| **Open overlay** | `Enter` | Pull the active card into the reading overlay. |
| **Edit (in-app)** | `e` | Open the active card in the built-in vim-style editor. If another editor is suspended, opens as a companion pane. Set `editMode: "external"` to redirect `e` to `$EDITOR` instead. |
| **Edit (external)** | `E` | Open the active card in `$EDITOR`. |
| **Continue** | `c` | Create a Luhmann continuation card (e.g. `16a` → `16a1`). Opens in a split pane by default (`autoSplitOnLink`). |
| **Branch** | `C` | Create a Luhmann sibling card (e.g. `16a` → `16b`). Opens in a split pane by default (`autoSplitOnLink`). |
| **Next root** | `N` | Create the next integer root card (e.g. `17` if `16` is highest). |
| **Mark card** | `m` | Toggle mark (`*`). |
| **Filter** | `t` | Toggle "Marked-Only" view. |
| **Box picker** | `b` | Switch between open boxes (when more than one is loaded). |
| **Reload** | `R` | Reload the card list from disk. |
| **Help** | `?` / `h` | Show current keybindings. |
| **Quit** | `q` | Press twice within 3 seconds to confirm quit. `Ctrl+c` quits immediately. |

### Overlay

| Action | Keybinding | Notes |
| :--- | :--- | :--- |
| **Close** | `Esc` / `Enter` | Return to the stack. |
| **Scroll** | `j` / `k` | Scroll content line-by-line. |
| **Page** | `n` / `p` | Next / previous page (when content overflows). |
| **Flip** | `f` | Flip to the back side of the card (or front, if already flipped). If no back exists, shows a prompt; press `e` to write one. |
| **Edit** | `e` | Open whichever side is currently visible in the in-app editor. Saving reconstructs the full file automatically. |

### Editor

The in-app editor uses vim-style modes.

#### Navigation

| Action | Keybinding | Notes |
| :--- | :--- | :--- |
| **Insert mode** | `i` / `a` | Enter insert mode before / after cursor. |
| **Append at EOL** | `A` | Enter insert mode at end of line. |
| **Open line below** | `o` | Insert new line below and enter insert mode. |
| **Open line above** | `O` | Insert new line above and enter insert mode. |
| **Normal mode** | `Esc` / `Ctrl+[` | Return to normal mode. |
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
| **Delete to end** | `dG` | Delete from current line to end of file. |
| **Delete to top** | `dgg` | Delete from current line to start of file. |
| **Delete word** | `dw` / `diw` | Delete to next word boundary / delete inner word. |
| **Change word** | `cw` / `ciw` | Replace to next word boundary / replace inner word (enters insert mode). |
| **Yank line** | `yy` | Copy current line to register. |
| **Yank to end** | `yG` | Yank from current line to end of file. |
| **Yank to top** | `ygg` | Yank from current line to start of file. |
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
| **Save & quit** | `:wq` / `:x` | Save and close active pane. |
| **Save & quit all** | `:wqa` | Save all open panes and exit the editor. |
| **Sort lines** | `:sort` | Sort all lines in the file alphabetically. |
| **Fill char** | `:fill [c]` | Expand or contract a run of character `c` on the current line so the total line length fits the card face width. Default char is `.`. |
| **Rename file** | `:rename <name>` | Rename the current file (no extension needed). Does not update links. |
| **Edit back** | `:back` | Switch to editing the back face of the card. |
| **Edit back (portrait)** | `:back:portrait` | Switch to editing the back face in portrait orientation. |
| **Edit front** | `:front` | Switch to editing the front face of the card. |
| **Help** | `:help` | Show the full command reference overlay. Any key closes it. |

#### Pane Management

| Action | Keybinding | Notes |
| :--- | :--- | :--- |
| **Switch pane** | `Ctrl+w` | Switch focus between top and bottom panes (split view only). |
| **Suspend editor** | `Ctrl+b` | Suspend editor and return to browse; press `e` to resume. |

#### Count Prefix

Prefix any normal-mode motion or insert command with a count to repeat it: `5j` moves down 5 lines, `3x` deletes 3 characters, `10i.<Esc>` inserts ten dots. The count is displayed in the footer while you type it.

> Keybindings can be customized via the configuration file. The debug overlay (`d`) is disabled by default; enable it with `enableDebugUI: true`.

## Card Backs

A card can have a back side by adding a `---back---` line on its own:

```
The front of the card.

Some notes here.
---back---
The back of the card.

Additional thoughts, added later.
```

The `↻` indicator appears on the active stack card and in the overlay header when a back side exists. In the overlay, `f` flips between sides. Pressing `e` opens whichever side is currently visible; saving reconstructs the full file transparently.

#### Portrait backs

For notes that work better in portrait orientation (taller than wide), use `---back:portrait---` instead:

```
Front of the card (landscape).
---back:portrait---
Back of the card (portrait).
```

When editing or viewing a portrait back, the card dimensions use a 4:3 aspect ratio. Use `:back:portrait` in the editor to create or convert a back section to portrait orientation.

Luhmann rarely used card backs. When you do, keep the same discipline: one idea per side.

## Printing

`scripts/print_cards.py` generates a print-ready PDF. Two 4"×6" landscape cards are laid out per 8.5"×11" sheet with corner tick marks as cutting guides. Cards are sorted in natural Luhmann order.

```bash
pip install reportlab   # one-time setup

# Print an entire box
python3 scripts/print_cards.py ~/notes/zettel out.pdf

# Print a specific list of cards (paths from stdin)
cat changed.txt | python3 scripts/print_cards.py - out.pdf
```

Cards that exceed the card face (16 lines at the default font size) are refused with an error and must be split before printing.

### Duplex (two-sided) printing

With `--duplex`, the PDF is structured for **long-edge** duplex printing: page 1 carries fronts, page 2 carries backs in the same slot positions, so the backs land physically behind their fronts after the printer flips the sheet. Cut horizontally to produce two-sided cards.

```bash
python3 scripts/print_cards.py ~/notes/zettel out.pdf --duplex
```

Cards without a back side print normally; their back slot on page 2 is left blank.

### Registration correction

If the back side lands slightly off-center relative to the front, use `--back-offset` to compensate. Values are in millimetres; positive = right / up.

```bash
python3 scripts/print_cards.py ~/notes/zettel out.pdf --duplex --back-offset -2,-2
```

Adjust in 1–2mm increments until the cut cards align. Once dialled in, the values are stable for a given printer.

### Printing only changed cards

Pass `-` as the directory to read a newline-separated list of file paths from stdin. Combine with any tool that identifies changed cards (e.g. `git log --name-only`) to print only what's new:

```bash
git log --pretty=format: --name-only --diff-filter=AM --since="2024-01-01" -- "*.txt" \
  | sort -u \
  | sed "s|^|$(git rev-parse --show-toplevel)/|" \
  | python3 scripts/print_cards.py - out.pdf --duplex --back-offset -2,-2
```

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
- **Card size limit:** Disabled with `--no-card-limit`; applies globally regardless of free mode.

Full reference lives in `config.annotated.toml` (TOML with inline docs); JSON and YAML are also supported.

On panic, Thumbr writes a crash log to `~/.thumbr/crash.log` (override with `--crash-log`).

## Development

### Project Layout

- `cmd/thumbr`: CLI entry point and Bubble Tea program initialization.
- `internal/notes`: Card discovery, file parsing, and filtering logic.
- `internal/ui`: The Bubble Tea model, input handling, and rendering.
- `samples/notes`: Sample notes for smoke testing.
- `samples/print-test`: Minimal two-card set for testing the print script.
- `scripts/`: Utility scripts. `print_cards.py` generates print-ready PDFs.

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

# Run with no arguments — prints --help (the image default CMD)
docker run --rm thumbr
```

## Roadmap / Known Gaps

- Release pipelines (GoReleaser/Homebrew) — on hold pending demand.
