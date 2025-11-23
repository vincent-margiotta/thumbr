package ui

import "testing"

func TestFilterMarkedShowsOnlyMarkedAndSnapsCursor(t *testing.T) {
	m := newTestModel(4, 0)
	m.cursor = 2
	m = m.toggleMark() // mark card 2

	// Toggle filter on.
	m = m.toggleFilter()

	vis := m.visibleIndices()
	if len(vis) != 1 || vis[0] != 2 {
		t.Fatalf("expected only marked card visible, got %v", vis)
	}

	if m.cursor != 2 {
		t.Fatalf("expected cursor to stay on marked card, got %d", m.cursor)
	}

	geoms := m.computeStackGeometry()
	if len(geoms) != 1 || geoms[0].index != 2 {
		t.Fatalf("expected geometry for marked card only, got %+v", geoms)
	}

	// Unmark while filtered should drop filter and keep cursor valid.
	m = m.toggleMark()
	if m.filterMarked {
		t.Fatalf("expected filter to turn off after unmarking last card")
	}
	vis = m.visibleIndices()
	if len(vis) != 4 {
		t.Fatalf("expected all cards visible after dropping filter, got %v", vis)
	}
}
