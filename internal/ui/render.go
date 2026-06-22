// render.go — grid rendering, card drawing, overlays, status bar, and help screen.

package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// highlightOverLimit applies a subtle background to textarea view lines that
// exceed the card face capacity. cursorLine is es.ta.Line() — that visual row
// is left untouched so the textarea cursor remains visible.
func (m Model) highlightOverLimit(body string, cursorLine int, portrait bool) string {
	if !m.settings.HighlightOverLimit {
		return body
	}
	_, cardH := m.cardSizeOriented(portrait)
	maxLines := cardH - 4
	if maxLines <= 0 {
		return body
	}
	lines := strings.Split(body, "\n")
	if len(lines) <= maxLines {
		return body
	}
	overStyle := lipgloss.NewStyle().Background(m.settings.ColorOverLimit)
	for i := maxLines; i < len(lines); i++ {
		if i == cursorLine {
			continue // keep original textarea rendering so the cursor stays visible
		}
		text := strings.TrimRight(stripANSI(lines[i]), " ")
		if text != "" {
			lines[i] = overStyle.Render(text)
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderEditorHelp() string {
	dim := lipgloss.NewStyle().Foreground(m.settings.ColorStatusDim)
	hi := lipgloss.NewStyle().Foreground(m.settings.ColorHiFG)
	col := func(label, keys, desc string) string {
		return hi.Render(fmt.Sprintf("  %-16s", label)) + dim.Render(fmt.Sprintf("%-14s", keys)) + dim.Render(desc)
	}
	sec := func(title string) string { return hi.Render(title) }
	lines := []string{
		sec("Navigation"),
		col("", "h j k l", "left / down / up / right"),
		col("", "w  b  e", "word forward / back / end"),
		col("", "0  $", "line start / end"),
		col("", "G", "file end  (ctrl+home = start)"),
		col("", "{n}<motion>", "repeat n times"),
		"",
		sec("Editing"),
		col("", "i  a  A", "insert before / after / end of line"),
		col("", "o  O", "open line below / above"),
		col("", "x", "delete char under cursor"),
		col("", "u   ctrl+r", "undo / redo"),
		col("", ".", "repeat last change"),
		col("", "yy  dd  p", "yank line / delete line / paste"),
		"",
		sec("Commands"),
		col(":w", "", "save"),
		col(":wq  :x", "", "save and quit"),
		col(":q", "", "quit (warns if unsaved)"),
		col(":q!", "", "discard and quit"),
		col(":wqa", "", "save all panes and quit"),
		col(":sort", "", "sort lines alphabetically"),
		col(":fill [c]", "", "pad fill char c to card width"),
		col(":rename <n>", "", "rename file (links not updated)"),
		col(":back", "", "edit back face"),
		col(":back:portrait", "", "edit back face (portrait)"),
		col(":front", "", "edit front face"),
		col(":help", "", "this screen"),
		"",
		sec("Global"),
		col("", "ctrl+s", "save"),
		col("", "ctrl+w", "switch pane (split mode)"),
		col("", "ctrl+b", "suspend to browser"),
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderEditor() string {
	if m.paneCount == 2 {
		return m.renderSplitEditor()
	}
	return m.renderSingleEditor()
}

func (m Model) renderSingleEditor() string {
	es := m.editors[0]

	hi := lipgloss.NewStyle().Foreground(m.settings.ColorHiFG).Bold(true)
	dim := lipgloss.NewStyle().Foreground(m.settings.ColorStatusDim)

	label := modeLabel(es.mode)
	modeStr := hi.Render("[" + label + "]")
	sectionTag := ""
	if es.section == "back" && es.backPortrait {
		sectionTag = " [back:portrait]"
	} else if es.section == "back" {
		sectionTag = " [back]"
	} else if es.section == "front" {
		sectionTag = " [front]"
	}
	dirtyFlag := ""
	if es.dirty {
		dirtyFlag = " [*]"
	}
	posStr := dim.Render(fmt.Sprintf("L%d C%d", es.ta.Line()+1, es.ta.LineInfo().CharOffset+1))
	headerLeft := hi.Render(filepath.Base(es.path) + sectionTag + dirtyFlag)
	padLen := max(0, m.viewport.Width-len(stripANSI(headerLeft))-len(stripANSI(posStr))-1-len(stripANSI(modeStr)))
	header := headerLeft + strings.Repeat(" ", padLen) + posStr + " " + modeStr

	portrait := es.section == "back" && es.backPortrait
	var body string
	if m.editorHelpVisible {
		body = m.renderEditorHelp()
	} else {
		body = m.highlightOverLimit(es.ta.View(), es.ta.Line(), portrait)
	}

	var footer string
	if m.editorHelpVisible {
		footer = dim.Render("press any key to close")
	} else if es.mode == vimCommand {
		footer = hi.Render(":") + es.cmdLine + "█"
	} else if es.mode == vimNormal && es.countBuf != "" {
		footer = dim.Render(es.countBuf)
	}

	return header + "\n" + body + "\n" + footer + "\n" + m.renderStatusBar()
}

func (m Model) renderSplitEditor() string {
	hi := lipgloss.NewStyle().Foreground(m.settings.ColorHiFG).Bold(true)
	dim := lipgloss.NewStyle().Foreground(m.settings.ColorStatusDim)

	renderPaneHeader := func(es editorState, active bool) string {
		sectionTag := ""
		if es.section == "back" && es.backPortrait {
			sectionTag = " [back:portrait]"
		} else if es.section == "back" {
			sectionTag = " [back]"
		} else if es.section == "front" {
			sectionTag = " [front]"
		}
		dirtyFlag := ""
		if es.dirty {
			dirtyFlag = " [*]"
		}
		label := modeLabel(es.mode)
		modeStr := "[" + label + "]"
		posStr := fmt.Sprintf("L%d C%d", es.ta.Line()+1, es.ta.LineInfo().CharOffset+1)
		headerLeft := filepath.Base(es.path) + sectionTag + dirtyFlag
		if active {
			modeStr = hi.Render(modeStr)
			headerLeft = hi.Render(headerLeft)
		} else {
			modeStr = dim.Render(modeStr)
			headerLeft = dim.Render(headerLeft)
		}
		posStr = dim.Render(posStr)
		padLen := max(0, m.viewport.Width-len(stripANSI(headerLeft))-len(stripANSI(posStr))-1-len(stripANSI(modeStr)))
		return headerLeft + strings.Repeat(" ", padLen) + posStr + " " + modeStr
	}

	top0Portrait := m.editors[0].section == "back" && m.editors[0].backPortrait
	top1Portrait := m.editors[1].section == "back" && m.editors[1].backPortrait
	topHeader := renderPaneHeader(m.editors[0], m.activePane == 0)
	topBody := m.highlightOverLimit(m.editors[0].ta.View(), m.editors[0].ta.Line(), top0Portrait)
	divider := strings.Repeat(string(m.settings.BorderH), m.viewport.Width)
	botHeader := renderPaneHeader(m.editors[1], m.activePane == 1)
	botBody := m.highlightOverLimit(m.editors[1].ta.View(), m.editors[1].ta.Line(), top1Portrait)

	if m.editorHelpVisible {
		helpBody := m.renderEditorHelp()
		if m.activePane == 0 {
			topBody = helpBody
		} else {
			botBody = helpBody
		}
	}

	activeES := m.editors[m.activePane]
	var footer string
	if m.editorHelpVisible {
		footer = dim.Render("press any key to close")
	} else if activeES.mode == vimCommand {
		footer = hi.Render(":") + activeES.cmdLine + "█"
	} else if activeES.mode == vimNormal && activeES.countBuf != "" {
		footer = dim.Render(activeES.countBuf)
	}

	return topHeader + "\n" + topBody + "\n" + divider + "\n" + botHeader + "\n" + botBody + "\n" + footer + "\n" + m.renderStatusBar()
}

// Rendering-related methods: drawing cards, overlay, status bar, etc.

func (m Model) drawCardOntoGrid(grid [][]cell, g cardGeom, activeDepth int) {
	if g.w <= 0 || g.h <= 0 {
		return
	}

	card := m.cards[g.index]

	// Style IDs (cached styles elsewhere)
	borderID := styleCardDim
	headerID := styleCardDim
	isMarked := m.isMarked(g.index)
	if g.active {
		borderID = styleCardHi
		headerID = styleCardHi
	}

	maxY := len(grid)
	if maxY == 0 {
		return
	}
	maxX := len(grid[0])

	// Cards at depth <= activeDepth occlude the active card and must be solid
	// so the active card doesn't bleed through their wireframe interiors.
	fillInterior := g.depth <= activeDepth

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
				switch {
				case dy == 0 && dx == 0:
					ch = m.settings.BorderTL
				case dy == 0:
					ch = m.settings.BorderTR
				case dx == 0:
					ch = m.settings.BorderBL
				default:
					ch = m.settings.BorderBR
				}
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
	for i, r := range headerRunes {
		x := g.x + 1 + i
		if x < 0 || x >= maxX {
			continue
		}
		style := headerID
		if isMarked && i < markerLen {
			style = styleCardMark
		}
		grid[headerY][x] = cell{
			ch:      r,
			styleID: style,
		}
	}

	// Back indicator: show ↻ at top-right of the active (depth-0) card when it has a back side.
	if g.active && card.ContentLoaded {
		_, _, hasBack, _ := splitCardSides(card.Content)
		if hasBack {
			indicX := g.x + g.w - 3
			if indicX >= 0 && indicX < maxX && headerY >= 0 && headerY < maxY {
				grid[headerY][indicX] = cell{ch: '↻', styleID: styleCardMuted}
			}
		}
	}

	// Content preview. Depth-0 (frontmost) fills all available interior rows;
	// inner cards only expose their left edge so one line is enough.
	// Always use the front side only — strip any back content.
	if g.h > 3 && card.ContentLoaded && card.Content != "" {
		frontContent, _, _, _ := splitCardSides(card.Content)
		previewWidth := g.w - 4 // 1-char inner margin each side, matching the overlay
		previewX := g.x + 2
		maxRows := 1
		if g.depth == 0 {
			maxRows = g.h - 3 // header row + content rows + bottom border
		}
		if previewWidth > 0 && maxRows > 0 {
			// Front card: show raw content as-is (blank lines, links, everything).
			// Back cards: skip leading blanks/links so the one visible row is useful.
			var displayLines []string
			if g.depth == 0 {
				displayLines = wrapText(frontContent, previewWidth, maxRows)
			} else {
				var filteredLines []string
				seenContent := false
				for _, line := range strings.SplitN(frontContent, "\n", 500) {
					t := strings.TrimSpace(line)
					if !seenContent && (strings.HasPrefix(t, "-->") || t == "") {
						continue
					}
					seenContent = true
					filteredLines = append(filteredLines, strings.TrimLeft(t, "# "))
				}
				displayLines = wrapText(strings.Join(filteredLines, "\n"), previewWidth, maxRows)
			}
			for row, line := range displayLines {
				if row >= maxRows {
					break
				}
				py := g.y + 2 + row
				if py < 0 || py >= maxY {
					break
				}
				runes := []rune(line)
				if len(runes) > previewWidth {
					if previewWidth > 1 {
						runes = append(runes[:previewWidth-1], '…')
					} else {
						runes = runes[:previewWidth]
					}
				}
				contentStyle := styleCardDim
				if g.active {
					contentStyle = styleOverlayBody
				}
				for i, r := range runes {
					x := previewX + i
					if x < 0 || x >= maxX {
						continue
					}
					grid[py][x] = cell{ch: r, styleID: contentStyle}
				}
			}
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
	_, _, hasBack, _ := splitCardSides(card.Content)
	content := m.overlayContent()
	backIsEmpty := m.overlayFlipped && !hasBack

	maxY := len(grid)
	if maxY == 0 {
		return
	}
	maxX := len(grid[0])

	cardW, cardH := m.cardSizeOriented(false)
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
	isMarked := m.isMarked(m.cursor)

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
				switch {
				case dy == 0 && dx == 0:
					ch = m.settings.BorderTL
				case dy == 0:
					ch = m.settings.BorderTR
				case dx == 0:
					ch = m.settings.BorderBL
				default:
					ch = m.settings.BorderBR
				}
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

	// Reserve right-side space in the header for the back indicator.
	backIndicator := []rune{}
	if m.overlayFlipped {
		backIndicator = []rune("BACK")
	} else if hasBack {
		backIndicator = []rune("↻")
	}

	headerY := y + 1
	if headerY >= 0 && headerY < maxY {
		headerMaxWidth := cardW - 2
		if headerMaxWidth > 0 {
			// Truncate header text to leave room for the back indicator.
			availWidth := headerMaxWidth
			if len(backIndicator) > 0 {
				availWidth -= len(backIndicator) + 1 // +1 for a space gap
				if availWidth < 0 {
					availWidth = 0
				}
			}
			if len(headerRunes) > availWidth {
				if availWidth > 3 {
					headerRunes = append(headerRunes[:availWidth-3], []rune("...")...)
				} else {
					headerRunes = headerRunes[:availWidth]
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
			// Draw back indicator right-aligned in the header row.
			if len(backIndicator) > 0 {
				indicStyle := styleCardMuted
				if m.overlayFlipped {
					indicStyle = styleOverlayHeader
				}
				startX := x + cardW - 2 - len(backIndicator)
				for i, r := range backIndicator {
					xx := startX + i
					if xx < 0 || xx >= maxX {
						continue
					}
					grid[headerY][xx] = cell{ch: r, styleID: indicStyle}
				}
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

	// Placeholder when flipped to a back side that doesn't exist yet.
	if backIsEmpty && bodyH > 0 && bodyY < maxY {
		placeholder := []rune("(no back — press e to write one)")
		if len(placeholder) > bodyW {
			placeholder = placeholder[:bodyW]
		}
		for i, r := range placeholder {
			xx := bodyX + i
			if xx < 0 || xx >= maxX {
				continue
			}
			grid[bodyY][xx] = cell{ch: r, styleID: styleCardMuted}
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
	hi := lipgloss.NewStyle().
		Background(m.settings.ColorStatusBG).
		Foreground(m.settings.ColorStatusFG)

	dim := lipgloss.NewStyle().
		Background(m.settings.ColorStatusBG).
		Foreground(m.settings.ColorStatusDim)

	bg := lipgloss.NewStyle().
		Background(m.settings.ColorStatusBG)

	// ---- Left: box name · position · marks ----

	boxLabel := filepath.Base(m.noteRoot)
	if boxLabel == "" || boxLabel == "." || boxLabel == "/" {
		boxLabel = m.noteRoot
	}
	if len(m.boxes) > 1 {
		boxLabel = fmt.Sprintf("%s [%d/%d]", boxLabel, m.activeBox+1, len(m.boxes))
	}

	left := hi.Render(" " + boxLabel + " ")

	vis := m.visibleIndices()
	if len(vis) == 0 {
		left += dim.Render(" · no cards")
	} else {
		posIdx := m.visibleCursorIndex(vis)
		if posIdx < 0 {
			posIdx = 0
		}
		left += dim.Render(fmt.Sprintf(" · %d/%d", posIdx+1, len(vis)))
		if m.state == StateViewing {
			bodyH, total := m.overlayLimits()
			if bodyH > 0 && total > bodyH {
				pages := (total + bodyH - 1) / bodyH
				pageIdx := m.overlayPage/bodyH + 1
				if pageIdx > pages {
					pageIdx = pages
				}
				left += dim.Render(fmt.Sprintf(" · p%d/%d", pageIdx, pages))
			}
		}
	}

	markedCount := m.markedCountCurrent()
	if m.filterMarked && markedCount > 0 {
		left += hi.Render(" [filtered] ")
	} else if markedCount > 0 {
		left += dim.Render(fmt.Sprintf(" · %d marked", markedCount))
	}

	if m.paneCount >= 1 && m.state != StateEditing {
		if m.paneCount == 2 {
			left += dim.Render(fmt.Sprintf(" · ↩ %s / %s",
				filepath.Base(m.editors[0].path), filepath.Base(m.editors[1].path)))
		} else {
			left += dim.Render(fmt.Sprintf(" · ↩ %s", filepath.Base(m.editors[0].path)))
		}
	}

	// ---- Right: ephemeral message or context hints ----

	var right string
	if m.navPending != "" {
		right = hi.Render(" " + m.navPending + "… ")
	} else if m.statusMsg != "" && time.Now().Before(m.statusMsgUntil) {
		right = hi.Render(" " + m.statusMsg + " ")
	} else {
		var hint string
		switch {
		case m.showHelp:
			hint = "esc close · ?/h toggle"
		case m.state == StatePrompting:
			hint = "enter create · esc cancel"
		case m.state == StateEditing:
			if m.paneCount == 2 {
				hint = "ctrl+s save · ctrl+w switch · :wq quit · ctrl+b browse"
			} else {
				hint = "ctrl+s save · :wq quit · ctrl+b browse"
			}
		case m.state == StateViewing:
			if m.overlayFlipped {
				hint = "e edit back · f front · esc back"
			} else if len(m.cards) > 0 {
				_, _, hasBack, _ := splitCardSides(m.cards[m.cursor].Content)
				if hasBack {
					hint = "j/k scroll · n/p page · f flip · esc back"
				} else {
					hint = "j/k scroll · n/p page · f add back · esc back"
				}
			} else {
				hint = "j/k scroll · n/p page · esc back"
			}
		default:
			hint = "j/k · r random · ? help · q quit"
		}
		right = dim.Render(hint + " ")
	}

	// ---- Pad the gap with background so the bar fills the full width ----

	padN := max(0, m.viewport.Width-len(stripANSI(left))-len(stripANSI(right)))
	return left + bg.Render(strings.Repeat(" ", padN)) + right
}

func (m Model) renderEmpty() string {
	status := m.renderStatusBar()
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	msg := "No cards found.\n\n" +
		"Point Thumbr at a directory of .txt notes:\n\n" +
		"  thumbr /path/to/notes\n\n" +
		"Press 'N' to create the first root card, or 'c'/'C' to continue/branch.\n"

	body := bodyStyle.Render(msg)

	return body + "\n" + status
}

func (m Model) renderPrompt() string {
	if m.viewport.Width == 0 {
		return "New Card\n"
	}
	hi := lipgloss.NewStyle().Foreground(m.settings.ColorHiFG).Bold(true)
	dim := lipgloss.NewStyle().Foreground(m.settings.ColorStatusDim)

	var sb strings.Builder
	sb.WriteString(hi.Render("New Card"))
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("Directory: %s\n\n", m.noteRoot))
	sb.WriteString(m.prompt.input.View())
	sb.WriteString("\n\n")
	sb.WriteString(dim.Render("[enter] create  [esc] cancel"))
	return sb.String() + "\n" + m.renderStatusBar()
}

func (m Model) renderBoxMenu() string {
	hi := lipgloss.NewStyle().Foreground(m.settings.ColorHiFG).Bold(true)
	dim := lipgloss.NewStyle().Foreground(m.settings.ColorStatusDim)

	var sb strings.Builder
	sb.WriteString(hi.Render("Switch Box"))
	sb.WriteString("\n\n")

	for i, box := range m.boxes {
		label := filepath.Base(box)
		if label == "" || label == "." || label == "/" {
			label = box
		}
		var tags []string
		if i == m.activeBox {
			tags = append(tags, "active")
		}
		if m.boxFreeMode[box] {
			tags = append(tags, "free")
		}
		if len(tags) > 0 {
			label += " (" + strings.Join(tags, ", ") + ")"
		}
		cursor := "  "
		if i == m.boxMenuCursor {
			cursor = "▶ "
			sb.WriteString(hi.Render(cursor+label) + "\n")
		} else {
			sb.WriteString(dim.Render(cursor+label) + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(dim.Render("j/k navigate  enter select  esc cancel"))
	sb.WriteString("\n")
	return sb.String() + "\n" + m.renderStatusBar()
}

func (m Model) renderHelp() string {
	if m.viewport.Width == 0 {
		return "Help\n"
	}

	hi := lipgloss.NewStyle().Foreground(m.settings.ColorHiFG).Bold(true)
	dim := lipgloss.NewStyle().Foreground(m.settings.ColorStatusDim)

	type row struct {
		section string // non-empty → render as section header, ignore keys/desc
		keys    []string
		desc    string
	}

	chunk := fmt.Sprintf("%d", m.settings.NavChunkSize)
	rows := []row{
		{section: "Movement"},
		{keys: m.bindings.Up, desc: "move into stack"},
		{keys: m.bindings.Down, desc: "move back"},
		{keys: m.bindings.ChunkForward, desc: "jump forward ~" + chunk + " cards"},
		{keys: m.bindings.ChunkBackward, desc: "jump backward ~" + chunk + " cards"},
		{keys: m.bindings.BisectForward, desc: "bisect toward end of deck"},
		{keys: m.bindings.BisectBackward, desc: "bisect toward start of deck"},
		{keys: m.bindings.NavFirst, desc: "first card  (gg)"},
		{keys: m.bindings.NavLast, desc: "last card"},
		{keys: m.bindings.Random, desc: "random card"},

		{section: "View"},
		{keys: m.bindings.OverlayToggle, desc: "open / close overlay"},
		{keys: m.bindings.OpenInApp, desc: func() string {
			if m.settings.ExternalEditMode {
				return "open card in $EDITOR"
			}
			return "open in editor  (companion pane if one is suspended)"
		}()},
		{keys: m.bindings.OpenExternal, desc: "open card in $EDITOR"},

		{section: "Cards"},
		{keys: m.bindings.Mark, desc: "mark / unmark"},
		{keys: m.bindings.Filter, desc: "toggle marked-only filter"},
		{keys: m.bindings.OpenBox, desc: "box picker  (when multiple boxes are open)"},
	}
	if len(m.bindings.Reload) > 0 {
		rows = append(rows, row{keys: m.bindings.Reload, desc: "reload box from disk"})
	}
	rows = append(rows,
		row{section: "Create"},
		row{keys: m.bindings.Continue, desc: func() string {
			if m.settings.FreeMode {
				return "continue card (Luhmann) — disabled in free mode"
			}
			return "continue card  (Luhmann alphanumeric)"
		}()},
		row{keys: m.bindings.Branch, desc: func() string {
			if m.settings.FreeMode {
				return "branch card (Luhmann) — disabled in free mode"
			}
			return "branch card  (Luhmann alphanumeric)"
		}()},
		row{keys: m.bindings.NextRoot, desc: func() string {
			if m.settings.FreeMode {
				return "new card  (prompts for filename)"
			}
			return "next integer root card"
		}()},

		row{section: "Editor"},
		row{keys: m.bindings.SwitchPane, desc: "switch focus between split panes"},
		row{keys: m.bindings.SuspendEditor, desc: "suspend editor, return to browse"},

		row{section: "Overlay"},
		row{keys: m.bindings.OverlayDown, desc: "scroll down"},
		row{keys: m.bindings.OverlayUp, desc: "scroll up"},
		row{keys: m.bindings.PageNext, desc: "next page"},
		row{keys: m.bindings.PagePrev, desc: "previous page"},
		row{keys: []string{"f"}, desc: "flip to back / front"},

		row{section: "General"},
		row{keys: m.bindings.Quit, desc: "quit  (press twice within 3s to confirm; or close overlay)"},
		row{keys: []string{"esc"}, desc: "close overlay or help"},
		row{keys: m.bindings.Help, desc: "this help screen"},
		row{keys: []string{"ctrl+c"}, desc: "quit immediately"},
	)
	if m.enableDebug {
		rows = append(rows, row{keys: m.bindings.Debug, desc: "toggle debug overlay"})
	}

	// Measure key-column width (binding rows only).
	leftWidth := 0
	for _, r := range rows {
		if r.section != "" {
			continue
		}
		w := len(strings.Join(r.keys, " / "))
		if w > leftWidth {
			leftWidth = w
		}
	}

	var sb strings.Builder
	sb.WriteString(hi.Render("Thumbr Help"))
	sb.WriteString("\n")

	for _, r := range rows {
		if r.section != "" {
			sb.WriteString("\n")
			sb.WriteString(hi.Render(r.section))
			sb.WriteString("\n")
			continue
		}
		keyStr := strings.Join(r.keys, " / ")
		pad := strings.Repeat(" ", leftWidth-len(keyStr))
		sb.WriteString("  ")
		sb.WriteString(dim.Render(keyStr + pad))
		sb.WriteString("  ")
		sb.WriteString(r.desc)
		sb.WriteString("\n")
	}

	return sb.String() + "\n" + m.renderStatusBar()
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
		fmt.Sprintf("Cards: %d (visible: %d, marked: %d, filter: %v)", len(m.cards), len(vis), m.markedCountCurrent(), m.filterMarked),
		fmt.Sprintf("Page step: %d, bodyH: %d, totalLines: %d", m.pageStep(), bodyH, totalLines),
		fmt.Sprintf("Nav chunk: %d, jitter: %.2f", m.settings.NavChunkSize, m.settings.NavJitter),
		fmt.Sprintf("Cursor depth max: %d", m.settings.MaxCursorDepth),
	}
	if len(m.updateSamples) > 0 || m.updateMax > 0 {
		lines = append(lines, fmt.Sprintf("Update: avg %v (last %d), max %v, >16ms: %d, >33ms: %d", updateAvg, len(m.updateSamples), m.updateMax, m.updateOver16, m.updateOver33))
	}
	if m.loadDuration > 0 {
		lines = append(lines, fmt.Sprintf("Load: %v (walk %v, sort %v)", m.loadDuration, m.loadWalk, m.loadSort))
	}
	if m.keyCount > 0 {
		lines = append(lines, fmt.Sprintf("Keys processed: %d", m.keyCount))
	}
	if m.contentErrors > 0 || m.editorErrors > 0 {
		lines = append(lines, fmt.Sprintf("Errors: content %d, editor %d", m.contentErrors, m.editorErrors))
	}
	for _, line := range lines {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String() + "\n" + m.renderStatusBar()
}
