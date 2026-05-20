package ui

import "testing"

// default curve params matching DefaultSettings
const (
	testTau   = 300.0
	testGamma = 1.75
	testMinDt = 50.0
)

func ns(dt float64, sameDir bool, maxStep int) int {
	return navStep(dt, testTau, testGamma, testMinDt, sameDir, maxStep)
}

// ---- navStep ----

func TestNavStep_FirstPressAlwaysOne(t *testing.T) {
	// Direction change (sameDir=false) always yields step=1.
	if step := ns(50, false, 20); step != 1 {
		t.Errorf("first press (sameDir=false): want step=1, got %d", step)
	}
}

func TestNavStep_DirectionChangeResetsToOne(t *testing.T) {
	if step := ns(30, false, 20); step != 1 {
		t.Errorf("after direction change: want step=1, got %d", step)
	}
}

func TestNavStep_FastPressingAccelerates(t *testing.T) {
	// Power law: fast pressing (80 ms) must produce a dramatically larger step
	// than moderate pressing (150 ms).
	slow := ns(150, true, 50)
	fast := ns(80, true, 50)
	if fast <= slow {
		t.Errorf("fast (80ms, step=%d) should exceed moderate (150ms, step=%d)", fast, slow)
	}
	if fast < 5 {
		t.Errorf("fast pressing (80 ms) should produce step >= 5, got %d", fast)
	}
}

func TestNavStep_SlowPressingYieldsOne(t *testing.T) {
	if step := ns(400, true, 20); step > 1 {
		t.Errorf("slow pressing (400 ms) should produce step=1, got %d", step)
	}
}

func TestNavStep_NeverExceedsMaxStep(t *testing.T) {
	maxStep := 5
	if step := ns(1, true, maxStep); step > maxStep {
		t.Fatalf("step %d exceeds maxStep %d", step, maxStep)
	}
}

func TestNavStep_AlwaysAtLeastOne(t *testing.T) {
	for _, dt := range []float64{0, 1000, 5000} {
		if s := ns(dt, false, 20); s < 1 {
			t.Errorf("step < 1 (dt=%.0f, sameDir=false)", dt)
		}
	}
	for _, dt := range []float64{1000, 5000} {
		if s := ns(dt, true, 20); s < 1 {
			t.Errorf("step < 1 (dt=%.0f, sameDir=true)", dt)
		}
	}
}

func TestNavStep_CustomCurve(t *testing.T) {
	// A user-supplied gamma=1.0 (linear) with tau=200 should behave like the old
	// linear model: step = int(200/dt), capped at maxStep.
	step := navStep(100, 200.0, 1.0, 50.0, true, 20)
	if step != 2 {
		t.Errorf("linear curve (gamma=1, tau=200, dt=100): want step=2, got %d", step)
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
	vis := m.visibleIndices()
	mNeg := m.jumpToPercent(-10)
	if mNeg.cursor < vis[0] || mNeg.cursor > vis[len(vis)-1] {
		t.Errorf("jumpToPercent(-10): cursor %d out of visible range", mNeg.cursor)
	}
}
