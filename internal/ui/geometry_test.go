// internal/ui/geometry_test.go
package ui

import (
	"fmt"
	"testing"

	"thumbr/internal/notes"
)

func newTestModel(totalCards, cursor int) Model {
	cards := make([]notes.Card, totalCards)
	for i := 0; i < totalCards; i++ {
		cards[i] = notes.Card{
			ID:    fmt.Sprintf("%d", i+1),
			Title: fmt.Sprintf("Card %d", i+1),
		}
	}
	m := Model{
		cards:    cards,
		cursor:   cursor,
		state:    StateBrowsing,
		settings: DefaultSettings,
		viewport: Viewport{
			Width:  120,
			Height: 40,
		},
	}
	return m
}

func TestComputeStackGeometry_Front(t *testing.T) {
	m := newTestModel(10, 0)
	m.settings.StackVisibleCount = 5
	m.settings.MaxCursorDepth = 2

	geoms := m.computeStackGeometry()
	if len(geoms) != 5 {
		t.Fatalf("expected 5 visible cards, got %d", len(geoms))
	}

	// At the very front, the window should start at index 0
	if geoms[0].index != 0 {
		t.Fatalf("expected front card index 0, got %d", geoms[0].index)
	}

	// The active card is index 0 and should be at depth 0 (front-most)
	var active cardGeom
	for _, g := range geoms {
		if g.index == m.cursor {
			active = g
			break
		}
	}
	if active.index != 0 {
		t.Fatalf("expected active card index 0, got %d", active.index)
	}
	if active.depth != 0 {
		t.Fatalf("expected active card depth 0 at front, got %d", active.depth)
	}
}

func TestComputeStackGeometry_MiddleKeepsCursorAtMaxDepth(t *testing.T) {
	m := newTestModel(10, 5)
	m.settings.StackVisibleCount = 5
	m.settings.MaxCursorDepth = 2

	geoms := m.computeStackGeometry()
	if len(geoms) != 5 {
		t.Fatalf("expected 5 visible cards, got %d", len(geoms))
	}

	// In the middle, the window should be [3..7] and the cursor (5)
	// should be at depth MaxCursorDepth (2).
	if geoms[0].index != 3 || geoms[len(geoms)-1].index != 7 {
		t.Fatalf("expected visible window [3..7], got [%d..%d]",
			geoms[0].index, geoms[len(geoms)-1].index)
	}

	var active cardGeom
	for _, g := range geoms {
		if g.index == m.cursor {
			active = g
			break
		}
	}
	if active.depth != m.settings.MaxCursorDepth {
		t.Fatalf("expected active card at depth %d, got %d",
			m.settings.MaxCursorDepth, active.depth)
	}
}

func TestComputeStackGeometry_Back(t *testing.T) {
	m := newTestModel(10, 9) // last card
	m.settings.StackVisibleCount = 5
	m.settings.MaxCursorDepth = 2

	geoms := m.computeStackGeometry()
	if len(geoms) != 5 {
		t.Fatalf("expected 5 visible cards, got %d", len(geoms))
	}

	// Near the back, the window should clamp to the last possible start
	// index (10 - 5 = 5), so we expect indices [5..9].
	if geoms[0].index != 5 || geoms[len(geoms)-1].index != 9 {
		t.Fatalf("expected visible window [5..9], got [%d..%d]",
			geoms[0].index, geoms[len(geoms)-1].index)
	}

	// Active card is the last one and should sit at the deepest layer.
	var active cardGeom
	for _, g := range geoms {
		if g.index == m.cursor {
			active = g
			break
		}
	}
	expectedDepth := len(geoms) - 1
	if active.depth != expectedDepth {
		t.Fatalf("expected active card depth %d at back, got %d",
			expectedDepth, active.depth)
	}
}

func TestCardSize_MonotonicWithViewport(t *testing.T) {
	// Smaller viewport
	mSmall := newTestModel(1, 0)
	mSmall.viewport = Viewport{Width: 80, Height: 24}
	w1, h1 := mSmall.cardSize()
	if w1 <= 0 || h1 <= 0 {
		t.Fatalf("expected positive card size for small viewport, got %dx%d", w1, h1)
	}

	// Larger viewport
	mLarge := newTestModel(1, 0)
	mLarge.viewport = Viewport{Width: 160, Height: 48}
	w2, h2 := mLarge.cardSize()
	if w2 <= 0 || h2 <= 0 {
		t.Fatalf("expected positive card size for large viewport, got %dx%d", w2, h2)
	}

	if w2 < w1 {
		t.Fatalf("expected card width to grow with viewport, got %d -> %d", w1, w2)
	}
	if h2 < h1 {
		t.Fatalf("expected card height to grow with viewport, got %d -> %d", h1, h2)
	}
}

func TestCardSize_AspectReasonable(t *testing.T) {
	m := newTestModel(1, 0)
	m.viewport = Viewport{Width: 120, Height: 40}
	w, h := m.cardSize()
	if w <= 0 || h <= 0 {
		t.Fatalf("expected positive card size, got %dx%d", w, h)
	}

	ratio := float64(w) / float64(h)
	// We don't demand exact equality with visualAspect (3.0), just that
	// it's in a plausible "landscape card" range.
	if ratio < 2.0 || ratio > 4.0 {
		t.Fatalf("expected width/height ratio in [2.0, 4.0], got %.2f (%dx%d)", ratio, w, h)
	}
}
