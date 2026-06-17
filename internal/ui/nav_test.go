package ui

import "testing"

// ---- jumpToEdge ----

func TestJumpToEdge_First(t *testing.T) {
	m := newTestModel(10, 9)
	m = m.jumpToEdge(-1)
	vis := m.visibleIndices()
	if m.cursor != vis[0] {
		t.Errorf("jumpToEdge(-1): want cursor=%d (first), got %d", vis[0], m.cursor)
	}
}

func TestJumpToEdge_Last(t *testing.T) {
	m := newTestModel(10, 0)
	m = m.jumpToEdge(1)
	vis := m.visibleIndices()
	if m.cursor != vis[len(vis)-1] {
		t.Errorf("jumpToEdge(1): want cursor=%d (last), got %d", vis[len(vis)-1], m.cursor)
	}
}

// ---- jumpToPercent ----

func TestJumpToPercent_Edges(t *testing.T) {
	m := newTestModel(10, 5)
	vis := m.visibleIndices()

	m0 := m.jumpToPercent(0)
	if m0.cursor != vis[0] {
		t.Errorf("jumpToPercent(0): want %d, got %d", vis[0], m0.cursor)
	}

	m100 := m.jumpToPercent(100)
	if m100.cursor != vis[len(vis)-1] {
		t.Errorf("jumpToPercent(100): want %d, got %d", vis[len(vis)-1], m100.cursor)
	}
}

func TestJumpToPercent_Middle(t *testing.T) {
	m := newTestModel(10, 0)
	vis := m.visibleIndices()
	m50 := m.jumpToPercent(50)
	want := vis[5]
	if m50.cursor != want {
		t.Errorf("jumpToPercent(50) on 10 cards: want card %d, got %d", want, m50.cursor)
	}
}

func TestJumpToPercent_OutOfRange(t *testing.T) {
	m := newTestModel(10, 5)
	vis := m.visibleIndices()
	mNeg := m.jumpToPercent(-10)
	if mNeg.cursor < vis[0] || mNeg.cursor > vis[len(vis)-1] {
		t.Errorf("jumpToPercent(-10): cursor %d out of visible range", mNeg.cursor)
	}
}
