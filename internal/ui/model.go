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
	StateEditing                // in-app vim editor is active
	StatePrompting              // text prompt (free-mode new card)
	StateBoxMenu                // box-picker menu
)

func (s State) String() string {
	switch s {
	case StateBrowsing:
		return "Browsing"
	case StateViewing:
		return "Viewing"
	case StateEditing:
		return "Editing"
	case StatePrompting:
		return "Prompting"
	case StateBoxMenu:
		return "BoxMenu"
	default:
		return "Unknown"
	}
}

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

	navPending string // pending nav prefix key ("g" waits for a second key)

	rng               *rand.Rand
	showHelp          bool
	showDebug         bool
	marked            map[string]bool // absolute file paths of marked cards
	filterMarked      bool
	prompt            promptState
	stateBeforePrompt State
	overlayPage       int
	overlayPageStep   int  // overrides computed half-page step when > 0
	overlayFlipped    bool // true when the overlay is showing the back side of a card

	loadOpts notes.LoadOptions

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
	editors           [2]editorState
	activePane        int  // 0 or 1
	paneCount         int  // 0=none, 1=single, 2=split
	editorHelpVisible bool // true while :help overlay is shown

	// watchStop is closed to signal the current watcher goroutine to exit.
	watchStop chan struct{}
	// watchingBox is the box path currently being watched.
	watchingBox string

	// Multi-box support: boxes holds all open directories; activeBox is the current index.
	boxes         []string
	activeBox     int
	boxFreeMode   map[string]bool   // which boxes run in free mode
	boxLastPath   map[string]string // last visited card path per box (for cursor restore)
	boxMenuCursor int               // selected index in the box-picker menu

	// pendingEditorPath causes the next boxLoadResult to open this file in-app.
	pendingEditorPath string
	pendingEditorLine int
	// pendingSourcePath, when non-empty alongside pendingEditorPath, triggers auto-split.
	pendingSourcePath string
}

