package ui

import "testing"

// ---- navStep ----

func TestNavStep_FirstPressAlwaysOne(t *testing.T) {
	// Direction change (sameDir=false) always yields step=1.
	step := navStep(50, false, 20)
	if step != 1 {
		t.Errorf("first press (sameDir=false): want step=1, got %d", step)
	}
}

func TestNavStep_DirectionChangeResetsToOne(t *testing.T) {
	// Fast pressing then a direction change must reset to step=1.
	step := navStep(30, false, 20)
	if step != 1 {
		t.Errorf("after direction change: want step=1, got %d", step)
	}
}

func TestNavStep_FastPressingAccelerates(t *testing.T) {
	// Pressing at 30 ms (very fast) should yield step > 1.
	step := navStep(30, true, 20)
	if step <= 1 {
		t.Errorf("fast pressing (30 ms) should produce step > 1, got %d", step)
	}
}

func TestNavStep_SlowPressingYieldsOne(t *testing.T) {
	// Pressing at 300 ms (deliberate) should yield step=1.
	step := navStep(300, true, 20)
	if step > 1 {
		t.Errorf("slow pressing (300 ms) should produce step=1, got %d", step)
	}
}

func TestNavStep_NeverExceedsMaxStep(t *testing.T) {
	maxStep := 5
	// Even the fastest conceivable interval should be capped.
	step := navStep(1, true, maxStep)
	if step > maxStep {
		t.Fatalf("step %d exceeds maxStep %d", step, maxStep)
	}
}

func TestNavStep_AlwaysAtLeastOne(t *testing.T) {
	// Any input must produce step >= 1.
	for _, dt := range []float64{0, 1000, 5000} {
		s := navStep(dt, false, 20)
		if s < 1 {
			t.Errorf("step < 1 (dt=%.0f, sameDir=false)", dt)
		}
	}
	for _, dt := range []float64{1000, 5000} {
		s := navStep(dt, true, 20)
		if s < 1 {
			t.Errorf("step < 1 (dt=%.0f, sameDir=true)", dt)
		}
	}
}

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

func TestJumpToEdge_ResetsMomentum(t *testing.T) {
	m := newTestModel(10, 5)
	m.lastNavDir = 1
	m = m.jumpToEdge(-1)
	if m.lastNavDir != 0 {
		t.Errorf("jumpToEdge should reset lastNavDir, got %d", m.lastNavDir)
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

	// 100% should land at or near the last card (integer division may not reach exactly).
	m100 := m.jumpToPercent(100)
	if m100.cursor != vis[len(vis)-1] {
		t.Errorf("jumpToPercent(100): want %d, got %d", vis[len(vis)-1], m100.cursor)
	}
}

func TestJumpToPercent_Middle(t *testing.T) {
	m := newTestModel(10, 0)
	vis := m.visibleIndices()
	m50 := m.jumpToPercent(50)
	want := vis[5] // 50 * 10 / 100 = 5
	if m50.cursor != want {
		t.Errorf("jumpToPercent(50) on 10 cards: want card %d, got %d", want, m50.cursor)
	}
}

func TestJumpToPercent_ResetsMomentum(t *testing.T) {
	m := newTestModel(10, 5)
	m.lastNavDir = -1
	m = m.jumpToPercent(30)
	if m.lastNavDir != 0 {
		t.Errorf("jumpToPercent should reset lastNavDir, got %d", m.lastNavDir)
	}
}

func TestJumpToPercent_OutOfRange(t *testing.T) {
	m := newTestModel(10, 5)
	// Values outside 0-100 should still produce a valid cursor.
	vis := m.visibleIndices()
	mNeg := m.jumpToPercent(-10)
	if mNeg.cursor < vis[0] || mNeg.cursor > vis[len(vis)-1] {
		t.Errorf("jumpToPercent(-10): cursor %d out of visible range", mNeg.cursor)
	}
}
