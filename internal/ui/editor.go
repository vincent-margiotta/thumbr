// editor.go — modal in-app vim-style text editor state and key handling.

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// taKey sends a synthetic key event to ta and returns the updated model.
// The tea.Cmd return from textarea.Update is discarded because textarea key
// messages never produce commands that need to be dispatched.
func taKey(ta textarea.Model, key tea.KeyType) textarea.Model {
	ta, _ = ta.Update(tea.KeyMsg{Type: key})
	return ta
}

type vimMode int

const (
	vimNormal vimMode = iota
	vimInsert
	vimCommand
)

type undoEntry struct {
	content string
	line    int
	col     int
}

type editorState struct {
	ta             textarea.Model
	mode           vimMode
	path           string
	dirty          bool
	pending        string // partial multi-char sequence: "g", "d", "y", "r", "c"
	yankBuf        string
	cmdLine        string // content after : in command mode
	saveErr        error
	undoStack      []undoEntry
	redoStack      []undoEntry
	insertSnapshot *undoEntry                    // state captured on insert-mode entry
	insertLog      string                        // chars typed since insert-mode entry (for dot repeat)
	insertEntry    func(editorState) editorState // pre-insert action to replay on dot repeat
	lastRepeat     func(editorState) editorState // nil until a repeatable change is made
}

type editorAction int

const (
	editorActionNone editorAction = iota
	editorActionSave
	editorActionQuit
	editorActionSaveQuit
	editorActionSaveQuitAll
	editorActionForceQuit
	editorActionReformat
)

// splitViewports returns per-pane Viewports for a 35/65 split.
// Heights are inflated by 3 so newEditorState's (vp.Height − 3) formula
// yields the correct textarea row count.
// Reserved lines: 2 pane headers + divider + footer + status bar = 5.
func splitViewports(vp Viewport) (top, bottom Viewport) {
	usable := vp.Height - 5
	if usable < 4 {
		usable = 4
	}
	topH := int(float64(usable) * 0.35)
	if topH < 1 {
		topH = 1
	}
	botH := usable - topH
	return Viewport{Width: vp.Width, Height: topH + 3},
		Viewport{Width: vp.Width, Height: botH + 3}
}

// modeLabel returns the display string for the given vim mode.
func modeLabel(mode vimMode) string {
	switch mode {
	case vimInsert:
		return "INSERT"
	case vimCommand:
		return "COMMAND"
	default:
		return "NORMAL"
	}
}

func newEditorState(path, content string, vp Viewport, cursorLine int) (editorState, tea.Cmd) {
	ta := textarea.New()
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	h := vp.Height - 3
	if h < 1 {
		h = 1
	}
	ta.SetWidth(vp.Width)
	ta.SetHeight(h)
	ta.SetValue(content)
	cmd := ta.Focus()
	// SetValue leaves cursor at end; always go to top first, then to cursorLine.
	for i := 0; i < 1000; i++ {
		prev := ta.Line()
		ta = taKey(ta, tea.KeyUp)
		if ta.Line() >= prev {
			break
		}
	}
	for i := 0; i < cursorLine; i++ {
		ta = taKey(ta, tea.KeyDown)
	}
	return editorState{
		ta:   ta,
		mode: vimNormal,
		path: path,
	}, cmd
}

func (es *editorState) save() error {
	err := os.WriteFile(es.path, []byte(es.ta.Value()), 0o644)
	if err != nil {
		es.saveErr = err
		return err
	}
	es.dirty = false
	es.saveErr = nil
	return nil
}

func (es editorState) snapshot() undoEntry {
	return undoEntry{
		content: es.ta.Value(),
		line:    es.ta.Line(),
		col:     es.ta.LineInfo().CharOffset,
	}
}

// pushUndo saves the current state onto the undo stack and clears the redo stack.
func (es editorState) pushUndo() editorState {
	es.undoStack = append(es.undoStack, es.snapshot())
	es.redoStack = nil
	return es
}

// ---------------------------------------------------------------------------
// Top-level editor key dispatcher (method on Model so it can call applyEditorAction)
// ---------------------------------------------------------------------------

