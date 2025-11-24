// internal/notes/notes_test.go
package notes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCardFilename_WithIDAndTitle(t *testing.T) {
	id, title := parseCardFilename("1.1a Some title here.md")
	if id != "1.1a" {
		t.Fatalf("expected id '1.1a', got %q", id)
	}
	if title != "Some title here" {
		t.Fatalf("expected title 'Some title here', got %q", title)
	}
}

func TestParseCardFilename_TitleOnly(t *testing.T) {
	id, title := parseCardFilename("JustATitle.md")
	if id != "" {
		t.Fatalf("expected empty id, got %q", id)
	}
	if title != "JustATitle" {
		t.Fatalf("expected title 'JustATitle', got %q", title)
	}
}

func TestParseCardFilename_TrimsExtraSpace(t *testing.T) {
	id, title := parseCardFilename("2.0a   Spaced   Title.md")
	if id != "2.0a" {
		t.Fatalf("expected id '2.0a', got %q", id)
	}
	if title != "Spaced   Title" {
		t.Fatalf("expected trimmed title 'Spaced   Title', got %q", title)
	}
}

func TestLoadCardsFromDir_DefaultsToMarkdown(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"keep.md":    "# md",
		"skip.txt":   "# text",
		".hidden.md": "# hidden ok",
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

	for _, name := range []string{"keep.md", ".hidden.md"} {
		if !got[name] {
			t.Fatalf("expected to include %s", name)
		}
	}
	if got["skip.txt"] {
		t.Fatalf("did not expect to include skip.txt")
	}
}

func TestLoadCardsFromDir_FiltersByExtAndIgnore(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"keep1.md":             "# one",
		"keep2.txt":            "# two",
		"skip.tmp":             "tmp",
		"ignored/keep3.md":     "# nested",
		"ignored/skip2.txt":    "# nested skip",
		"nested/keep4.md":      "# nested keep",
		"nested/skipme.tmp":    "tmp",
		"nested/.hidden.md":    "# hidden",
		"nested/file.TXT":      "# case",
		"nested/ignore.me":     "meh",
		"Archive/keep5.md":     "# archived",
		"Archive/skip.foo":     "foo",
		"Archive/deep/keep.md": "# deep keep",
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
