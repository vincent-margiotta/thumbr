package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Rendering-related methods: drawing cards, overlay, status bar, etc.

func (m Model) drawCardOntoGrid(grid [][]cell, g cardGeom) {
	if g.w <= 0 || g.h <= 0 {
		return
	}

	card := m.cards[g.index]

	// Style IDs (cached styles elsewhere)
	borderID := styleCardDim
	headerID := styleCardDim
	if g.active {
		borderID = styleCardHi
		headerID = styleCardHi
	}

	maxY := len(grid)
	if maxY == 0 {
		return
	}
	maxX := len(grid[0])

	// Front-of-window card (depth 0) should also be filled,
	// even if it's not the active card.
	fillInterior := g.active || g.depth == 0

	// Draw border / interior
	for dy := 0; dy < g.h; dy++ {
		for dx := 0; dx < g.w; dx++ {
			x := g.x + dx
			y := g.y + dy
			if y < 0 || y >= maxY || x < 0 || x >= maxX {
				continue
			}

			isBorderY := (dy == 0 || dy == g.h-1)
			isBorderX := (dx == 0 || dx == g.w-1)

			// inside the left border for all cards.
			isLeftBand := (dx == 1)

			// For non-active, non-front cards, skip interior cells
			// *except* the left band.
			if !fillInterior && !isBorderX && !isBorderY && !isLeftBand {
				continue
			}

			var ch rune
			if isBorderY && isBorderX {
				ch = '+'
			} else if isBorderY {
				ch = '-'
			} else if isBorderX {
				ch = '|'
			} else {
				// Interior: only for active card and front card.
				ch = ' '
			}

			grid[y][x] = cell{
				ch:      ch,
				styleID: borderID,
			}
		}
	}

	// Header: ID + Title
	headerText := strings.TrimSpace(card.ID + " " + card.Title)
	if len(headerText) == 0 {
		headerText = "(untitled)"
	}

	headerY := g.y + 1
	if headerY < 0 || headerY >= maxY {
		return
	}

	headerMaxWidth := g.w - 2
	if headerMaxWidth <= 0 {
		return
	}

	// Truncate header if needed
	if len([]rune(headerText)) > headerMaxWidth {
		runes := []rune(headerText)
		if headerMaxWidth > 3 {
			runes = append(runes[:headerMaxWidth-3], []rune("...")...)
		} else {
			runes = runes[:headerMaxWidth]
		}
		headerText = string(runes)
	}

	for i, r := range []rune(headerText) {
		x := g.x + 1 + i
		if x < 0 || x >= maxX {
			continue
		}
		grid[headerY][x] = cell{
			ch:      r,
			styleID: headerID,
		}
	}
}

