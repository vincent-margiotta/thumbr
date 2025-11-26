package ui

import "testing"

func TestWrapTextUnlimitedAllowsNegativeMaxLines(t *testing.T) {
	lines := wrapText("a b c d e f g", 3, -1)
	if len(lines) == 0 {
		t.Fatalf("expected lines when maxLines = -1, got none")
	}
	if lines[0] != "a b" {
		t.Fatalf("expected first line to wrap as 'a b', got %q", lines[0])
	}
}
