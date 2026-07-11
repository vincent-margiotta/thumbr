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

func TestWrapTextPreservesIndentation(t *testing.T) {
	lines := wrapText("Header:\n  Foo\n  Bar", 20, -1)
	want := []string{"Header:", "  Foo", "  Bar"}
	if len(lines) != len(want) {
		t.Fatalf("expected %d lines, got %d: %v", len(want), len(lines), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d mismatch: want %q got %q", i, want[i], lines[i])
		}
	}
}

func TestWrapTextPreservesInternalMultiSpace(t *testing.T) {
	lines := wrapText("this space:          will be eaten", 60, -1)
	want := []string{"this space:          will be eaten"}
	if len(lines) != len(want) {
		t.Fatalf("expected %d lines, got %d: %v", len(want), len(lines), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d mismatch: want %q got %q", i, want[i], lines[i])
		}
	}
}

func TestWrapTextIndentCarriesToWrappedContinuation(t *testing.T) {
	lines := wrapText("  one two three", 8, -1)
	want := []string{"  one", "  two", "  three"}
	if len(lines) != len(want) {
		t.Fatalf("expected %d lines, got %d: %v", len(want), len(lines), lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d mismatch: want %q got %q", i, want[i], lines[i])
		}
	}
}
