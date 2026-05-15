package ui

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"thumbr/internal/notes"
)

// Update and navigation-related methods live here.

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	start := time.Now()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		if msg.Height > 1 {
			m.viewport.Height = msg.Height - 1
		} else {
			m.viewport.Height = msg.Height
		}
		if m.state == StatePrompting {
			width := m.viewport.Width - 4
			if width < 10 {
				width = m.viewport.Width
			}
			if width < 10 {
				width = 10
			}
			m.prompt.input.Width = width
		}
		if m.state == StateEditing {
			m.editor.ta.SetWidth(msg.Width)
			h := msg.Height - 3
			if h < 1 {
				h = 1
			}
			m.editor.ta.SetHeight(h)
		}
		m.ready = true
		return m.withUpdateSample(start), nil

	case boxLoadResult:
		if msg.err != nil {
			m.err = msg.err
			m = m.setStatus(fmt.Sprintf("Load failed: %v", msg.err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		m = m.resetAfterLoad(msg.cards, msg.path)
		m = m.setStatus(fmt.Sprintf("Loaded %d cards", len(msg.cards)), 2*time.Second)
		if m.pendingEditorPath != "" {
			path := m.pendingEditorPath
			m.pendingEditorPath = ""
			return m.withUpdateSample(start), openInAppCmd(path)
		}
		vis := m.visibleIndices()
		n := m.settings.StackVisibleCount + 2
		if n > len(vis) {
			n = len(vis)
		}
		return m.withUpdateSample(start), preloadCardsCmd(m.cards, vis[:n])

	case newFileResult:
		if msg.err != nil {
			m.err = msg.err
			m = m.setStatus(fmt.Sprintf("New file error: %v", msg.err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		m = m.setActiveBox(msg.box)
		status := "Opening file in editor"
		if msg.path != "" {
			status = fmt.Sprintf("Opening %s", filepath.Base(msg.path))
		}
		m.pendingSeekPath = msg.seekPath
		if msg.openInApp && msg.path != "" {
			m.pendingEditorPath = msg.path
			status = fmt.Sprintf("Created %s", filepath.Base(msg.path))
		}
		m = m.setStatus(status, 2*time.Second)
		return m.withUpdateSample(start), m.loadBoxCmd(msg.box)

	case openInAppResult:
		if msg.err != nil {
			m.err = msg.err
			m = m.setStatus(fmt.Sprintf("Open failed: %v", msg.err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		cursorLine := m.pendingEditorLine
		m.pendingEditorLine = 0
		es, cmd := newEditorState(msg.path, msg.content, m.viewport, cursorLine)
		m.editor = es
		m.state = StateEditing
		return m.withUpdateSample(start), cmd

	case cardContentResult:
		for i := range m.cards {
			if m.cards[i].Path == msg.path && !m.cards[i].ContentLoaded {
				m.cards[i].Content = msg.content
				m.cards[i].ContentLoaded = true
				m.cards[i].ContentErr = msg.err
				break
			}
		}
		return m.withUpdateSample(start), nil

	case editorResult:
		if msg.err != nil {
			m.err = msg.err
			m.editorErrors++
			m = m.setStatus(fmt.Sprintf("Open failed: %v", msg.err), 3*time.Second)
		} else {
			m = m.setStatus("Opening in editor…", 2*time.Second)
		}
		return m.withUpdateSample(start), nil

	case tea.KeyMsg:
		key := msg.String()
		m.keyCount++

		// Reset pending quit on any non-quit key or if it expired.
		if !m.isBinding(key, m.bindings.Quit) {
			m.pendingQuit = false
		}
		if m.pendingQuit && time.Now().After(m.pendingQuitUntil) {
			m.pendingQuit = false
		}

		// ctrl+c always quits
		if key == "ctrl+c" {
			return m.withUpdateSample(start), tea.Quit
		}

		if m.state == StatePrompting {
			return m.handlePromptKey(msg, start)
		}

		// When editing, all keys go to the editor — no global actions should fire.
		if m.state == StateEditing {
			return m.handleEditorKey(msg, start)
		}

		if m.enableDebug && m.isBinding(key, m.bindings.Debug) {
			m.showDebug = !m.showDebug
			if m.showDebug {
				m.showHelp = false
			}
			return m, nil
		}

		if m.isBinding(key, m.bindings.Help) {
			m.showHelp = !m.showHelp
			if m.showHelp {
				m.showDebug = false
			}
			return m.withUpdateSample(start), nil
		}

		// When help is open, close on any key (without applying it).
		if m.showHelp {
			m.showHelp = false
			return m.withUpdateSample(start), nil
		}
		// When debug is open, close on any key (without applying it).
		if m.showDebug {
			m.showDebug = false
			return m.withUpdateSample(start), nil
		}

		// Global actions (apply in all states).
		switch {
		case m.isBinding(key, m.bindings.Mark):
			m = m.toggleMark()
			return m, nil
		case m.isBinding(key, m.bindings.Filter):
			m = m.toggleFilter()
			return m, nil
		case m.isBinding(key, m.bindings.OpenBox):
			m = m.startBoxPrompt()
			return m.withUpdateSample(start), nil
		case m.isBinding(key, m.bindings.NewFile):
			m = m.startNewFilePrompt()
			return m.withUpdateSample(start), nil
		case m.isBinding(key, m.bindings.Continue):
			return m.startContinueOrBranch(true, start)
		case m.isBinding(key, m.bindings.Branch):
			return m.startContinueOrBranch(false, start)
		case m.isBinding(key, m.bindings.NextRoot):
			return m.startNextRoot(start)
		case m.isBinding(key, m.bindings.Reload):
			m = m.setStatus("Reloading…", 1*time.Second)
			return m.withUpdateSample(start), m.loadBoxCmd(m.currentBox())
		case m.isBinding(key, m.bindings.OpenInApp):
			if m.editor.path != "" {
				// Resume suspended editor. If it's dirty and the user is on a
				// different card, warn rather than silently discarding their work.
				if m.editor.dirty && len(m.cards) > 0 && m.cards[m.cursor].Path != m.editor.path {
					m = m.setStatus(fmt.Sprintf("Suspended editor has unsaved changes — press e on %s to resume, or :q! to discard", filepath.Base(m.editor.path)), 4*time.Second)
					return m.withUpdateSample(start), nil
				}
				m.state = StateEditing
				return m.withUpdateSample(start), nil
			}
			if len(m.cards) > 0 {
				return m.withUpdateSample(start), openInAppCmd(m.cards[m.cursor].Path)
			}
			return m.withUpdateSample(start), nil
		case m.isBinding(key, m.bindings.OpenExternal):
			if len(m.cards) > 0 {
				cmd := m.openInEditorCmd(m.cards[m.cursor].Path)
				m = m.setStatus("Opening in editor…", 2*time.Second)
				return m.withUpdateSample(start), cmd
			}
			return m.withUpdateSample(start), nil
		}

		// If we have no cards, let q also quit, otherwise do nothing
		if len(m.cards) == 0 {
			if key == "q" {
				return m, tea.Quit
			}
			return m, nil
		}

		var navCmd tea.Cmd

		switch m.state {

		case StateBrowsing:
			switch {
			case m.isBinding(key, m.bindings.Quit):
				m, cmd := m.attemptQuit()
				return m.withUpdateSample(start), cmd
			case m.isBinding(key, m.bindings.OverlayToggle):
				// Pull current card out into viewing overlay
				m.state = StateViewing
				m.overlayPage = 0
				m = m.ensureCardContent(m.cursor)
			case m.isBinding(key, m.bindings.Down):
				m = m.moveCursor(-1)
				navCmd = m.preloadVisibleCmd()
			case m.isBinding(key, m.bindings.Up):
				m = m.moveCursor(1)
				navCmd = m.preloadVisibleCmd()
			case m.isBinding(key, m.bindings.Random):
				m = m.randomCursor()
				navCmd = m.preloadVisibleCmd()
			case m.isBinding(key, m.bindings.NavFirst):
				m = m.jumpToEdge(-1)
				navCmd = m.preloadVisibleCmd()
			case m.isBinding(key, m.bindings.NavLast):
				m = m.jumpToEdge(1)
				navCmd = m.preloadVisibleCmd()
			}

		case StateViewing:
			m = m.ensureCardContent(m.cursor)
			switch {
			case key == "esc" || m.isBinding(key, m.bindings.Quit):
				// Drop back into stack browsing
				m.state = StateBrowsing
				m.overlayPage = 0

			case m.isBinding(key, m.bindings.OverlayToggle):
				// Toggle viewing off (back to browsing)
				m.state = StateBrowsing
				m.overlayPage = 0

			case m.isBinding(key, m.bindings.OverlayDown):
				m = m.scrollOverlayLines(1)
			case m.isBinding(key, m.bindings.OverlayUp):
				m = m.scrollOverlayLines(-1)

			case m.isBinding(key, m.bindings.Random):
				if m.settings.StickyOverlayNav {
					m = m.randomCursor()
					m = m.ensureCardContent(m.cursor)
				} else {
					m.state = StateBrowsing
					m.overlayPage = 0
					m = m.randomCursor()
				}
			case m.isBinding(key, m.bindings.PageNext):
				m = m.nextPage()
			case m.isBinding(key, m.bindings.PagePrev):
				m = m.prevPage()
			}
		}

		return m.withUpdateSample(start), navCmd
	}

	return m.withUpdateSample(start), nil
}

func (m Model) handlePromptKey(msg tea.KeyMsg, start time.Time) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m = m.clearPrompt()
		return m.withUpdateSample(start), nil
	case "tab":
		m = m.cyclePromptBox(1)
		return m.withUpdateSample(start), nil
	case "shift+tab":
		m = m.cyclePromptBox(-1)
		return m.withUpdateSample(start), nil
	case "enter":
		return m.submitPrompt(start)
	}

	var cmd tea.Cmd
	m.prompt.input, cmd = m.prompt.input.Update(msg)
	return m.withUpdateSample(start), cmd
}

func (m Model) submitPrompt(start time.Time) (tea.Model, tea.Cmd) {
	switch m.prompt.kind {
	case promptBox:
		path := strings.TrimSpace(m.prompt.input.Value())
		if path == "" {
			m = m.setStatus("Enter a box path", 2*time.Second)
			return m.withUpdateSample(start), nil
		}
		m = m.clearPrompt()
		return m.withUpdateSample(start), m.loadBoxCmd(path)

	case promptNewFile:
		name := strings.TrimSpace(m.prompt.input.Value())
		if name == "" {
			m = m.setStatus("Enter a file name", 2*time.Second)
			return m.withUpdateSample(start), nil
		}
		target := m.promptTargetBox()
		cmd := m.createFileCmd(target, name)
		m = m.clearPrompt()
		m = m.setStatus("Creating file…", 1*time.Second)
		return m.withUpdateSample(start), cmd
	}

	m = m.clearPrompt()
	return m.withUpdateSample(start), nil
}

func (m Model) cyclePromptBox(delta int) Model {
	if len(m.boxes) == 0 {
		return m
	}
	count := len(m.boxes)
	m.prompt.selectedBox = (m.prompt.selectedBox + delta + count) % count
	if m.prompt.kind == promptBox {
		m.prompt.input.SetValue(m.boxes[m.prompt.selectedBox])
	}
	return m
}

func (m Model) promptTargetBox() string {
	if len(m.boxes) == 0 {
		return m.noteRoot
	}
	if m.prompt.selectedBox >= 0 && m.prompt.selectedBox < len(m.boxes) {
		return m.boxes[m.prompt.selectedBox]
	}
	return m.currentBox()
}

// ==== Navigation ====

func (m Model) moveCursor(dir int) Model {
	vis := m.visibleIndices()
	if dir == 0 || len(vis) == 0 {
		return m
	}

	pos := m.visibleCursorIndex(vis)
	if pos < 0 {
		m.cursor = vis[0]
		pos = 0
	}

	now := time.Now()
	dt := float64(now.Sub(m.lastNavTime).Milliseconds())
	sameDir := dir == m.lastNavDir

	m.lastNavDir = dir
	m.lastNavTime = now

	// Physics-based momentum: each same-direction press multiplies navVelocity by
	// a growth factor; elapsed time decays it exponentially. Pressing faster than
	// ~160ms/press accelerates; slower causes gradual decay. Direction change resets.
	const growth = 1.6  // multiplier per press
	const decay = 0.75  // fraction remaining per 100ms of elapsed time
	const initV = 1.0 / growth // so first press always yields step=1

	if !sameDir || m.avgInterval == 0 {
		m.avgInterval = initV
	} else if dt > 0 {
		m.avgInterval *= math.Pow(decay, dt/100.0)
		if m.avgInterval < initV {
			m.avgInterval = initV
		}
	}
	m.avgInterval *= growth

	maxStep := len(vis) / 10
	if maxStep < m.settings.NavMaxStep {
		maxStep = m.settings.NavMaxStep
	}

	step := int(m.avgInterval)
	if step < 1 {
		step = 1
	}
	if step > maxStep {
		step = maxStep
	}

	remaining := pos
	if dir > 0 {
		remaining = len(vis) - 1 - pos
	}
	if step > remaining {
		step = remaining
	}
	newPos := pos + dir*step
	if newPos < 0 {
		newPos = 0
	}
	if newPos >= len(vis) {
		newPos = len(vis) - 1
	}

	m.cursor = vis[newPos]
	return m
}

// jumpToEdge moves to the first (dir>0) or last (dir<0) visible card and resets momentum.
func (m Model) jumpToEdge(dir int) Model {
	vis := m.visibleIndices()
	if len(vis) == 0 {
		return m
	}
	if dir > 0 {
		m.cursor = vis[len(vis)-1]
	} else {
		m.cursor = vis[0]
	}
	m.avgInterval = 0
	m.lastNavDir = 0
	return m
}

func (m Model) randomCursor() Model {
	vis := m.visibleIndices()
	if len(vis) == 0 {
		return m
	}
	if m.rng != nil {
		m.cursor = vis[m.rng.Intn(len(vis))]
	} else {
		m.cursor = vis[rand.Intn(len(vis))]
	}
	m.avgInterval = 0
	m.lastNavDir = 0
	return m
}

func (m Model) startContinueOrBranch(isContinue bool, start time.Time) (tea.Model, tea.Cmd) {
	if len(m.cards) == 0 {
		return m.withUpdateSample(start), nil
	}

	card := m.cards[m.cursor]
	base := filepath.Base(card.Path)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	if ext == "" {
		ext = m.defaultExt()
	}

	var newStem string
	var ok bool

	if isContinue {
		if m.settings.ContinueNameCmd != "" {
			if derived, err := deriveStemViaCmd(m.settings.ContinueNameCmd, stem); err == nil && derived != "" {
				newStem, ok = derived, true
			}
		}
		if !ok {
			newStem, ok = notes.DeriveContinuation(stem)
		}
	} else {
		if m.settings.BranchNameCmd != "" {
			if derived, err := deriveStemViaCmd(m.settings.BranchNameCmd, stem); err == nil && derived != "" {
				newStem, ok = derived, true
			}
		}
		if !ok {
			newStem, ok = notes.DeriveBranch(stem)
		}
	}

	if !ok {
		m = m.startNewFilePrompt()
		return m.withUpdateSample(start), nil
	}

	var targetDir string
	if m.settings.NewFileSameDir {
		targetDir = filepath.Dir(card.Path)
	} else {
		targetDir = m.currentBox()
	}

	// Walk forward through DeriveBranch until we find a name that doesn't exist on disk.
	for {
		if _, err := os.Stat(filepath.Join(targetDir, newStem+ext)); os.IsNotExist(err) {
			break
		}
		next, ok2 := notes.DeriveBranch(newStem)
		if !ok2 {
			m = m.startNewFilePrompt()
			return m.withUpdateSample(start), nil
		}
		newStem = next
	}

	// For branch, the backwards link points to the parent (prefix), not the source card.
	// For continue, the source card IS the parent.
	linkStem := stem
	if !isContinue {
		if prefix, _, _, ok2 := notes.SplitLuhmannStem(stem); ok2 && prefix != "" {
			linkStem = prefix
		} else {
			linkStem = ""
		}
	}

	var linkContent string
	if linkStem != "" && m.settings.NewFileLinkTemplate != "" {
		linkContent = fmt.Sprintf(m.settings.NewFileLinkTemplate, linkStem)
	}

	var cmd tea.Cmd
	switch m.settings.NewFileEditor {
	case "external":
		cmd = m.createLinkedFileCmd(m.currentBox(), targetDir, newStem, ext, linkContent)
	case "none":
		cmd = m.createLinkedFileCmdNoEditor(m.currentBox(), targetDir, newStem, ext, linkContent, false)
	default: // "inapp" or ""
		// Layout: \n[cursor]\n\n[reference]\n — cursor at line 1, above the link.
		inAppContent := linkContent
		if inAppContent != "" {
			inAppContent = "\n\n\n" + strings.TrimSuffix(inAppContent, "\n")
			m.pendingEditorLine = 1
		}
		cmd = m.createLinkedFileCmdNoEditor(m.currentBox(), targetDir, newStem, ext, inAppContent, true)
	}
	m = m.setStatus(fmt.Sprintf("Creating %s…", newStem+ext), 1*time.Second)
	return m.withUpdateSample(start), cmd
}

func (m Model) startNextRoot(start time.Time) (tea.Model, tea.Cmd) {
	stems := make([]string, 0, len(m.cards))
	for _, card := range m.cards {
		base := filepath.Base(card.Path)
		stem := strings.TrimSuffix(base, filepath.Ext(base))
		stems = append(stems, stem)
	}

	newStem, ok := notes.NextRootInteger(stems)
	if !ok {
		m = m.startNewFilePrompt()
		return m.withUpdateSample(start), nil
	}

	ext := m.defaultExt()
	if len(m.cards) > 0 {
		if e := filepath.Ext(filepath.Base(m.cards[m.cursor].Path)); e != "" {
			ext = e
		}
	}

	targetDir := m.currentBox()
	for {
		if _, err := os.Stat(filepath.Join(targetDir, newStem+ext)); os.IsNotExist(err) {
			break
		}
		n, _ := strconv.Atoi(newStem)
		newStem = strconv.Itoa(n + 1)
	}

	var cmd tea.Cmd
	switch m.settings.NewFileEditor {
	case "external":
		cmd = m.createLinkedFileCmd(m.currentBox(), targetDir, newStem, ext, "")
	case "none":
		cmd = m.createLinkedFileCmdNoEditor(m.currentBox(), targetDir, newStem, ext, "", false)
	default:
		cmd = m.createLinkedFileCmdNoEditor(m.currentBox(), targetDir, newStem, ext, "", true)
	}
	m = m.setStatus(fmt.Sprintf("Creating %s…", newStem+ext), 1*time.Second)
	return m.withUpdateSample(start), cmd
}
