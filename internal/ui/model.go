package ui

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
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

type promptKind int

const (
	promptNone promptKind = iota
	promptBox
	promptNewFile
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

	BorderCorner rune
	BorderH      rune
	BorderV      rune
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

	BorderCorner: '+',
	BorderH:      '-',
	BorderV:      '|',
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
	MaxCursorDepth    int
	BorderCorner      rune
	BorderH           rune
	BorderV           rune
}

type KeyBindings struct {
	Up            []string
	Down          []string
	Random        []string
	OverlayToggle []string
	OpenEditor    []string
	OpenBox       []string
	NewFile       []string
	Mark          []string
	Filter        []string
	Help          []string
	Debug         []string
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
		OpenEditor:    []string{"e"},
		OpenBox:       []string{"b"},
		NewFile:       []string{"a"},
		Mark:          []string{"m"},
		Filter:        []string{"t"},
		Help:          []string{"?", "h"},
		Debug:         []string{"d"},
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
	// showDebug toggles the debug overlay.
	showDebug bool
	// marked tracks marked cards by absolute file path (persists across box switches).
	marked map[string]bool
	// boxFilters remembers filter state per box/root.
	boxFilters map[string]bool
	// filterMarked toggles showing only marked cards.
	filterMarked bool
	// overlayPage is the current page offset when viewing overlay content.
	overlayPage int
	// overlayPageStep overrides computed half-page step when >0.
	overlayPageStep int

	// loadOpts keeps the include/ignore globs so we can reload boxes in-session.
	loadOpts notes.LoadOptions
	// boxes tracks visited roots; activeBox indexes boxes.
	boxes     []string
	activeBox int

	// Prompt UI state
	prompt promptState
	// stateBeforePrompt lets us restore browsing/viewing after prompt dismissal.
	stateBeforePrompt State

	err       error
	noteRoot  string
	ready     bool
	lastWidth int

	statusMsg      string
	statusMsgUntil time.Time

	updateSamples []time.Duration
}

type promptState struct {
	kind        promptKind
	input       textinput.Model
	selectedBox int
}

type boxLoadResult struct {
	path  string
	cards []notes.Card
	err   error
}

type newFileResult struct {
	box  string
	path string
	err  error
}

type editorResult struct {
	path string
	err  error
}

