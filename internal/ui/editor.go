// editor.go — modal in-app vim-style text editor state and key handling.

package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type vimMode int

const (
	vimNormal  vimMode = iota
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
	insertSnapshot *undoEntry // state captured on insert-mode entry
}

type editorAction int

const (
	editorActionNone editorAction = iota
	editorActionSave
	editorActionQuit
	editorActionSaveQuit
	editorActionForceQuit
)

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
		ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyUp})
		if ta.Line() >= prev {
			break
		}
	}
	for i := 0; i < cursorLine; i++ {
		ta, _ = ta.Update(tea.KeyMsg{Type: tea.KeyDown})
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
	es := m.editor

	if m.isBinding(key, m.bindings.SuspendEditor) {
		m.state = StateBrowsing
		return m.withUpdateSample(start), nil
	}

	if key == "ctrl+s" {
		if err := es.save(); err != nil {
			m.editor = es
			m = m.setStatus(fmt.Sprintf("Save failed: %v", err), 3*time.Second)
		} else {
			m.editor = es
			m = m.setStatus("Saved", 2*time.Second)
		}
		return m.withUpdateSample(start), nil
	}

	switch es.mode {
	case vimNormal:
		if es.pending != "" {
			es = es.handlePending(key)
			m.editor = es
			return m.withUpdateSample(start), nil
		}
		var action editorAction
		es, action = es.handleNormal(key)
		m.editor = es
		if action != editorActionNone {
			return m.applyEditorAction(action, start)
		}
		return m.withUpdateSample(start), nil

	case vimInsert:
		var cmd tea.Cmd
		es, cmd = es.handleInsert(msg)
		m.editor = es
		return m.withUpdateSample(start), cmd

	case vimCommand:
		var action editorAction
		es, action = es.handleCommand(key)
		m.editor = es
		if action != editorActionNone {
			return m.applyEditorAction(action, start)
		}
		return m.withUpdateSample(start), nil
	}

	return m.withUpdateSample(start), nil
}

func (m Model) applyEditorAction(action editorAction, start time.Time) (tea.Model, tea.Cmd) {
	es := m.editor
	switch action {
	case editorActionSave:
		if err := es.save(); err != nil {
			m.editor = es
			m = m.setStatus(fmt.Sprintf("Save failed: %v", err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		m.editor = es
		m = m.setStatus("Saved", 2*time.Second)
		return m.withUpdateSample(start), nil

	case editorActionQuit:
		if es.dirty {
			m.editor = es
			m = m.setStatus("Unsaved changes — use :q! to discard", 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		m.state = StateBrowsing
		m.editor = editorState{}
		return m.withUpdateSample(start), nil

	case editorActionForceQuit:
		m.state = StateBrowsing
		m.editor = editorState{}
		return m.withUpdateSample(start), nil

	case editorActionSaveQuit:
		if err := es.save(); err != nil {
			m.editor = es
			m = m.setStatus(fmt.Sprintf("Save failed: %v", err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		m.state = StateBrowsing
		m.editor = editorState{}
		return m.withUpdateSample(start), nil
	}

	return m.withUpdateSample(start), nil
}

// ---------------------------------------------------------------------------
// Mode-specific handlers
// ---------------------------------------------------------------------------

func (es editorState) handleNormal(key string) (editorState, editorAction) {
	switch key {
	case "h":
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyLeft})
	case "l":
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyRight})
	case "j":
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyDown})
	case "k":
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyUp})
	case "w":
		es = es.moveWordForward()
	case "b":
		es = es.moveWordBackward()
	case "0":
		es.ta.CursorStart()
	case "$":
		es.ta.CursorEnd()
	case "G":
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyCtrlEnd})
	case "x":
		es = es.pushUndo()
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyDelete})
		es.dirty = true
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
		es.mode = vimInsert
	case "a":
		snap := es.snapshot()
		es.insertSnapshot = &snap
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyRight})
		es.mode = vimInsert
	case "A":
		snap := es.snapshot()
		es.insertSnapshot = &snap
		es.ta.CursorEnd()
		es.mode = vimInsert
	case "o":
		es = es.pushUndo()
		es.ta.CursorEnd()
		es.ta.InsertString("\n")
		es.dirty = true
		snap := es.snapshot()
		es.insertSnapshot = &snap
		es.mode = vimInsert
	case "O":
		es = es.pushUndo()
		es.ta.CursorStart()
		es.ta.InsertString("\n")
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyUp})
		es.dirty = true
		snap := es.snapshot()
		es.insertSnapshot = &snap
		es.mode = vimInsert
	case "p":
		if es.yankBuf != "" {
			es = es.pushUndo()
			es.ta.CursorEnd()
			es.ta.InsertString("\n" + es.yankBuf)
			es.dirty = true
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

func (es editorState) handlePending(key string) editorState {
	prev := es.pending
	es.pending = ""

	switch prev {
	case "g":
		if key == "g" {
			es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyCtrlHome})
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
			es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyDelete})
			es.ta.InsertString(key)
			es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyLeft})
			es.dirty = true
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
			es.mode = vimInsert
		}
	}
	return es
}

// setCursorToLineCol navigates the textarea cursor to the given line and column
// after a SetValue call that resets cursor position.
func (es editorState) setCursorToLineCol(line, col int) editorState {
	es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyCtrlHome})
	for i := 0; i < line; i++ {
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	es.ta.CursorStart()
	for i := 0; i < col; i++ {
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyRight})
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
// crossing lines if the current word runs to the end of the line.
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
		for i := col; i < newCol; i++ {
			es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyRight})
		}
	} else if lineIdx+1 < len(lines) {
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyDown})
		es.ta.CursorStart()
		nextRunes := []rune(lines[lineIdx+1])
		i := 0
		for i < len(nextRunes) && (nextRunes[i] == ' ' || nextRunes[i] == '\t') {
			es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyRight})
			i++
		}
	}
	return es
}

// moveWordBackward implements vim `b`: jump to the start of the current or
// previous word, crossing lines when already at column 0.
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
		for i := col; i > newCol; i-- {
			es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyLeft})
		}
	} else if lineIdx > 0 {
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyUp})
		prevRunes := []rune(lines[lineIdx-1])
		es.ta.CursorEnd()
		if len(prevRunes) > 0 {
			i := len(prevRunes) - 1
			isSpace := func(r rune) bool { return r == ' ' || r == '\t' }
			for i >= 0 && isSpace(prevRunes[i]) {
				i--
			}
			if i >= 0 {
				start, _ := wordBoundary(prevRunes, i)
				for j := len(prevRunes); j > start; j-- {
					es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyLeft})
				}
			}
		}
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

func (es editorState) handleInsert(msg tea.KeyMsg) (editorState, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc", "ctrl+[":
		if es.insertSnapshot != nil && es.ta.Value() != es.insertSnapshot.content {
			es.undoStack = append(es.undoStack, *es.insertSnapshot)
			es.redoStack = nil
		}
		es.insertSnapshot = nil
		es.mode = vimNormal
		return es, nil
	}
	var cmd tea.Cmd
	es.ta, cmd = es.ta.Update(msg)
	es.dirty = true
	return es, cmd
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
