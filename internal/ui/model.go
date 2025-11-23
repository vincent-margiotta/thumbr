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
	ColorMutFG     lipgloss.Color
	ColorMarkFG    lipgloss.Color
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
	ColorMutFG:        lipgloss.Color("#444444"), // muted for unmarked when marks exist
	ColorMarkFG:       lipgloss.Color("#6CCB5F"), // distinct mark indicator
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

type Colors struct {
	ColorHiFG      lipgloss.Color
	ColorMutFG     lipgloss.Color
	ColorMarkFG    lipgloss.Color
	ColorDimFG     lipgloss.Color
	ColorStatusBG  lipgloss.Color
	ColorStatusFG  lipgloss.Color
	ColorStatusDim lipgloss.Color
}

type Layout struct {
	StackVisibleCount int
	StackOffsetX      int
	StackOffsetY      int
	CardWidthFrac     float64
	CardHeightFrac    float64
	ActiveLiftY       int
	StickyOverlayNav  *bool
}

type KeyBindings struct {
	Up            []string
	Down          []string
	Random        []string
	OverlayToggle []string
	Mark          []string
	Filter        []string
	Help          []string
	Quit          []string
	PageNext      []string
	PagePrev      []string
	OverlayUp     []string
	OverlayDown   []string
}

func DefaultBindings() KeyBindings {
	return KeyBindings{
		Up:            []string{"k", "up"},
		Down:          []string{"j", "down"},
		Random:        []string{"r"},
		OverlayToggle: []string{"enter"},
		Mark:          []string{"m"},
		Filter:        []string{"t"},
		Help:          []string{"?", "h"},
		Quit:          []string{"q"},
		PageNext:      []string{"n"},
		PagePrev:      []string{"p"},
		OverlayUp:     []string{"k", "up"},
		OverlayDown:   []string{"j", "down"},
	}
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
	styleCardMuted
	styleCardMark
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
	bindings KeyBindings

	lastNavDir  int
	lastNavTime time.Time
	velocity    int

	rng *rand.Rand
	// showHelp toggles the keybinding overlay.
	showHelp bool
	// marked tracks marked cards by absolute index in m.cards.
	marked map[int]bool
	// filterMarked toggles showing only marked cards.
	filterMarked bool
	// overlayPage is the current page offset when viewing overlay content.
	overlayPage int
	// overlayPageStep overrides computed half-page step when >0.
	overlayPageStep int

	err       error
	noteRoot  string
	ready     bool
	lastWidth int

	statusMsg      string
	statusMsgUntil time.Time
}

// NewModel constructs the initial UI model. The RNG controls random jumps; pass
// nil to use the global math/rand instance.
func NewModel(cards []notes.Card, noteRoot string, rng *rand.Rand) Model {
	return Model{
		cards:    cards,
		cursor:   0,
		state:    StateBrowsing,
		settings: DefaultSettings,
		bindings: DefaultBindings(),
		rng:      rng,
		marked:   make(map[int]bool),
		noteRoot: noteRoot,
	}
}

func (m *Model) ApplyColors(c Colors) {
	if c.ColorHiFG != "" {
		m.settings.ColorHiFG = c.ColorHiFG
	}
	if c.ColorMutFG != "" {
		m.settings.ColorMutFG = c.ColorMutFG
	}
	if c.ColorMarkFG != "" {
		m.settings.ColorMarkFG = c.ColorMarkFG
	}
	if c.ColorDimFG != "" {
		m.settings.ColorDimFG = c.ColorDimFG
	}
	if c.ColorStatusBG != "" {
		m.settings.ColorStatusBG = c.ColorStatusBG
	}
	if c.ColorStatusFG != "" {
		m.settings.ColorStatusFG = c.ColorStatusFG
	}
	if c.ColorStatusDim != "" {
		m.settings.ColorStatusDim = c.ColorStatusDim
	}
}

func (m *Model) SetPageStep(step int) {
	if step > 0 {
		m.overlayPageStep = step
	}
}

func (m *Model) ApplyNav(navAccelMs, navMaxStep int) {
	if navAccelMs > 0 {
		m.settings.NavAccelWindow = time.Duration(navAccelMs) * time.Millisecond
	}
	if navMaxStep > 0 {
		m.settings.NavMaxStep = navMaxStep
	}
}

// ApplyBindings overrides default keybindings with provided values (non-empty slices).
func (m *Model) ApplyBindings(b KeyBindings) {
	override := func(dst *[]string, src []string) {
		if len(src) > 0 {
			*dst = src
		}
	}
	override(&m.bindings.Up, b.Up)
	override(&m.bindings.Down, b.Down)
	override(&m.bindings.Random, b.Random)
	override(&m.bindings.OverlayToggle, b.OverlayToggle)
	override(&m.bindings.Mark, b.Mark)
	override(&m.bindings.Filter, b.Filter)
	override(&m.bindings.Help, b.Help)
	override(&m.bindings.Quit, b.Quit)
	override(&m.bindings.PageNext, b.PageNext)
	override(&m.bindings.PagePrev, b.PagePrev)
	override(&m.bindings.OverlayUp, b.OverlayUp)
	override(&m.bindings.OverlayDown, b.OverlayDown)
}

