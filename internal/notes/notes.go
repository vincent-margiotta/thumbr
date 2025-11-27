package notes

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Card is the basic unit Thumbr browses over.
type Card struct {
	ID            string
	Title         string
	Path          string
	Content       string
	ContentLoaded bool
	ContentErr    error
}

type LoadOptions struct {
	IncludeExts      []string // e.g. []string{".txt", ".md"}; empty means default .txt
	IgnoreGlobs      []string // file/dir patterns to skip
	SortMode         string   // "natural" (default) or "lexical"
	SortPattern      string   // regex to apply sort mode to; others fall back to lexical (default numeric-ish)
	SortPatternFirst *bool    // when true (default), names matching SortPattern come first; when false they come after
	Timings          *LoadTimings
}

// LoadTimings captures coarse timings for load phases.
type LoadTimings struct {
	Walk time.Duration
	Sort time.Duration
}

// LoadCardsFromDir walks the given root directory and returns all matching files as Cards,
// using the filename (sans extension) as the title. Content is not read; callers
// should load it lazily when needed (e.g., when opening a card).
func LoadCardsFromDir(root string, opts ...LoadOptions) ([]Card, error) {
	var cards []Card
	var opt LoadOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if len(opt.IncludeExts) == 0 {
		opt.IncludeExts = []string{".txt"}
	}

	walkStart := time.Now()
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
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = filepath.Clean(path)
		}

		card := Card{
			ID:    "",
			Title: title,
			Path:  absPath,
		}
		cards = append(cards, card)
		return nil
	})

	if err != nil {
		return nil, err
	}

	walkDur := time.Since(walkStart)

	sortStart := time.Now()
	if err := applySort(cards, opt); err != nil {
		return nil, err
	}
	sortDur := time.Since(sortStart)

	if opt.Timings != nil {
		opt.Timings.Walk = walkDur
		opt.Timings.Sort = sortDur
	}

	return cards, nil
}

// LoadContent reads the card's file into Content and marks it as loaded. On
// error, Content is left empty and ContentErr is set.
func (c *Card) LoadContent() error {
	data, err := os.ReadFile(c.Path)
	if err != nil {
		c.Content = ""
		c.ContentLoaded = true
		c.ContentErr = err
		return err
	}
	c.Content = string(data)
	c.ContentLoaded = true
	c.ContentErr = nil
	return nil
}

const defaultSortPattern = `^[0-9]+[A-Za-z0-9]*$`

func applySort(cards []Card, opt LoadOptions) error {
	mode := strings.ToLower(opt.SortMode)
	if mode == "" {
		mode = "natural"
	}
	patternFirst := false
	if opt.SortPatternFirst != nil {
		patternFirst = *opt.SortPatternFirst
	}
	pattern := opt.SortPattern
	if pattern == "" {
		pattern = defaultSortPattern
	}

	var re *regexp.Regexp
	if pattern != "" {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			return err
		}
	}

	less := func(a, b Card) bool {
		an := a.Title
		bn := b.Title

		matchAll := re == nil
		matchA := matchAll || re.MatchString(an)
		matchB := matchAll || re.MatchString(bn)

		useMode := func() int {
			if mode == "lexical" {
				return strings.Compare(an, bn)
			}
			return naturalCompare(an, bn)
		}

		switch {
		case matchA && matchB:
			return useMode() < 0
		case matchA && !matchB:
			// Matched names before/after based on config.
			return patternFirst
		case !matchA && matchB:
			return !patternFirst
		default:
			return strings.Compare(an, bn) < 0
		}
	}

	sort.SliceStable(cards, func(i, j int) bool { return less(cards[i], cards[j]) })
	return nil
}

type segment struct {
	num  bool
	text string
}

func splitSegments(s string) []segment {
	var out []segment
	var cur strings.Builder
	var curNum *bool
	flush := func() {
		if cur.Len() == 0 || curNum == nil {
			return
		}
		out = append(out, segment{num: *curNum, text: cur.String()})
		cur.Reset()
	}
	for _, r := range s {
		isNum := r >= '0' && r <= '9'
		if curNum == nil {
			curNum = new(bool)
			*curNum = isNum
		}
		if *curNum != isNum {
			flush()
			curNum = new(bool)
			*curNum = isNum
		}
		cur.WriteRune(r)
	}
	flush()
	return out
}

func cmpNumeric(a, b string) int {
	aTrim := strings.TrimLeft(a, "0")
	bTrim := strings.TrimLeft(b, "0")
	if aTrim == "" {
		aTrim = "0"
	}
	if bTrim == "" {
		bTrim = "0"
	}
	if len(aTrim) != len(bTrim) {
		if len(aTrim) < len(bTrim) {
			return -1
		}
		return 1
	}
	return strings.Compare(aTrim, bTrim)
}

func naturalCompare(a, b string) int {
	as := splitSegments(a)
	bs := splitSegments(b)
	for i := 0; i < len(as) && i < len(bs); i++ {
		sa := as[i]
		sb := bs[i]
		if sa.num && sb.num {
			if cmp := cmpNumeric(sa.text, sb.text); cmp != 0 {
				return cmp
			}
		} else {
			if cmp := strings.Compare(sa.text, sb.text); cmp != 0 {
				return cmp
			}
		}
	}
	if len(as) == len(bs) {
		return 0
	}
	if len(as) < len(bs) {
		return -1
	}
	return 1
}
