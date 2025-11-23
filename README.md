# Thumbr

Terminal note browser that lets you thumb through a stack of Markdown cards. Thumbr walks a directory of `.md` files, renders them as index cards in the terminal, and pulls the active card into an overlay for reading.

## Quick start
- Requires Go 1.24 or newer.
- Run directly: `go run ./cmd/thumbr [path/to/notes]` (defaults to the current directory).
- Optional: `make run ARGS=/path/to/notes` to build and launch in one step; `make build` to produce the `thumbr` binary (embeds `git describe` as the version when available).
- Sample config: `config.example.json` shows `includeExts` and `ignoreGlobs`; pass with `--config path/to/config`.
- Sample notes: `samples/notes` contains a handful of cards (including nested and hidden) for quick smoke testing; point Thumbr at that path to try it.

## CLI options
- Positional `path`: root directory to scan (defaults to current directory).
- `-v, --version`: print the build version.
- `--random-seed <int>`: set RNG seed for reproducible `r` jumps (defaults to a time-based seed).
- `--alt-screen` / `--alt-screen=false` or `--no-alt-screen`: toggle use of the terminal alt screen.
- `--config path/to/config.(json|yaml|toml)`: optional config with fields `noteRoot`, `randomSeed`, `altScreen`, `includeExts`, `ignoreGlobs`; CLI flags take precedence. Paths are resolved from the current working directory—there is no default search. See `config.example.json` for a starting point.
- `--include-exts .md,.txt`: comma-separated list of file extensions to include (defaults to `.md` if not set).
- `--ignore pattern1,pattern2`: comma-separated glob patterns to skip (matched against full path and basename).

## Controls
- `k`/`up`, `j`/`down`: move through the stack (accelerates with rapid presses). `k/up` moves into the stack; `j/down` moves back out.
- `enter`: open or close the overlay view for the current card.
- `r`: jump to a random card.
- `q` or `ctrl+c`: quit; `esc` leaves the overlay.
- `m`: mark/unmark the current card.
- `t`: toggle marked-only filter (shows only marked cards when on).
- `?` or `h`: toggle the help overlay with keybindings.
- Marked cards show a `*` marker; unmarked cards dim when any marks exist.

## Notes and filenames
- Markdown files are loaded by default; configure extensions with `--include-exts` or config.
- Directories are walked recursively; use `--ignore`/`ignoreGlobs` to skip paths (globs match basename or full path).
- Filenames shaped like `ID Title.md` are parsed into an `ID` and `Title` (e.g. `1.1a Some idea.md`). A lone title such as `Draft.md` also works; the ID is left empty.
- File contents are shown best-effort in the overlay; unreadable files are skipped without stopping the scan.

## Project layout
- `cmd/thumbr/main.go`: CLI entry point; flags/config, RNG seeding, Bubble Tea program.
- `internal/notes`: card discovery, filename parsing, include/ignore filters.
- `internal/ui`: Bubble Tea model, input handling, geometry, rendering, help overlay.
- `Makefile`: build/test/lint targets; `config.example.json`: config reference; `Dockerfile`: dev shell + release image.

## Development
- `make fmt` (gofmt), `make lint` (go vet), `make test` (with local `GOCACHE`, `-count=1`), `make check` (lint+test).
- Run tests manually: `GOCACHE=$(pwd)/.gocache go test ./...`
- Keep modules tidy: `go mod tidy`
- Quick smoke: `make build` then `./thumbr obsidian/Main` (or your notes path).

## Containers
- Dev shell: `docker build -t thumbr-dev --target dev .` then `docker run --rm -it -v "$PWD":/app -w /app thumbr-dev`
- Build release image (default target): `docker build -t thumbr --build-arg VERSION=$(git describe --tags --dirty --always 2>/dev/null || echo dev) .`
- Run built image: `docker run --rm thumbr --help`
- Quick sample: `docker run --rm -v "$PWD/samples/notes":/notes thumbr /notes`
 
## Requirements
For the full set of functional and non-functional requirements, see `REQUIREMENTS.md`.
