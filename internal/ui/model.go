// Package ui provides the Bubble Tea model, key handling, and rendering for Thumbr.
package ui

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vincent-margiotta/thumbr/internal/notes"
)

// ---------------------------------------------------------------------------
// State machine types
// ---------------------------------------------------------------------------

// State represents the current interaction mode of the UI.
type State int

const (
	StateBrowsing  State = iota // navigating the card stack
	StateViewing                // reading a card in the overlay
	StatePrompting              // entering text in a prompt (box path, new file)
	StateEditing                // in-app vim editor is active
)

func (s State) String() string {
	switch s {
	case StateBrowsing:
		return "Browsing"
	case StateViewing:
		return "Viewing"
	case StatePrompting:
		return "Prompting"
	case StateEditing:
		return "Editing"
	default:
		return "Unknown"
	}
}

type promptKind int

const (
	promptNone promptKind = iota
	promptBox
	promptNewFile
)

// ---------------------------------------------------------------------------
// Rendering internals
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// Model is the Bubble Tea model for Thumbr.
type Model struct {
	cards    []notes.Card
	cursor   int
	state    State
	viewport Viewport
	settings Settings
	bindings KeyBindings

	enableDebug bool

	lastNavDir  int
	lastNavTime time.Time
	navPending  string // pending nav prefix key ("g" waits for a second key)

	rng             *rand.Rand
	showHelp        bool
	showDebug       bool
	marked          map[string]bool // absolute file paths of marked cards
	boxFilters      map[string]bool // per-box filter state
	filterMarked    bool
	overlayPage     int
	overlayPageStep int // overrides computed half-page step when > 0

	loadOpts  notes.LoadOptions
	boxes     []string
	activeBox int

	prompt            promptState
	stateBeforePrompt State

	err      error
	noteRoot string
	ready    bool

	statusMsg      string
	statusMsgUntil time.Time

	updateSamples []time.Duration
	loadDuration  time.Duration
	loadWalk      time.Duration
	loadSort      time.Duration

	keyCount     int
	updateMax    time.Duration
	updateOver16 int
	updateOver33 int

	contentErrors int
	editorErrors  int

	pendingQuit      bool
	pendingQuitUntil time.Time

	// pendingSeekPath causes the next resetAfterLoad to navigate to this path.
	pendingSeekPath string

	// editors[0] is the top pane, editors[1] is the bottom pane.
	editors    [2]editorState
	activePane int // 0 or 1
	paneCount  int // 0=none, 1=single, 2=split

	// watchStop is closed to signal the current watcher goroutine to exit.
	watchStop chan struct{}
	// watchingBox is the box path currently being watched.
	watchingBox string

	// pendingEditorPath causes the next boxLoadResult to open this file in-app.
	pendingEditorPath string
	pendingEditorLine int
	// pendingSourcePath, when non-empty alongside pendingEditorPath, triggers auto-split.
	pendingSourcePath string
}

type promptState struct {
	kind        promptKind
	input       textinput.Model
	selectedBox int
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewModel constructs the initial UI model. Pass nil for rng to use the
// global math/rand instance.
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
		noteRoot:    root,
		loadOpts:    loadOpts,
		watchStop:   make(chan struct{}),
		watchingBox: root,
	}
	m = m.setActiveBox(root)
	return m
}

// ---------------------------------------------------------------------------
// Bubble Tea interface
// ---------------------------------------------------------------------------

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if len(m.cards) > 0 {
		vis := m.visibleIndices()
		n := m.settings.StackVisibleCount + 2
		if n > len(vis) {
			n = len(vis)
		}
		cmds = append(cmds, preloadCardsCmd(m.cards, vis[:n]))
	}
	if m.settings.LiveReload {
		cmds = append(cmds, watchDirCmd(m.watchStop, m.currentBox(), m.loadOpts))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.pendingQuit && time.Now().After(m.pendingQuitUntil) {
		m.pendingQuit = false
	}
	if m.state == StatePrompting {
		return m.renderPrompt()
	}
	if m.state == StateEditing {
		return m.renderEditor()
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

	grid := make([][]cell, m.viewport.Height)
	for y := range grid {
		grid[y] = make([]cell, m.viewport.Width)
		for x := range grid[y] {
			grid[y][x] = cell{ch: ' ', styleID: styleBase}
		}
	}

	geoms := m.computeStackGeometry()

	if m.state == StateViewing {
		for i := range geoms {
			geoms[i].active = false
		}
	}

	sort.Slice(geoms, func(i, j int) bool {
		return geoms[i].depth > geoms[j].depth
	})
	for _, g := range geoms {
		m.drawCardOntoGrid(grid, g)
	}

	if m.state == StateViewing {
		m.drawOverlayCardOntoGrid(grid)
	}

	return m.renderGrid(grid, styles) + "\n" + m.renderStatusBar()
}

// ---------------------------------------------------------------------------
// Pure state helpers
// ---------------------------------------------------------------------------

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

// visibleIndices returns card indices respecting the current filter.
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

// visibleCursorIndex returns the cursor's position within the visible list,
// or -1 if the cursor is not visible.
func (m Model) visibleCursorIndex(vis []int) int {
	for i, idx := range vis {
		if idx == m.cursor {
			return i
		}
	}
	return -1
}

// ensureCursorVisible snaps the cursor to the first visible index if needed.
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
	d := time.Since(start)
	m.updateSamples = appendSample(m.updateSamples, d)
	if d > m.updateMax {
		m.updateMax = d
	}
	if d > 16*time.Millisecond {
		m.updateOver16++
	}
	if d > 33*time.Millisecond {
		m.updateOver33++
	}
	return m
}

