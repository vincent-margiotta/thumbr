// editor.go — modal in-app vim-style text editor state and key handling.

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	pending        string // partial multi-char sequence: "g", "d", "dg", "y", "yg", "r", "c"
	yankBuf        string
	yankLinewise   bool   // true for line yanks (yy/dd), false for char yanks (yw/D/x)
	cmdLine        string // content after : in command mode
	saveErr        error
	undoStack      []undoEntry
	redoStack      []undoEntry
	insertSnapshot *undoEntry                    // state captured on insert-mode entry
	insertLog      string                        // chars typed since insert-mode entry (for dot repeat)
	insertEntry    func(editorState) editorState // pre-insert action to replay on dot repeat
	lastRepeat     func(editorState) editorState // nil until a repeatable change is made

	// section editing: when section is "front" or "back", the textarea holds only
	// that side of the card and otherSide holds the other half. save() reconstructs
	// the full file. section == "" means the whole file is in the textarea.
	section      string
	otherSide    string
	backPortrait bool // true when the back side uses portrait orientation (---back:portrait---)

	// count prefix support (normal mode)
	countBuf              string // accumulated digit count for pending motion/insert
	insertCount           int    // count passed into current insert session
	insertEntryRepeatable bool   // true for o/O: entry should be repeated per count
	pendingFillChar       rune   // character to use for :fill command
	pendingRenameTarget   string // new filename stem for :rename command
}

// fullContent returns the complete file content, reconstructing from section parts if needed.
// When editing the front and the back is empty, the delimiter is omitted so saving a
// front-only edit does not create a spurious empty back section.
func (es editorState) fullContent() string {
	v := es.ta.Value()
	switch es.section {
	case "front":
		if strings.TrimSpace(es.otherSide) == "" {
			return v
		}
		return joinCardSides(v, es.otherSide)
	case "back":
		if es.backPortrait {
			return joinCardSidesPortrait(es.otherSide, v)
		}
		return joinCardSides(es.otherSide, v)
	}
	return v
}

type editorAction int

