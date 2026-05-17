// commands.go — async tea.Cmd factories, result message types, and OS-level helpers.

package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"

	"github.com/vincent-margiotta/thumbr/internal/notes"
)

// ---------------------------------------------------------------------------
// Result message types
// ---------------------------------------------------------------------------

type boxLoadResult struct {
	path  string
	cards []notes.Card
	err   error
}

type newFileResult struct {
	box       string
	path      string
	seekPath  string // if non-empty, navigate to this path after the box reloads
	openInApp bool   // if true, open path in the in-app editor after reload
	err       error
}

type editorResult struct {
	path string
	err  error
}

type openInAppResult struct {
	path    string
	content string
	err     error
}

type cardContentResult struct {
	path    string
	content string
	err     error
}

// watchEventMsg is sent when the file watcher detects a relevant change in the note root.
type watchEventMsg struct{}

type openSplitResult struct {
	topPath, topContent       string
	bottomPath, bottomContent string
	err                       error
}

// ---------------------------------------------------------------------------
// Async command methods
// ---------------------------------------------------------------------------

// loadBoxCmd loads the given path with the model's load options in a goroutine.
func (m Model) loadBoxCmd(path string) tea.Cmd {
	root := cleanBoxPath(path)
	opts := m.loadOpts
	return func() tea.Msg {
		cards, err := notes.LoadCardsFromDir(root, opts)
		return boxLoadResult{path: root, cards: cards, err: err}
	}
}

// openInEditorCmd launches the external editor for the path.
func (m Model) openInEditorCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return editorResult{path: path, err: launchEditor(path)}
	}
}

// preloadVisibleCmd returns a preload command for the cards currently visible
// in the stack, centered on the cursor. Already-loaded cards are skipped.
func (m Model) preloadVisibleCmd() tea.Cmd {
	vis := m.visibleIndices()
	if len(vis) == 0 {
		return nil
	}
	pos := m.visibleCursorIndex(vis)
	if pos < 0 {
		pos = 0
	}
	n := m.settings.StackVisibleCount + 2
	start := pos - 1
	if start < 0 {
		start = 0
	}
	end := start + n
	if end > len(vis) {
		end = len(vis)
	}
	return preloadCardsCmd(m.cards, vis[start:end])
}

// preloadCardsCmd returns a batch of commands that read content for the given
// card paths in the background. Already-loaded cards are skipped.
func preloadCardsCmd(cards []notes.Card, indices []int) tea.Cmd {
	var cmds []tea.Cmd
	for _, i := range indices {
		if i < 0 || i >= len(cards) || cards[i].ContentLoaded {
			continue
		}
		path := cards[i].Path
		cmds = append(cmds, func() tea.Msg {
			data, err := os.ReadFile(path)
			return cardContentResult{path: path, content: string(data), err: err}
		})
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// openSplitCmd reads both files and returns an openSplitResult to open a split-pane editor.
func openSplitCmd(topPath, bottomPath string) tea.Cmd {
	return func() tea.Msg {
		top, err := os.ReadFile(topPath)
		if err != nil {
			return openSplitResult{err: err}
		}
		bottom, err := os.ReadFile(bottomPath)
		return openSplitResult{
			topPath:       topPath,
			topContent:    string(top),
			bottomPath:    bottomPath,
			bottomContent: string(bottom),
			err:           err,
		}
	}
}

// openInAppCmd reads the file at path and returns an openInAppResult so the
// caller can transition to StateEditing.
func openInAppCmd(path string) tea.Cmd {
	return func() tea.Msg {
		content, err := os.ReadFile(path)
		return openInAppResult{path: path, content: string(content), err: err}
	}
}

// createFileCmd makes sure the file exists under boxRoot, creates directories,
// and opens it in the external editor.
func (m Model) createFileCmd(boxRoot, userPath string) tea.Cmd {
	root := cleanBoxPath(boxRoot)
	defaultExt := m.defaultExt()
	return func() tea.Msg {
		raw := strings.TrimSpace(userPath)
		if raw == "" {
			return newFileResult{box: root, err: errors.New("file name required")}
		}

		target := raw
		if !filepath.IsAbs(target) {
			target = filepath.Join(root, target)
		}
		target = filepath.Clean(target)
		if filepath.Ext(target) == "" && defaultExt != "" {
			target += defaultExt
		}

		dir := filepath.Dir(target)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return newFileResult{box: root, path: target, err: fmt.Errorf("make dir: %w", err)}
		}

		// Create the file if missing; preserve existing content otherwise.
		if _, err := os.Stat(target); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return newFileResult{box: root, path: target, err: fmt.Errorf("stat: %w", err)}
			}
			if err := os.WriteFile(target, []byte{}, 0o644); err != nil {
				return newFileResult{box: root, path: target, err: fmt.Errorf("create: %w", err)}
			}
		}

		if err := launchEditor(target); err != nil {
			return newFileResult{box: root, path: target, err: fmt.Errorf("open: %w", err)}
		}

		return newFileResult{box: root, path: target, seekPath: target}
	}
}

