package ui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"thumbr/internal/notes"
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

		return newFileResult{box: root, path: target}
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