// NewModel constructs the initial UI model. The RNG controls random jumps; pass
// nil to use the global math/rand instance.
func NewModel(cards []notes.Card, noteRoot string, loadOpts notes.LoadOptions, rng *rand.Rand) Model {
	root := cleanBoxPath(noteRoot)
	m := Model{
		cards:    cards,
		cursor:   0,
		state:    StateBrowsing,
		settings: DefaultSettings,
		bindings: DefaultBindings(),
		rng:      rng,
		marked:   make(map[string]bool),
		boxFilters: map[string]bool{
			root: false,
		},
		noteRoot: root,
		loadOpts: loadOpts,
	}
	m = m.setActiveBox(root)
	return m
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
	override(&m.bindings.OpenEditor, b.OpenEditor)
	override(&m.bindings.OpenBox, b.OpenBox)
	override(&m.bindings.NewFile, b.NewFile)
	override(&m.bindings.Up, b.Up)
	override(&m.bindings.Down, b.Down)
	override(&m.bindings.Random, b.Random)
	override(&m.bindings.OverlayToggle, b.OverlayToggle)
	override(&m.bindings.Mark, b.Mark)
	override(&m.bindings.Filter, b.Filter)
	override(&m.bindings.Help, b.Help)
	override(&m.bindings.Debug, b.Debug)
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
	if l.MaxCursorDepth > 0 {
		m.settings.MaxCursorDepth = l.MaxCursorDepth
	}
	if l.BorderCorner != 0 {
		m.settings.BorderCorner = l.BorderCorner
	}
	if l.BorderH != 0 {
		m.settings.BorderH = l.BorderH
	}
	if l.BorderV != 0 {
		m.settings.BorderV = l.BorderV
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	if m.state == StatePrompting {
		return m.renderPrompt()
	}
	if m.showDebug {
		return m.renderDebug()
	}
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

func (m Model) isMarked(idx int) bool {
	if idx < 0 || idx >= len(m.cards) {
		return false
	}
	return m.marked[m.cards[idx].Path]
}

func (m Model) markedCountCurrent() int {
	count := 0
	for i := range m.cards {
		if m.isMarked(i) {
			count++
		}
	}
	return count
}

func (m Model) setFilterForCurrent(state bool) Model {
	if m.boxFilters == nil {
		m.boxFilters = make(map[string]bool)
	}
	m.filterMarked = state
	m.boxFilters[m.currentBox()] = state
	return m
}

// visibleIndices returns the card indices respecting the current filter.
func (m Model) visibleIndices() []int {
	if m.filterMarked {
		if m.markedCountCurrent() == 0 {
			return nil
		}
		vis := make([]int, 0, len(m.marked))
		for i := range m.cards {
			if m.isMarked(i) {
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

func (m Model) isBinding(key string, set []string) bool {
	for _, k := range set {
		if key == k {
			return true
		}
	}
	return false
}

func appendSample(samples []time.Duration, d time.Duration) []time.Duration {
	const maxSamples = 50
	samples = append(samples, d)
	if len(samples) > maxSamples {
		samples = samples[len(samples)-maxSamples:]
	}
	return samples
}

func (m Model) withUpdateSample(start time.Time) Model {
	m.updateSamples = appendSample(m.updateSamples, time.Since(start))
	return m
}

func (m Model) defaultExt() string {
	if len(m.loadOpts.IncludeExts) > 0 {
		return m.loadOpts.IncludeExts[0]
	}
	return ".md"
}

func cleanBoxPath(path string) string {
	if strings.TrimSpace(path) == "" {
		path = "."
	}
	path = filepath.Clean(path)
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return path
}

func (m Model) setActiveBox(path string) Model {
	path = cleanBoxPath(path)
	for i, b := range m.boxes {
		if b == path {
			m.activeBox = i
			m.noteRoot = path
			return m
		}
	}
	m.boxes = append(m.boxes, path)
	m.activeBox = len(m.boxes) - 1
	m.noteRoot = path
	return m
}

func (m Model) currentBox() string {
	if len(m.boxes) == 0 {
		return m.noteRoot
	}
	if m.activeBox < 0 || m.activeBox >= len(m.boxes) {
		return m.noteRoot
	}
	return m.boxes[m.activeBox]
}

func (m Model) startPrompt(kind promptKind, initial string) Model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.SetValue(initial)
	if m.viewport.Width > 4 {
		ti.Width = m.viewport.Width - 4
	} else {
		ti.Width = 40
	}
	ti.Focus()
	m.prompt = promptState{
		kind:        kind,
		input:       ti,
		selectedBox: m.activeBox,
	}
	m.stateBeforePrompt = m.state
	m.state = StatePrompting
	m.showHelp = false
	m.showDebug = false
	return m
}

func (m Model) startBoxPrompt() Model {
	return m.startPrompt(promptBox, m.noteRoot)
}

func (m Model) startNewFilePrompt() Model {
	return m.startPrompt(promptNewFile, "")
}

// loadBoxCmd loads the given path with the model's load options in a goroutine.
func (m Model) loadBoxCmd(path string) tea.Cmd {
	root := cleanBoxPath(path)
	opts := m.loadOpts
	return func() tea.Msg {
		cards, err := notes.LoadCardsFromDir(root, opts)
		return boxLoadResult{path: root, cards: cards, err: err}
	}
}

// openInEditorCmd launches the editor (VISUAL/EDITOR or OS default) for the path.
func (m Model) openInEditorCmd(path string) tea.Cmd {
	return func() tea.Msg {
		return editorResult{path: path, err: launchEditor(path)}
	}
}

// createFileCmd makes sure the file exists under boxRoot, creates directories,
// and opens it in the editor.
func (m Model) createFileCmd(boxRoot, userPath string) tea.Cmd {
	root := cleanBoxPath(boxRoot)
	defaultExt := m.defaultExt()
	return func() tea.Msg {
		raw := strings.TrimSpace(userPath)
		if raw == "" {
			return newFileResult{box: root, err: errors.New("file name required")}
		}

		target := raw
		if !filepath.IsAbs(target) {
			target = filepath.Join(root, target)
		}
		target = filepath.Clean(target)
		if filepath.Ext(target) == "" && defaultExt != "" {
			target += defaultExt
		}

		dir := filepath.Dir(target)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return newFileResult{box: root, path: target, err: fmt.Errorf("make dir: %w", err)}
		}

		// Create the file if missing; preserve existing content otherwise.
		if _, err := os.Stat(target); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return newFileResult{box: root, path: target, err: fmt.Errorf("stat: %w", err)}
			}
			if err := os.WriteFile(target, []byte{}, 0o644); err != nil {
				return newFileResult{box: root, path: target, err: fmt.Errorf("create: %w", err)}
			}
		}

		if err := launchEditor(target); err != nil {
			return newFileResult{box: root, path: target, err: fmt.Errorf("open: %w", err)}
		}

		return newFileResult{box: root, path: target, err: nil}
	}
}

func (m Model) resetAfterLoad(cards []notes.Card, root string) Model {
	prev := m.currentBox()
	if m.boxFilters == nil {
		m.boxFilters = make(map[string]bool)
	}
	m.boxFilters[prev] = m.filterMarked

	m = m.setActiveBox(root)
	// Load previous filter state for this box (default false).
	if state, ok := m.boxFilters[m.currentBox()]; ok {
		m.filterMarked = state
	} else {
		m.filterMarked = false
		m.boxFilters[m.currentBox()] = false
	}
	m.cards = cards
	m.cursor = 0
	m.overlayPage = 0
	m.state = StateBrowsing
	return m.ensureCursorVisible()
}

func (m Model) clearPrompt() Model {
	m.prompt = promptState{}
	if m.state == StatePrompting {
		m.state = m.stateBeforePrompt
	}
	return m
}

func launchEditor(path string) error {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return runEditorCommand(editor, path)
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return runEditorCommand(editor, path)
	}
	return openWithSystem(path)
}

func runEditorCommand(cmdStr, path string) error {
	command := fmt.Sprintf("%s %s", cmdStr, strconv.Quote(path))
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", command).Start()
	}
	return exec.Command("sh", "-c", command).Start()
}

func openWithSystem(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", "", strconv.Quote(path)).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func (m Model) toggleMark() Model {
	if len(m.cards) == 0 {
		return m
	}
	path := m.cards[m.cursor].Path
	if m.marked[path] {
		delete(m.marked, path)
	} else {
		m.marked[path] = true
	}

	// If filtering and we just removed the last marked card, drop the filter.
	if m.filterMarked {
		vis := m.visibleIndices()
		if len(vis) == 0 {
			m = m.setFilterForCurrent(false)
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
		m = m.setFilterForCurrent(false)
		return m.ensureCursorVisible()
	}
	// Turn on filtering only if something is marked.
	if m.markedCountCurrent() == 0 {
		return m.setStatus("No marked cards to filter", 2*time.Second)
	}
	m = m.setFilterForCurrent(true)
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
	content := m.cards[m.cursor].Content
	lines := wrapText(content, bodyW, -1)
	return bodyH, len(lines)
}
