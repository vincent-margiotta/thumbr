package ui

import (
	"math/rand"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"thumbr/internal/notes"
)

type State int

const (
	StateBrowsing State = iota
	StateViewing
	StatePrompting
)

func (s State) String() string {
	switch s {
	case StateBrowsing:
		return "Browsing"
	case StateViewing:
		return "Viewing"
	case StatePrompting:
		return "Prompting"
	default:
		return "Unknown"
	}
}

type Viewport struct {
	Width  int
	Height int
}

type Settings struct {
	StackVisibleCount int
	StackOffsetX      int
	StackOffsetY      int

	CardWidthFrac  float64
	CardHeightFrac float64

	ColorHiFG      lipgloss.Color
	ColorDimFG     lipgloss.Color
	ColorStatusBG  lipgloss.Color
	ColorStatusFG  lipgloss.Color
	ColorStatusDim lipgloss.Color

	NavAccelWindow time.Duration
	NavMaxStep     int

	MaxCursorDepth   int
	ActiveLiftY      int
	StickyOverlayNav bool
}

var DefaultSettings = Settings{
	StackVisibleCount: 7,
	StackOffsetX:      2,
	StackOffsetY:      1,
	CardWidthFrac:     0.6,
	CardHeightFrac:    0.6,
	ColorHiFG:         lipgloss.Color("#FFD166"), // warm highlight
	ColorDimFG:        lipgloss.Color("#666666"),
	ColorStatusBG:     lipgloss.Color("#222222"),
	ColorStatusFG:     lipgloss.Color("#F5F5F5"),
	ColorStatusDim:    lipgloss.Color("#999999"),
	NavAccelWindow:    350 * time.Millisecond,
	NavMaxStep:        8,
	MaxCursorDepth:    2,
	ActiveLiftY:       2,
	StickyOverlayNav:  false,
}

type cardGeom struct {
	index  int
	x, y   int
	w, h   int
	active bool
	depth  int
}

type styleID int

const (
	styleBase styleID = iota
	styleCardDim
	styleCardHi
	styleOverlayBorder
	styleOverlayHeader
	styleOverlayBody
)

type cell struct {
	ch      rune
	styleID styleID
}

// Model is the Bubble Tea model for Thumbr.
type Model struct {
	cards    []notes.Card
	cursor   int
	state    State
	viewport Viewport
	settings Settings

	lastNavDir  int
	lastNavTime time.Time
	velocity    int

	rng *rand.Rand
	// showHelp toggles the keybinding overlay.
	showHelp bool

	err       error
	noteRoot  string
	ready     bool
	lastWidth int
}

// NewModel constructs the initial UI model. The RNG controls random jumps; pass
// nil to use the global math/rand instance.
func NewModel(cards []notes.Card, noteRoot string, rng *rand.Rand) Model {
	return Model{
		cards:    cards,
		cursor:   0,
		state:    StateBrowsing,
		settings: DefaultSettings,
		rng:      rng,
		noteRoot: noteRoot,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	if !m.ready || m.viewport.Width == 0 || m.viewport.Height == 0 {
		return "Thumbr – initializing…\n"
	}

	if m.showHelp {
		return m.renderHelp()
	}

	// No cards? Simple empty view.
	if len(m.cards) == 0 {
		return m.renderEmpty()
	}

	// ---- Build style palette for this frame ----

	// Background base (slightly dimmed when viewing).
	baseFG := lipgloss.Color("#CCCCCC")
	if m.state == StateViewing {
		baseFG = m.settings.ColorDimFG
	}
	baseStyle := lipgloss.NewStyle().Foreground(baseFG)

	cardDimStyle := lipgloss.NewStyle().Foreground(m.settings.ColorDimFG)
	cardHiStyle := lipgloss.NewStyle().Foreground(m.settings.ColorHiFG).Bold(true)
	overlayBorderStyle := cardHiStyle
	overlayHeaderStyle := cardHiStyle
	overlayBodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#DDDDDD"))

	styles := []lipgloss.Style{
		styleBase:          baseStyle,
		styleCardDim:       cardDimStyle,
		styleCardHi:        cardHiStyle,
		styleOverlayBorder: overlayBorderStyle,
		styleOverlayHeader: overlayHeaderStyle,
		styleOverlayBody:   overlayBodyStyle,
	}

	// ---- Build grid with style IDs ----

	grid := make([][]cell, m.viewport.Height)
	for y := range grid {
		grid[y] = make([]cell, m.viewport.Width)
		for x := range grid[y] {
			grid[y][x] = cell{
				ch:      ' ',
				styleID: styleBase,
			}
		}
	}

	geoms := m.computeStackGeometry()

	// In Viewing mode, the stack behind the overlay should be fully dim;
	// only the overlay card should be highlighted. So we clear "active"
	// on all stack geoms.
	if m.state == StateViewing {
		for i := range geoms {
			geoms[i].active = false
		}
	}

	// Draw stack from back to front based on depth
	sort.Slice(geoms, func(i, j int) bool {
		return geoms[i].depth > geoms[j].depth
	})
	for _, g := range geoms {
		m.drawCardOntoGrid(grid, g)
	}

	// If we're in Viewing mode, pull the active card out on top
	if m.state == StateViewing {
		m.drawOverlayCardOntoGrid(grid)
	}

	body := m.renderGrid(grid, styles)
	status := m.renderStatusBar()

	return body + "\n" + status
}

// ==== Utilities ====

func stripANSI(s string) string {
	// crude but sufficient for computing padding length
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
