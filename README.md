# Thumbr

**Thumbr** is a terminal card box emulator for plain-text notes. It walks a directory of note files (`.txt` by default; extensions are configurable), renders them as index cards in the terminal, and pulls the active card into an overlay for reading (content is shown as plain text—no rendering).

## Influence

Thumbr favors the analog experience over digital workflow features. It emulates flipping through a physical card box: filename-as-title cards in a stack, rapid “thumbing” with acceleration, and an overlay that feels like pulling a card to the front. Navigation is biased toward serendipity (random jumps) and branching hub notes instead of rigid hierarchies or backlinks. The goal is to keep your notes as simple files while giving you an intuitive, tactile way to browse and continue ideas.

## Features
- **TUI Interface:** Built with Bubble Tea for a responsive terminal experience.
- **Simple Titles:** Uses the filename (minus extension) as the card title; order follows directory traversal.
- **Focus Mode:** Pull cards into an overlay to read long content without distraction.
- **Organization:** Mark important cards, toggle "marked-only" filters, and jump to random notes.
- **Multi-Box Sessions:** Switch note roots on the fly and create new notes from inside Thumbr.
- **Persistent Marks/Filters:** Marks and filter state stick to each box during a session (cleared when you quit).
- **Highly Configurable:** Tune extensions, sorting, layout, colors, and keys via flags or config files.
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

### Install/Run via Go

```bash
# Run directly on your current directory
go run ./cmd/thumbr .
```

## Usage

```
thumbr [options] [path]
```

- `path` is optional; defaults to the current working directory. Positional `path` beats the config file.
- Run `thumbr -h` to see all flags. `-v/--version` prints the embedded build version.

## Controls

Thumbr has two main interaction modes: the **Stack** (browsing files) and the **Overlay** (reading content).

### Navigation & Actions

| Context | Action | Keybindings | Notes |
| :--- | :--- | :--- | :--- |
| **Stack** | **Forward/In** | `k` / `Up` | Moves deeper into the stack. Accelerates with rapid presses. |
| **Stack** | **Back/Out** | `j` / `Down` | Moves back out of the stack. |
| **Stack** | **Random** | `r` | Jump to a random card. |
| **Stack** | **Mark Card** | `m` | Toggles mark (`*`). Unmarked cards dim if others are marked. |
| **Stack** | **Filter** | `t` | Toggle "Marked-Only" view. |
| **Overlay** | **Open/Close** | `Enter` | Opens active card / Closes overlay. |
| **Overlay** | **Scroll** | `j` / `k` | Scroll content line-by-line. |
| **Overlay** | **Page** | `n` / `p` | Next/Previous page (if content overflows). |
| **System** | **Help** | `?` / `h` | Shows current keybindings. |
| **System** | **Debug** | `d` | Toggle debug stats (counts, paging, nav metrics). |
| **System** | **Switch Box** | `b` | Open a different root (box) within the same session. |
| **System** | **New File** | `a` | Create a new file in a box and open it in your editor. |
| **System** | **Open in Editor** | `e` | Open the active card in `$EDITOR` (or system default). |
| **System** | **Quit** | `q` / `Ctrl+c` | Quit app. (`Esc` also quits, or closes overlay if open). |

> **Note:** Keybindings can be customized via the configuration file.

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

### Debug Overlay & Crash Logs

- The debug overlay is gated: enable it via `--enable-debug-ui` or `enableDebugUI: true` in your config to use the `d` keybinding. Disabled by default.
- On panic, Thumbr writes a crash log to `~/.thumbr/crash.log` (override with `--crash-log`). Only touched when a panic occurs.

### Configuration (quick scan)

Common tweaks:
- **Files:** `includeExts` (defaults to `.txt`), `ignoreGlobs`.
- **Sorting:** `sortMode`, `sortPattern`, `sortPatternFirst` to control natural vs lexical ordering and grouping.
- **Layout/Styling:** `stackVisible`, `cardWidthFrac`, `colors`, border characters.
- **Bindings:** `bind*` keys to remap navigation, overlay, marks, etc.

Full reference lives in `config.example.json` (JSON) and is supported via YAML/TOML too.


## Development

### Project Layout

  - `cmd/thumbr`: CLI entry point and Bubble Tea program initialization.
  - `internal/notes`: Card discovery, file parsing, and filtering logic.
  - `internal/ui`: The Bubble Tea model, input handling, rendering, and physics.
  - `samples/notes`: A test suite of notes (nested, hidden, long content) for smoke testing.

### Make Targets

Uses a local `.gocache` to keep your system clean.

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

  - Live box/config reloading.
  - Performance tuning for large boxes (target: 90k notes in \<2s).
  - Release pipelines (GoReleaser/Homebrew).
