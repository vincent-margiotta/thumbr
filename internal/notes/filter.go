// filter.go — include/ignore glob filtering for card loading.

package notes

import (
	"path/filepath"
	"strings"
)

// includeFilter returns true if the file should be considered based on allowed extensions.
// Extensions are compared case-insensitively and should be provided with a leading dot (e.g., ".md").
func includeFilter(name string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	lower := strings.ToLower(filepath.Ext(name))
	for _, ext := range exts {
		if strings.ToLower(ext) == lower {
			return true
		}
	}
	return false
}

// ignoreFilter returns true if the path should be skipped based on glob patterns.
// Patterns are matched using filepath.Match against the full path, and also against the base name
// to make simple patterns like "*.md" or "ignored" convenient.
func ignoreFilter(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	base := filepath.Base(path)
	clean := filepath.Clean(path)
	for _, pat := range patterns {
		if pat == "" {
			continue
		}
		if ok, _ := filepath.Match(pat, path); ok {
			return true
		}
		if ok, _ := filepath.Match(pat, base); ok {
			return true
		}

		// If the pattern ends with "/*", treat it as a directory ignore shorthand.
		if strings.HasSuffix(pat, string(filepath.Separator)+"*") {
			dirPat := strings.TrimSuffix(pat, string(filepath.Separator)+"*")
			if dirPat != "" {
				if base == dirPat {
					return true
				}
				if strings.HasSuffix(clean, string(filepath.Separator)+dirPat) {
					return true
				}
			}
		}
	}
	return false
}