const (
	editorActionNone editorAction = iota
	editorActionSave
	editorActionQuit
	editorActionSaveQuit
	editorActionSaveQuitAll
	editorActionForceQuit
	editorActionSort
	editorActionSwitchBack
	editorActionSwitchBackPortrait
	editorActionSwitchFront
	editorActionFill
	editorActionRename
	editorActionHelp
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

// editorWidth returns the effective textarea width: capped at textWidth when set,
// so text soft-wraps at the configured column without inserting literal newlines.
func editorWidth(vpWidth, textWidth int) int {
	if textWidth > 0 && textWidth < vpWidth {
		return textWidth
	}
	return vpWidth
}

func newEditorState(path, content string, vp Viewport, textWidth, cursorLine int) (editorState, tea.Cmd) {
	ta := textarea.New()
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	h := vp.Height - 3
	if h < 1 {
		h = 1
	}
	ta.SetWidth(editorWidth(vp.Width, textWidth))
	ta.SetHeight(h)
	ta.SetValue(content)
	cmd := ta.Focus()
	// SetValue leaves cursor at end; jump directly to the top then step down to cursorLine.
	ta = taKey(ta, tea.KeyCtrlHome)
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
	content := es.fullContent()
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	err := os.WriteFile(es.path, []byte(content), 0o644)
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

	if m.editorHelpVisible {
		m.editorHelpVisible = false
		return m.withUpdateSample(start), nil
	}

	es := m.editors[m.activePane]

	if key == "ctrl+s" {
		if err := es.save(); err != nil {
			m.editors[m.activePane] = es
			m = m.setStatus(fmt.Sprintf("Save failed: %v", err), 3*time.Second)
		} else {
			m.editors[m.activePane] = es
			m = m.syncCardContent(es.path, es.fullContent())
			m = m.setStatus("Saved", 2*time.Second)
		}
		return m.withUpdateSample(start), nil
	}

	if m.settings.CardSizeLimit {
		_, cardH := m.cardSize()
		maxLines := cardH - 4
		if maxLines > 0 {
			totalLines := len(strings.Split(es.ta.Value(), "\n"))
			if totalLines >= maxLines {
				blocked := false
				switch es.mode {
				case vimInsert:
					blocked = msg.Type == tea.KeyEnter
				case vimNormal:
					blocked = es.pending == "" && (key == "o" || key == "O" ||
						(key == "p" && es.yankLinewise))
				}
				if blocked {
					msg := "Card full — use --no-card-limit to disable"
					if !m.settings.FreeMode {
						msg = "Card full — use c/C to continue or branch"
					}
					m = m.setStatus(msg, 3*time.Second)
					return m.withUpdateSample(start), nil
				}
			}
		}
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
		m = m.syncCardContent(es.path, es.fullContent())
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
		m = m.syncCardContent(es.path, es.fullContent())
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
			m = m.syncCardContent(pane.path, pane.fullContent())
		}
		m.editors = [2]editorState{}
		m.paneCount = 0
		m.activePane = 0
		m.state = StateBrowsing
		return m.withUpdateSample(start), nil

	case editorActionSwitchBack:
		return m.switchEditorSection(m.activePane, "back", false, start)

	case editorActionSwitchBackPortrait:
		return m.switchEditorSection(m.activePane, "back", true, start)

	case editorActionSwitchFront:
		return m.switchEditorSection(m.activePane, "front", false, start)

	case editorActionFill:
		cardW, _ := m.cardSize()
		// targetW is one less than the card body width: the textarea's wrap function
		// uses >= (not >) for its final word flush, so a line of exactly bodyW chars
		// wraps onto a second visual row. Filling to bodyW-1 avoids this edge case.
		targetW := cardW - 5
		if targetW <= 0 {
			m = m.setStatus(":fill requires a visible card (resize terminal or set textWidth)", 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		fillChar := es.pendingFillChar
		if fillChar == 0 {
			fillChar = '.'
		}
		es.pendingFillChar = 0

		lines := strings.Split(es.ta.Value(), "\n")
		lineIdx := es.ta.Line()
		if lineIdx >= 0 && lineIdx < len(lines) {
			runes := []rune(lines[lineIdx])
			// Find the first contiguous run of fillChar.
			runStart, runEnd := -1, -1
			for i, r := range runes {
				if r == fillChar {
					if runStart == -1 {
						runStart = i
					}
					runEnd = i + 1
				} else if runStart != -1 {
					break
				}
			}
			if runStart == -1 {
				m.editors[m.activePane] = es
				m = m.setStatus(fmt.Sprintf("No run of '%c' on current line", fillChar), 3*time.Second)
				return m.withUpdateSample(start), nil
			}
			needed := targetW - (len(runes) - (runEnd - runStart))
			if needed < 0 {
				needed = 0
			}
			es = es.pushUndo()
			lines[lineIdx] = string(runes[:runStart]) + strings.Repeat(string(fillChar), needed) + string(runes[runEnd:])
			es.ta.SetValue(strings.Join(lines, "\n"))
			es = es.setCursorToLineCol(lineIdx, runStart)
			es.dirty = true
			m.editors[m.activePane] = es
			m = m.setStatus(fmt.Sprintf("Line filled to %d chars", targetW), 2*time.Second)
			return m.withUpdateSample(start), nil
		}
		m.editors[m.activePane] = es
		return m.withUpdateSample(start), nil

	case editorActionRename:
		newName := strings.TrimSpace(es.pendingRenameTarget)
		es.pendingRenameTarget = ""
		m.editors[m.activePane] = es
		if newName == "" {
			m = m.setStatus("Usage: :rename <newname>", 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		if !strings.HasSuffix(newName, ".txt") {
			newName += ".txt"
		}
		dir := filepath.Dir(es.path)
		newPath := filepath.Join(dir, newName)
		if newPath == es.path {
			m = m.setStatus("Same filename — nothing to rename", 2*time.Second)
			return m.withUpdateSample(start), nil
		}
		if _, err := os.Stat(newPath); err == nil {
			m = m.setStatus(fmt.Sprintf("File already exists: %s", newName), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		if err := os.Rename(es.path, newPath); err != nil {
			m = m.setStatus(fmt.Sprintf("Rename failed: %v", err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		oldPath := es.path
		es.path = newPath
		m.editors[m.activePane] = es
		// Update the matching card in the current list so the overlay stays correct.
		stem := strings.TrimSuffix(newName, ".txt")
		for i := range m.cards {
			if m.cards[i].Path == oldPath {
				m.cards[i].Path = newPath
				m.cards[i].Title = stem
				break
			}
		}
		m = m.setStatus(fmt.Sprintf("Renamed to %s", newName), 3*time.Second)
		return m.withUpdateSample(start), m.loadBoxCmd(m.noteRoot)

	case editorActionHelp:
		m.editorHelpVisible = true
		return m.withUpdateSample(start), nil

	case editorActionSort:
		lines := strings.Split(es.ta.Value(), "\n")
		sorted := make([]string, len(lines))
		copy(sorted, lines)
		sort.Slice(sorted, func(i, j int) bool {
			ki, kj := dotLeaderSortKey(sorted[i]), dotLeaderSortKey(sorted[j])
			if ki != kj {
				return ki < kj
			}
			return sorted[i] < sorted[j]
		})
		newContent := strings.Join(sorted, "\n")
		if newContent != es.ta.Value() {
			es = es.pushUndo()
			es.ta.SetValue(newContent)
			es = es.setCursorToLineCol(0, 0)
			es.dirty = true
			m.editors[m.activePane] = es
			m = m.setStatus("Sorted", 2*time.Second)
		} else {
			m = m.setStatus("Already sorted", 2*time.Second)
		}
		return m.withUpdateSample(start), nil
	}

	return m.withUpdateSample(start), nil
}

// switchEditorSection transitions the active pane to editing only the front or back
// section of a card. It handles three entry states:
//
//   - section == "": whole file is in the textarea; split at ---back--- (or treat
//     the entire content as front if no delimiter exists) and load the target section.
//   - section == "front" → "back" (or vice versa): swap current content into otherSide
//     and load the other half.
//   - section == target: no-op with a status hint.
func (m Model) switchEditorSection(pane int, target string, forcePortrait bool, start time.Time) (tea.Model, tea.Cmd) {
	es := m.editors[pane]
	current := es.ta.Value()

	if es.section == target {
		m = m.setStatus("Already editing "+target, 2*time.Second)
		return m.withUpdateSample(start), nil
	}

	var newContent, newOther string
	var newPortrait bool
	switch es.section {
	case "":
		front, back, _, filePortrait := splitCardSides(current)
		if target == "back" {
			newContent, newOther = back, front
			newPortrait = filePortrait || forcePortrait
		} else {
			newContent, newOther = front, back
		}
	default:
		// section is "front" or "back" — swap
		newContent, newOther = es.otherSide, current
		newPortrait = es.backPortrait || forcePortrait
	}

	vp := m.viewport
	if m.paneCount == 2 {
		topVP, botVP := splitViewports(m.viewport)
		if pane == 0 {
			vp = topVP
		} else {
			vp = botVP
		}
	}

	newES, cmd := newEditorState(es.path, newContent, vp, m.settings.TextWidth, 0)
	newES.section = target
	newES.otherSide = newOther
	newES.dirty = es.dirty
	if target == "back" {
		newES.backPortrait = newPortrait
	}
	m.editors[pane] = newES
	return m.withUpdateSample(start), cmd
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
		m.editors[remaining].ta.SetWidth(editorWidth(m.viewport.Width, m.settings.TextWidth))
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
	// Accumulate count prefix: digits 1-9 start a count; 0 continues one.
	if len([]rune(key)) == 1 {
		d := []rune(key)[0]
		if d >= '1' && d <= '9' || (d == '0' && es.countBuf != "") {
			es.countBuf += key
			return es, editorActionNone
		}
	}
	count, _ := strconv.Atoi(es.countBuf)
	if count < 1 {
		count = 1
	}
	es.countBuf = ""

	switch key {
	case "h":
		for i := 0; i < count; i++ {
			es.ta = taKey(es.ta, tea.KeyLeft)
		}
	case "l":
		for i := 0; i < count; i++ {
			es.ta = taKey(es.ta, tea.KeyRight)
		}
	case "j":
		for i := 0; i < count; i++ {
			es.ta = taKey(es.ta, tea.KeyDown)
		}
	case "k":
		for i := 0; i < count; i++ {
			es.ta = taKey(es.ta, tea.KeyUp)
		}
	case "w":
		for i := 0; i < count; i++ {
			es = es.moveWordForward()
		}
	case "b":
		for i := 0; i < count; i++ {
			es = es.moveWordBackward()
		}
	case "e":
		for i := 0; i < count; i++ {
			es = es.moveWordEnd()
		}
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
		lines := strings.Split(es.ta.Value(), "\n")
		lineIdx := es.ta.Line()
		col := es.ta.LineInfo().CharOffset
		if lineIdx >= 0 && lineIdx < len(lines) {
			runes := []rune(lines[lineIdx])
			end := col + count
			if end > len(runes) {
				end = len(runes)
			}
			if col < end {
				es.yankBuf = string(runes[col:end])
				es.yankLinewise = false
				runes = append(runes[:col:col], runes[end:]...)
				lines[lineIdx] = string(runes)
				es.ta.SetValue(strings.Join(lines, "\n"))
				es = es.setCursorToLineCol(lineIdx, col)
				es.dirty = true
			}
		}
		finalCount := count
		es.lastRepeat = func(s editorState) editorState {
			s = s.pushUndo()
			ls := strings.Split(s.ta.Value(), "\n")
			li := s.ta.Line()
			c := s.ta.LineInfo().CharOffset
			if li >= 0 && li < len(ls) {
				r := []rune(ls[li])
				e := c + finalCount
				if e > len(r) {
					e = len(r)
				}
				if c < e {
					s.yankBuf = string(r[c:e])
					s.yankLinewise = false
					r = append(r[:c:c], r[e:]...)
					ls[li] = string(r)
					s.ta.SetValue(strings.Join(ls, "\n"))
					s = s.setCursorToLineCol(li, c)
					s.dirty = true
				}
			}
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
		es.insertEntryRepeatable = false
		es.insertCount = count
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
		es.insertEntryRepeatable = false
		es.insertCount = count
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
		es.insertEntryRepeatable = false
		es.insertCount = count
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
		es.insertEntryRepeatable = true
		es.insertCount = count
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
		es.insertEntryRepeatable = true
		es.insertCount = count
		es.mode = vimInsert
	case "p":
		if es.yankBuf != "" {
			es = es.pushUndo()
			if es.yankLinewise {
				es.ta.CursorEnd()
				es.ta.InsertString("\n" + es.yankBuf)
			} else {
				es.ta = taKey(es.ta, tea.KeyRight)
				es.ta.InsertString(es.yankBuf)
			}
			es.dirty = true
			es.lastRepeat = func(s editorState) editorState {
				if s.yankBuf != "" {
					s = s.pushUndo()
					if s.yankLinewise {
						s.ta.CursorEnd()
						s.ta.InsertString("\n" + s.yankBuf)
					} else {
						s.ta = taKey(s.ta, tea.KeyRight)
						s.ta.InsertString(s.yankBuf)
					}
					s.dirty = true
				}
				return s
			}
		}
	case "D":
		es = es.pushUndo()
		lines := strings.Split(es.ta.Value(), "\n")
		lineIdx := es.ta.Line()
		col := es.ta.LineInfo().CharOffset
		if lineIdx >= 0 && lineIdx < len(lines) {
			runes := []rune(lines[lineIdx])
			if col < len(runes) {
				es.yankBuf = string(runes[col:])
				es.yankLinewise = false
				lines[lineIdx] = string(runes[:col])
				es.ta.SetValue(strings.Join(lines, "\n"))
				es = es.setCursorToLineCol(lineIdx, col)
				es.dirty = true
			}
		}
		es.lastRepeat = func(s editorState) editorState {
			s = s.pushUndo()
			ls := strings.Split(s.ta.Value(), "\n")
			li := s.ta.Line()
			c := s.ta.LineInfo().CharOffset
			if li >= 0 && li < len(ls) {
				r := []rune(ls[li])
				if c < len(r) {
					s.yankBuf = string(r[c:])
					s.yankLinewise = false
					ls[li] = string(r[:c])
					s.ta.SetValue(strings.Join(ls, "\n"))
					s = s.setCursorToLineCol(li, c)
					s.dirty = true
				}
			}
			return s
		}
	case "C":
		es = es.pushUndo()
		lines := strings.Split(es.ta.Value(), "\n")
		lineIdx := es.ta.Line()
		col := es.ta.LineInfo().CharOffset
		if lineIdx >= 0 && lineIdx < len(lines) {
			runes := []rune(lines[lineIdx])
			lines[lineIdx] = string(runes[:col])
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
				ls[li] = string(r[:c])
				s.ta.SetValue(strings.Join(ls, "\n"))
				s = s.setCursorToLineCol(li, c)
				s.dirty = true
			}
			return s
		}
		es.insertEntryRepeatable = false
		es.insertCount = 1
		es.mode = vimInsert
	case ":":
		es.countBuf = ""
		es.mode = vimCommand
		es.cmdLine = ""
	case "esc", "ctrl+[":
		es.pending = ""
		es.countBuf = ""
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
		}

	case "d":
		switch key {
		case "d":
			es = es.pushUndo()
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			if lineIdx >= 0 && lineIdx < len(lines) {
				es.yankBuf = lines[lineIdx]
				es.yankLinewise = true
				newLines := append(lines[:lineIdx:lineIdx], lines[lineIdx+1:]...)
				es.ta.SetValue(strings.Join(newLines, "\n"))
				es = es.setCursorToLineCol(lineIdx, 0)
				es.dirty = true
				es.lastRepeat = func(s editorState) editorState {
					s = s.pushUndo()
					ls := strings.Split(s.ta.Value(), "\n")
					li := s.ta.Line()
					if li >= 0 && li < len(ls) {
						s.yankBuf = ls[li]
						s.yankLinewise = true
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
				es.yankBuf = string(runes[col:end])
				es.yankLinewise = false
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
						s.yankBuf = string(r[c:e])
						s.yankLinewise = false
						ls[li] = string(r[:c]) + string(r[e:])
						s.ta.SetValue(strings.Join(ls, "\n"))
						s = s.setCursorToLineCol(li, c)
						s.dirty = true
					}
					return s
				}
			}
		case "G":
			es = es.pushUndo()
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			if lineIdx >= 0 && lineIdx < len(lines) {
				es.yankBuf = strings.Join(lines[lineIdx:], "\n")
				es.yankLinewise = true
				es.ta.SetValue(strings.Join(lines[:lineIdx], "\n"))
				es = es.setCursorToLineCol(max(0, lineIdx-1), 0)
				es.dirty = true
			}
		case "g":
			es.pending = "dg"
		case "i":
			es.pending = "di"
		}

	case "dg":
		if key == "g" {
			es = es.pushUndo()
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			if lineIdx >= 0 && lineIdx < len(lines) {
				es.yankBuf = strings.Join(lines[:lineIdx+1], "\n")
				es.yankLinewise = true
				es.ta.SetValue(strings.Join(lines[lineIdx+1:], "\n"))
				es = es.setCursorToLineCol(0, 0)
				es.dirty = true
			}
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
				es.yankBuf = string(runes[start:end])
				es.yankLinewise = false
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
						s.yankBuf = string(r[st:en])
						s.yankLinewise = false
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
		switch key {
		case "y":
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			if lineIdx >= 0 && lineIdx < len(lines) {
				es.yankBuf = lines[lineIdx]
				es.yankLinewise = true
			}
		case "w":
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			col := es.ta.LineInfo().CharOffset
			if lineIdx >= 0 && lineIdx < len(lines) {
				runes := []rune(lines[lineIdx])
				end := wordForwardEnd(runes, col)
				es.yankBuf = string(runes[col:end])
				es.yankLinewise = false
			}
		case "G":
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			if lineIdx >= 0 && lineIdx < len(lines) {
				es.yankBuf = strings.Join(lines[lineIdx:], "\n")
				es.yankLinewise = true
			}
		case "g":
			es.pending = "yg"
		case "i":
			es.pending = "yi"
		}

	case "yg":
		if key == "g" {
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			if lineIdx >= 0 && lineIdx < len(lines) {
				es.yankBuf = strings.Join(lines[:lineIdx+1], "\n")
				es.yankLinewise = true
			}
		}

	case "yi":
		if key == "w" {
			lines := strings.Split(es.ta.Value(), "\n")
			lineIdx := es.ta.Line()
			col := es.ta.LineInfo().CharOffset
			if lineIdx >= 0 && lineIdx < len(lines) {
				runes := []rune(lines[lineIdx])
				start, end := wordBoundary(runes, col)
				es.yankBuf = string(runes[start:end])
				es.yankLinewise = false
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

// moveWordEnd implements vim `e`: jump to the last character of the current or
// next word, crossing to the next line when needed.
func (es editorState) moveWordEnd() editorState {
	lines := strings.Split(es.ta.Value(), "\n")
	lineIdx := es.ta.Line()
	col := es.ta.LineInfo().CharOffset
	if lineIdx >= len(lines) {
		return es
	}
	runes := []rune(lines[lineIdx])
	newCol := wordEndForward(runes, col)
	if newCol >= 0 {
		return es.setCursorToLineCol(lineIdx, newCol)
	}
	if lineIdx+1 < len(lines) {
		next := []rune(lines[lineIdx+1])
		i := 0
		isWord := func(r rune) bool {
			return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		}
		isSpace := func(r rune) bool { return r == ' ' || r == '\t' }
		for i < len(next) && isSpace(next[i]) {
			i++
		}
		if i >= len(next) {
			return es.setCursorToLineCol(lineIdx+1, max(0, len(next)-1))
		}
		if isWord(next[i]) {
			for i+1 < len(next) && isWord(next[i+1]) {
				i++
			}
		} else {
			for i+1 < len(next) && !isWord(next[i+1]) && !isSpace(next[i+1]) {
				i++
			}
		}
		return es.setCursorToLineCol(lineIdx+1, i)
	}
	return es
}

// wordEndForward returns the column of the last character of the current or next
// word, implementing vim `e` semantics (cursor lands ON the final char).
// Returns -1 when the motion exhausts the current line and should cross to the next.
func wordEndForward(runes []rune, col int) int {
	n := len(runes)
	if n == 0 {
		return -1
	}
	isWord := func(r rune) bool {
		return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
	}
	isSpace := func(r rune) bool { return r == ' ' || r == '\t' }

	i := col
	// If the cursor is already at a word/token end, advance past it so we seek
	// the *next* word end rather than staying put.
	if i < n {
		if i+1 >= n {
			return -1 // last char of line; cross to next line
		}
		curr, next := runes[i], runes[i+1]
		atWordEnd := (isWord(curr) && !isWord(next)) ||
			(!isWord(curr) && !isSpace(curr) && (isWord(next) || isSpace(next)))
		if atWordEnd {
			i++
		}
	}
	// Skip whitespace.
	for i < n && isSpace(runes[i]) {
		i++
	}
	if i >= n {
		return -1
	}
	// Advance to the last character of this token.
	if isWord(runes[i]) {
		for i+1 < n && isWord(runes[i+1]) {
			i++
		}
	} else {
		for i+1 < n && !isWord(runes[i+1]) && !isSpace(runes[i+1]) {
			i++
		}
	}
	return i
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

func (es editorState) handleInsert(msg tea.KeyMsg, textwidth int) (editorState, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc", "ctrl+[":
		count := es.insertCount
		if count < 1 {
			count = 1
		}
		es.insertCount = 0
		if es.insertSnapshot != nil && es.ta.Value() != es.insertSnapshot.content {
			es.undoStack = append(es.undoStack, *es.insertSnapshot)
			es.redoStack = nil
			log := es.insertLog
			entry := es.insertEntry
			entryRepeatable := es.insertEntryRepeatable

			// Replay typed log (count-1) more times.
			replayLog := func(dst editorState) editorState {
				for _, r := range []rune(log) {
					if r == '\n' {
						dst.ta = taKey(dst.ta, tea.KeyEnter)
					} else {
						dst.ta.InsertString(string(r))
					}
					dst.dirty = true
				}
				return dst
			}
			if entryRepeatable {
				for i := 1; i < count; i++ {
					if entry != nil {
						es = entry(es)
					}
					es = replayLog(es)
				}
			} else {
				for i := 1; i < count; i++ {
					es = replayLog(es)
				}
			}

			es.lastRepeat = func(s editorState) editorState {
				s = s.pushUndo()
				if entry != nil {
					s = entry(s)
				}
				if entryRepeatable {
					for i := 0; i < count; i++ {
						if i > 0 && entry != nil {
							s = entry(s)
						}
						for _, r := range []rune(log) {
							if r == '\n' {
								s.ta = taKey(s.ta, tea.KeyEnter)
							} else {
								s.ta.InsertString(string(r))
							}
							s.dirty = true
						}
					}
				} else {
					for i := 0; i < count; i++ {
						for _, r := range []rune(log) {
							if r == '\n' {
								s.ta = taKey(s.ta, tea.KeyEnter)
							} else {
								s.ta.InsertString(string(r))
							}
							s.dirty = true
						}
					}
				}
				return s
			}
		}
		es.insertSnapshot = nil
		es.insertLog = ""
		es.insertEntry = nil
		es.insertEntryRepeatable = false
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
		case "wqa":
			return es, editorActionSaveQuitAll
		case "sort":
			return es, editorActionSort
		case "help":
			return es, editorActionHelp
		case "back":
			return es, editorActionSwitchBack
		case "back:portrait":
			return es, editorActionSwitchBackPortrait
		case "front":
			return es, editorActionSwitchFront
		default:
			if cmd == "fill" || strings.HasPrefix(cmd, "fill ") {
				fillChar := rune('.')
				rest := strings.TrimSpace(strings.TrimPrefix(cmd, "fill"))
				if runes := []rune(rest); len(runes) > 0 {
					fillChar = runes[0]
				}
				es.pendingFillChar = fillChar
				return es, editorActionFill
			}
			if strings.HasPrefix(cmd, "rename ") {
				es.pendingRenameTarget = strings.TrimSpace(strings.TrimPrefix(cmd, "rename"))
				return es, editorActionRename
			}
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

// dotLeaderSortKey returns the subject heading portion of a dot-leader register
// line (everything before the first " .."), so lines with parenthetical variants
// sort immediately after their base entry rather than after all dot characters.
func dotLeaderSortKey(line string) string {
	if idx := strings.Index(line, " .."); idx >= 0 {
		return strings.TrimSpace(line[:idx])
	}
	return line
}
