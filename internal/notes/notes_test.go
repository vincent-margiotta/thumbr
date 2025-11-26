// internal/notes/notes_test.go
package notes

import (
	"os"
	"path/filepath"
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

func TestLoadCardsFromDir_FiltersByExtAndIgnore(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"keep1.md":              "# one",
		"keep2.txt":             "# two",
		"skip.tmp":              "tmp",
		"ignored/keep3.md":      "# nested",
		"ignored/skip2.txt":     "# nested skip",
		"nested/keep4.md":       "# nested keep",
		"nested/skipme.tmp":     "tmp",
		"nested/.hidden.md":     "# hidden",
		"nested/file.TXT":       "# case",
		"nested/ignore.me":      "meh",
		"Archive/keep5.md":      "# archived",
		"Archive/skip.foo":      "foo",
		"Archive/deep/keep.md":  "# deep keep",
		"Archive/deep/keep2.md": "# deep keep2",
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	opts := LoadOptions{
		IncludeExts: []string{".md", ".txt"},
		IgnoreGlobs: []string{"Archive/*", "ignored", "*.tmp"},
	}
	cards, err := LoadCardsFromDir(dir, opts)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	got := map[string]bool{}
	for _, c := range cards {
		got[filepath.Base(c.Path)] = true
	}

	want := []string{"keep1.md", "keep2.txt", "keep4.md", ".hidden.md", "file.TXT"}
	for _, name := range want {
		if !got[name] {
			t.Fatalf("expected to include %s", name)
		}
	}

	notWanted := []string{"skip.tmp", "skipme.tmp", "keep3.md", "skip2.txt", "keep5.md", "keep.md"}
	for _, name := range notWanted {
		if got[name] {
			t.Fatalf("did not expect to include %s", name)
		}
	}
}

func TestApplySort_NaturalDefault(t *testing.T) {
	cases := []struct {
		name   string
		input  []string
		expect []string
	}{
		{
			name: "case1 lexical sorting",
			input: []string{
				"1", "2", "3", "9", "10", "11", "20",
			},
			expect: []string{"1", "2", "3", "9", "10", "11", "20"},
		},
		{
			name:   "case2 double-digit siblings",
			input:  []string{"1", "1a", "1a1", "1a2", "1a3", "1a9", "1a10", "1a11", "1b"},
			expect: []string{"1", "1a", "1a1", "1a2", "1a3", "1a9", "1a10", "1a11", "1b"},
		},
		{
			name:   "case3 deep nesting",
			input:  []string{"1", "1a", "1a1", "1a1a", "1a1b", "1a1b1", "1a1b2", "1a1b9", "1a1b10", "1a1b11", "1a2", "1b", "2"},
			expect: []string{"1", "1a", "1a1", "1a1a", "1a1b", "1a1b1", "1a1b2", "1a1b9", "1a1b10", "1a1b11", "1a2", "1b", "2"},
		},
		{
			name:   "case4 mixed depth",
			input:  []string{"1", "1a", "1a1", "1a1a", "1a1a1", "1a1a2", "1a1a9", "1a1a10", "1a1b", "1a2", "1a9", "1a10", "1a10a", "1a10b", "1a11", "1b", "1b1", "1b2", "1b9", "1b10", "2", "10", "10a"},
			expect: []string{"1", "1a", "1a1", "1a1a", "1a1a1", "1a1a2", "1a1a9", "1a1a10", "1a1b", "1a2", "1a9", "1a10", "1a10a", "1a10b", "1a11", "1b", "1b1", "1b2", "1b9", "1b10", "2", "10", "10a"},
		},
		{
			name:   "case5 smoke",
			input:  []string{"1", "1a1", "1a2", "1a9", "1a10", "1a11", "2", "10"},
			expect: []string{"1", "1a1", "1a2", "1a9", "1a10", "1a11", "2", "10"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cards := cardsFromNames(tc.input)
			if err := applySort(cards, LoadOptions{}); err != nil {
				t.Fatalf("applySort error: %v", err)
			}
			got := namesFromCards(cards)
			assertSliceEquals(t, tc.expect, got)
		})
	}
}

func TestApplySort_LexicalFallback(t *testing.T) {
	input := []string{"1", "1a", "1a10", "1a2"}
	want := []string{"1", "1a", "1a10", "1a2"} // lexical puts 10 before 2
	cards := cardsFromNames(input)
	if err := applySort(cards, LoadOptions{SortMode: "lexical", SortPattern: ".*"}); err != nil {
		t.Fatalf("applySort error: %v", err)
	}
	got := namesFromCards(cards)
	assertSliceEquals(t, want, got)
}

func TestApplySort_PatternOnlyMatchesSome(t *testing.T) {
	input := []string{"abc", "1", "10", "2"}
	want := []string{"1", "2", "10", "abc"} // numeric matches come first sorted naturally; abc last via lexical
	cards := cardsFromNames(input)
	if err := applySort(cards, LoadOptions{SortMode: "natural", SortPattern: "^[0-9]+$"}); err != nil {
		t.Fatalf("applySort error: %v", err)
	}
	got := namesFromCards(cards)
	assertSliceEquals(t, want, got)
}

func TestApplySort_InvalidRegex(t *testing.T) {
	if err := applySort(cardsFromNames([]string{"1"}), LoadOptions{SortPattern: "("}); err == nil {
		t.Fatalf("expected regex error")
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
	if len(expect) != len(got) {
		t.Fatalf("length mismatch: want %d got %d\nwant: %v\ngot:  %v", len(expect), len(got), expect, got)
	}
	for i := range expect {
		if expect[i] != got[i] {
			t.Fatalf("mismatch at %d: want %q got %q\nwant: %v\ngot:  %v", i, expect[i], got[i], expect, got)
		}
	}
}
