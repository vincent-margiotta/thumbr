package ui

import "testing"

func TestWrapTextHardBreaksLongWords(t *testing.T) {
	lines := wrapText("supercalifragilistic", 4, -1)
	want := []string{"supe", "rcal", "ifra", "gili", "stic"}
	if len(lines) != len(want) {
		t.Fatalf("expected %d lines, got %d: %v", len(want), len(lines), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d mismatch: want %q got %q", i, want[i], lines[i])
		}
	}
}

func TestWrapTextPreservesBlankLines(t *testing.T) {
	lines := wrapText("first\n\nsecond", 10, -1)
	want := []string{"first", "", "second"}
	if len(lines) != len(want) {
		t.Fatalf("expected %d lines, got %d: %v", len(want), len(lines), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d mismatch: want %q got %q", i, want[i], lines[i])
		}
	}
}

func TestWrapTextZeroMaxLinesReturnsNil(t *testing.T) {
	if got := wrapText("content", 5, 0); got != nil {
		t.Fatalf("expected nil when maxLines=0, got %v", got)
	}
}
