package ui

import (
	"math/rand"
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
		m.ready = true
		return m.withUpdateSample(start), nil

	case tea.KeyMsg:
		key := msg.String()

		// ctrl+c always quits
		if key == "ctrl+c" {
			return m.withUpdateSample(start), tea.Quit
		}

		if m.isBinding(key, m.bindings.Debug) {
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
			case m.isBinding(key, m.bindings.Down):
				m = m.moveCursor(-1)
			case m.isBinding(key, m.bindings.Up):
				m = m.moveCursor(1)
			case m.isBinding(key, m.bindings.Random):
				m = m.randomCursor()
			}

		case StateViewing:
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

		case StatePrompting:
			// not used yet; ignore keys for now
		}

		return m.withUpdateSample(start), nil
	}

	return m.withUpdateSample(start), nil
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
