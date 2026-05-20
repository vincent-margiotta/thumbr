package ui

import "testing"

// ---- wordEndForward ----

func TestWordEndForward_BasicWord(t *testing.T) {
	// "hello world" — cursor at 0, should land on 'o' (index 4).
	runes := []rune("hello world")
	if got := wordEndForward(runes, 0); got != 4 {
		t.Errorf("wordEndForward at 0: want 4, got %d", got)
	}
}

func TestWordEndForward_AlreadyAtWordEnd(t *testing.T) {
	// "hello world" — cursor at 4 (end of "hello"), should advance to 10 (end of "world").
	runes := []rune("hello world")
	if got := wordEndForward(runes, 4); got != 10 {
		t.Errorf("wordEndForward at 4 (word end): want 10, got %d", got)
	}
}

func TestWordEndForward_MidWord(t *testing.T) {
	// "hello world" — cursor at 2 (mid-word), should land on 4.
	runes := []rune("hello world")
	if got := wordEndForward(runes, 2); got != 4 {
		t.Errorf("wordEndForward at 2 (mid-word): want 4, got %d", got)
	}
}

func TestWordEndForward_AtLastChar_ReturnsMinusOne(t *testing.T) {
	// Cursor at the last char of the line → -1 (caller crosses to next line).
	runes := []rune("hello")
	if got := wordEndForward(runes, 4); got != -1 {
		t.Errorf("wordEndForward at last char: want -1, got %d", got)
	}
}

func TestWordEndForward_EmptyLine(t *testing.T) {
	if got := wordEndForward([]rune{}, 0); got != -1 {
		t.Errorf("wordEndForward on empty line: want -1, got %d", got)
	}
}

func TestWordEndForward_Punctuation(t *testing.T) {
	// "foo, bar" — indices: f=0 o=1 o=2 ,=3 ' '=4 b=5 a=6 r=7
	runes := []rune("foo, bar")

	// From col 2 (end of "foo"): advances past word-end, skips ',', lands on col 3.
	if got := wordEndForward(runes, 2); got != 3 {
		t.Errorf("wordEndForward at 2 (end of foo): want 3 (comma), got %d", got)
	}
	// From col 3 (',', end of its punct token): advances past it, skips space,
	// lands on col 7 (end of "bar").
	if got := wordEndForward(runes, 3); got != 7 {
		t.Errorf("wordEndForward at 3 (comma, word-end): want 7 (end of bar), got %d", got)
	}
}

func TestWordEndForward_TrailingSpaces_CrossesLine(t *testing.T) {
	// Line ends with spaces: "foo   " — from col 2 (end of "foo"), cross to next line.
	runes := []rune("foo   ")
	got := wordEndForward(runes, 2)
	if got != -1 {
		t.Errorf("wordEndForward with only trailing spaces left: want -1, got %d", got)
	}
}
