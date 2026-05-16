package ui

import "testing"

// ---- navStep ----

func TestNavStep_FirstPressAlwaysOne(t *testing.T) {
	// avgInterval=0 signals "no history"; first press must yield step=1.
	_, step := navStep(0, 999, true, 20)
	if step != 1 {
		t.Errorf("first press (avgInterval=0): want step=1, got %d", step)
	}
}

func TestNavStep_DirectionChangeResetsToOne(t *testing.T) {
	// Build up some momentum then flip direction.
	avg := 0.0
	for i := 0; i < 10; i++ {
		avg, _ = navStep(avg, 50, true, 20)
	}
	_, step := navStep(avg, 50, false, 20)
	if step != 1 {
		t.Errorf("after direction change: want step=1, got %d", step)
	}
}

func TestNavStep_FastPressingAccelerates(t *testing.T) {
	// Pressing at 50 ms intervals (very fast) should grow the step over time.
	avg := 0.0
	maxStep := 20
	var last int
	for i := 0; i < 15; i++ {
		var s int
		avg, s = navStep(avg, 50, i > 0, maxStep)
		last = s
	}
	if last <= 1 {
		t.Errorf("sustained fast pressing should produce step > 1, got %d", last)
	}
}

func TestNavStep_SlowPressingDecelerates(t *testing.T) {
	// Build up momentum, then slow down; step should shrink back toward 1.
	avg := 0.0
	maxStep := 20
	for i := 0; i < 12; i++ {
		avg, _ = navStep(avg, 50, i > 0, maxStep)
	}
	var slowStep int
	for i := 0; i < 8; i++ {
		avg, slowStep = navStep(avg, 400, true, maxStep)
	}
	if slowStep > 2 {
		t.Errorf("after slowing down, want step <= 2, got %d", slowStep)
	}
}

func TestNavStep_NeverExceedsMaxStep(t *testing.T) {
	maxStep := 5
	avg := 0.0
	for i := 0; i < 30; i++ {
		var s int
		avg, s = navStep(avg, 30, i > 0, maxStep)
		if s > maxStep {
			t.Fatalf("step %d exceeds maxStep %d at iteration %d", s, maxStep, i)
		}
	}
}

func TestNavStep_AlwaysAtLeastOne(t *testing.T) {
	// Even with direction change and a long pause the step must be >= 1.
	avg := 0.0
	for _, dt := range []float64{0, 1000, 5000} {
		_, s := navStep(avg, dt, false, 20)
		if s < 1 {
			t.Errorf("step < 1 (dt=%.0f)", dt)
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
	m.avgInterval = 99.9
	m.lastNavDir = 1
	m = m.jumpToEdge(-1)
	if m.avgInterval != 0 {
		t.Errorf("jumpToEdge should reset avgInterval, got %f", m.avgInterval)
	}
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
	m.avgInterval = 42.0
	m.lastNavDir = -1
	m = m.jumpToPercent(30)
	if m.avgInterval != 0 {
		t.Errorf("jumpToPercent should reset avgInterval, got %f", m.avgInterval)
	}
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
