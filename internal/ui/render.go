package ui

import (
	"fmt"
	"strings"
	"time"

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
	isMarked := m.marked[g.index]
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
				ch = m.settings.BorderCorner
			} else if isBorderY {
				ch = m.settings.BorderH
			} else if isBorderX {
				ch = m.settings.BorderV
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

	// Header: ID + Title with marker for marked cards
	headerText := strings.TrimSpace(card.ID + " " + card.Title)
	if len(headerText) == 0 {
		headerText = "(untitled)"
	}
	headerRunes := []rune(headerText)
	markerRunes := []rune{}
	if isMarked {
		markerRunes = []rune("* ")
		headerRunes = append(markerRunes, headerRunes...)
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
	if len(headerRunes) > headerMaxWidth {
		if headerMaxWidth > 3 {
			headerRunes = append(headerRunes[:headerMaxWidth-3], []rune("...")...)
		} else {
			headerRunes = headerRunes[:headerMaxWidth]
		}
	}

	markerLen := len(markerRunes)
	hasMarks := len(m.marked) > 0
	for i, r := range headerRunes {
		x := g.x + 1 + i
		if x < 0 || x >= maxX {
			continue
		}
		style := headerID
		if hasMarks && !isMarked {
			style = styleCardMuted
		}
		if isMarked && i < markerLen {
			style = styleCardMark
		}
		grid[headerY][x] = cell{
			ch:      r,
			styleID: style,
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
	content := sanitizeContent(card.Content)

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
	isMarked := m.marked[m.cursor]

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
				ch = m.settings.BorderCorner
			} else if isBorderY {
				ch = m.settings.BorderH
			} else if isBorderX {
				ch = m.settings.BorderV
			} else {
				ch = ' '
				styleID = bodyID
			}

			grid[yy][xx] = cell{ch: ch, styleID: styleID}
		}
	}

	// Header: ID + Title with marker
	headerText := strings.TrimSpace(card.ID + " " + card.Title)
	if len(headerText) == 0 {
		headerText = "(untitled)"
	}
	headerRunes := []rune(headerText)
	markerRunes := []rune{}
	if isMarked {
		markerRunes = []rune("* ")
		headerRunes = append(markerRunes, headerRunes...)
	}

	headerY := y + 1
	if headerY >= 0 && headerY < maxY {
		headerMaxWidth := cardW - 2
		if headerMaxWidth > 0 {
			if len(headerRunes) > headerMaxWidth {
				if headerMaxWidth > 3 {
					headerRunes = append(headerRunes[:headerMaxWidth-3], []rune("...")...)
				} else {
					headerRunes = headerRunes[:headerMaxWidth]
				}
			}
			markerLen := len(markerRunes)
			for i, r := range headerRunes {
				xx := x + 1 + i
				if xx < 0 || xx >= maxX {
					continue
				}
				style := headerID
				if isMarked && i < markerLen {
					style = styleCardMark
				}
				grid[headerY][xx] = cell{ch: r, styleID: style}
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

	allLines := wrapText(content, bodyW, -1) // effectively unlimited

	start := m.overlayPage
	if start < 0 {
		start = 0
	}
	end := start + bodyH
	if end > len(allLines) {
		end = len(allLines)
	}
	if start > end {
		start = max(0, end-bodyH)
	}
	lines := allLines[start:end]

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

	// Edge cues
	if start > 0 && bodyH > 0 && bodyY < maxY {
		cx := bodyX + bodyW - 1
		if cx >= 0 && cx < maxX {
			grid[bodyY][cx] = cell{ch: '^', styleID: styleCardMuted}
		}
	}
	if end < len(allLines) && bodyH > 0 {
		lastLine := bodyY + bodyH - 1
		cx := bodyX + bodyW - 1
		if lastLine < maxY && cx >= 0 && cx < maxX {
			grid[lastLine][cx] = cell{ch: 'v', styleID: styleCardMuted}
		}
	}
}

// wrapText wraps a string into lines of at most `width` runes,
// up to `maxLines` lines, preferring to break on spaces.
// If a single word is longer than `width`, it is hard-broken.
func wrapText(s string, width, maxLines int) []string {
	if width <= 0 {
		return nil
	}

	if maxLines < 0 {
		maxLines = int(^uint(0) >> 1)
	}
	if maxLines == 0 {
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
	vis := m.visibleIndices()
	if len(vis) == 0 {
		leftParts = append(leftParts, dimStyle.Render("no cards"))
	} else {
		posIdx := m.visibleCursorIndex(vis)
		if posIdx < 0 {
			posIdx = 0
		}
		pos := fmt.Sprintf("Card %d/%d", posIdx+1, len(vis))
		leftParts = append(leftParts, dimStyle.Render(pos))
		if m.state == StateViewing {
			bodyH, total := m.overlayLimits()
			if bodyH > 0 && total > bodyH {
				pages := (total + bodyH - 1) / bodyH
				pageIdx := m.overlayPage/bodyH + 1
				if pageIdx > pages {
					pageIdx = pages
				}
				leftParts = append(leftParts, dimStyle.Render(fmt.Sprintf("Page %d/%d", pageIdx, pages)))
			}
		}
	}
	if m.filterMarked && len(m.marked) > 0 {
		leftParts = append(leftParts, statusStyle.Render("[Marked filter]"))
	} else if len(m.marked) > 0 {
		leftParts = append(leftParts, dimStyle.Render(fmt.Sprintf("%d marked", len(m.marked))))
	}

	// Ephemeral status message
	var rightParts []string
	if m.statusMsg != "" && time.Now().Before(m.statusMsgUntil) {
		rightParts = append(rightParts, statusStyle.Render(m.statusMsg))
	}

	modeStr := m.state.String()
	if m.showHelp {
		modeStr = "Help"
	}

	navHint := " [j/k] move  [r] random  [q] quit"
	if m.showHelp {
		navHint = " [esc] close help  [?/h] toggle"
	}

	rightText := modeStr + navHint
	if len(rightParts) > 0 {
		rightText = rightText + "  " + strings.Join(rightParts, " | ")
	}
	right := dimStyle.Render(rightText)
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
		keys []string
		desc string
	}

	bindings := []binding{
		{keys: m.bindings.Up, desc: "move into stack"},
		{keys: m.bindings.Down, desc: "move back/out"},
		{keys: m.bindings.Random, desc: "jump to random card"},
		{keys: m.bindings.Mark, desc: "mark/unmark card"},
		{keys: m.bindings.Filter, desc: "toggle marked-only filter"},
		{keys: m.bindings.OverlayToggle, desc: "toggle overlay view"},
		{keys: nil, desc: ""},
		{keys: []string{"(overlay only)"}, desc: ""},
		{keys: m.bindings.OverlayDown, desc: "scroll overlay by 1 line"},
		{keys: m.bindings.PageNext, desc: "next page"},
		{keys: m.bindings.PagePrev, desc: "previous page"},
		{keys: nil, desc: ""},
		{keys: m.bindings.Quit, desc: "quit (from stack) / back (from overlay)"},
		{keys: []string{"esc"}, desc: "close overlay or help"},
		{keys: m.bindings.Help, desc: "toggle this help overlay"},
		{keys: m.bindings.Debug, desc: "toggle debug overlay"},
		{keys: []string{"ctrl+c"}, desc: "quit immediately"},
	}

	leftWidth := 0
	for _, b := range bindings {
		keyStr := strings.Join(b.keys, " / ")
		if len(keyStr) > leftWidth {
			leftWidth = len(keyStr)
		}
	}

	var sb strings.Builder
	title := lipgloss.NewStyle().Foreground(m.settings.ColorHiFG).Bold(true).Render("Thumbr Help")
	sb.WriteString(title)
	sb.WriteString("\n\n")

	for _, b := range bindings {
		keyStr := strings.Join(b.keys, " / ")
		pad := strings.Repeat(" ", leftWidth-len(keyStr))
		line := fmt.Sprintf("  %s%s  %s\n", keyStr, pad, b.desc)
		sb.WriteString(line)
	}

	body := sb.String()
	return body + "\n" + m.renderStatusBar()
}

func (m Model) renderDebug() string {
	if m.viewport.Width == 0 {
		return "Debug\n"
	}

	vis := m.visibleIndices()
	bodyH, totalLines := m.overlayLimits()
	updateAvg := time.Duration(0)
	if len(m.updateSamples) > 0 {
		var sum time.Duration
		for _, d := range m.updateSamples {
			sum += d
		}
		updateAvg = sum / time.Duration(len(m.updateSamples))
	}

	var sb strings.Builder
	title := lipgloss.NewStyle().Foreground(m.settings.ColorHiFG).Bold(true).Render("Debug")
	sb.WriteString(title)
	sb.WriteString("\n\n")

	lines := []string{
		fmt.Sprintf("Root: %s", m.noteRoot),
		fmt.Sprintf("Cards: %d (visible: %d, marked: %d, filter: %v)", len(m.cards), len(vis), len(m.marked), m.filterMarked),
		fmt.Sprintf("Page step: %d, bodyH: %d, totalLines: %d", m.pageStep(), bodyH, totalLines),
		fmt.Sprintf("Nav accel: %v, max step: %d", m.settings.NavAccelWindow, m.settings.NavMaxStep),
		fmt.Sprintf("Cursor depth max: %d", m.settings.MaxCursorDepth),
		fmt.Sprintf("Update avg (last %d): %v", len(m.updateSamples), updateAvg),
	}
	for _, line := range lines {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String() + "\n" + m.renderStatusBar()
}
