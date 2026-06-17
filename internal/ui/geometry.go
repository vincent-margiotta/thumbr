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

// cardSizeOriented returns the card dimensions for the given orientation.
// portrait=false → landscape (6"×4", aspect 3.0); portrait=true → portrait (4"×6", aspect 4/3).
func (m Model) cardSizeOriented(portrait bool) (int, int) {
	vw := float64(m.viewport.Width)
	vh := float64(m.viewport.Height)
	if vw <= 0 || vh <= 0 {
		return 0, 0
	}

	var aspect, maxW float64
	if portrait {
		// Portrait: 4"×6" — terminal col:row ratio = (4×2)/6 = 4/3.
		aspect = 4.0 / 3.0
		if m.settings.TextWidth > 0 {
			// Scale width from landscape textWidth by the physical 4:6 ratio.
			maxW = float64(m.settings.TextWidth)*2.0/3.0 + 4
		} else {
			maxW = vw * 0.4
		}
	} else {
		// Landscape: 6"×4" — terminal col:row ratio = (6×2)/4 = 3.
		aspect = 3.0
		if m.settings.TextWidth > 0 {
			maxW = float64(m.settings.TextWidth + 4)
		} else {
			maxW = vw * 0.6
		}
	}

	if maxW > vw-2 {
		maxW = vw - 2
	}

	maxH := vh - 2

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

	if w < 20 {
		w = 20
	}
	if h < 6 {
		h = 6
	}
	if w > vw-2 {
		w = vw - 2
	}
	if h > vh-2 {
		h = vh - 2
	}

	return int(w + 0.5), int(h + 0.5)
}

// cardSize returns dimensions for the current UI context (portrait or landscape).
func (m Model) cardSize() (int, int) {
	return m.cardSizeOriented(m.isPortraitContext())
}
