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
// using the filename (sans extension) as the title.
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

		title := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		contentBytes, _ := os.ReadFile(path) // ignore error; content not crucial yet

		card := Card{
			ID:      "",
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
