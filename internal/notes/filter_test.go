package notes

import (
	"path/filepath"
	"testing"
)

func TestIncludeFilterDefaultsAndCaseInsensitive(t *testing.T) {
	if !includeFilter("note.any", nil) {
		t.Fatalf("expected includeFilter to allow all when no extensions provided")
	}

	exts := []string{".md"}
	if !includeFilter("Card.MD", exts) {
		t.Fatalf("expected case-insensitive match for .md, got false")
	}
	if includeFilter("image.png", exts) {
		t.Fatalf("did not expect .png to match allowed exts %v", exts)
	}
}

func TestIgnoreFilterMatchesBasenameAndDirShorthand(t *testing.T) {
	sep := string(filepath.Separator)
	patterns := []string{"", "Archive" + sep + "*", "*.tmp", "skipme"}

	tests := []struct {
		path string
		want bool
	}{
		{path: filepath.Join("notes", "Archive"), want: true},             // dir/* shorthand (directory entry)
		{path: filepath.Join("notes", "Archive", "deeper"), want: false},  // nested dir requires parent to be skipped earlier
		{path: filepath.Join("notes", "draft.tmp"), want: true},           // basename glob
		{path: filepath.Join("notes", "keep.md"), want: false},            // normal file
		{path: filepath.Join("notes", "skipme"), want: true},              // basename exact match
		{path: filepath.Join("notes", "Archive", "card.md"), want: false}, // file inside ignored dir (dir would be skipped earlier)
		{path: filepath.Join("notes", "sub", "keep.txt"), want: false},    // no pattern match
	}

	for _, tt := range tests {
		if got := ignoreFilter(tt.path, patterns); got != tt.want {
			t.Fatalf("ignoreFilter(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
