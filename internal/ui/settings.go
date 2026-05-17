package ui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Viewport holds the current terminal dimensions.
type Viewport struct {
	Width  int
	Height int
}

// Settings holds all tuneable display and behaviour parameters for the UI.
// The zero value is not useful; start from DefaultSettings and apply overrides
// via the Apply* methods on Model.
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

	BorderTL rune
	BorderTR rune
	BorderBL rune
	BorderBR rune
	BorderH  rune
	BorderV  rune

	NewFileLinkTemplate string
	NewFileSameDir      bool
	NewFileEditor       string // "inapp" | "external" | "none"
	ContinueNameCmd     string
	BranchNameCmd       string
	AutoSplitOnLink     bool

	TextWidth int // hard-wrap column in the editor; 0 disables
}

// DefaultSettings is the out-of-the-box configuration used by NewModel.
var DefaultSettings = Settings{
	StackVisibleCount: 7,
	StackOffsetX:      2,
	StackOffsetY:      1,
	CardWidthFrac:     0.6,
	CardHeightFrac:    0.6,
	ColorHiFG:         lipgloss.Color("#FFD166"),
	ColorMutFG:        lipgloss.Color("#444444"),
	ColorMarkFG:       lipgloss.Color("#6CCB5F"),
	ColorDimFG:        lipgloss.Color("#666666"),
	ColorStatusBG:     lipgloss.Color("#222222"),
	ColorStatusFG:     lipgloss.Color("#F5F5F5"),
	ColorStatusDim:    lipgloss.Color("#999999"),
	NavAccelWindow:    350 * time.Millisecond,
	NavMaxStep:        8,
	MaxCursorDepth:    2,
	ActiveLiftY:       2,
	StickyOverlayNav:  false,

	BorderTL: '╭',
	BorderTR: '╮',
	BorderBL: '╰',
	BorderBR: '╯',
	BorderH:  '─',
	BorderV:  '│',

	NewFileLinkTemplate: "--> %s\n\n",
	NewFileSameDir:      true,
	NewFileEditor:       "inapp",
	ContinueNameCmd:     "",
	BranchNameCmd:       "",
	AutoSplitOnLink:     true,
	TextWidth:           80,
}

// Colors groups the colour overrides accepted by ApplyColors.
type Colors struct {
	ColorHiFG      lipgloss.Color
	ColorMutFG     lipgloss.Color
	ColorMarkFG    lipgloss.Color
	ColorDimFG     lipgloss.Color
	ColorStatusBG  lipgloss.Color
	ColorStatusFG  lipgloss.Color
	ColorStatusDim lipgloss.Color
}

// Layout groups the display-geometry overrides accepted by ApplyLayout.
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

// KeyBindings maps actions to their trigger key strings.
type KeyBindings struct {
	Up            []string
	Down          []string
	Random        []string
	OverlayToggle []string
	OpenInApp     []string
	OpenExternal  []string
	OpenBox       []string
	NewFile       []string
	Continue      []string
	Branch        []string
	NextRoot      []string
	SuspendEditor []string
	Mark          []string
	Filter        []string
	Help          []string
	Debug         []string
	Quit          []string
	PageNext      []string
	PagePrev      []string
	OverlayUp     []string
	OverlayDown   []string
	Reload        []string
	NavFirst      []string
	NavLast       []string
	SwitchPane    []string
}

// DefaultBindings returns the out-of-the-box keybinding set.
func DefaultBindings() KeyBindings {
	return KeyBindings{
		Up:            []string{"k", "up"},
		Down:          []string{"j", "down"},
		Random:        []string{"r"},
		OverlayToggle: []string{"enter"},
		OpenInApp:     []string{"e"},
		OpenExternal:  []string{"E"},
		OpenBox:       []string{"b"},
		NewFile:       []string{"a"},
		Continue:      []string{"c"},
		Branch:        []string{"C"},
		NextRoot:      []string{"N"},
		SuspendEditor: []string{"ctrl+b"},
		Mark:          []string{"m"},
		Filter:        []string{"t"},
		Help:          []string{"?", "h"},
		Debug:         []string{"d"},
		Quit:          []string{"q"},
		PageNext:      []string{"n"},
		PagePrev:      []string{"p"},
		OverlayUp:     []string{"k", "up"},
		OverlayDown:   []string{"j", "down"},
		Reload:        []string{"R"},
		NavFirst:      []string{"g"},
		NavLast:       []string{"G"},
		SwitchPane:    []string{"ctrl+w"},
	}
}

