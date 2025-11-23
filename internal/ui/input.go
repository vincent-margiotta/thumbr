package ui

import (
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Update and navigation-related methods live here.

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		if msg.Height > 1 {
			m.viewport.Height = msg.Height - 1
		} else {
			m.viewport.Height = msg.Height
		}
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		// ctrl+c always quits
		if key == "ctrl+c" {
			return m, tea.Quit
		}

		// Toggle help overlay.
		if key == "?" || key == "h" {
			m.showHelp = !m.showHelp
			return m, nil
		}

		// When help is open, ignore other keys except closing it.
		if m.showHelp {
			if key == "esc" {
				m.showHelp = false
			}
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
			switch key {
			case "q":
				return m, tea.Quit
			case "enter":
				// Pull current card out into viewing overlay
				m.state = StateViewing
			case "j", "down":
				m = m.moveCursor(-1)
			case "k", "up":
				m = m.moveCursor(1)
			case "r":
				m = m.randomCursor()
			}

		case StateViewing:
			switch key {
			case "esc", "q":
				// Drop back into stack browsing
				m.state = StateBrowsing

			case "enter":
				// Toggle viewing off (back to browsing)
				m.state = StateBrowsing

			case "j", "down":
				if m.settings.StickyOverlayNav {
					m = m.moveCursor(-1)
				} else {
					// Close overlay, then move
					m.state = StateBrowsing
					m = m.moveCursor(-1)
				}

			case "k", "up":
				if m.settings.StickyOverlayNav {
					m = m.moveCursor(1)
				} else {
					m.state = StateBrowsing
					m = m.moveCursor(1)
				}

			case "r":
				if m.settings.StickyOverlayNav {
					m = m.randomCursor()
				} else {
					m.state = StateBrowsing
					m = m.randomCursor()
				}
			}

		case StatePrompting:
			// not used yet; ignore keys for now
		}

		return m, nil
	}

	return m, nil
}

// ==== Navigation ====

func (m Model) moveCursor(dir int) Model {
	if dir == 0 || len(m.cards) == 0 {
		return m
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
			remaining = len(m.cards) - 1 - m.cursor
		} else {
			remaining = m.cursor
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

	newCursor := m.cursor + dir*step
	if newCursor < 0 {
		newCursor = 0
	}
	if newCursor >= len(m.cards) {
		newCursor = len(m.cards) - 1
	}

	m.cursor = newCursor
	return m
}

func (m Model) randomCursor() Model {
	if len(m.cards) == 0 {
		return m
	}
	if m.rng != nil {
		m.cursor = m.rng.Intn(len(m.cards))
	} else {
		m.cursor = rand.Intn(len(m.cards))
	}
	m.velocity = 1
	m.lastNavDir = 0
	return m
}
