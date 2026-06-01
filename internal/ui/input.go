// input.go — Bubble Tea Update method and key dispatch for all states.

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

	"github.com/vincent-margiotta/thumbr/internal/notes"
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
		if m.state == StateEditing || m.paneCount >= 1 {
			if m.paneCount == 2 {
				topVP, botVP := splitViewports(m.viewport)
				m.editors[0].ta.SetWidth(editorWidth(topVP.Width, m.settings.TextWidth))
				m.editors[0].ta.SetHeight(topVP.Height - 3)
				m.editors[1].ta.SetWidth(editorWidth(botVP.Width, m.settings.TextWidth))
				m.editors[1].ta.SetHeight(botVP.Height - 3)
			} else if m.paneCount == 1 {
				m.editors[0].ta.SetWidth(editorWidth(m.viewport.Width, m.settings.TextWidth))
				h := m.viewport.Height - 3
				if h < 1 {
					h = 1
				}
				m.editors[0].ta.SetHeight(h)
			}
		}
		m.ready = true
		return m.withUpdateSample(start), nil

	case watchEventMsg:
		if !m.settings.LiveReload {
			return m.withUpdateSample(start), nil
		}
		return m.withUpdateSample(start), tea.Batch(
			m.loadBoxCmd(m.noteRoot),
			watchDirCmd(m.watchStop, m.noteRoot, m.loadOpts),
		)

	case boxLoadResult:
		if msg.err != nil {
			m.err = msg.err
			m = m.setStatus(fmt.Sprintf("Load failed: %v", msg.err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		// When a pending editor open is queued, read the file(s) synchronously so we can
		// transition directly to StateEditing in this Update call — no intermediate browse
		// frame, no blip. Note: resetAfterLoad sets state=StateBrowsing, so we must open
		// the editor AFTER calling it but BEFORE returning.
		pendingPath := m.pendingEditorPath
		pendingSource := m.pendingSourcePath
		m.pendingEditorPath = ""
		m.pendingSourcePath = ""
		// Preserve editor state across background reloads (live-reload watcher firing while
		// editing). Only applies when no pending editor open is queued.
		editorActive := m.paneCount >= 1
		savedState, savedCursor := m.state, m.cursor
		m = m.resetAfterLoad(msg.cards, msg.path)
		// When the active box changes, cancel the old watcher and start one for the new root.
		var newWatchCmd tea.Cmd
		if m.settings.LiveReload && msg.path != m.watchingBox {
			close(m.watchStop)
			m.watchStop = make(chan struct{})
			m.watchingBox = msg.path
			newWatchCmd = watchDirCmd(m.watchStop, msg.path, m.loadOpts)
		}
		if pendingPath != "" {
			cursorLine := m.pendingEditorLine
			m.pendingEditorLine = 0
			if pendingSource != "" && m.settings.AutoSplitOnLink {
				topContent, err := os.ReadFile(pendingSource)
				if err != nil {
					m = m.setStatus(fmt.Sprintf("Open failed: %v", err), 3*time.Second)
					return m.withUpdateSample(start), newWatchCmd
				}
				botContent, err := os.ReadFile(pendingPath)
				if err != nil {
					m = m.setStatus(fmt.Sprintf("Open failed: %v", err), 3*time.Second)
					return m.withUpdateSample(start), newWatchCmd
				}
				topVP, botVP := splitViewports(m.viewport)
				topEs, _ := newEditorState(pendingSource, string(topContent), topVP, m.settings.TextWidth, 0)
				botEs, cmd := newEditorState(pendingPath, string(botContent), botVP, m.settings.TextWidth, cursorLine)
				m.editors[0] = topEs
				m.editors[1] = botEs
				m.paneCount = 2
				m.activePane = 1
				m.state = StateEditing
				return m.withUpdateSample(start), tea.Batch(newWatchCmd, cmd)
			}
			content, err := os.ReadFile(pendingPath)
			if err != nil {
				m = m.setStatus(fmt.Sprintf("Open failed: %v", err), 3*time.Second)
				return m.withUpdateSample(start), newWatchCmd
			}
			es, cmd := newEditorState(pendingPath, string(content), m.viewport, m.settings.TextWidth, cursorLine)
			m.editors[0] = es
			m.editors[1] = editorState{}
			m.paneCount = 1
			m.activePane = 0
			m.state = StateEditing
			return m.withUpdateSample(start), tea.Batch(newWatchCmd, cmd)
		}
		if editorActive {
			m.state = savedState
			m.cursor = savedCursor
		} else {
			m = m.setStatus(fmt.Sprintf("Loaded %d cards", len(msg.cards)), 2*time.Second)
		}
		m = m.ensureVisibleContent()
		return m.withUpdateSample(start), tea.Batch(m.preloadVisibleCmd(), newWatchCmd)

	case newFileResult:
		if msg.err != nil {
			m.err = msg.err
			m = m.setStatus(fmt.Sprintf("New file error: %v", msg.err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
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
		return m.withUpdateSample(start), m.loadBoxCmd(m.noteRoot)

	case openInAppResult:
		if msg.err != nil {
			m.err = msg.err
			m = m.setStatus(fmt.Sprintf("Open failed: %v", msg.err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		cursorLine := m.pendingEditorLine
		m.pendingEditorLine = 0
		// If the file has a back side, open only the front section so ---back---
		// never appears raw in the textarea. The user can reach the back via :back.
		editContent := msg.content
		editSection := ""
		editOther := ""
		if front, back, hasBack := splitCardSides(msg.content); hasBack {
			editContent = front
			editSection = "front"
			editOther = back
		}
		if m.paneCount == 1 && m.editors[0].path != "" {
			// Companion open: resize pane 0 to top, open new file as bottom pane.
			topVP, botVP := splitViewports(m.viewport)
			m.editors[0].ta.SetWidth(editorWidth(topVP.Width, m.settings.TextWidth))
			m.editors[0].ta.SetHeight(topVP.Height - 3)
			es, cmd := newEditorState(msg.path, editContent, botVP, m.settings.TextWidth, cursorLine)
			es.section = editSection
			es.otherSide = editOther
			m.editors[1] = es
			m.paneCount = 2
			m.activePane = 1
			m.state = StateEditing
			m = m.warnIfOversized(editContent)
			return m.withUpdateSample(start), cmd
		}
		// Fresh single-pane open.
		es, cmd := newEditorState(msg.path, editContent, m.viewport, m.settings.TextWidth, cursorLine)
		es.section = editSection
		es.otherSide = editOther
		m.editors[0] = es
		m.editors[1] = editorState{}
		m.paneCount = 1
		m.activePane = 0
		m.state = StateEditing
		m = m.warnIfOversized(editContent)
		return m.withUpdateSample(start), cmd

	case openSplitResult:
		if msg.err != nil {
			m.err = msg.err
			m = m.setStatus(fmt.Sprintf("Open failed: %v", msg.err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		topVP, botVP := splitViewports(m.viewport)
		topContent := msg.topContent
		topSection := ""
		topOther := ""
		if front, back, hasBack := splitCardSides(msg.topContent); hasBack {
			topContent = front
			topSection = "front"
			topOther = back
		}
		topEs, _ := newEditorState(msg.topPath, topContent, topVP, m.settings.TextWidth, 0)
		topEs.section = topSection
		topEs.otherSide = topOther
		cursorLine := m.pendingEditorLine
		m.pendingEditorLine = 0
		botEs, cmd := newEditorState(msg.bottomPath, msg.bottomContent, botVP, m.settings.TextWidth, cursorLine)
		m.editors[0] = topEs
		m.editors[1] = botEs
		m.paneCount = 2
		m.activePane = 1
		m.state = StateEditing
		prevStatus := m.statusMsg
		m = m.warnIfOversized(msg.topContent)
		if m.statusMsg == prevStatus {
			m = m.warnIfOversized(msg.bottomContent)
		}
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
			m = m.setStatus(fmt.Sprintf("Editor error: %v", msg.err), 3*time.Second)
			return m.withUpdateSample(start), nil
		}
		if content, err := os.ReadFile(msg.path); err == nil {
			m = m.syncCardContent(msg.path, string(content))
		}
		m.pendingSeekPath = msg.path
		return m.withUpdateSample(start), m.loadBoxCmd(m.noteRoot)

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

		if m.state == StateBoxMenu {
			return m.handleBoxMenuKey(msg, start)
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
		case m.isBinding(key, m.bindings.Continue):
			if m.settings.FreeMode {
				m = m.setStatus("Not available in free mode", 2*time.Second)
				return m.withUpdateSample(start), nil
			}
			return m.startContinueOrBranch(true, start)
		case m.isBinding(key, m.bindings.Branch):
			if m.settings.FreeMode {
				m = m.setStatus("Not available in free mode", 2*time.Second)
				return m.withUpdateSample(start), nil
			}
			return m.startContinueOrBranch(false, start)
		case m.isBinding(key, m.bindings.NextRoot):
			if m.settings.FreeMode {
				m = m.startNewFilePrompt()
				return m.withUpdateSample(start), nil
			}
			return m.startNextRoot(start)
		case m.isBinding(key, m.bindings.OpenBox):
			if len(m.boxes) <= 1 {
				return m.withUpdateSample(start), nil
			}
			m.boxMenuCursor = m.activeBox
			m.state = StateBoxMenu
			return m.withUpdateSample(start), nil
		case m.isBinding(key, m.bindings.Reload):
			m = m.setStatus("Reloading…", 1*time.Second)
			return m.withUpdateSample(start), m.loadBoxCmd(m.noteRoot)
		case m.isBinding(key, m.bindings.OpenInApp):
			if m.settings.ExternalEditMode {
				if len(m.cards) > 0 {
					return m.withUpdateSample(start), externalEditorCmd(m.cards[m.cursor].Path)
				}
				return m.withUpdateSample(start), nil
			}
			// Section-aware open when the overlay is showing a specific card side.
			if m.state == StateViewing && len(m.cards) > 0 {
				return m.openSectionInAppFromOverlay(start)
			}
			currentPath := ""
			if len(m.cards) > 0 {
				currentPath = m.cards[m.cursor].Path
			}
			// Resume if the current card is already open in a suspended pane.
			if m.paneCount >= 1 && m.editors[0].path == currentPath && currentPath != "" {
				m.state = StateEditing
				m.activePane = 0
				return m.withUpdateSample(start), nil
			}
			if m.paneCount == 2 && m.editors[1].path == currentPath && currentPath != "" {
				m.state = StateEditing
				m.activePane = 1
				return m.withUpdateSample(start), nil
			}
			if currentPath == "" {
				return m.withUpdateSample(start), nil
			}
			// Single suspended + different card → open as companion.
			if m.paneCount == 1 && m.editors[0].path != "" {
				if m.editors[0].dirty {
					m = m.setStatus(fmt.Sprintf("Unsaved changes in %s — save or :q! first", filepath.Base(m.editors[0].path)), 4*time.Second)
					return m.withUpdateSample(start), nil
				}
				return m.withUpdateSample(start), openInAppCmd(currentPath)
			}
			// Split suspended + different card → warn if dirty, else discard and open fresh.
			if m.paneCount == 2 {
				if m.editors[0].dirty || m.editors[1].dirty {
					m = m.setStatus("Unsaved changes — save or :q! first", 4*time.Second)
					return m.withUpdateSample(start), nil
				}
				m.editors = [2]editorState{}
				m.paneCount = 0
				m.activePane = 0
			}
			return m.withUpdateSample(start), openInAppCmd(currentPath)
		case m.isBinding(key, m.bindings.OpenExternal):
			if len(m.cards) > 0 {
				return m.withUpdateSample(start), externalEditorCmd(m.cards[m.cursor].Path)
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
				m.overlayFlipped = false
				m = m.ensureCardContent(m.cursor)
			case m.isBinding(key, m.bindings.Down):
				m = m.moveCursor(-1)
				m = m.ensureVisibleContent()
				navCmd = m.preloadVisibleCmd()
			case m.isBinding(key, m.bindings.Up):
				m = m.moveCursor(1)
				m = m.ensureVisibleContent()
				navCmd = m.preloadVisibleCmd()
			case m.isBinding(key, m.bindings.Random):
				m = m.randomCursor()
				m = m.ensureVisibleContent()
				navCmd = m.preloadVisibleCmd()
			case m.isBinding(key, m.bindings.NavFirst):
				// g enters pending mode; gg resolves to first card.
				if m.navPending == "g" {
					m.navPending = ""
					m = m.jumpToEdge(-1)
					m = m.ensureVisibleContent()
					navCmd = m.preloadVisibleCmd()
				} else {
					m.navPending = "g"
				}
			case m.isBinding(key, m.bindings.NavLast):
				m.navPending = ""
				m = m.jumpToEdge(1)
				m = m.ensureVisibleContent()
				navCmd = m.preloadVisibleCmd()
			case m.navPending == "g" && len(key) == 1 && key[0] >= '1' && key[0] <= '9':
				m.navPending = ""
				m = m.jumpToPercent(int(key[0]-'0') * 10)
				m = m.ensureVisibleContent()
				navCmd = m.preloadVisibleCmd()
			default:
				// Any unrecognised key cancels a pending prefix.
				m.navPending = ""
			}

		case StateViewing:
			m = m.ensureCardContent(m.cursor)
			switch {
			case key == "esc" || m.isBinding(key, m.bindings.Quit):
				// Drop back into stack browsing
				m.state = StateBrowsing
				m.overlayPage = 0
				m.overlayFlipped = false

			case m.isBinding(key, m.bindings.OverlayToggle):
				// Toggle viewing off (back to browsing)
				m.state = StateBrowsing
				m.overlayPage = 0
				m.overlayFlipped = false

			case key == "f":
				m.overlayFlipped = !m.overlayFlipped
				m.overlayPage = 0

			case m.isBinding(key, m.bindings.OverlayDown):
				m = m.scrollOverlayLines(1)
			case m.isBinding(key, m.bindings.OverlayUp):
				m = m.scrollOverlayLines(-1)

			case m.isBinding(key, m.bindings.Random):
				m.state = StateBrowsing
				m.overlayPage = 0
				m = m.randomCursor()
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

// openSectionInAppFromOverlay opens the currently visible card side (front or back)
// in a fresh single-pane editor, with section tracking so save reconstructs the full file.
func (m Model) openSectionInAppFromOverlay(start time.Time) (tea.Model, tea.Cmd) {
	m = m.ensureCardContent(m.cursor)
	card := m.cards[m.cursor]
	front, back, hasBack := splitCardSides(card.Content)

	var sectionContent, section, otherSide string
	if m.overlayFlipped {
		// Open back side — may be empty if no back exists yet; save creates the delimiter.
		sectionContent = back // "" when !hasBack
		section = "back"
		otherSide = front
	} else if hasBack {
		sectionContent = front
		section = "front"
		otherSide = back
	} else {
		sectionContent = card.Content
	}

	// Warn if dirty editor already suspended.
	if m.paneCount >= 1 {
		for i := 0; i < m.paneCount; i++ {
			if m.editors[i].dirty {
				m = m.setStatus("Unsaved changes in editor — save or :q! first", 4*time.Second)
				return m.withUpdateSample(start), nil
			}
		}
	}

	m.overlayFlipped = false
	m.overlayPage = 0
	es, cmd := newEditorState(card.Path, sectionContent, m.viewport, m.settings.TextWidth, 0)
	es.section = section
	es.otherSide = otherSide
	m.editors[0] = es
	m.editors[1] = editorState{}
	m.paneCount = 1
	m.activePane = 0
	m.state = StateEditing
	m = m.warnIfOversized(sectionContent)
	return m.withUpdateSample(start), cmd
}

func (m Model) handleBoxMenuKey(msg tea.KeyMsg, start time.Time) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "esc" || key == "q" || m.isBinding(key, m.bindings.OpenBox):
		m.state = StateBrowsing
	case m.isBinding(key, m.bindings.Down):
		m.boxMenuCursor = (m.boxMenuCursor + 1) % len(m.boxes)
	case m.isBinding(key, m.bindings.Up):
		m.boxMenuCursor = (m.boxMenuCursor - 1 + len(m.boxes)) % len(m.boxes)
	case key == "enter":
		if m.boxMenuCursor != m.activeBox {
			m.state = StateBrowsing
			m, cmd := m.switchToBox(m.boxMenuCursor)
			return m.withUpdateSample(start), cmd
		}
		m.state = StateBrowsing
	}
	return m.withUpdateSample(start), nil
}

func (m Model) handlePromptKey(msg tea.KeyMsg, start time.Time) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m = m.clearPrompt()
		return m.withUpdateSample(start), nil
	case "enter":
		return m.submitPrompt(start)
	}
	var cmd tea.Cmd
	m.prompt.input, cmd = m.prompt.input.Update(msg)
	return m.withUpdateSample(start), cmd
}

func (m Model) submitPrompt(start time.Time) (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.prompt.input.Value())
	if name == "" {
		m = m.setStatus("Enter a file name", 2*time.Second)
		return m.withUpdateSample(start), nil
	}
	if filepath.Ext(name) == "" {
		name += ".txt"
	}
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	ext := filepath.Ext(name)
	m = m.clearPrompt()
	m = m.setStatus(fmt.Sprintf("Creating %s…", name), 1*time.Second)
	var cmd tea.Cmd
	switch m.settings.NewFileEditor {
	case "external":
		cmd = m.createLinkedFileCmd(m.noteRoot, m.noteRoot, stem, ext, "")
	case "none":
		cmd = m.createLinkedFileCmdNoEditor(m.noteRoot, m.noteRoot, stem, ext, "", false)
	default: // "inapp"
		cmd = m.createLinkedFileCmdNoEditor(m.noteRoot, m.noteRoot, stem, ext, "", true)
	}
	return m.withUpdateSample(start), cmd
}

// ==== Navigation ====

// navStep returns the step count for a single directional keypress.
// sameDir is false on direction change or the first press, which always yields 1.
// dt is milliseconds since the last same-direction press.
//
// Power-law model: step = round((tau/dt)^gamma).
// tau is the breakeven interval — pressing every tau ms gives exactly step=1.
// gamma > 1 makes the response non-linear: a modest speedup in pressing rate
// produces a dramatic jump in step size, matching the feel of physical thumbing.
// minDt floors dt so auto-repeat (key-hold) converges with fast deliberate pressing.
func navStep(dt, tau, gamma, minDt float64, sameDir bool, maxStep int) int {
	if !sameDir || dt <= 0 {
		return 1
	}
	if dt < minDt {
		dt = minDt
	}
	ratio := tau / dt
	if ratio <= 1 {
		return 1
	}
	step := int(math.Round(math.Pow(ratio, gamma)))
	if step < 1 {
		step = 1
	}
	if step > maxStep {
		step = maxStep
	}
	return step
}

func (m Model) warnIfOversized(content string) Model {
	if !m.settings.CardSizeLimit {
		return m
	}
	_, cardH := m.cardSize()
	maxLines := cardH - 4
	if maxLines <= 0 {
		return m
	}
	n := len(strings.Split(content, "\n"))
	if n > maxLines {
		return m.setStatus(fmt.Sprintf("Oversized card (%d lines, limit %d) — new lines blocked; use --no-card-limit to disable", n, maxLines), 6*time.Second)
	}
	return m
}

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

	maxStep := len(vis) / 10
	if maxStep < m.settings.NavMaxStep {
		maxStep = m.settings.NavMaxStep
	}

	step := navStep(dt, m.settings.NavTau, m.settings.NavGamma, m.settings.NavMinDt, sameDir, maxStep)

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
	m.lastNavDir = 0
	return m
}

// jumpToPercent moves to pct% through the visible deck (pct in range 0–100).
func (m Model) jumpToPercent(pct int) Model {
	vis := m.visibleIndices()
	if len(vis) == 0 {
		return m
	}
	idx := pct * len(vis) / 100
	if idx >= len(vis) {
		idx = len(vis) - 1
	}
	if idx < 0 {
		idx = 0
	}
	m.cursor = vis[idx]
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
		m = m.setStatus("Cannot derive next card name", 3*time.Second)
		return m.withUpdateSample(start), nil
	}

	targetDir := filepath.Dir(card.Path)

	// Walk forward through DeriveBranch until we find a name that doesn't exist on disk.
	for {
		if _, err := os.Stat(filepath.Join(targetDir, newStem+ext)); os.IsNotExist(err) {
			break
		}
		next, ok2 := notes.DeriveBranch(newStem)
		if !ok2 {
			m = m.setStatus("Cannot derive next card name", 3*time.Second)
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
	if linkStem != "" {
		linkContent = fmt.Sprintf("--> %s\n\n", linkStem)
	}

	var cmd tea.Cmd
	switch m.settings.NewFileEditor {
	case "external":
		cmd = m.createLinkedFileCmd(m.noteRoot, targetDir, newStem, ext, linkContent)
	case "none":
		cmd = m.createLinkedFileCmdNoEditor(m.noteRoot, targetDir, newStem, ext, linkContent, false)
	default: // "inapp" or ""
		// Layout: \n[reference]\n[cursor]\n — cursor at line 2, below the link.
		inAppContent := linkContent
		if inAppContent != "" {
			inAppContent = "\n" + strings.TrimRight(inAppContent, "\n") + "\n\n"
			m.pendingEditorLine = 2
		}
		if m.settings.AutoSplitOnLink {
			m.pendingSourcePath = card.Path
		}
		cmd = m.createLinkedFileCmdNoEditor(m.noteRoot, targetDir, newStem, ext, inAppContent, true)
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
		m = m.setStatus("Cannot derive next root card name", 3*time.Second)
		return m.withUpdateSample(start), nil
	}

	ext := m.defaultExt()

	targetDir := m.noteRoot
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
		cmd = m.createLinkedFileCmd(m.noteRoot, targetDir, newStem, ext, "")
	case "none":
		cmd = m.createLinkedFileCmdNoEditor(m.noteRoot, targetDir, newStem, ext, "", false)
	default:
		cmd = m.createLinkedFileCmdNoEditor(m.noteRoot, targetDir, newStem, ext, "", true)
	}
	m = m.setStatus(fmt.Sprintf("Creating %s…", newStem+ext), 1*time.Second)
	return m.withUpdateSample(start), cmd
}
