// Package notes handles loading, sorting, and addressing Zettelkasten note cards.
package notes

import (
	"os"
	"path/filepath"
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

// LoadOptions controls optional behaviour during a load. All fields are
// optional; the zero value is valid.
type LoadOptions struct {
	Timings *LoadTimings
}

// LoadTimings captures coarse timings for load phases.
type LoadTimings struct {
	Walk time.Duration
	Sort time.Duration
}

// LoadCardsFromDir walks root and returns all .txt files as Cards, sorted in
// Luhmann natural order. Subdirectories are traversed but all cards land in a
// single flat list. Content is not read; call Card.LoadContent lazily.
func LoadCardsFromDir(root string, opts ...LoadOptions) ([]Card, error) {
	var opt LoadOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	var cards []Card

	walkStart := time.Now()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".txt") {
			return nil
		}

		stem := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = filepath.Clean(path)
		}

		cards = append(cards, Card{
			Title: stem,
			Path:  absPath,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	walkDur := time.Since(walkStart)

	sortStart := time.Now()
	sort.SliceStable(cards, func(i, j int) bool {
		return naturalCompare(cards[i].Title, cards[j].Title) < 0
	})
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
			// Case-insensitive first so "and" and "Greek" sort by letter
			// (a before g) rather than by ASCII case (all uppercase before
			// all lowercase). Fall back to a raw comparison only to keep
			// differently-cased variants of the same word in a stable order.
			if cmp := strings.Compare(strings.ToLower(sa.text), strings.ToLower(sb.text)); cmp != 0 {
				return cmp
			}
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
