# Changelog

## v0.1.0 -- 2026-06-22

Initial public release.

### TUI

- Stack view: cards rendered as physical 4"x6" index cards, sized from `textWidth` and terminal dimensions.
- Content preview on stack cards with soft-wrap and consistent margins.
- Focus overlay: pull the active card into a centered reading view with scroll and page navigation.
- Marks (`m`) and marked-only filter (`t`) -- the digital equivalent of tabbing or orienting a card sideways.
- Confirm-before-quit: `q` requires a second press within 3 seconds; `ctrl+c` exits immediately.
- Reload (`R`) to refresh the card list from disk on demand.
- Debug overlay gated behind `enableDebugUI` config flag.

### Navigation

- `j`/`k` (or arrow keys): move one card at a time.
- `J`/`K`: chunk jumps (~7 cards, configurable via `navChunkSize`).
- `[`/`]`: bisect -- jump to the midpoint between cursor and the deck edge; converges in O(log N) presses.
- `r`: random jump.
- `gg`/`G`: first / last card.
- `navJitter` (default 0.15) adds slight randomness to chunk and bisect landings for a more physical feel.

### Luhmann Addressing

- Filename is the card address. No database, no metadata.
- `c`: continue -- appends the next alternating component (`16a` -> `16a1`).
- `C`: branch -- increments the last component as a sibling (`16a` -> `16b`).
- `N`: create the next integer root card (`17` if `16` is the highest).
- `continueNameCmd` / `branchNameCmd`: override derivation with a custom shell command.
- Natural Luhmann sort order throughout.

### Card Backs

- Add a back side with a `---back---` line; use `---back:portrait---` for portrait orientation (4:3 aspect).
- Flip indicator on stack cards and in the overlay header when a back exists.
- `f` in the overlay flips between front and back.
- `e` in the overlay opens whichever side is currently visible.

### In-App Editor

- Vim-style modal editing: normal, insert, and command modes.
- Navigation: `h`/`l`/`j`/`k`, `w`/`b`/`e` word motions (crossing lines), `0`/`$`, `gg`/`G`.
- Editing: `x`, `r`, `D`, `C`, `dd`, `dG`/`dgg`, `dw`/`diw`, `cw`/`ciw`, `yy`/`yG`/`ygg`, `yw`/`yiw`, `p`.
- Undo (`u`) / redo (`ctrl+r`) / dot repeat (`.`).
- Count prefix: `5j`, `3x`, `10i.<Esc>`.
- Commands: `:w` / `ctrl+s` save; `:q` / `:q!` / `:wq` / `:x` / `:wqa` close/exit; `:sort`; `:fill [c]`; `:rename <name>`; `:back` / `:back:portrait` / `:front`; `:help`.
- `:sort` sorts by subject heading, ignoring dot-leader suffixes, so parenthetical variants group correctly.
- `editMode: "external"` redirects `e` to `$EDITOR`; `E` always opens `$EDITOR`.
- `ctrl+b` suspends the editor and returns to browse; `e` resumes.

### Split-Pane Editor

- `c`/`C` opens source and new card in a 35/65 horizontal split by default (`autoSplitOnLink: true`).
- Pressing `e` on a different card while a single-pane editor is suspended opens it as a companion pane.
- `ctrl+w` switches focus between panes.
- Each pane tracks dirty state independently; `:q`/`:wq` close the focused pane (split -> single -> browse).
- Portrait-orientation card backs render at the correct aspect in the editor without affecting the stack.

### Multi-Box

- `--box <dir>` / `--free-box <dir>`: open multiple note directories; `b` opens a picker to switch.
- Free-mode boxes disable `c`/`C`; `N` prompts for an arbitrary filename.
- Flags can be interleaved to control which box is active first.

### Live Reload

- `liveReload: true` (default): card list refreshes automatically when files change on disk via fsnotify.
- Editor state is preserved across background reloads.
- Set `liveReload: false` on networked filesystems where FS events are unreliable.

### Physical Printing

- `scripts/print_cards.py`: two 4"x6" landscape cards per 8.5"x11" sheet with cutting guides.
- `--duplex`: long-edge duplex layout so backs land physically behind their fronts after cutting.
- `--back-offset`: millimetre registration correction for printer variance.
- Pass `-` as the directory to read a list of paths from stdin (e.g., from `git log`).

### Configuration

- Config file auto-discovery: `config.json/.yaml/.yml/.toml` in current directory then `~/.thumbr/`.
- All keybindings, colors, layout, and editor settings configurable via file or CLI flags.
- `config.annotated.toml` provides a fully documented reference.
- `config.example.json` for a minimal starting point.

### Performance

- ~0.23s to load and sort ~90k notes on an i7-4770HQ with warm cache (`make bench`).
- Lazy content loading: visible stack previews preloaded on box open.

### Reliability

- Panic handler writes a crash log to `~/.thumbr/crash.log` (override with `--crash-log`).
- Empty-box crash fix: `ensureVisibleContent` guards against zero visible cards.
- Saves always append a trailing newline for clean diffs.
