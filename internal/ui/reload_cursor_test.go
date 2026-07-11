package ui

import (
	"testing"

	"github.com/vincent-margiotta/thumbr/internal/notes"
)

// A stray reload (e.g. the live-reload watcher firing concurrently with the
// explicit reload after an external-editor edit) must not yank the cursor
// back to the top of the deck when no seek is pending.
func TestResetAfterLoadPreservesCursorWithoutPendingSeek(t *testing.T) {
	m := newTestModel(5, 2) // cursor on card-3.txt
	m.noteRoot = cleanBoxPath(".")

	reloaded := m.resetAfterLoad(m.cards, ".")

	if got, want := reloaded.cards[reloaded.cursor].Path, "/tmp/card-3.txt"; got != want {
		t.Fatalf("cursor path: want %q got %q (cursor=%d)", want, got, reloaded.cursor)
	}
}

func TestResetAfterLoadHonorsPendingSeekPath(t *testing.T) {
	m := newTestModel(5, 0)
	m.noteRoot = cleanBoxPath(".")
	m.pendingSeekPath = "/tmp/card-4.txt"

	reloaded := m.resetAfterLoad(m.cards, ".")

	if got, want := reloaded.cards[reloaded.cursor].Path, "/tmp/card-4.txt"; got != want {
		t.Fatalf("cursor path: want %q got %q (cursor=%d)", want, got, reloaded.cursor)
	}
	if reloaded.pendingSeekPath != "" {
		t.Fatalf("pendingSeekPath should be cleared after use, got %q", reloaded.pendingSeekPath)
	}
}

func TestResetAfterLoadDefaultsToTopOnBoxSwitch(t *testing.T) {
	m := newTestModel(5, 2)
	m.noteRoot = cleanBoxPath("/some/other/box")

	cards := make([]notes.Card, 3)
	for i := range cards {
		cards[i] = notes.Card{Path: "/tmp/new-box/card.txt"}
	}

	reloaded := m.resetAfterLoad(cards, ".")

	if reloaded.cursor != 0 {
		t.Fatalf("expected cursor reset to 0 on box switch without pending seek, got %d", reloaded.cursor)
	}
}