type promptState struct {
	input textinput.Model
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewModel constructs the initial UI model. Pass nil for rng to use the
// global math/rand instance.
func NewModel(cards []notes.Card, noteRoot string, loadOpts notes.LoadOptions, rng *rand.Rand) Model {
	root := cleanBoxPath(noteRoot)
	m := Model{
		cards:       cards,
		cursor:      0,
		state:       StateBrowsing,
		settings:    DefaultSettings,
		bindings:    DefaultBindings(),
		rng:         rng,
		marked:      make(map[string]bool),
		noteRoot:    root,
		loadOpts:    loadOpts,
		watchStop:   make(chan struct{}),
		watchingBox: root,
		boxes:       []string{root},
		activeBox:   0,
		boxFreeMode: make(map[string]bool),
		boxLastPath: make(map[string]string),
	}
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
		cmds = append(cmds, watchDirCmd(m.watchStop, m.noteRoot, m.loadOpts))
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
	if m.state == StateBoxMenu {
		return m.renderBoxMenu()
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

	// Cards at depth <= activeDepth occlude the active card and must be solid.
	// Scan before clearing active flags (StateViewing clears them below).
	activeDepth := 0
	for _, g := range geoms {
		if g.active {
			activeDepth = g.depth
			break
		}
	}

	if m.state == StateViewing {
		for i := range geoms {
			geoms[i].active = false
		}
		activeDepth = 0 // overlay mode: only front card fills
	}

	sort.Slice(geoms, func(i, j int) bool {
		return geoms[i].depth > geoms[j].depth
	})
	for _, g := range geoms {
		m.drawCardOntoGrid(grid, g, activeDepth)
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
	m.filterMarked = state
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
	return ".txt"
}

// ensureVisibleContent synchronously loads content for the depth-0 card in the
// current stack view — the card whose full preview is displayed. Called after
// each navigation move so the preview is present in the same render frame.
func (m Model) ensureVisibleContent() Model {
	vis := m.visibleIndices()
	if len(vis) == 0 {
		return m
	}
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

func (m Model) startNewFilePrompt() Model {
	ti := textinput.New()
	ti.Prompt = "> "
	if m.viewport.Width > 4 {
		ti.Width = m.viewport.Width - 4
	} else {
		ti.Width = 40
	}
	ti.Focus()
	m.prompt = promptState{input: ti}
	m.stateBeforePrompt = m.state
	m.state = StatePrompting
	m.showHelp = false
	m.showDebug = false
	return m
}

func (m Model) clearPrompt() Model {
	m.prompt = promptState{}
	if m.state == StatePrompting {
		m.state = m.stateBeforePrompt
	}
	return m
}

func (m Model) resetAfterLoad(cards []notes.Card, root string) Model {
	m.noteRoot = cleanBoxPath(root)
	m.cards = cards
	m.cursor = 0
	m.filterMarked = false
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
	lines := wrapText(m.overlayContent(), bodyW, -1)
	return bodyH, len(lines)
}

// ---------------------------------------------------------------------------
// Card back support
// ---------------------------------------------------------------------------

const backDelimiter = "---back---"
const backPortraitDelimiter = "---back:portrait---"

// splitCardSides splits content on the first line that is exactly "---back---"
// or "---back:portrait---". If no delimiter is found, the full content is
// returned as front with hasBack=false.
func splitCardSides(content string) (front, back string, hasBack, backPortrait bool) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == backDelimiter {
			return strings.Join(lines[:i], "\n"), strings.Join(lines[i+1:], "\n"), true, false
		}
		if trimmed == backPortraitDelimiter {
			return strings.Join(lines[:i], "\n"), strings.Join(lines[i+1:], "\n"), true, true
		}
	}
	return content, "", false, false
}

func joinCardSides(front, back string) string {
	return front + "\n" + backDelimiter + "\n" + back
}

func joinCardSidesPortrait(front, back string) string {
	return front + "\n" + backPortraitDelimiter + "\n" + back
}

// overlayContent returns the content to display in the overlay, respecting
// overlayFlipped to show either the front or back side of the card.
func (m Model) overlayContent() string {
	if len(m.cards) == 0 {
		return ""
	}
	content := m.cards[m.cursor].Content
	front, back, hasBack, _ := splitCardSides(content)
	if m.overlayFlipped {
		if hasBack {
			return back
		}
		return "" // back doesn't exist yet
	}
	if hasBack {
		return front
	}
	return content
}

// isPortraitContext returns true when the current UI state calls for portrait card dimensions:
// editing the back section of a portrait-flagged card, or viewing the flipped side of one.
func (m Model) isPortraitContext() bool {
	switch m.state {
	case StateEditing:
		es := m.editors[m.activePane]
		return es.section == "back" && es.backPortrait
	case StateViewing:
		if m.overlayFlipped && len(m.cards) > 0 {
			_, _, _, backPortrait := splitCardSides(m.cards[m.cursor].Content)
			return backPortrait
		}
	}
	return false
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

// SetBoxes registers one or more boxes from BoxConfig values. The first entry
// becomes the active box and its free-mode setting is applied immediately.
// Called from main after model construction.
func (m *Model) SetBoxes(boxes []BoxConfig) {
	if len(boxes) == 0 {
		return
	}
	paths := make([]string, 0, len(boxes))
	seen := map[string]bool{}
	freeMode := make(map[string]bool)
	for _, b := range boxes {
		c := cleanBoxPath(b.Path)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		paths = append(paths, c)
		if b.Free {
			freeMode[c] = true
		}
	}
	if len(paths) == 0 {
		return
	}
	m.boxes = paths
	m.boxFreeMode = freeMode
	m.activeBox = 0
	m.noteRoot = paths[0]
	m.watchingBox = paths[0]
	m.settings.FreeMode = freeMode[paths[0]]
}

// switchToBox saves the current cursor position and switches to boxes[i], triggering a reload.
func (m Model) switchToBox(i int) (Model, tea.Cmd) {
	if len(m.boxes) <= 1 || i < 0 || i >= len(m.boxes) || i == m.activeBox {
		return m, nil
	}
	if len(m.cards) > 0 && m.cursor < len(m.cards) {
		m.boxLastPath[m.boxes[m.activeBox]] = m.cards[m.cursor].Path
	}
	m.activeBox = i
	m.noteRoot = m.boxes[i]
	m.pendingSeekPath = m.boxLastPath[m.noteRoot]
	m.settings.FreeMode = m.boxFreeMode[m.noteRoot]
	return m, m.loadBoxCmd(m.noteRoot)
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