// drawOverlayCardOntoGrid draws the currently active card as a larger,
// centered card that shows content, on top of the existing stack.
func (m Model) drawOverlayCardOntoGrid(grid [][]cell) {
	if len(m.cards) == 0 {
		return
	}
	if m.viewport.Width <= 0 || m.viewport.Height <= 0 {
		return
	}

	card := m.cards[m.cursor]

	maxY := len(grid)
	if maxY == 0 {
		return
	}
	maxX := len(grid[0])

	cardW, cardH := m.cardSize()
	if cardW <= 0 || cardH <= 0 {
		return
	}

	x := (m.viewport.Width - cardW) / 2
	y := (m.viewport.Height - cardH) / 2

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+cardW > m.viewport.Width {
		x = m.viewport.Width - cardW
	}
	if y+cardH > m.viewport.Height {
		y = m.viewport.Height - cardH
	}

	// Style IDs
	borderID := styleOverlayBorder
	headerID := styleOverlayHeader
	bodyID := styleOverlayBody

	// Draw border and clear interior
	for dy := 0; dy < cardH; dy++ {
		for dx := 0; dx < cardW; dx++ {
			xx := x + dx
			yy := y + dy
			if yy < 0 || yy >= maxY || xx < 0 || xx >= maxX {
				continue
			}

			isBorderY := (dy == 0 || dy == cardH-1)
			isBorderX := (dx == 0 || dx == cardW-1)

			var ch rune
			styleID := borderID

			if isBorderY && isBorderX {
				ch = '+'
			} else if isBorderY {
				ch = '-'
			} else if isBorderX {
				ch = '|'
			} else {
				ch = ' '
				styleID = bodyID
			}

			grid[yy][xx] = cell{ch: ch, styleID: styleID}
		}
	}

	// Header: ID + Title
	headerText := strings.TrimSpace(card.ID + " " + card.Title)
	if len(headerText) == 0 {
		headerText = "(untitled)"
	}

	headerY := y + 1
	if headerY >= 0 && headerY < maxY {
		headerMaxWidth := cardW - 2
		if headerMaxWidth > 0 {
			runes := []rune(headerText)
			if len(runes) > headerMaxWidth {
				if headerMaxWidth > 3 {
					runes = append(runes[:headerMaxWidth-3], []rune("...")...)
				} else {
					runes = runes[:headerMaxWidth]
				}
			}
			for i, r := range runes {
				xx := x + 1 + i
				if xx < 0 || xx >= maxX {
					continue
				}
				grid[headerY][xx] = cell{ch: r, styleID: headerID}
			}
		}
	}

	// Body text area (inside the card, below header)
	bodyX := x + 2     // 1 char margin inside the border
	bodyY := y + 2     // line after header
	bodyW := cardW - 4 // 1 char margin on both sides
	bodyH := cardH - 4 // top border + header + bottom border

	if bodyW <= 0 || bodyH <= 0 {
		return
	}

	lines := wrapText(card.Content, bodyW, bodyH)

	for i, line := range lines {
		yy := bodyY + i
		if yy < 0 || yy >= maxY {
			continue
		}
		runes := []rune(line)
		if len(runes) > bodyW {
			runes = runes[:bodyW]
		}
		for j, r := range runes {
			xx := bodyX + j
			if xx < 0 || xx >= maxX {
				continue
			}
			grid[yy][xx] = cell{ch: r, styleID: bodyID}
		}
	}
}

// wrapText wraps a string into lines of at most `width` runes,
// up to `maxLines` lines, preferring to break on spaces.
// If a single word is longer than `width`, it is hard-broken.
func wrapText(s string, width, maxLines int) []string {
	if width <= 0 || maxLines <= 0 {
		return nil
	}

	var lines []string

	// Treat each original line as a separate paragraph to preserve
	// intentional breaks in the note.
	paras := strings.Split(s, "\n")

	for _, para := range paras {
		if len(lines) >= maxLines {
			break
		}

		para = strings.TrimRight(para, " \t\r")
		if para == "" {
			// Blank line
			lines = append(lines, "")
			if len(lines) >= maxLines {
				break
			}
			continue
		}

		words := strings.Fields(para)
		var current []rune

		addLine := func() {
			if len(current) == 0 {
				return
			}
			lines = append(lines, string(current))
			current = nil
		}

		for _, w := range words {
			if len(lines) >= maxLines {
				break
			}

			wRunes := []rune(w)
			if len(current) == 0 {
				// First word on the line
				if len(wRunes) <= width {
					current = append(current, wRunes...)
				} else {
					// Hard-break long word
					for len(wRunes) > 0 && len(lines) < maxLines {
						chunk := wRunes
						if len(chunk) > width {
							chunk = chunk[:width]
						}
						lines = append(lines, string(chunk))
						wRunes = wRunes[len(chunk):]
					}
				}
				continue
			}

			// Check if adding " space + word" fits
			if len(current)+1+len(wRunes) <= width {
				current = append(current, ' ')
				current = append(current, wRunes...)
			} else {
				// Flush current line
				addLine()
				if len(lines) >= maxLines {
					break
				}
				// Start new line with this word, possibly hard-breaking it
				if len(wRunes) <= width {
					current = append(current, wRunes...)
				} else {
					for len(wRunes) > 0 && len(lines) < maxLines {
						chunk := wRunes
						if len(chunk) > width {
							chunk = chunk[:width]
						}
						lines = append(lines, string(chunk))
						wRunes = wRunes[len(chunk):]
					}
				}
			}
		}

		if len(lines) >= maxLines {
			break
		}
		if len(current) > 0 {
			addLine()
		}
	}

	if len(lines) > maxLines {
		return lines[:maxLines]
	}
	return lines
}