// ApplyLayout overrides layout-related settings.
func (m *Model) ApplyLayout(l Layout) {
	if l.StackVisibleCount > 0 {
		m.settings.StackVisibleCount = l.StackVisibleCount
	}
	if l.StackOffsetX != 0 {
		m.settings.StackOffsetX = l.StackOffsetX
	}
	if l.StackOffsetY != 0 {
		m.settings.StackOffsetY = l.StackOffsetY
	}
	if l.CardWidthFrac > 0 {
		m.settings.CardWidthFrac = l.CardWidthFrac
	}
	if l.CardHeightFrac > 0 {
		m.settings.CardHeightFrac = l.CardHeightFrac
	}
	if l.ActiveLiftY != 0 {
		m.settings.ActiveLiftY = l.ActiveLiftY
	}
	if l.StickyOverlayNav != nil {
		m.settings.StickyOverlayNav = *l.StickyOverlayNav
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

	vis := m.visibleIndices()
	if len(vis) == 0 {
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
	cardMutedStyle := lipgloss.NewStyle().Foreground(m.settings.ColorMutFG)

	styles := []lipgloss.Style{
		styleBase:          baseStyle,
		styleCardDim:       cardDimStyle,
		styleCardHi:        cardHiStyle,
		styleCardMuted:     cardMutedStyle,
		styleCardMark:      lipgloss.NewStyle().Foreground(m.settings.ColorMarkFG).Bold(true),
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

// visibleIndices returns the card indices respecting the current filter.
func (m Model) visibleIndices() []int {
	if m.filterMarked {
		if len(m.marked) == 0 {
			return nil
		}
		vis := make([]int, 0, len(m.marked))
		for i := range m.cards {
			if m.marked[i] {
				vis = append(vis, i)
			}
		}
		return vis
	}

	vis := make([]int, len(m.cards))
	for i := range m.cards {
		vis[i] = i
	}
	return vis
}

// visibleCursorIndex returns the cursor's position within the visible list.
// If not present, returns -1.
func (m Model) visibleCursorIndex(vis []int) int {
	for i, idx := range vis {
		if idx == m.cursor {
			return i
		}
	}
	return -1
}

// ensureCursorVisible snaps the cursor to a valid visible index if needed.
func (m Model) ensureCursorVisible() Model {
	vis := m.visibleIndices()
	if len(vis) == 0 {
		m.cursor = 0
		m.overlayPage = 0
		return m
	}
	if m.visibleCursorIndex(vis) == -1 {
		m.cursor = vis[0]
		m.overlayPage = 0
	}
	return m
}

func (m Model) setStatus(msg string, dur time.Duration) Model {
	m.statusMsg = msg
	m.statusMsgUntil = time.Now().Add(dur)
	return m
}

func sanitizeContent(s string) string {
	// Replace Obsidian-style links [[note]] or [[note|alias]] with visible text.
	for {
		start := strings.Index(s, "[[")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "]]")
		if end == -1 {
			break
		}
		end += start
		inner := s[start+2 : end]
		parts := strings.SplitN(inner, "|", 2)
		repl := parts[0]
		if len(parts) == 2 && parts[1] != "" {
			repl = parts[1]
		}
		s = s[:start] + repl + s[end+2:]
	}
	return s
}

func (m Model) isBinding(key string, set []string) bool {
	for _, k := range set {
		if key == k {
			return true
		}
	}
	return false
}

func (m Model) toggleMark() Model {
	if len(m.cards) == 0 {
		return m
	}
	if m.marked[m.cursor] {
		delete(m.marked, m.cursor)
	} else {
		m.marked[m.cursor] = true
	}

	// If filtering and we just removed the last marked card, drop the filter.
	if m.filterMarked {
		vis := m.visibleIndices()
		if len(vis) == 0 {
			m.filterMarked = false
			return m.ensureCursorVisible()
		}
		if m.visibleCursorIndex(vis) == -1 {
			m.cursor = vis[0]
		}
	}
	return m
}

func (m Model) toggleFilter() Model {
	if m.filterMarked {
		m.filterMarked = false
		return m.ensureCursorVisible()
	}
	// Turn on filtering only if something is marked.
	if len(m.marked) == 0 {
		return m.setStatus("No marked cards to filter", 2*time.Second)
	}
	m.filterMarked = true
	m = m.ensureCursorVisible()
	m.overlayPage = 0
	return m
}

func (m Model) nextPage() Model {
	if m.state != StateViewing {
		return m
	}
	step := m.pageStep()
	return m.scrollOverlayLines(step)
}

func (m Model) prevPage() Model {
	if m.state != StateViewing {
		return m
	}
	step := m.pageStep()
	return m.scrollOverlayLines(-step)
}

func (m Model) pageStep() int {
	if m.overlayPageStep > 0 {
		return m.overlayPageStep
	}
	_, cardH := m.cardSize()
	bodyH := cardH - 4 // header + borders
	if bodyH < 1 {
		return 1
	}
	return max(1, bodyH/2)
}

func (m Model) scrollOverlayLines(delta int) Model {
	if m.state != StateViewing {
		return m
	}
	bodyH, total := m.overlayLimits()
	if bodyH <= 0 || total == 0 {
		return m
	}
	maxStart := total - bodyH
	if maxStart < 0 {
		maxStart = 0
	}

	offset := m.overlayPage + delta
	if offset < 0 {
		offset = 0
	}
	if offset > maxStart {
		offset = maxStart
	}
	m.overlayPage = offset
	return m
}

// overlayLimits computes body height and total wrapped lines for the current card.
func (m Model) overlayLimits() (bodyH int, totalLines int) {
	if len(m.cards) == 0 {
		return 0, 0
	}
	cardW, cardH := m.cardSize()
	bodyW := cardW - 4
	bodyH = cardH - 4
	if bodyW <= 0 || bodyH <= 0 {
		return bodyH, 0
	}
	content := sanitizeContent(m.cards[m.cursor].Content)
	lines := wrapText(content, bodyW, -1)
	return bodyH, len(lines)
}