// FileCreation holds configuration for the continue/branch file-creation feature.
type FileCreation struct {
	LinkTemplate    string
	SameDir         *bool
	NewFileEditor   string
	ContinueCmd     string
	BranchCmd       string
	AutoSplitOnLink *bool
}

// ---------------------------------------------------------------------------
// Apply* methods — merge configuration into the model.
// ---------------------------------------------------------------------------

// ApplyColors overrides colour settings with any non-zero values in c.
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

// ApplyLayout overrides layout settings with any non-zero values in l.
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
		m.settings.BorderTL = l.BorderCorner
		m.settings.BorderTR = l.BorderCorner
		m.settings.BorderBL = l.BorderCorner
		m.settings.BorderBR = l.BorderCorner
	}
	if l.BorderH != 0 {
		m.settings.BorderH = l.BorderH
	}
	if l.BorderV != 0 {
		m.settings.BorderV = l.BorderV
	}
}

// ApplyBindings overrides default keybindings with provided values (non-empty slices only).
func (m *Model) ApplyBindings(b KeyBindings) {
	override := func(dst *[]string, src []string) {
		if len(src) > 0 {
			*dst = src
		}
	}
	override(&m.bindings.OpenInApp, b.OpenInApp)
	override(&m.bindings.OpenExternal, b.OpenExternal)
	override(&m.bindings.OpenBox, b.OpenBox)
	override(&m.bindings.NewFile, b.NewFile)
	override(&m.bindings.Continue, b.Continue)
	override(&m.bindings.Branch, b.Branch)
	override(&m.bindings.NextRoot, b.NextRoot)
	override(&m.bindings.SuspendEditor, b.SuspendEditor)
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
	override(&m.bindings.NavFirst, b.NavFirst)
	override(&m.bindings.NavLast, b.NavLast)
	override(&m.bindings.SwitchPane, b.SwitchPane)
}

// ApplyFileCreation overrides file-creation settings with non-zero values.
func (m *Model) ApplyFileCreation(fc FileCreation) {
	if fc.LinkTemplate != "" {
		m.settings.NewFileLinkTemplate = fc.LinkTemplate
	}
	if fc.SameDir != nil {
		m.settings.NewFileSameDir = *fc.SameDir
	}
	if fc.NewFileEditor != "" {
		m.settings.NewFileEditor = fc.NewFileEditor
	}
	if fc.ContinueCmd != "" {
		m.settings.ContinueNameCmd = fc.ContinueCmd
	}
	if fc.BranchCmd != "" {
		m.settings.BranchNameCmd = fc.BranchCmd
	}
	if fc.AutoSplitOnLink != nil {
		m.settings.AutoSplitOnLink = *fc.AutoSplitOnLink
	}
}

// ApplyNav overrides navigation acceleration settings with any positive values.
func (m *Model) ApplyNav(navAccelMs, navMaxStep int) {
	if navAccelMs > 0 {
		m.settings.NavAccelWindow = time.Duration(navAccelMs) * time.Millisecond
	}
	if navMaxStep > 0 {
		m.settings.NavMaxStep = navMaxStep
	}
}

// SetPageStep sets the overlay page-scroll step in lines. Values ≤ 0 are
// ignored; the default is half the visible body height.
func (m *Model) SetPageStep(step int) {
	if step > 0 {
		m.overlayPageStep = step
	}
}

// SetTextWidth sets the hard-wrap column for the in-app editor. Pass 0 to disable.
func (m *Model) SetTextWidth(n int) {
	if n < 0 {
		n = 0
	}
	m.settings.TextWidth = n
}

// EnableDebugUI gates the debug overlay; when false, debug toggles are ignored.
func (m *Model) EnableDebugUI(enabled bool) {
	m.enableDebug = enabled
	if !enabled {
		m.showDebug = false
	}
}

// SetLoadDuration stores the initial load duration for the debug view.
func (m *Model) SetLoadDuration(d time.Duration) {
	m.loadDuration = d
}

// SetLoadBreakdown stores optional walk/sort timings for the debug view.
func (m *Model) SetLoadBreakdown(walk, sort time.Duration) {
	m.loadWalk = walk
	m.loadSort = sort
}