func (m Model) handleEditorKey(msg tea.KeyMsg, start time.Time) (tea.Model, tea.Cmd) {
	key := msg.String()

	tw := m.settings.TextWidth

	if m.isBinding(key, m.bindings.SwitchPane) && m.paneCount == 2 {
		m.activePane = 1 - m.activePane
		return m.withUpdateSample(start), nil
	}

	if m.isBinding(key, m.bindings.SuspendEditor) {
		m.state = StateBrowsing
		return m.withUpdateSample(start), nil
	}

	es := m.editors[m.activePane]

	if key == "ctrl+s" {
		if err := es.save(); err != nil {
			m.editors[m.activePane] = es
			m = m.setStatus(fmt.Sprintf("Save failed: %v", err), 3*time.Second)
		} else {
			m.editors[m.activePane] = es
			m = m.syncCardContent(es.path, es.ta.Value())
			m = m.setStatus("Saved", 2*time.Second)
		}
		return m.withUpdateSample(start), nil
	}

	switch es.mode {
	case vimNormal:
		if es.pending != "" {
			es = es.handlePending(key, tw)
			m.editors[m.activePane] = es
			return m.withUpdateSample(start), nil
		}
		var action editorAction
		es, action = es.handleNormal(key)
		m.editors[m.activePane] = es
		if action != editorActionNone {
			return m.applyEditorAction(action, start)
		}
		return m.withUpdateSample(start), nil

	case vimInsert:
		var cmd tea.Cmd
		es, cmd = es.handleInsert(msg, tw)
		m.editors[m.activePane] = es
		return m.withUpdateSample(start), cmd

	case vimCommand:
		var action editorAction
		es, action = es.handleCommand(key)
		m.editors[m.activePane] = es
		if action != editorActionNone {
			return m.applyEditorAction(action, start)
		}
		return m.withUpdateSample(start), nil
	}

	return m.withUpdateSample(start), nil
}

// syncCardContent updates the cached content of the card at path so the
// overlay reflects the saved state without a full reload.
func (m Model) syncCardContent(path, content string) Model {
	for i := range m.cards {
		if m.cards[i].Path == path {
			m.cards[i].Content = content
			m.cards[i].ContentLoaded = true
			m.cards[i].ContentErr = nil
			break
		}
	}
	return m
}

