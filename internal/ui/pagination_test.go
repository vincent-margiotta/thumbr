package ui

import "testing"

func TestPaginationHalfPageSteps(t *testing.T) {
	// Viewport height influences step size.
	m := newTestModel(1, 0)
	m.viewport = Viewport{Width: 80, Height: 20}
	m.state = StateViewing

	// Seed some content longer than one page.
	m.cards[0].Content = ""
	for i := 0; i < 50; i++ {
		m.cards[0].Content += "Line\n"
	}

	step := m.pageStep()
	if step < 1 {
		t.Fatalf("expected positive step, got %d", step)
	}

	if m.overlayPage != 0 {
		t.Fatalf("expected initial overlayPage 0, got %d", m.overlayPage)
	}

	m = m.nextPage()
	if m.overlayPage != step {
		t.Fatalf("expected overlayPage step of %d, got %d", step, m.overlayPage)
	}

	m = m.nextPage()
	if m.overlayPage != step*2 {
		t.Fatalf("expected overlayPage step of %d twice, got %d", step, m.overlayPage)
	}

	// Prev should step back.
	m = m.prevPage()
	if m.overlayPage != step {
		t.Fatalf("expected overlayPage to decrement by %d, got %d", step, m.overlayPage)
	}

	m = m.prevPage()
	if m.overlayPage != 0 {
		t.Fatalf("expected overlayPage to clamp at 0, got %d", m.overlayPage)
	}
}
