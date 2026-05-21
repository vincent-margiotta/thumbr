// internal/notes/notes_test.go
package notes

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestLoadCardsFromDir_DefaultsToTxtExtension(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"keep.txt":    "# txt",
		"skip.md":     "# md",
		".hidden.txt": "# hidden ok",
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	cards, err := LoadCardsFromDir(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	got := map[string]bool{}
	for _, c := range cards {
		got[filepath.Base(c.Path)] = true
	}

	for _, name := range []string{"keep.txt", ".hidden.txt"} {
		if !got[name] {
			t.Fatalf("expected to include %s", name)
		}
	}
	if got["skip.md"] {
		t.Fatalf("did not expect to include skip.md")
	}
}

func TestLoadCardsFromDir_OnlyTxtLoaded(t *testing.T) {
	dir := t.TempDir()
	files := []string{"1.txt", "1a.txt", "skip.md", "skip.tmp", "2.txt"}
	for _, name := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	cards, err := LoadCardsFromDir(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	got := map[string]bool{}
	for _, c := range cards {
		got[filepath.Base(c.Path)] = true
	}

	for _, name := range []string{"1.txt", "1a.txt", "2.txt"} {
		if !got[name] {
			t.Fatalf("expected to include %s", name)
		}
	}
	for _, name := range []string{"skip.md", "skip.tmp"} {
		if got[name] {
			t.Fatalf("did not expect to include %s", name)
		}
	}
}

func TestNaturalSort(t *testing.T) {
	cases := []struct {
		name   string
		input  []string
		expect []string
	}{
		{
			name:   "numeric ordering",
			input:  []string{"1", "2", "3", "9", "10", "11", "20"},
			expect: []string{"1", "2", "3", "9", "10", "11", "20"},
		},
		{
			name:   "double-digit siblings",
			input:  []string{"1", "1a", "1a1", "1a2", "1a3", "1a9", "1a10", "1a11", "1b"},
			expect: []string{"1", "1a", "1a1", "1a2", "1a3", "1a9", "1a10", "1a11", "1b"},
		},
		{
			name:   "deep nesting",
			input:  []string{"1", "1a", "1a1", "1a1a", "1a1b", "1a1b1", "1a1b2", "1a1b9", "1a1b10", "1a1b11", "1a2", "1b", "2"},
			expect: []string{"1", "1a", "1a1", "1a1a", "1a1b", "1a1b1", "1a1b2", "1a1b9", "1a1b10", "1a1b11", "1a2", "1b", "2"},
		},
		{
			name:   "mixed depth",
			input:  []string{"1", "1a", "1a1", "1a1a", "1a1a1", "1a1a2", "1a1a9", "1a1a10", "1a1b", "1a2", "1a9", "1a10", "1a10a", "1a10b", "1a11", "1b", "1b1", "1b2", "1b9", "1b10", "2", "10", "10a"},
			expect: []string{"1", "1a", "1a1", "1a1a", "1a1a1", "1a1a2", "1a1a9", "1a1a10", "1a1b", "1a2", "1a9", "1a10", "1a10a", "1a10b", "1a11", "1b", "1b1", "1b2", "1b9", "1b10", "2", "10", "10a"},
		},
		{
			name:   "smoke",
			input:  []string{"1", "1a1", "1a2", "1a9", "1a10", "1a11", "2", "10"},
			expect: []string{"1", "1a1", "1a2", "1a9", "1a10", "1a11", "2", "10"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cards := cardsFromNames(tc.input)
			sort.SliceStable(cards, func(i, j int) bool {
				return naturalCompare(cards[i].Title, cards[j].Title) < 0
			})
			got := namesFromCards(cards)
			assertSliceEquals(t, tc.expect, got)
		})
	}
}

func cardsFromNames(names []string) []Card {
	out := make([]Card, len(names))
	for i, n := range names {
		out[i] = Card{Title: n}
	}
	return out
}

func namesFromCards(cards []Card) []string {
	out := make([]string, len(cards))
	for i, c := range cards {
		out[i] = c.Title
	}
	return out
}

func assertSliceEquals(t *testing.T, expect, got []string) {
	t.Helper()
	if len(expect) != len(got) {
		t.Fatalf("length mismatch: want %d got %d\nwant: %v\ngot:  %v", len(expect), len(got), expect, got)
	}
	for i := range expect {
		if expect[i] != got[i] {
			t.Fatalf("mismatch at %d: want %q got %q\nwant: %v\ngot:  %v", i, expect[i], got[i], expect, got)
		}
	}
}
