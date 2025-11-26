package notes

import (
	"os"
	"path/filepath"
	"strings"
)

// Card is the basic unit Thumbr browses over.
type Card struct {
	ID      string
	Title   string
	Path    string
	Content string
}

type LoadOptions struct {
	IncludeExts []string // e.g. []string{".txt", ".md"}; empty means default .txt
	IgnoreGlobs []string // file/dir patterns to skip
}

// LoadCardsFromDir walks the given root directory and returns all matching files as Cards,
// parsing the ID and title from the filename.
func LoadCardsFromDir(root string, opts ...LoadOptions) ([]Card, error) {
	var cards []Card
	var opt LoadOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if len(opt.IncludeExts) == 0 {
		opt.IncludeExts = []string{".txt"}
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if ignoreFilter(path, opt.IgnoreGlobs) {
				return filepath.SkipDir
			}
			return nil
		}
		if ignoreFilter(path, opt.IgnoreGlobs) {
			return nil
		}
		if !includeFilter(d.Name(), opt.IncludeExts) {
			return nil
		}

		id, title := parseCardFilename(d.Name())
		contentBytes, _ := os.ReadFile(path) // ignore error; content not crucial yet

		card := Card{
			ID:      id,
			Title:   title,
			Path:    path,
			Content: string(contentBytes),
		}
		cards = append(cards, card)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return cards, nil
}

// parseCardFilename splits "ID Title.md" into ("ID", "Title").
func parseCardFilename(name string) (string, string) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	parts := strings.SplitN(base, " ", 2)
	if len(parts) == 1 {
		return "", base
	}
	id := parts[0]
	title := strings.TrimSpace(parts[1])
	return id, title
}
