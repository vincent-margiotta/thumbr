package ui

import "testing"

func TestPageStepUsesOverride(t *testing.T) {
	m := newTestModel(1, 0)
	m.viewport = Viewport{Width: 80, Height: 20}

	defaultStep := m.pageStep()
	if defaultStep < 1 {
		t.Fatalf("expected positive default page step, got %d", defaultStep)
	}

	m.SetPageStep(5)
	if step := m.pageStep(); step != 5 {
		t.Fatalf("expected override page step 5, got %d", step)
	}
}