func (m Model) applyEditorAction(action editorAction, start time.Time) (tea.Model, tea.Cmd) {
	es := m.editors[m.activePane]
	switch action {
	case editorActionSave:
		if err := es.save(); err != nil {
			m.editors[m.activePane] = es
			m = m.setStatus(fmt.Sprintf("Save failed: %v", err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		m.editors[m.activePane] = es
		m = m.syncCardContent(es.path, es.ta.Value())
		m = m.setStatus("Saved", 2*time.Second)
		return m.withUpdateSample(start), nil

	case editorActionQuit:
		if es.dirty {
			m.editors[m.activePane] = es
			m = m.setStatus("Unsaved changes — use :q! to discard", 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		return m.closeFocusedPane(start)

	case editorActionForceQuit:
		return m.closeFocusedPane(start)

	case editorActionSaveQuit:
		if err := es.save(); err != nil {
			m.editors[m.activePane] = es
			m = m.setStatus(fmt.Sprintf("Save failed: %v", err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		m = m.syncCardContent(es.path, es.ta.Value())
		return m.closeFocusedPane(start)

	case editorActionSaveQuitAll:
		for i := 0; i < m.paneCount; i++ {
			pane := m.editors[i]
			if err := pane.save(); err != nil {
				m.editors[i] = pane
				m = m.setStatus(fmt.Sprintf("Save failed (%s): %v", filepath.Base(pane.path), err), 3*time.Second)
				return m.withUpdateSample(start), nil
			}
			m.editors[i] = pane
			m = m.syncCardContent(pane.path, pane.ta.Value())
		}
		m.editors = [2]editorState{}
		m.paneCount = 0
		m.activePane = 0
		m.state = StateBrowsing
		return m.withUpdateSample(start), nil

	case editorActionReformat:
		tw := m.settings.TextWidth
		if tw <= 0 {
			m = m.setStatus("textWidth is disabled — set it in config to use :fmt", 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		lines := strings.Split(es.ta.Value(), "\n")
		var newLines []string
		for _, line := range lines {
			newLines = append(newLines, reflowLine(line, tw)...)
		}
		newContent := strings.Join(newLines, "\n")
		if newContent != es.ta.Value() {
			es = es.pushUndo()
			es.ta.SetValue(newContent)
			es = es.setCursorToLineCol(es.ta.Line(), 0)
			es.dirty = true
			m.editors[m.activePane] = es
			m = m.setStatus("Reformatted", 2*time.Second)
		} else {
			m = m.setStatus("Already within column limit", 2*time.Second)
		}
		return m.withUpdateSample(start), nil
	}

	return m.withUpdateSample(start), nil
}

// closeFocusedPane closes the active pane. If split, the other pane expands to
// fill the screen; if single, the editor exits to browsing.
func (m Model) closeFocusedPane(start time.Time) (tea.Model, tea.Cmd) {
	if m.paneCount == 2 {
		remaining := 1 - m.activePane
		h := m.viewport.Height - 3
		if h < 1 {
			h = 1
		}
		m.editors[remaining].ta.SetWidth(m.viewport.Width)
		m.editors[remaining].ta.SetHeight(h)
		m.editors[0] = m.editors[remaining]
		m.editors[1] = editorState{}
		m.paneCount = 1
		m.activePane = 0
	} else {
		m.editors[0] = editorState{}
		m.editors[1] = editorState{}
		m.paneCount = 0
		m.activePane = 0
		m.state = StateBrowsing
	}
	return m.withUpdateSample(start), nil
}

// ---------------------------------------------------------------------------
// Mode-specific handlers
// ---------------------------------------------------------------------------

func (es editorState) handleNormal(key string) (editorState, editorAction) {
	switch key {
	case "h":
		es.ta = taKey(es.ta, tea.KeyLeft)
	case "l":
		es.ta = taKey(es.ta, tea.KeyRight)
	case "j":
		es.ta = taKey(es.ta, tea.KeyDown)
	case "k":
		es.ta = taKey(es.ta, tea.KeyUp)
	case "w":
		es = es.moveWordForward()
	case "b":
		es = es.moveWordBackward()
	case "0":
		es.ta.CursorStart()
	case "$":
		es.ta.CursorEnd()
	case "G":
		es.ta = taKey(es.ta, tea.KeyCtrlEnd)
	case ".":
		if es.lastRepeat != nil {
			es = es.lastRepeat(es)
		}
	case "x":
		es = es.pushUndo()
		es.ta = taKey(es.ta, tea.KeyDelete)
		es.dirty = true
		es.lastRepeat = func(s editorState) editorState {
			s = s.pushUndo()
			s.ta = taKey(s.ta, tea.KeyDelete)
			s.dirty = true
			return s
		}
	case "u":
		if len(es.undoStack) > 0 {
			es.redoStack = append(es.redoStack, es.snapshot())
			entry := es.undoStack[len(es.undoStack)-1]
			es.undoStack = es.undoStack[:len(es.undoStack)-1]
			es.ta.SetValue(entry.content)
			es = es.setCursorToLineCol(entry.line, entry.col)
			es.dirty = true
		}
	case "ctrl+r":
		if len(es.redoStack) > 0 {
			es.undoStack = append(es.undoStack, es.snapshot())
			entry := es.redoStack[len(es.redoStack)-1]
			es.redoStack = es.redoStack[:len(es.redoStack)-1]
			es.ta.SetValue(entry.content)
			es = es.setCursorToLineCol(entry.line, entry.col)
			es.dirty = true
		}
	case "i":
		snap := es.snapshot()
		es.insertSnapshot = &snap
		es.insertLog = ""
		es.insertEntry = nil
		es.mode = vimInsert
	case "a":
		snap := es.snapshot()
		es.insertSnapshot = &snap
		es.ta = taKey(es.ta, tea.KeyRight)
		es.insertLog = ""
		es.insertEntry = func(s editorState) editorState {
			s.ta = taKey(s.ta, tea.KeyRight)
			return s
		}
		es.mode = vimInsert
	case "A":
		snap := es.snapshot()
		es.insertSnapshot = &snap
		es.ta.CursorEnd()
		es.insertLog = ""
		es.insertEntry = func(s editorState) editorState {
			s.ta.CursorEnd()
			return s
		}
		es.mode = vimInsert
	case "o":
		es = es.pushUndo()
		es.ta.CursorEnd()
		es.ta.InsertString("\n")
		es.dirty = true
		snap := es.snapshot()
		es.insertSnapshot = &snap
		es.insertLog = ""
		es.insertEntry = func(s editorState) editorState {
			s = s.pushUndo()
			s.ta.CursorEnd()
			s.ta.InsertString("\n")
			s.dirty = true
			return s
		}
		es.mode = vimInsert
	case "O":
		es = es.pushUndo()
		es.ta.CursorStart()
		es.ta.InsertString("\n")
		es.ta = taKey(es.ta, tea.KeyUp)
		es.dirty = true
		snap := es.snapshot()
		es.insertSnapshot = &snap
		es.insertLog = ""
		es.insertEntry = func(s editorState) editorState {
			s = s.pushUndo()
			s.ta.CursorStart()
			s.ta.InsertString("\n")
			s.ta = taKey(s.ta, tea.KeyUp)
			s.dirty = true
			return s
		}
		es.mode = vimInsert
	case "p":
		if es.yankBuf != "" {
			es = es.pushUndo()
			es.ta.CursorEnd()
			es.ta.InsertString("\n" + es.yankBuf)
			es.dirty = true
			es.lastRepeat = func(s editorState) editorState {
				if s.yankBuf != "" {
					s = s.pushUndo()
					s.ta.CursorEnd()
					s.ta.InsertString("\n" + s.yankBuf)
					s.dirty = true
				}
				return s
			}
		}
	case ":":
		es.mode = vimCommand
		es.cmdLine = ""
	case "esc", "ctrl+[":
		es.pending = ""
	case "g", "d", "y", "r", "c":
		es.pending = key
	}
	return es, editorActionNone
}

func (es editorState) handlePending(key string, textwidth int) editorState {
	prev := es.pending
	es.pending = ""

	switch prev {
	case "g":
		if key == "g" {
			es.ta = taKey(es.ta, tea.KeyCtrlHome)
		} else if key == "q" {
			es.pending = "gq"
		}

	case "gq":
		if key == "q" && textwidth > 0 {
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			if lineIdx >= 0 && lineIdx < len(lines) {
				reflowed := reflowLine(lines[lineIdx], textwidth)
				if len(reflowed) > 1 || (len(reflowed) == 1 && reflowed[0] != lines[lineIdx]) {
					es = es.pushUndo()
					newLines := make([]string, 0, len(lines)+len(reflowed)-1)
					newLines = append(newLines, lines[:lineIdx]...)
					newLines = append(newLines, reflowed...)
					newLines = append(newLines, lines[lineIdx+1:]...)
					es.ta.SetValue(strings.Join(newLines, "\n"))
					es = es.setCursorToLineCol(lineIdx, 0)
					es.dirty = true
					tw := textwidth
					es.lastRepeat = func(s editorState) editorState {
						s = s.pushUndo()
						ls := strings.Split(s.ta.Value(), "\n")
						li := s.ta.Line()
						if li >= 0 && li < len(ls) {
							rf := reflowLine(ls[li], tw)
							nl := make([]string, 0, len(ls)+len(rf)-1)
							nl = append(nl, ls[:li]...)
							nl = append(nl, rf...)
							nl = append(nl, ls[li+1:]...)
							s.ta.SetValue(strings.Join(nl, "\n"))
							s = s.setCursorToLineCol(li, 0)
							s.dirty = true
						}
						return s
					}
				}
			}
		}

	case "d":
		switch key {
		case "d":
			es = es.pushUndo()
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			if lineIdx >= 0 && lineIdx < len(lines) {
				newLines := append(lines[:lineIdx:lineIdx], lines[lineIdx+1:]...)
				es.ta.SetValue(strings.Join(newLines, "\n"))
				es = es.setCursorToLineCol(lineIdx, 0)
				es.dirty = true
				es.lastRepeat = func(s editorState) editorState {
					s = s.pushUndo()
					ls := strings.Split(s.ta.Value(), "\n")
					li := s.ta.Line()
					if li >= 0 && li < len(ls) {
						nl := append(ls[:li:li], ls[li+1:]...)
						s.ta.SetValue(strings.Join(nl, "\n"))
						s = s.setCursorToLineCol(li, 0)
						s.dirty = true
					}
					return s
				}
			}
		case "w":
			es = es.pushUndo()
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			col := es.ta.LineInfo().CharOffset
			if lineIdx >= 0 && lineIdx < len(lines) {
				runes := []rune(lines[lineIdx])
				end := wordForwardEnd(runes, col)
				lines[lineIdx] = string(runes[:col]) + string(runes[end:])
				es.ta.SetValue(strings.Join(lines, "\n"))
				es = es.setCursorToLineCol(lineIdx, col)
				es.dirty = true
				es.lastRepeat = func(s editorState) editorState {
					s = s.pushUndo()
					ls := strings.Split(s.ta.Value(), "\n")
					li := s.ta.Line()
					c := s.ta.LineInfo().CharOffset
					if li >= 0 && li < len(ls) {
						r := []rune(ls[li])
						e := wordForwardEnd(r, c)
						ls[li] = string(r[:c]) + string(r[e:])
						s.ta.SetValue(strings.Join(ls, "\n"))
						s = s.setCursorToLineCol(li, c)
						s.dirty = true
					}
					return s
				}
			}
		case "i":
			es.pending = "di"
		}

	case "di":
		if key == "w" {
			es = es.pushUndo()
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			col := es.ta.LineInfo().CharOffset
			if lineIdx >= 0 && lineIdx < len(lines) {
				runes := []rune(lines[lineIdx])
				start, end := wordBoundary(runes, col)
				lines[lineIdx] = string(runes[:start]) + string(runes[end:])
				es.ta.SetValue(strings.Join(lines, "\n"))
				es = es.setCursorToLineCol(lineIdx, start)
				es.dirty = true
				es.lastRepeat = func(s editorState) editorState {
					s = s.pushUndo()
					ls := strings.Split(s.ta.Value(), "\n")
					li := s.ta.Line()
					c := s.ta.LineInfo().CharOffset
					if li >= 0 && li < len(ls) {
						r := []rune(ls[li])
						st, en := wordBoundary(r, c)
						ls[li] = string(r[:st]) + string(r[en:])
						s.ta.SetValue(strings.Join(ls, "\n"))
						s = s.setCursorToLineCol(li, st)
						s.dirty = true
					}
					return s
				}
			}
		}

	case "y":
		if key == "y" {
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			if lineIdx >= 0 && lineIdx < len(lines) {
				es.yankBuf = lines[lineIdx]
			}
		}

	case "r":
		if len([]rune(key)) == 1 {
			es = es.pushUndo()
			es.ta = taKey(es.ta, tea.KeyDelete)
			es.ta.InsertString(key)
			es.ta = taKey(es.ta, tea.KeyLeft)
			es.dirty = true
			ch := key
			es.lastRepeat = func(s editorState) editorState {
				s = s.pushUndo()
				s.ta = taKey(s.ta, tea.KeyDelete)
				s.ta.InsertString(ch)
				s.ta = taKey(s.ta, tea.KeyLeft)
				s.dirty = true
				return s
			}
		}

	case "c":
		switch key {
		case "w":
			es = es.pushUndo()
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			col := es.ta.LineInfo().CharOffset
			if lineIdx >= 0 && lineIdx < len(lines) {
				runes := []rune(lines[lineIdx])
				end := wordForwardEnd(runes, col)
				lines[lineIdx] = string(runes[:col]) + string(runes[end:])
				es.ta.SetValue(strings.Join(lines, "\n"))
				es = es.setCursorToLineCol(lineIdx, col)
				es.dirty = true
			}
			snap := es.snapshot()
			es.insertSnapshot = &snap
			es.insertLog = ""
			es.insertEntry = func(s editorState) editorState {
				s = s.pushUndo()
				ls := strings.Split(s.ta.Value(), "\n")
				li := s.ta.Line()
				c := s.ta.LineInfo().CharOffset
				if li >= 0 && li < len(ls) {
					r := []rune(ls[li])
					e := wordForwardEnd(r, c)
					ls[li] = string(r[:c]) + string(r[e:])
					s.ta.SetValue(strings.Join(ls, "\n"))
					s = s.setCursorToLineCol(li, c)
					s.dirty = true
				}
				return s
			}
			es.mode = vimInsert
		case "i":
			es.pending = "ci"
		}

	case "ci":
		if key == "w" {
			es = es.pushUndo()
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			col := es.ta.LineInfo().CharOffset
			if lineIdx >= 0 && lineIdx < len(lines) {
				runes := []rune(lines[lineIdx])
				start, end := wordBoundary(runes, col)
				lines[lineIdx] = string(runes[:start]) + string(runes[end:])
				es.ta.SetValue(strings.Join(lines, "\n"))
				es = es.setCursorToLineCol(lineIdx, start)
				es.dirty = true
			}
			snap := es.snapshot()
			es.insertSnapshot = &snap
			es.insertLog = ""
			es.insertEntry = func(s editorState) editorState {
				s = s.pushUndo()
				ls := strings.Split(s.ta.Value(), "\n")
				li := s.ta.Line()
				c := s.ta.LineInfo().CharOffset
				if li >= 0 && li < len(ls) {
					r := []rune(ls[li])
					st, en := wordBoundary(r, c)
					ls[li] = string(r[:st]) + string(r[en:])
					s.ta.SetValue(strings.Join(ls, "\n"))
					s = s.setCursorToLineCol(li, st)
					s.dirty = true
				}
				return s
			}
			es.mode = vimInsert
		}
	}
	return es
}

// setCursorToLineCol navigates the textarea cursor to the given line and column
// using absolute character offset from the start of the document. This avoids
// the unreliable Down-key navigation that misfires after SetValue calls.
func (es editorState) setCursorToLineCol(line, col int) editorState {
	lines := strings.Split(es.ta.Value(), "\n")
	if line < 0 {
		line = 0
	}
	if len(lines) == 0 {
		return es
	}
	if line >= len(lines) {
		line = len(lines) - 1
	}
	lineRunes := []rune(lines[line])
	if col < 0 {
		col = 0
	}
	if col > len(lineRunes) {
		col = len(lineRunes)
	}
	abs := 0
	for i := 0; i < line; i++ {
		abs += len([]rune(lines[i])) + 1 // +1 for the \n
	}
	abs += col
	es.ta = taKey(es.ta, tea.KeyCtrlHome)
	for i := 0; i < abs; i++ {
		es.ta = taKey(es.ta, tea.KeyRight)
	}
	return es
}

// wordBoundary returns the [start, end) rune range of the word/token under col.
// Words are runs of word-chars (letters/digits/_); punctuation runs form their
// own tokens; whitespace forms its own runs.
func wordBoundary(runes []rune, col int) (start, end int) {
	n := len(runes)
	if n == 0 {
		return 0, 0
	}
	if col >= n {
		col = n - 1
	}
	isWord := func(r rune) bool {
		return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
	}
	isSpace := func(r rune) bool { return r == ' ' || r == '\t' }

	cur := runes[col]
	var same func(rune) bool
	switch {
	case isWord(cur):
		same = isWord
	case isSpace(cur):
		same = isSpace
	default:
		same = func(r rune) bool { return !isWord(r) && !isSpace(r) }
	}

	start = col
	for start > 0 && same(runes[start-1]) {
		start--
	}
	end = col + 1
	for end < n && same(runes[end]) {
		end++
	}
	return start, end
}

// moveWordForward implements vim `w`: jump to the start of the next word,
// crossing to the next line when the motion runs off the end of the current one.
func (es editorState) moveWordForward() editorState {
	lines := strings.Split(es.ta.Value(), "\n")
	lineIdx := es.ta.Line()
	col := es.ta.LineInfo().CharOffset
	if lineIdx >= len(lines) {
		return es
	}
	runes := []rune(lines[lineIdx])
	newCol := wordForwardEnd(runes, col)
	if newCol < len(runes) {
		return es.setCursorToLineCol(lineIdx, newCol)
	}
	if lineIdx+1 < len(lines) {
		nextRunes := []rune(lines[lineIdx+1])
		i := 0
		for i < len(nextRunes) && (nextRunes[i] == ' ' || nextRunes[i] == '\t') {
			i++
		}
		return es.setCursorToLineCol(lineIdx+1, i)
	}
	return es
}

// moveWordBackward implements vim `b`: jump to the start of the current or
// previous word, crossing to the previous line when already at column 0.
func (es editorState) moveWordBackward() editorState {
	lines := strings.Split(es.ta.Value(), "\n")
	lineIdx := es.ta.Line()
	col := es.ta.LineInfo().CharOffset
	if lineIdx >= len(lines) {
		return es
	}
	runes := []rune(lines[lineIdx])
	newCol := prevWordStart(runes, col)
	if newCol >= 0 {
		return es.setCursorToLineCol(lineIdx, newCol)
	}
	if lineIdx > 0 {
		prevRunes := []rune(lines[lineIdx-1])
		if len(prevRunes) == 0 {
			return es.setCursorToLineCol(lineIdx-1, 0)
		}
		isSpace := func(r rune) bool { return r == ' ' || r == '\t' }
		i := len(prevRunes) - 1
		for i >= 0 && isSpace(prevRunes[i]) {
			i--
		}
		if i < 0 {
			return es.setCursorToLineCol(lineIdx-1, 0)
		}
		start, _ := wordBoundary(prevRunes, i)
		return es.setCursorToLineCol(lineIdx-1, start)
	}
	return es
}

// prevWordStart returns the column of the start of the previous/current word
// relative to col. Returns -1 when the motion should cross to the previous line.
func prevWordStart(runes []rune, col int) int {
	if col <= 0 {
		return -1
	}
	isSpace := func(r rune) bool { return r == ' ' || r == '\t' }
	start, _ := wordBoundary(runes, col)
	if start < col {
		return start
	}
	// Already at start of a token — skip whitespace leftward then find prev token.
	i := col - 1
	for i >= 0 && isSpace(runes[i]) {
		i--
	}
	if i < 0 {
		return -1
	}
	start, _ = wordBoundary(runes, i)
	return start
}

// reflowLine wraps a single line to fit within textwidth by replacing spaces
// with newlines at word boundaries. Lines with no breakable space are returned
// unchanged. The result always contains at least one element.
func reflowLine(line string, textwidth int) []string {
	if len([]rune(line)) <= textwidth {
		return []string{line}
	}
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{line}
	}
	var result []string
	cur := ""
	for _, word := range words {
		switch {
		case cur == "":
			cur = word
		case len([]rune(cur))+1+len([]rune(word)) <= textwidth:
			cur += " " + word
		default:
			result = append(result, cur)
			cur = word
		}
	}
	if cur != "" {
		result = append(result, cur)
	}
	return result
}

// wordForwardEnd returns the rune index just past the end of the word/token
// motion from col — matching vim's `w` / `dw` target (skips trailing spaces).
func wordForwardEnd(runes []rune, col int) int {
	n := len(runes)
	if col >= n {
		return n
	}
	isWord := func(r rune) bool {
		return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
	}
	isSpace := func(r rune) bool { return r == ' ' || r == '\t' }

	i := col
	if isSpace(runes[i]) {
		for i < n && isSpace(runes[i]) {
			i++
		}
	} else if isWord(runes[i]) {
		for i < n && isWord(runes[i]) {
			i++
		}
		for i < n && isSpace(runes[i]) {
			i++
		}
	} else {
		for i < n && !isWord(runes[i]) && !isSpace(runes[i]) {
			i++
		}
		for i < n && isSpace(runes[i]) {
			i++
		}
	}
	return i
}

func (es editorState) handleInsert(msg tea.KeyMsg, textwidth int) (editorState, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc", "ctrl+[":
		if es.insertSnapshot != nil && es.ta.Value() != es.insertSnapshot.content {
			es.undoStack = append(es.undoStack, *es.insertSnapshot)
			es.redoStack = nil
			log := es.insertLog
			entry := es.insertEntry
			tw := textwidth
			es.lastRepeat = func(s editorState) editorState {
				s = s.pushUndo()
				if entry != nil {
					s = entry(s)
				}
				for _, r := range []rune(log) {
					if r == '\n' {
						s.ta = taKey(s.ta, tea.KeyEnter)
					} else {
						s.ta.InsertString(string(r))
						if tw > 0 {
							s = s.applyHardWrap(tw)
						}
					}
					s.dirty = true
				}
				return s
			}
		}
		es.insertSnapshot = nil
		es.insertLog = ""
		es.insertEntry = nil
		es.mode = vimNormal
		return es, nil
	case "backspace":
		if len(es.insertLog) > 0 {
			runes := []rune(es.insertLog)
			es.insertLog = string(runes[:len(runes)-1])
		}
	default:
		if msg.Type == tea.KeyRunes {
			es.insertLog += string(msg.Runes)
		} else if msg.Type == tea.KeyEnter {
			es.insertLog += "\n"
		}
	}
	var cmd tea.Cmd
	es.ta, cmd = es.ta.Update(msg)
	es.dirty = true
	if textwidth > 0 && msg.Type == tea.KeyRunes {
		es = es.applyHardWrap(textwidth)
	}
	return es, cmd
}

// applyHardWrap checks whether the current line exceeds textwidth and, if so,
// finds the last space at or before textwidth and replaces it with a newline.
// Lines with no space before textwidth are left unchanged (long words are not split).
func (es editorState) applyHardWrap(textwidth int) editorState {
	lines := strings.Split(es.ta.Value(), "\n")
	lineIdx := es.ta.Line()
	col := es.ta.LineInfo().CharOffset
	if lineIdx >= len(lines) {
		return es
	}
	runes := []rune(lines[lineIdx])
	if len(runes) <= textwidth {
		return es
	}
	wrapAt := -1
	for i := textwidth - 1; i >= 0; i-- {
		if runes[i] == ' ' {
			wrapAt = i
			break
		}
	}
	if wrapAt < 0 {
		return es
	}
	before := string(runes[:wrapAt])
	after := string(runes[wrapAt+1:])
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:lineIdx]...)
	newLines = append(newLines, before, after)
	newLines = append(newLines, lines[lineIdx+1:]...)
	es.ta.SetValue(strings.Join(newLines, "\n"))
	if col >= wrapAt {
		es = es.setCursorToLineCol(lineIdx+1, col-wrapAt-1)
	} else {
		es = es.setCursorToLineCol(lineIdx, col)
	}
	return es
}

func (es editorState) handleCommand(key string) (editorState, editorAction) {
	switch key {
	case "esc", "ctrl+[":
		es.mode = vimNormal
		es.cmdLine = ""
		return es, editorActionNone
	case "enter":
		cmd := strings.TrimSpace(es.cmdLine)
		es.mode = vimNormal
		es.cmdLine = ""
		switch cmd {
		case "w":
			return es, editorActionSave
		case "q":
			return es, editorActionQuit
		case "q!":
			return es, editorActionForceQuit
		case "wq", "x":
			return es, editorActionSaveQuit
		case "wqa":
			return es, editorActionSaveQuitAll
		case "fmt":
			return es, editorActionReformat
		}
		return es, editorActionNone
	case "backspace":
		if len(es.cmdLine) > 0 {
			runes := []rune(es.cmdLine)
			es.cmdLine = string(runes[:len(runes)-1])
		}
		return es, editorActionNone
	default:
		if len([]rune(key)) == 1 {
			es.cmdLine += key
		}
		return es, editorActionNone
	}
}
