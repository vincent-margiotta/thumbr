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

type editorState struct {
	ta      textarea.Model
	mode    vimMode
	path    string
	dirty   bool
	pending string // partial multi-char sequence: "g", "d", "y", "r"
	yankBuf string
	cmdLine string // content after : in command mode
	saveErr error
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

// ---------------------------------------------------------------------------
// Top-level editor key dispatcher (method on Model so it can call applyEditorAction)
// ---------------------------------------------------------------------------

func (m Model) handleEditorKey(msg tea.KeyMsg, start time.Time) (tea.Model, tea.Cmd) {
	key := msg.String()
	es := m.editor

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
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyCtrlRight})
	case "b":
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyCtrlLeft})
	case "0":
		es.ta.CursorStart()
	case "$":
		es.ta.CursorEnd()
	case "G":
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyCtrlEnd})
	case "x":
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyDelete})
		es.dirty = true
	case "u":
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	case "ctrl+r":
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	case "i":
		es.mode = vimInsert
	case "a":
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyRight})
		es.mode = vimInsert
	case "A":
		es.ta.CursorEnd()
		es.mode = vimInsert
	case "o":
		es.ta.CursorEnd()
		es.ta.InsertString("\n")
		es.dirty = true
		es.mode = vimInsert
	case "O":
		es.ta.CursorStart()
		es.ta.InsertString("\n")
		es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyUp})
		es.dirty = true
		es.mode = vimInsert
	case "p":
		if es.yankBuf != "" {
			es.ta.CursorEnd()
			es.ta.InsertString("\n" + es.yankBuf)
			es.dirty = true
		}
	case ":":
		es.mode = vimCommand
		es.cmdLine = ""
	case "esc", "ctrl+[":
		es.pending = ""
	case "g", "d", "y", "r":
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
		if key == "d" {
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			if lineIdx >= 0 && lineIdx < len(lines) {
				newLines := append(lines[:lineIdx:lineIdx], lines[lineIdx+1:]...)
				es.ta.SetValue(strings.Join(newLines, "\n"))
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
			es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyDelete})
			es.ta.InsertString(key)
			es.ta, _ = es.ta.Update(tea.KeyMsg{Type: tea.KeyLeft})
			es.dirty = true
		}
	}
	return es
}

func (es editorState) handleInsert(msg tea.KeyMsg) (editorState, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc", "ctrl+[":
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
