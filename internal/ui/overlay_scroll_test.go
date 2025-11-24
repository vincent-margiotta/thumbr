package ui

import (
	"strings"
	"testing"
)

func TestScrollOverlayLinesClampsBounds(t *testing.T) {
	m := newTestModel(1, 0)
	m.state = StateViewing
	m.cards[0].Content = strings.Repeat("line\n", 200) // plenty of content to scroll

	bodyH, total := m.overlayLimits()
	if bodyH <= 0 || total <= bodyH {
		t.Fatalf("expected overlay with positive body height and multiple pages, got bodyH=%d total=%d", bodyH, total)
	}

	// Overshoot forward should clamp to last full page start.
	m.overlayPage = total // way past the end
	m = m.scrollOverlayLines(5)
	maxStart := total - bodyH
	if m.overlayPage != maxStart {
		t.Fatalf("expected forward scroll to clamp at %d, got %d", maxStart, m.overlayPage)
	}

	// Overshoot backward should clamp to zero.
	m = m.scrollOverlayLines(-999)
	if m.overlayPage != 0 {
		t.Fatalf("expected backward scroll to clamp at 0, got %d", m.overlayPage)
	}
}
