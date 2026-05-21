package ui

// Geometry-related methods: stack layout and card sizing.

func (m Model) computeStackGeometry() []cardGeom {
	if len(m.cards) == 0 || m.viewport.Width <= 0 || m.viewport.Height <= 0 {
		return nil
	}

	vis := m.visibleIndices()
	if len(vis) == 0 {
		return nil
	}

	s := m.settings

	total := len(vis)

	visible := s.StackVisibleCount
	if visible > total {
		visible = total
	}
	if visible <= 0 {
		return nil
	}

	// Clamp MaxCursorDepth into [0, visible-1]
	maxCursorDepth := s.MaxCursorDepth
	if maxCursorDepth < 0 {
		maxCursorDepth = 0
	}
	if maxCursorDepth > visible-1 {
		maxCursorDepth = visible - 1
	}

	// Last possible start index if we want exactly `visible` cards.
	lastWindowStart := total - visible
	if lastWindowStart < 0 {
		lastWindowStart = 0
	}

	// 1) Compute the theoretical window start that would keep the cursor
	//    at `maxCursorDepth` layers from the front.
	cursorPos := m.visibleCursorIndex(vis)
	if cursorPos < 0 {
		cursorPos = 0
	}

	startIdx := cursorPos - maxCursorDepth

	// 2) Clamp that start into [0, lastWindowStart].
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx > lastWindowStart {
		startIdx = lastWindowStart
	}

	endIdx := startIdx + visible - 1
	if endIdx >= total {
		endIdx = total - 1
	}

	if endIdx < startIdx {
		return nil
	}

	visibleCount := endIdx - startIdx + 1
	maxLayer := visibleCount - 1 // deepest layer index

	cardW, cardH := m.cardSize()
	if cardW <= 0 || cardH <= 0 {
		return nil
	}

	// --- Stack placement: bottom-right → top-left ---

	dx := s.StackOffsetX
	dy := s.StackOffsetY

	stackWidth := cardW + maxLayer*dx
	stackHeight := cardH + maxLayer*dy

	// Base position is where the *front* (layer 0) card lives.
	baseX := (m.viewport.Width-stackWidth)/2 + maxLayer*dx
	baseY := (m.viewport.Height-stackHeight)/2 + maxLayer*dy

	var geoms []cardGeom
	for idx := startIdx; idx <= endIdx; idx++ {
		layer := idx - startIdx // 0 = front, increasing towards back

		x := baseX - layer*dx
		y := baseY - layer*dy

		// Active card gets a vertical boost so the title peeks — but not
		// in StateViewing, where the card is "set down" to read via the overlay.
		cardIdx := vis[idx]
		if cardIdx == m.cursor && m.state != StateViewing {
			y -= s.ActiveLiftY
		}

		// Clamp to viewport
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

		geoms = append(geoms, cardGeom{
			index:  cardIdx,
			x:      x,
			y:      y,
			w:      cardW,
			h:      cardH,
			active: cardIdx == m.cursor,
			depth:  layer,
		})
	}

	return geoms
}

// cardSize computes the width/height of an index-card-shaped rectangle in
// terminal cells, preserving a 4"×6" landscape aspect ratio.
//
// Width priority: CardWidthFrac (explicit) → TextWidth+4 (auto) → 60% viewport.
// Height priority: CardHeightFrac (explicit) → derived from aspect ratio.
// Either fraction set to 0 activates the automatic behaviour for that axis.
func (m Model) cardSize() (int, int) {
	vw := float64(m.viewport.Width)
	vh := float64(m.viewport.Height)
	if vw <= 0 || vh <= 0 {
		return 0, 0
	}

	// A 4"×6" index card (landscape) has a 6:4 = 1.5 physical aspect ratio.
	// Terminal cells are roughly 2× taller than wide, so the visual column:row
	// ratio that reproduces the physical shape is 1.5 × 2 = 3.0.
	const aspect = 3.0

	// Target width.
	var maxW float64
	switch {
	case m.settings.CardWidthFrac > 0:
		maxW = vw * m.settings.CardWidthFrac
	case m.settings.TextWidth > 0:
		// Anchor to the configured line width: +2 for left/right borders,
		// +2 for one column of inner margin on each side.
		maxW = float64(m.settings.TextWidth + 4)
	default:
		maxW = vw * 0.6
	}
	if maxW > vw-2 {
		maxW = vw - 2
	}

	// Target height.
	var maxH float64
	if m.settings.CardHeightFrac > 0 {
		maxH = vh * m.settings.CardHeightFrac
	} else {
		maxH = vh - 2 // unconstrained; aspect ratio drives the actual height
	}
	if maxH > vh-2 {
		maxH = vh - 2
	}

	// Fit within maxW × maxH while preserving the aspect ratio.
	var w, h float64
	hFromW := maxW / aspect
	if hFromW <= maxH {
		w, h = maxW, hFromW
	} else {
		h = maxH
		w = maxH * aspect
		if w > maxW {
			w = maxW
		}
	}

	// Minimums so it still looks like a card.
	if w < 20 {
		w = 20
	}
	if h < 6 {
		h = 6
	}

	// Final viewport clamp.
	if w > vw-2 {
		w = vw - 2
	}
	if h > vh-2 {
		h = vh - 2
	}

	return int(w + 0.5), int(h + 0.5)
}