// createLinkedFileCmd creates stem+ext in targetDir with initialContent, then
// opens it in the external editor.
func (m Model) createLinkedFileCmd(box, targetDir, stem, ext, initialContent string) tea.Cmd {
	return func() tea.Msg {
		target := filepath.Join(targetDir, stem+ext)
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return newFileResult{box: box, path: target, err: fmt.Errorf("make dir: %w", err)}
		}
		if _, err := os.Stat(target); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return newFileResult{box: box, path: target, err: fmt.Errorf("stat: %w", err)}
			}
			if err := os.WriteFile(target, []byte(initialContent), 0o644); err != nil {
				return newFileResult{box: box, path: target, err: fmt.Errorf("create: %w", err)}
			}
		}
		if err := launchEditor(target); err != nil {
			return newFileResult{box: box, path: target, err: fmt.Errorf("open: %w", err)}
		}
		return newFileResult{box: box, path: target, seekPath: target}
	}
}

// createLinkedFileCmdNoEditor creates stem+ext in targetDir with initialContent
// but does not open any editor. openInApp controls the newFileResult flag.
func (m Model) createLinkedFileCmdNoEditor(box, targetDir, stem, ext, initialContent string, openInApp bool) tea.Cmd {
	return func() tea.Msg {
		target := filepath.Join(targetDir, stem+ext)
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return newFileResult{box: box, path: target, err: fmt.Errorf("make dir: %w", err)}
		}
		if _, err := os.Stat(target); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return newFileResult{box: box, path: target, err: fmt.Errorf("stat: %w", err)}
			}
			if err := os.WriteFile(target, []byte(initialContent), 0o644); err != nil {
				return newFileResult{box: box, path: target, err: fmt.Errorf("create: %w", err)}
			}
		}
		return newFileResult{box: box, path: target, seekPath: target, openInApp: openInApp}
	}
}

// watchDirCmd watches root for file-system events relevant to the card list
// (creates, removes, and renames of files whose extension matches opts.IncludeExts).
// It blocks until a debounced event fires or stop is closed, then returns
// watchEventMsg or nil respectively. Newly created subdirectories are added
// to the watch set automatically.
func watchDirCmd(stop <-chan struct{}, root string, opts notes.LoadOptions) tea.Cmd {
	return func() tea.Msg {
		w, err := fsnotify.NewWatcher()
		if err != nil {
			return nil
		}
		defer w.Close()
		if err := watchAddDirs(w, root, opts.IgnoreGlobs); err != nil {
			return nil
		}

		timerCh := make(chan struct{}, 1)
		var debounce *time.Timer

		for {
			select {
			case <-stop:
				return nil
			case event, ok := <-w.Events:
				if !ok {
					return nil
				}
				if event.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
					continue
				}
				// Watch newly created subdirectories.
				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						_ = watchAddDirs(w, event.Name, opts.IgnoreGlobs)
					}
				}
				// Only trigger on files with a matching extension.
				if !watchFileMatches(event.Name, opts.IncludeExts, opts.IgnoreGlobs) {
					continue
				}
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(300*time.Millisecond, func() {
					select {
					case timerCh <- struct{}{}:
					default:
					}
				})
			case <-timerCh:
				return watchEventMsg{}
			case _, ok := <-w.Errors:
				if !ok {
					return nil
				}
			}
		}
	}
}

// watchAddDirs recursively adds root and all subdirectories to w, skipping
// directories that match ignoreGlobs.
func watchAddDirs(w *fsnotify.Watcher, root string, ignoreGlobs []string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if watchPathIgnored(path, ignoreGlobs) {
			return filepath.SkipDir
		}
		_ = w.Add(path)
		return nil
	})
}

// watchFileMatches returns true when path has a matching extension and is not ignored.
func watchFileMatches(path string, exts []string, ignoreGlobs []string) bool {
	if watchPathIgnored(path, ignoreGlobs) {
		return false
	}
	if len(exts) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range exts {
		if strings.ToLower(e) == ext {
			return true
		}
	}
	return false
}

// watchPathIgnored returns true when path or its base name matches any ignore glob.
func watchPathIgnored(path string, patterns []string) bool {
	base := filepath.Base(path)
	for _, pat := range patterns {
		if pat == "" {
			continue
		}
		if ok, _ := filepath.Match(pat, path); ok {
			return true
		}
		if ok, _ := filepath.Match(pat, base); ok {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// OS-level helpers
// ---------------------------------------------------------------------------

// deriveStemViaCmd runs shellCmd with stem as the sole argument and returns trimmed stdout.
func deriveStemViaCmd(shellCmd, stem string) (string, error) {
	out, err := exec.Command("sh", "-c", shellCmd+" "+strconv.Quote(stem)).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func launchEditor(path string) error {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return runEditorCommand(editor, path)
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return runEditorCommand(editor, path)
	}
	return openWithSystem(path)
}

func runEditorCommand(cmdStr, path string) error {
	command := fmt.Sprintf("%s %s", cmdStr, strconv.Quote(path))
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", command).Start()
	}
	return exec.Command("sh", "-c", command).Start()
}

func openWithSystem(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", "", strconv.Quote(path)).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}
