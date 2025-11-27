package ui

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
		return m.withUpdateSample(start), nil

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
		m = m.setStatus(status, 2*time.Second)
		return m.withUpdateSample(start), m.loadBoxCmd(msg.box)

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

		// ctrl+c always quits
		if key == "ctrl+c" {
			return m.withUpdateSample(start), tea.Quit
		}

		if m.state == StatePrompting {
			return m.handlePromptKey(msg, start)
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
		case m.isBinding(key, m.bindings.Reload):
			m = m.setStatus("Reloading…", 1*time.Second)
			return m.withUpdateSample(start), m.loadBoxCmd(m.currentBox())
		case m.isBinding(key, m.bindings.OpenEditor):
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

		switch m.state {

		case StateBrowsing:
			switch {
			case m.isBinding(key, m.bindings.Quit):
				return m.withUpdateSample(start), tea.Quit
			case m.isBinding(key, m.bindings.OverlayToggle):
				// Pull current card out into viewing overlay
				m.state = StateViewing
				m.overlayPage = 0
				m = m.ensureCardContent(m.cursor)
			case m.isBinding(key, m.bindings.Down):
				m = m.moveCursor(-1)
			case m.isBinding(key, m.bindings.Up):
				m = m.moveCursor(1)
			case m.isBinding(key, m.bindings.Random):
				m = m.randomCursor()
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

		return m.withUpdateSample(start), nil
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

	// --- Velocity update (same idea as before) ---

	if dir == m.lastNavDir && now.Sub(m.lastNavTime) < m.settings.NavAccelWindow {
		if m.velocity < m.settings.NavMaxStep {
			m.velocity++
		}
	} else {
		m.velocity = 1
	}

	m.lastNavDir = dir
	m.lastNavTime = now

	v := m.velocity
	const fastThreshold = 7 // how many rapid taps before we start "thumbing"

	step := 1

	// --- Thumbr behavior for sustained rapid presses ---

	if v > fastThreshold {
		// How many cards remain in the direction we're moving?
		var remaining int
		if dir > 0 {
			remaining = len(vis) - 1 - pos
		} else {
			remaining = pos
		}

		if remaining > 0 {
			// Velocity 7..NavMaxStep gets mapped to (0,1]
			maxExtra := m.settings.NavMaxStep - fastThreshold
			if maxExtra < 1 {
				maxExtra = 1
			}
			speedFactor := float64(v-fastThreshold) / float64(maxExtra)

			// Base fraction of remaining we jump by when "thumbing".
			baseFraction := 0.18
			frac := baseFraction * speedFactor

			// Keep it in a sane band.
			if frac < 0.10 {
				frac = 0.10
			}
			if frac > 0.45 {
				frac = 0.45
			}

			raw := int(float64(remaining)*frac + 0.5)

			// Near the ends, automatically take smaller bites.
			switch {
			case remaining <= 10:
				// Fine control close to the end.
				step = 1
			case remaining <= 20:
				// A bit chunkier, but never huge.
				if raw < 2 {
					step = 2
				} else if raw > 3 {
					step = 3
				} else {
					step = raw
				}
			default:
				// In the fat middle of the stack, allow larger jumps.
				if raw < 2 {
					step = 2
				} else {
					step = raw
				}
			}
		}
	}

	// --- Apply step ---

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
	m.velocity = 1
	m.lastNavDir = 0
	return m
}