func (m Model) defaultExt() string {
	if len(m.loadOpts.IncludeExts) > 0 {
		return m.loadOpts.IncludeExts[0]
	}
	return ".md"
}

// ensureVisibleContent synchronously loads content for the depth-0 card in the
// current stack view — the card whose full preview is displayed. Called after
// each navigation move so the preview is present in the same render frame.
func (m Model) ensureVisibleContent() Model {
	vis := m.visibleIndices()
	pos := m.visibleCursorIndex(vis)
	if pos < 0 {
		pos = 0
	}
	front := pos - m.settings.MaxCursorDepth
	if front < 0 {
		front = 0
	}
	return m.ensureCardContent(vis[front])
}

func (m Model) ensureCardContent(idx int) Model {
	if idx < 0 || idx >= len(m.cards) {
		return m
	}
	card := &m.cards[idx]
	if card.ContentLoaded {
		return m
	}
	if err := card.LoadContent(); err != nil {
		m.contentErrors++
		msg := fmt.Sprintf("Read failed: %v", err)
		card.Content = msg
		m.err = err
		m = m.setStatus(msg, 3*time.Second)
	}
	return m
}

func (m Model) attemptQuit() (Model, tea.Cmd) {
	if m.pendingQuit && time.Now().Before(m.pendingQuitUntil) {
		return m, tea.Quit
	}
	m.pendingQuit = true
	m.pendingQuitUntil = time.Now().Add(3 * time.Second)
	m = m.setStatus("Press quit again within 3s to exit", 3*time.Second)
	return m, nil
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

func (m Model) resetAfterLoad(cards []notes.Card, root string) Model {
	prev := m.currentBox()
	if m.boxFilters == nil {
		m.boxFilters = make(map[string]bool)
	}
	m.boxFilters[prev] = m.filterMarked

	m = m.setActiveBox(root)
	if state, ok := m.boxFilters[m.currentBox()]; ok {
		m.filterMarked = state
	} else {
		m.filterMarked = false
		m.boxFilters[m.currentBox()] = false
	}
	m.cards = cards
	m.cursor = 0
	if m.pendingSeekPath != "" {
		for i, c := range cards {
			if c.Path == m.pendingSeekPath {
				m.cursor = i
				break
			}
		}
		m.pendingSeekPath = ""
	}
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
	if m.filterMarked {
		vis := m.visibleIndices()
		if len(vis) == 0 {
			m = m.setFilterForCurrent(false)
			return m.ensureCursorVisible()
		}
		if m.visibleCursorIndex(vis) == -1 {
			// Snap to the nearest remaining card: first one after the removed
			// card's position, or the last one before it if none follows.
			nearest := vis[0]
			for _, idx := range vis {
				if idx > m.cursor {
					nearest = idx
					break
				}
				nearest = idx
			}
			m.cursor = nearest
		}
	}
	return m
}

func (m Model) toggleFilter() Model {
	if m.filterMarked {
		m = m.setFilterForCurrent(false)
		return m.ensureCursorVisible()
	}
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
	return m.scrollOverlayLines(m.pageStep())
}

func (m Model) prevPage() Model {
	if m.state != StateViewing {
		return m
	}
	return m.scrollOverlayLines(-m.pageStep())
}

func (m Model) pageStep() int {
	if m.overlayPageStep > 0 {
		return m.overlayPageStep
	}
	_, cardH := m.cardSize()
	bodyH := cardH - 4
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

// overlayLimits returns body height and total wrapped lines for the current card.
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
	lines := wrapText(m.cards[m.cursor].Content, bodyW, -1)
	return bodyH, len(lines)
}

// ---------------------------------------------------------------------------
// Low-level utilities
// ---------------------------------------------------------------------------

func stripANSI(s string) string {
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
