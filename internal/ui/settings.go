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

	ColorHiFG      lipgloss.Color
	ColorMutFG     lipgloss.Color
	ColorMarkFG    lipgloss.Color
	ColorDimFG     lipgloss.Color
	ColorStatusBG  lipgloss.Color
	ColorStatusFG  lipgloss.Color
	ColorStatusDim lipgloss.Color

	NavMaxStep int
	NavTau     float64 // breakeven interval (ms): pressing at this rate → step=1
	NavGamma   float64 // power-law exponent; >1 = sharper acceleration with speed
	NavMinDt   float64 // minimum interval floor (ms); hold == pressing at top speed

	MaxCursorDepth int
	ActiveLiftY    int

	BorderTL rune
	BorderTR rune
	BorderBL rune
	BorderBR rune
	BorderH  rune
	BorderV  rune

	NewFileEditor string // "inapp" | "external" | "none"
	ContinueNameCmd     string
	BranchNameCmd       string
	AutoSplitOnLink     bool

	TextWidth        int  // hard-wrap column in the editor; 0 disables
	LiveReload       bool // reload card list when files change on disk
	ExternalEditMode bool // e key opens $EDITOR instead of the in-app editor
	FreeMode         bool // disables c/C; N prompts for arbitrary filename
	CardSizeLimit       bool // block new lines when content exceeds one card face
	HighlightOverLimit  bool // subtly highlight editor lines that exceed the card face
	ColorOverLimit      lipgloss.Color
}

// DefaultSettings is the out-of-the-box configuration used by NewModel.
var DefaultSettings = Settings{
	StackVisibleCount: 7,
	StackOffsetX:      2,
	StackOffsetY:      1,
	ColorHiFG:         lipgloss.Color("#FFD166"),
	ColorMutFG:        lipgloss.Color("#444444"),
	ColorMarkFG:       lipgloss.Color("#6CCB5F"),
	ColorDimFG:        lipgloss.Color("#666666"),
	ColorStatusBG:     lipgloss.Color("#222222"),
	ColorStatusFG:     lipgloss.Color("#F5F5F5"),
	ColorStatusDim:    lipgloss.Color("#999999"),
	NavMaxStep: 8,
	NavTau:     300.0,
	NavGamma:   1.75,
	NavMinDt:   50.0,
	MaxCursorDepth: 2,
	ActiveLiftY:    2,

	BorderTL: '╭',
	BorderTR: '╮',
	BorderBL: '╰',
	BorderBR: '╯',
	BorderH:  '─',
	BorderV:  '│',

	NewFileEditor: "inapp",
	ContinueNameCmd:     "",
	BranchNameCmd:       "",
	AutoSplitOnLink:     true,
	TextWidth:           80,
	LiveReload:          true,
	CardSizeLimit:       true,
	HighlightOverLimit:  true,
	ColorOverLimit:      lipgloss.Color("#3a0000"),
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
	ColorOverLimit lipgloss.Color
}

// Layout groups the display-geometry overrides accepted by ApplyLayout.
type Layout struct {
	StackVisibleCount int
	StackOffsetX      int
	StackOffsetY      int
	ActiveLiftY       int
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
	OpenInApp    []string
	OpenExternal []string
	Continue     []string
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
	OpenBox       []string
}

// DefaultBindings returns the out-of-the-box keybinding set.
func DefaultBindings() KeyBindings {
	return KeyBindings{
		Up:            []string{"k", "up"},
		Down:          []string{"j", "down"},
		Random:        []string{"r"},
		OverlayToggle: []string{"enter"},
		OpenInApp:    []string{"e"},
		OpenExternal: []string{"E"},
		Continue:     []string{"c"},
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
		OpenBox:       []string{"b"},
	}
}

// BoxConfig describes a single note directory and its per-box settings.
type BoxConfig struct {
	Path string
	Free bool // when true, Luhmann c/C are disabled and N prompts for an arbitrary filename
}

// FileCreation holds configuration for the continue/branch file-creation feature.
type FileCreation struct {
	NewFileEditor      string
	ContinueCmd        string
	BranchCmd          string
	AutoSplitOnLink    *bool
	HighlightOverLimit *bool
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
	if c.ColorOverLimit != "" {
		m.settings.ColorOverLimit = c.ColorOverLimit
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
	if l.ActiveLiftY != 0 {
		m.settings.ActiveLiftY = l.ActiveLiftY
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
	override(&m.bindings.OpenBox, b.OpenBox)
}

// ApplyFileCreation overrides file-creation settings with non-zero values.
func (m *Model) ApplyFileCreation(fc FileCreation) {
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
	if fc.HighlightOverLimit != nil {
		m.settings.HighlightOverLimit = *fc.HighlightOverLimit
	}
}

// ApplyNav overrides the maximum navigation step size (positive values only).
// ApplyNav overrides navigation tuning. Zero values are ignored (keep default).
// navMaxStep caps the step size per keypress.
// tau, gamma, minDt control the power-law curve; see NavTau/NavGamma/NavMinDt docs.
func (m *Model) ApplyNav(navMaxStep int, tau, gamma, minDt float64) {
	if navMaxStep > 0 {
		m.settings.NavMaxStep = navMaxStep
	}
	if tau > 0 {
		m.settings.NavTau = tau
	}
	if gamma > 0 {
		m.settings.NavGamma = gamma
	}
	if minDt > 0 {
		m.settings.NavMinDt = minDt
	}
}

// SetPageStep sets the overlay page-scroll step in lines. Values ≤ 0 are
// ignored; the default is half the visible body height.
func (m *Model) SetPageStep(step int) {
	if step > 0 {
		m.overlayPageStep = step
	}
}

// EnableLiveReload enables or disables automatic card-list reload on file-system changes.
func (m *Model) EnableLiveReload(enabled bool) {
	m.settings.LiveReload = enabled
}

// SetExternalEditMode controls whether the e key opens $EDITOR instead of the
// in-app editor. c/C/N always use the in-app split-pane regardless of this setting.
func (m *Model) SetExternalEditMode(enabled bool) {
	m.settings.ExternalEditMode = enabled
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

// EnableFreeMode disables Luhmann c/C and replaces N with an arbitrary filename prompt.
func (m *Model) EnableFreeMode(enabled bool) {
	m.settings.FreeMode = enabled
}

// EnableCardSizeLimit controls whether the editor blocks new lines once the
// content would exceed a single card face. Pass false (or use --no-card-limit)
// to allow unlimited content with editor scrolling.
func (m *Model) EnableCardSizeLimit(enabled bool) {
	m.settings.CardSizeLimit = enabled
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
