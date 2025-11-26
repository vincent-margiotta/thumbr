package ui

import "testing"

func TestVisibleIndicesRespectFilterAndOrder(t *testing.T) {
	m := newTestModel(3, 0)

	// No marks + filter → no visible cards.
	m.filterMarked = true
	if vis := m.visibleIndices(); vis != nil {
		t.Fatalf("expected no visible cards when filter on and none marked, got %v", vis)
	}

	// Marks should be returned in card order.
	m.marked[m.cards[2].Path] = true
	m.marked[m.cards[0].Path] = true
	vis := m.visibleIndices()
	if len(vis) != 2 || vis[0] != 0 || vis[1] != 2 {
		t.Fatalf("expected marked indices [0 2], got %v", vis)
	}
}

func TestEnsureCursorVisibleSnapsAndResetsOverlay(t *testing.T) {
	m := newTestModel(3, 2)
	m.filterMarked = true
	m.marked[m.cards[1].Path] = true
	m.overlayPage = 5

	m = m.ensureCursorVisible()
	if m.cursor != 1 {
		t.Fatalf("expected cursor to snap to first visible marked card, got %d", m.cursor)
	}
	if m.overlayPage != 0 {
		t.Fatalf("expected overlayPage to reset when snapping cursor, got %d", m.overlayPage)
	}
}
