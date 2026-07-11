package ui

import "testing"

func TestExpandTabsSimpleIndent(t *testing.T) {
	got := expandTabs("\tfoo", 4)
	want := "    foo"
	if got != want {
		t.Fatalf("want %q got %q", want, got)
	}
}

func TestExpandTabsAlignsToNextStop(t *testing.T) {
	// "ab" occupies columns 0-1; a tab from column 2 should pad to column 4.
	got := expandTabs("ab\tc", 4)
	want := "ab  c"
	if got != want {
		t.Fatalf("want %q got %q", want, got)
	}
}

func TestExpandTabsResetsColumnPerLine(t *testing.T) {
	got := expandTabs("a\tb\n\tc", 4)
	want := "a   b\n    c"
	if got != want {
		t.Fatalf("want %q got %q", want, got)
	}
}

func TestExpandTabsNoTabsReturnsUnchanged(t *testing.T) {
	s := "no tabs here"
	if got := expandTabs(s, 4); got != s {
		t.Fatalf("want unchanged %q got %q", s, got)
	}
}