func (m Model) renderGrid(grid [][]cell, styles []lipgloss.Style) string {
	if len(grid) == 0 {
		return ""
	}
	var b strings.Builder

	for y := 0; y < len(grid); y++ {
		row := grid[y]
		if len(row) == 0 {
			b.WriteRune('\n')
			continue
		}

		var (
			currentID styleID
			buf       []rune
			hasRun    bool
		)

		flushRun := func() {
			if !hasRun || len(buf) == 0 {
				return
			}
			style := styles[int(currentID)]
			b.WriteString(style.Render(string(buf)))
			buf = buf[:0]
		}

		for x := 0; x < len(row); x++ {
			c := row[x]

			if !hasRun {
				currentID = c.styleID
				buf = append(buf, c.ch)
				hasRun = true
				continue
			}

			if c.styleID == currentID {
				buf = append(buf, c.ch)
			} else {
				flushRun()
				currentID = c.styleID
				buf = append(buf, c.ch)
			}
		}

		flushRun()

		if y < len(grid)-1 {
			b.WriteRune('\n')
		}
	}

	return b.String()
}

func (m Model) renderStatusBar() string {
	statusStyle := lipgloss.NewStyle().
		Background(m.settings.ColorStatusBG).
		Foreground(m.settings.ColorStatusFG)

	dimStyle := lipgloss.NewStyle().
		Background(m.settings.ColorStatusBG).
		Foreground(m.settings.ColorStatusDim)

	var leftParts []string
	leftParts = append(leftParts, statusStyle.Render(" Thumbr "))
	if len(m.cards) == 0 {
		leftParts = append(leftParts, dimStyle.Render("no cards"))
	} else {
		pos := fmt.Sprintf("Card %d/%d", m.cursor+1, len(m.cards))
		leftParts = append(leftParts, dimStyle.Render(pos))
	}

	modeStr := m.state.String()
	if m.showHelp {
		modeStr = "Help"
	}

	navHint := " [j/k] move  [r] random  [q] quit"
	if m.showHelp {
		navHint = " [esc] close help  [?/h] toggle"
	}

	right := dimStyle.Render(modeStr + navHint)
	left := strings.Join(leftParts, " ")

	line := left + strings.Repeat(" ", max(0, m.viewport.Width-len(stripANSI(left))-len(stripANSI(right)))) + right
	return line
}

func (m Model) renderEmpty() string {
	status := m.renderStatusBar()
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	msg := "No cards found.\n\n" +
		"Point Thumbr at a directory of markdown notes:\n\n" +
		"  thumbr /path/to/notes\n"

	body := bodyStyle.Render(msg)

	return body + "\n" + status
}

func (m Model) renderHelp() string {
	if m.viewport.Width == 0 {
		return "Help\n"
	}

	type binding struct {
		keys string
		desc string
	}

	bindings := []binding{
		{keys: "k / up", desc: "move into stack"},
		{keys: "j / down", desc: "move back/out"},
		{keys: "enter", desc: "toggle overlay view"},
		{keys: "r", desc: "jump to random card"},
		{keys: "q", desc: "quit (from stack) / back (from overlay)"},
		{keys: "esc", desc: "close overlay or help"},
		{keys: "?, h", desc: "toggle this help overlay"},
		{keys: "ctrl+c", desc: "quit immediately"},
	}

	leftWidth := 0
	for _, b := range bindings {
		if len(b.keys) > leftWidth {
			leftWidth = len(b.keys)
		}
	}

	var sb strings.Builder
	title := lipgloss.NewStyle().Foreground(m.settings.ColorHiFG).Bold(true).Render("Thumbr Help")
	sb.WriteString(title)
	sb.WriteString("\n\n")

	for _, b := range bindings {
		pad := strings.Repeat(" ", leftWidth-len(b.keys))
		line := fmt.Sprintf("  %s%s  %s\n", b.keys, pad, b.desc)
		sb.WriteString(line)
	}

	body := sb.String()
	return body + "\n" + m.renderStatusBar()
}
