package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"github.com/vincent-margiotta/thumbr/internal/notes"
	"github.com/vincent-margiotta/thumbr/internal/ui"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

type config struct {
	NoteRoot       string   `json:"noteRoot" yaml:"noteRoot" toml:"noteRoot"`
	RandomSeed     *int64   `json:"randomSeed" yaml:"randomSeed" toml:"randomSeed"`
	AltScreen      *bool    `json:"altScreen" yaml:"altScreen" toml:"altScreen"`
	ColorMark      string   `json:"colorMark" yaml:"colorMark" toml:"colorMark"`
	ColorMuted     string   `json:"colorMuted" yaml:"colorMuted" toml:"colorMuted"`
	ColorHi        string   `json:"colorHi" yaml:"colorHi" toml:"colorHi"`
	ColorDim       string   `json:"colorDim" yaml:"colorDim" toml:"colorDim"`
	ColorStatusBG  string   `json:"colorStatusBG" yaml:"colorStatusBG" toml:"colorStatusBG"`
	ColorStatusFG  string   `json:"colorStatusFG" yaml:"colorStatusFG" toml:"colorStatusFG"`
	ColorStatusDim string   `json:"colorStatusDim" yaml:"colorStatusDim" toml:"colorStatusDim"`
	PageStep       int      `json:"pageStep" yaml:"pageStep" toml:"pageStep"`
	NavChunkSize   int      `json:"navChunkSize"  yaml:"navChunkSize"  toml:"navChunkSize"`
	NavJitter      float64  `json:"navJitter"     yaml:"navJitter"     toml:"navJitter"`
	StackVisible   int      `json:"stackVisible" yaml:"stackVisible" toml:"stackVisible"`
	StackOffsetX   int      `json:"stackOffsetX" yaml:"stackOffsetX" toml:"stackOffsetX"`
	StackOffsetY   int      `json:"stackOffsetY" yaml:"stackOffsetY" toml:"stackOffsetY"`
	ActiveLiftY    int      `json:"activeLiftY" yaml:"activeLiftY" toml:"activeLiftY"`
	MaxCursorDepth int      `json:"maxCursorDepth" yaml:"maxCursorDepth" toml:"maxCursorDepth"`
	BorderCorner   string   `json:"borderCorner" yaml:"borderCorner" toml:"borderCorner"`
	BorderH        string   `json:"borderH" yaml:"borderH" toml:"borderH"`
	BorderV        string   `json:"borderV" yaml:"borderV" toml:"borderV"`
	BindUp             []string `json:"bindUp" yaml:"bindUp" toml:"bindUp"`
	BindDown           []string `json:"bindDown" yaml:"bindDown" toml:"bindDown"`
	BindRandom         []string `json:"bindRandom" yaml:"bindRandom" toml:"bindRandom"`
	BindOverlay        []string `json:"bindOverlay" yaml:"bindOverlay" toml:"bindOverlay"`
	BindEdit           []string `json:"bindEdit" yaml:"bindEdit" toml:"bindEdit"`
	BindMark           []string `json:"bindMark" yaml:"bindMark" toml:"bindMark"`
	BindFilter         []string `json:"bindFilter" yaml:"bindFilter" toml:"bindFilter"`
	BindHelp           []string `json:"bindHelp" yaml:"bindHelp" toml:"bindHelp"`
	BindQuit           []string `json:"bindQuit" yaml:"bindQuit" toml:"bindQuit"`
	BindPageNext       []string `json:"bindPageNext" yaml:"bindPageNext" toml:"bindPageNext"`
	BindPagePrev       []string `json:"bindPagePrev" yaml:"bindPagePrev" toml:"bindPagePrev"`
	BindOverlayUp      []string `json:"bindOverlayUp" yaml:"bindOverlayUp" toml:"bindOverlayUp"`
	BindOverlayDown    []string `json:"bindOverlayDown" yaml:"bindOverlayDown" toml:"bindOverlayDown"`
	BindReload         []string `json:"bindReload"          yaml:"bindReload"          toml:"bindReload"`
	BindContinue       []string `json:"bindContinue"        yaml:"bindContinue"        toml:"bindContinue"`
	BindBranch         []string `json:"bindBranch"          yaml:"bindBranch"          toml:"bindBranch"`
	BindNextRoot       []string `json:"bindNextRoot"        yaml:"bindNextRoot"        toml:"bindNextRoot"`
	BindSuspendEditor  []string `json:"bindSuspendEditor"   yaml:"bindSuspendEditor"   toml:"bindSuspendEditor"`
	BindNavFirst       []string `json:"bindNavFirst"        yaml:"bindNavFirst"        toml:"bindNavFirst"`
	BindNavLast        []string `json:"bindNavLast"         yaml:"bindNavLast"         toml:"bindNavLast"`
	BindBisectForward  []string `json:"bindBisectForward"   yaml:"bindBisectForward"   toml:"bindBisectForward"`
	BindBisectBackward []string `json:"bindBisectBackward"  yaml:"bindBisectBackward"  toml:"bindBisectBackward"`
	BindChunkForward   []string `json:"bindChunkForward"    yaml:"bindChunkForward"    toml:"bindChunkForward"`
	BindChunkBackward  []string `json:"bindChunkBackward"   yaml:"bindChunkBackward"   toml:"bindChunkBackward"`
	ContinueNameCmd    string   `json:"continueNameCmd"     yaml:"continueNameCmd"     toml:"continueNameCmd"`
	BranchNameCmd      string   `json:"branchNameCmd"       yaml:"branchNameCmd"       toml:"branchNameCmd"`
	NewFileEditor      string   `json:"newFileEditor"       yaml:"newFileEditor"       toml:"newFileEditor"`
	BindInApp          []string `json:"bindInApp"           yaml:"bindInApp"           toml:"bindInApp"`
	BindSwitchPane     []string `json:"bindSwitchPane"      yaml:"bindSwitchPane"      toml:"bindSwitchPane"`
	AutoSplitOnLink    *bool    `json:"autoSplitOnLink"     yaml:"autoSplitOnLink"     toml:"autoSplitOnLink"`
	HighlightOverLimit *bool    `json:"highlightOverLimit"  yaml:"highlightOverLimit"  toml:"highlightOverLimit"`
	TextWidth          *int     `json:"textWidth"      yaml:"textWidth"      toml:"textWidth"`
	LiveReload         *bool    `json:"liveReload"     yaml:"liveReload"     toml:"liveReload"`
	EditMode           string   `json:"editMode"       yaml:"editMode"       toml:"editMode"`
	EnableDebugUI      *bool    `json:"enableDebugUI" yaml:"enableDebugUI" toml:"enableDebugUI"`
	CrashLogPath       string   `json:"crashLogPath" yaml:"crashLogPath" toml:"crashLogPath"`
}

type int64Flag struct {
	value int64
	set   bool
}

func (f *int64Flag) Set(s string) error {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid int value %q: %w", s, err)
	}
	f.value = v
	f.set = true
	return nil
}

func (f *int64Flag) String() string {
	if f.set {
		return fmt.Sprintf("%d", f.value)
	}
	return ""
}

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, fmt.Errorf("read config: %w", err)
	}

	var (
		cfg config
		ext = strings.ToLower(filepath.Ext(path))
	)

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return config{}, fmt.Errorf("parse config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return config{}, fmt.Errorf("parse config: %w", err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return config{}, fmt.Errorf("parse config: %w", err)
		}
	default:
		if err := json.Unmarshal(data, &cfg); err != nil {
			return config{}, fmt.Errorf("parse config (default JSON): %w", err)
		}
	}
	return cfg, nil
}

func findDefaultConfig() string {
	names := []string{"config.json", "config.yaml", "config.yml", "config.toml"}
	dirs := []string{"."}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".thumbr"))
	}

	for _, dir := range dirs {
		for _, name := range names {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}

// orderedBoxFlag appends to a shared []ui.BoxConfig slice so that --box and
// --free-box flags are recorded in the order they appear on the command line.
type orderedBoxFlag struct {
	entries *[]ui.BoxConfig
	free    bool
}

func (f *orderedBoxFlag) String() string { return "" }
func (f *orderedBoxFlag) Set(s string) error {
	*f.entries = append(*f.entries, ui.BoxConfig{Path: s, Free: f.free})
	return nil
}

func main() {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] [path]\n\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		flagSet.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), "\nFor multiple boxes use --box/--free-box flags (all flags must precede positional args).")
		fmt.Fprintln(flag.CommandLine.Output(), "Path defaults to the current directory if omitted.")
	}

	var (
		configPath        string
		showVersion       bool
		randomSeedArg     int64Flag
		altScreenArg      bool
		noAltScreen       bool
		colorMarkArg      string
		colorMutedArg     string
		colorHiArg        string
		colorDimArg       string
		colorStatusBGArg  string
		colorStatusFGArg  string
		colorStatusDimArg string
		pageStepArg       int
		cfgNavChunkSize   int
		cfgNavJitter      float64
		stackVisibleArg   int
		stackOffsetXArg   int
		stackOffsetYArg   int
		activeLiftYArg    int
		maxCursorDepthArg int
		borderCornerArg   string
		borderHArg        string
		borderVArg        string
		textWidthArg      int
		noLiveReloadArg   bool
		externalEditModeArg bool
	)

	flagSet.StringVar(&configPath, "config", "", "path to optional JSON/YAML/TOML config file (fields: noteRoot, randomSeed, altScreen)")
	flagSet.BoolVar(&showVersion, "version", false, "print version and exit")
	flagSet.BoolVar(&showVersion, "v", false, "print version and exit (shorthand)")
	flagSet.Var(&randomSeedArg, "random-seed", "set RNG seed for reproducible random jumps (int64)")
	flagSet.BoolVar(&altScreenArg, "alt-screen", true, "use alternate screen buffer; set to false to stay on the main screen")
	flagSet.BoolVar(&noAltScreen, "no-alt-screen", false, "run without the alternate screen buffer (same as --alt-screen=false)")
	flagSet.StringVar(&colorMarkArg, "color-mark", "", "hex color for marked indicator (e.g., #6CCB5F)")
	flagSet.StringVar(&colorMutedArg, "color-muted", "", "hex color for muted/unmarked headers when marks exist (e.g., #444444)")
	flagSet.StringVar(&colorHiArg, "color-hi", "", "hex color for highlight accents")
	flagSet.StringVar(&colorDimArg, "color-dim", "", "hex color for dimmed accents")
	flagSet.StringVar(&colorStatusBGArg, "color-status-bg", "", "hex color for status bar background")
	flagSet.StringVar(&colorStatusFGArg, "color-status-fg", "", "hex color for status bar foreground")
	flagSet.StringVar(&colorStatusDimArg, "color-status-dim", "", "hex color for status bar muted text")
	flagSet.IntVar(&pageStepArg, "page-step", 0, "override overlay page step (lines); defaults to half the overlay body height")
	flagSet.IntVar(&cfgNavChunkSize, "nav-chunk-size", 0, "base step for J/K chunk jumps (0 = default 7)")
	flagSet.Float64Var(&cfgNavJitter, "nav-jitter", -1, "fractional randomness for bisect and chunk jumps 0..1 (-1 = default 0.15)")
	flagSet.IntVar(&stackVisibleArg, "stack-visible", 0, "number of cards visible in the stack (0 to use default)")
	flagSet.IntVar(&stackOffsetXArg, "stack-offset-x", 0, "horizontal offset between stacked cards (0 to use default)")
	flagSet.IntVar(&stackOffsetYArg, "stack-offset-y", 0, "vertical offset between stacked cards (0 to use default)")
	flagSet.IntVar(&activeLiftYArg, "active-lift-y", 0, "active card vertical lift (0 to use default)")
	flagSet.IntVar(&maxCursorDepthArg, "max-cursor-depth", 0, "max depth the active card can sit in the stack (0 to use default)")
	flagSet.StringVar(&borderCornerArg, "border-corner", "", "single rune for card corners (default '+')")
	flagSet.StringVar(&borderHArg, "border-h", "", "single rune for horizontal card borders (default '-')")
	flagSet.StringVar(&borderVArg, "border-v", "", "single rune for vertical card borders (default '|')")
	flagSet.IntVar(&textWidthArg, "text-width", -1, "hard-wrap column for the in-app editor (0 to disable, default 80)")
	flagSet.BoolVar(&noLiveReloadArg, "no-live-reload", false, "disable automatic card-list reload when notes change on disk")
	flagSet.BoolVar(&externalEditModeArg, "external-edit", false, "make e open $EDITOR instead of the in-app editor")
	enableDebugUIArg := flagSet.Bool("enable-debug-ui", false, "enable in-app debug overlay (default disabled)")
	crashLogPathArg := flagSet.String("crash-log", "", "path to write crash log on panic (default ~/.thumbr/crash.log)")
	freeModeArg := flagSet.Bool("free", false, "free mode for all boxes: c/C disabled, N prompts for arbitrary filename")
	noCardLimitArg := flagSet.Bool("no-card-limit", false, "allow card content to exceed one card face (enables editor scrolling)")
	var orderedBoxes []ui.BoxConfig
	flagSet.Var(&orderedBoxFlag{entries: &orderedBoxes, free: false}, "box", "add a note directory as a normal-mode box (repeatable)")
	flagSet.Var(&orderedBoxFlag{entries: &orderedBoxes, free: true}, "free-box", "add a note directory as a free-mode box (repeatable)")
	// Keybinding overrides (comma-separated lists)
	var (
		bindUpArg            string
		bindDownArg          string
		bindRandomArg        string
		bindOverlayArg       string
		bindEditArg          string
		bindMarkArg          string
		bindFilterArg        string
		bindHelpArg          string
		bindQuitArg          string
		bindPageNextArg      string
		bindPagePrevArg      string
		bindOverlayUpArg     string
		bindOverlayDnArg     string
		bindReloadArg        string
		bindContinueArg      string
		bindBranchArg        string
		bindNextRootArg      string
		bindSuspendEditorArg string
		bindInAppArg         string
		bindNavFirstArg       string
		bindNavLastArg        string
		bindBisectForwardArg  string
		bindBisectBackwardArg string
		bindChunkForwardArg   string
		bindChunkBackwardArg  string
		bindSwitchPaneArg     string
	)
	var (
		continueNameCmdArg string
		branchNameCmdArg   string
		newFileEditorArg   string
	)
	flagSet.StringVar(&bindUpArg, "bind-up", "", "comma-separated keys for up navigation")
	flagSet.StringVar(&bindDownArg, "bind-down", "", "comma-separated keys for down navigation")
	flagSet.StringVar(&bindRandomArg, "bind-random", "", "comma-separated keys for random jump")
	flagSet.StringVar(&bindOverlayArg, "bind-overlay", "", "comma-separated keys to toggle overlay")
	flagSet.StringVar(&bindEditArg, "bind-edit", "", "comma-separated keys to open the current card in the editor")
	flagSet.StringVar(&bindMarkArg, "bind-mark", "", "comma-separated keys to mark/unmark")
	flagSet.StringVar(&bindFilterArg, "bind-filter", "", "comma-separated keys to toggle marked filter")
	flagSet.StringVar(&bindHelpArg, "bind-help", "", "comma-separated keys to toggle help")
	flagSet.StringVar(&bindQuitArg, "bind-quit", "", "comma-separated keys to quit/back")
	flagSet.StringVar(&bindPageNextArg, "bind-page-next", "", "comma-separated keys for next page")
	flagSet.StringVar(&bindPagePrevArg, "bind-page-prev", "", "comma-separated keys for previous page")
	flagSet.StringVar(&bindOverlayUpArg, "bind-overlay-up", "", "comma-separated keys to scroll overlay up")
	flagSet.StringVar(&bindOverlayDnArg, "bind-overlay-down", "", "comma-separated keys to scroll overlay down")
	flagSet.StringVar(&bindReloadArg, "bind-reload", "", "comma-separated keys to reload the current box")
	flagSet.StringVar(&bindContinueArg, "bind-continue", "", "comma-separated keys for Luhmann continue")
	flagSet.StringVar(&bindBranchArg, "bind-branch", "", "comma-separated keys for Luhmann branch")
	flagSet.StringVar(&bindNextRootArg, "bind-next-root", "", "comma-separated keys to create the next integer root card")
	flagSet.StringVar(&bindSuspendEditorArg, "bind-suspend-editor", "", "comma-separated keys to suspend the in-app editor and return to browse")
	flagSet.StringVar(&bindNavFirstArg, "bind-nav-first", "", "comma-separated keys to jump to the first card")
	flagSet.StringVar(&bindNavLastArg, "bind-nav-last", "", "comma-separated keys to jump to the last card")
	flagSet.StringVar(&bindBisectForwardArg, "bind-bisect-forward", "", "comma-separated keys to bisect toward the end of the deck")
	flagSet.StringVar(&bindBisectBackwardArg, "bind-bisect-backward", "", "comma-separated keys to bisect toward the start of the deck")
	flagSet.StringVar(&bindChunkForwardArg, "bind-chunk-forward", "", "comma-separated keys to jump forward by a chunk (default J)")
	flagSet.StringVar(&bindChunkBackwardArg, "bind-chunk-backward", "", "comma-separated keys to jump backward by a chunk (default K)")
	flagSet.StringVar(&continueNameCmdArg, "continue-name-cmd", "", "shell command to derive continuation filename stem")
	flagSet.StringVar(&branchNameCmdArg, "branch-name-cmd", "", "shell command to derive branch filename stem")
	flagSet.StringVar(&newFileEditorArg, "new-file-editor", "", "editor to open after c/C creates a file: inapp, external, or none (default inapp)")
	flagSet.StringVar(&bindInAppArg, "bind-inapp", "", "comma-separated keys to open card in in-app editor")
	flagSet.StringVar(&bindSwitchPaneArg, "bind-switch-pane", "", "comma-separated keys to switch focus between split editor panes")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	if showVersion {
		fmt.Println(version)
		return
	}

	opts := struct {
		noteRoot           string
		randomSeed         int64
		useAlternateScreen bool
		colorMark          string
		colorMuted         string
		colorHi            string
		colorDim           string
		colorStatusBG      string
		colorStatusFG      string
		colorStatusDim     string
		pageStep           int
		navChunkSize       int
		navJitter          float64
		bindings           ui.KeyBindings
		stackVisible       int
		stackOffsetX       int
		stackOffsetY       int
		activeLiftY        int
		maxCursorDepth     int
		borderCorner       string
		borderH            string
		borderV            string
		enableDebugUI      bool
		crashLogPath       string
		continueNameCmd    string
		branchNameCmd      string
		newFileEditor      string
		autoSplitOnLink    bool
		highlightOverLimit bool
		textWidth          int
		liveReload         bool
		externalEditMode   bool
	}{
		noteRoot:           ".",
		randomSeed:         time.Now().UnixNano(),
		useAlternateScreen: true,
		colorMark:          "",
		colorMuted:         "",
		colorHi:            "",
		colorDim:           "",
		colorStatusBG:      "",
		colorStatusFG:      "",
		colorStatusDim:     "",
		pageStep:           0,
		navChunkSize:       0,
		navJitter:          -1, // -1 means "use default from Settings"
		bindings:           ui.DefaultBindings(),
		stackVisible:       0,
		stackOffsetX:       0,
		stackOffsetY:       0,
		activeLiftY:        0,
		maxCursorDepth:     0,
		borderCorner:       "",
		borderH:            "",
		borderV:            "",
		enableDebugUI:      false,
		crashLogPath:       "",
		continueNameCmd:    "",
		branchNameCmd:      "",
		newFileEditor:      "",
		autoSplitOnLink:    ui.DefaultSettings.AutoSplitOnLink,
		highlightOverLimit: ui.DefaultSettings.HighlightOverLimit,
		textWidth:          ui.DefaultSettings.TextWidth,
		liveReload:         ui.DefaultSettings.LiveReload,
	}

	if configPath == "" {
		configPath = findDefaultConfig()
	}

	if configPath != "" {
		cfg, err := loadConfig(configPath)
		if err != nil {
			log.Fatalf("error loading config: %v", err)
		}
		if cfg.NoteRoot != "" {
			opts.noteRoot = cfg.NoteRoot
		}
		if cfg.RandomSeed != nil {
			opts.randomSeed = *cfg.RandomSeed
		}
		if cfg.AltScreen != nil {
			opts.useAlternateScreen = *cfg.AltScreen
		}
		if cfg.ColorMark != "" {
			opts.colorMark = cfg.ColorMark
		}
		if cfg.ColorMuted != "" {
			opts.colorMuted = cfg.ColorMuted
		}
		if cfg.ColorHi != "" {
			opts.colorHi = cfg.ColorHi
		}
		if cfg.ColorDim != "" {
			opts.colorDim = cfg.ColorDim
		}
		if cfg.ColorStatusBG != "" {
			opts.colorStatusBG = cfg.ColorStatusBG
		}
		if cfg.ColorStatusFG != "" {
			opts.colorStatusFG = cfg.ColorStatusFG
		}
		if cfg.ColorStatusDim != "" {
			opts.colorStatusDim = cfg.ColorStatusDim
		}
		if cfg.PageStep > 0 {
			opts.pageStep = cfg.PageStep
		}
		if cfg.MaxCursorDepth > 0 {
			opts.maxCursorDepth = cfg.MaxCursorDepth
		}
		if cfg.BorderCorner != "" {
			opts.borderCorner = cfg.BorderCorner
		}
		if cfg.BorderH != "" {
			opts.borderH = cfg.BorderH
		}
		if cfg.BorderV != "" {
			opts.borderV = cfg.BorderV
		}
		if cfg.EnableDebugUI != nil {
			opts.enableDebugUI = *cfg.EnableDebugUI
		}
		if cfg.CrashLogPath != "" {
			opts.crashLogPath = cfg.CrashLogPath
		}
		if cfg.StackVisible > 0 {
			opts.stackVisible = cfg.StackVisible
		}
		if cfg.StackOffsetX != 0 {
			opts.stackOffsetX = cfg.StackOffsetX
		}
		if cfg.StackOffsetY != 0 {
			opts.stackOffsetY = cfg.StackOffsetY
		}
		if cfg.ActiveLiftY != 0 {
			opts.activeLiftY = cfg.ActiveLiftY
		}
		mergeBinding := func(dst *[]string, src []string) {
			if len(src) > 0 {
				*dst = src
			}
		}
		mergeBinding(&opts.bindings.Up, cfg.BindUp)
		mergeBinding(&opts.bindings.Down, cfg.BindDown)
		mergeBinding(&opts.bindings.Random, cfg.BindRandom)
		mergeBinding(&opts.bindings.OverlayToggle, cfg.BindOverlay)
		mergeBinding(&opts.bindings.OpenExternal, cfg.BindEdit)
		mergeBinding(&opts.bindings.OpenInApp, cfg.BindInApp)
		mergeBinding(&opts.bindings.Mark, cfg.BindMark)
		mergeBinding(&opts.bindings.Filter, cfg.BindFilter)
		mergeBinding(&opts.bindings.Help, cfg.BindHelp)
		mergeBinding(&opts.bindings.Quit, cfg.BindQuit)
		mergeBinding(&opts.bindings.PageNext, cfg.BindPageNext)
		mergeBinding(&opts.bindings.PagePrev, cfg.BindPagePrev)
		mergeBinding(&opts.bindings.OverlayUp, cfg.BindOverlayUp)
		mergeBinding(&opts.bindings.OverlayDown, cfg.BindOverlayDown)
		mergeBinding(&opts.bindings.Reload, cfg.BindReload)
		mergeBinding(&opts.bindings.Continue, cfg.BindContinue)
		mergeBinding(&opts.bindings.Branch, cfg.BindBranch)
		mergeBinding(&opts.bindings.NextRoot, cfg.BindNextRoot)
		mergeBinding(&opts.bindings.SuspendEditor, cfg.BindSuspendEditor)
		mergeBinding(&opts.bindings.NavFirst, cfg.BindNavFirst)
		mergeBinding(&opts.bindings.NavLast, cfg.BindNavLast)
		mergeBinding(&opts.bindings.BisectForward, cfg.BindBisectForward)
		mergeBinding(&opts.bindings.BisectBackward, cfg.BindBisectBackward)
		mergeBinding(&opts.bindings.ChunkForward, cfg.BindChunkForward)
		mergeBinding(&opts.bindings.ChunkBackward, cfg.BindChunkBackward)
		mergeBinding(&opts.bindings.SwitchPane, cfg.BindSwitchPane)
		if cfg.NavChunkSize > 0 {
			opts.navChunkSize = cfg.NavChunkSize
		}
		if cfg.NavJitter >= 0 {
			opts.navJitter = cfg.NavJitter
		}
		if cfg.AutoSplitOnLink != nil {
			opts.autoSplitOnLink = *cfg.AutoSplitOnLink
		}
		if cfg.HighlightOverLimit != nil {
			opts.highlightOverLimit = *cfg.HighlightOverLimit
		}
		if cfg.TextWidth != nil {
			opts.textWidth = *cfg.TextWidth
		}
		if cfg.LiveReload != nil {
			opts.liveReload = *cfg.LiveReload
		}
		if cfg.EditMode == "external" {
			opts.externalEditMode = true
		}
		if cfg.ContinueNameCmd != "" {
			opts.continueNameCmd = cfg.ContinueNameCmd
		}
		if cfg.BranchNameCmd != "" {
			opts.branchNameCmd = cfg.BranchNameCmd
		}
		if cfg.NewFileEditor != "" {
			opts.newFileEditor = cfg.NewFileEditor
		}
	}

	// Flags override config/defaults.
	if randomSeedArg.set {
		opts.randomSeed = randomSeedArg.value
	}
	if noAltScreen {
		opts.useAlternateScreen = false
	} else {
		opts.useAlternateScreen = altScreenArg
	}
	if colorMarkArg != "" {
		opts.colorMark = colorMarkArg
	}
	if colorMutedArg != "" {
		opts.colorMuted = colorMutedArg
	}
	if colorHiArg != "" {
		opts.colorHi = colorHiArg
	}
	if colorDimArg != "" {
		opts.colorDim = colorDimArg
	}
	if colorStatusBGArg != "" {
		opts.colorStatusBG = colorStatusBGArg
	}
	if colorStatusFGArg != "" {
		opts.colorStatusFG = colorStatusFGArg
	}
	if colorStatusDimArg != "" {
		opts.colorStatusDim = colorStatusDimArg
	}
	if textWidthArg >= 0 {
		opts.textWidth = textWidthArg
	}
	if noLiveReloadArg {
		opts.liveReload = false
	}
	if externalEditModeArg {
		opts.externalEditMode = true
	}
	if pageStepArg > 0 {
		opts.pageStep = pageStepArg
	}
	if maxCursorDepthArg > 0 {
		opts.maxCursorDepth = maxCursorDepthArg
	}
	if borderCornerArg != "" {
		opts.borderCorner = borderCornerArg
	}
	if borderHArg != "" {
		opts.borderH = borderHArg
	}
	if borderVArg != "" {
		opts.borderV = borderVArg
	}
	if stackVisibleArg > 0 {
		opts.stackVisible = stackVisibleArg
	}
	if stackOffsetXArg != 0 {
		opts.stackOffsetX = stackOffsetXArg
	}
	if stackOffsetYArg != 0 {
		opts.stackOffsetY = stackOffsetYArg
	}
	if activeLiftYArg != 0 {
		opts.activeLiftY = activeLiftYArg
	}
	opts.enableDebugUI = *enableDebugUIArg
	if crashLogPathArg != nil && *crashLogPathArg != "" {
		opts.crashLogPath = *crashLogPathArg
	}
	parseBinding := func(arg string) []string {
		if arg == "" {
			return nil
		}
		parts := strings.Split(arg, ",")
		var out []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	if v := parseBinding(bindUpArg); len(v) > 0 {
		opts.bindings.Up = v
	}
	if v := parseBinding(bindDownArg); len(v) > 0 {
		opts.bindings.Down = v
	}
	if v := parseBinding(bindRandomArg); len(v) > 0 {
		opts.bindings.Random = v
	}
	if v := parseBinding(bindOverlayArg); len(v) > 0 {
		opts.bindings.OverlayToggle = v
	}
	if v := parseBinding(bindEditArg); len(v) > 0 {
		opts.bindings.OpenExternal = v
	}
	if v := parseBinding(bindInAppArg); len(v) > 0 {
		opts.bindings.OpenInApp = v
	}
	if v := parseBinding(bindMarkArg); len(v) > 0 {
		opts.bindings.Mark = v
	}
	if v := parseBinding(bindFilterArg); len(v) > 0 {
		opts.bindings.Filter = v
	}
	if v := parseBinding(bindHelpArg); len(v) > 0 {
		opts.bindings.Help = v
	}
	if v := parseBinding(bindQuitArg); len(v) > 0 {
		opts.bindings.Quit = v
	}
	if v := parseBinding(bindPageNextArg); len(v) > 0 {
		opts.bindings.PageNext = v
	}
	if v := parseBinding(bindPagePrevArg); len(v) > 0 {
		opts.bindings.PagePrev = v
	}
	if v := parseBinding(bindOverlayUpArg); len(v) > 0 {
		opts.bindings.OverlayUp = v
	}
	if v := parseBinding(bindOverlayDnArg); len(v) > 0 {
		opts.bindings.OverlayDown = v
	}
	if v := parseBinding(bindReloadArg); len(v) > 0 {
		opts.bindings.Reload = v
	}
	if v := parseBinding(bindContinueArg); len(v) > 0 {
		opts.bindings.Continue = v
	}
	if v := parseBinding(bindBranchArg); len(v) > 0 {
		opts.bindings.Branch = v
	}
	if v := parseBinding(bindNextRootArg); len(v) > 0 {
		opts.bindings.NextRoot = v
	}
	if v := parseBinding(bindSuspendEditorArg); len(v) > 0 {
		opts.bindings.SuspendEditor = v
	}
	if v := parseBinding(bindNavFirstArg); len(v) > 0 {
		opts.bindings.NavFirst = v
	}
	if v := parseBinding(bindNavLastArg); len(v) > 0 {
		opts.bindings.NavLast = v
	}
	if v := parseBinding(bindBisectForwardArg); len(v) > 0 {
		opts.bindings.BisectForward = v
	}
	if v := parseBinding(bindBisectBackwardArg); len(v) > 0 {
		opts.bindings.BisectBackward = v
	}
	if v := parseBinding(bindChunkForwardArg); len(v) > 0 {
		opts.bindings.ChunkForward = v
	}
	if v := parseBinding(bindChunkBackwardArg); len(v) > 0 {
		opts.bindings.ChunkBackward = v
	}
	if cfgNavChunkSize > 0 {
		opts.navChunkSize = cfgNavChunkSize
	}
	if cfgNavJitter >= 0 {
		opts.navJitter = cfgNavJitter
	}
	if v := parseBinding(bindSwitchPaneArg); len(v) > 0 {
		opts.bindings.SwitchPane = v
	}
	if continueNameCmdArg != "" {
		opts.continueNameCmd = continueNameCmdArg
	}
	if branchNameCmdArg != "" {
		opts.branchNameCmd = branchNameCmdArg
	}
	if newFileEditorArg != "" {
		opts.newFileEditor = newFileEditorArg
	}

	// Build the ordered box list.
	// --box / --free-box flags take precedence and preserve argument order;
	// positional arg is a fallback for the simple single-box case.
	var boxConfigs []ui.BoxConfig
	if len(orderedBoxes) > 0 {
		for _, b := range orderedBoxes {
			boxConfigs = append(boxConfigs, ui.BoxConfig{Path: b.Path, Free: b.Free || *freeModeArg})
		}
	} else {
		path := opts.noteRoot
		if args := flagSet.Args(); len(args) > 0 {
			path = args[0]
		}
		boxConfigs = []ui.BoxConfig{{Path: path, Free: *freeModeArg}}
	}
	opts.noteRoot = boxConfigs[0].Path

	// Seed RNG once for the whole process (random card jumps).
	rng := rand.New(rand.NewSource(opts.randomSeed))

	loadOpts := notes.LoadOptions{}

	var loadTimings notes.LoadTimings
	loadOpts.Timings = &loadTimings
	loadStart := time.Now()
	cards, err := notes.LoadCardsFromDir(opts.noteRoot, loadOpts)
	if err != nil {
		log.Printf("error loading cards from %s: %v", opts.noteRoot, err)
	}
	loadDuration := time.Since(loadStart)

	colors := ui.Colors{
		ColorMarkFG:    ui.DefaultSettings.ColorMarkFG,
		ColorMutFG:     ui.DefaultSettings.ColorMutFG,
		ColorHiFG:      ui.DefaultSettings.ColorHiFG,
		ColorDimFG:     ui.DefaultSettings.ColorDimFG,
		ColorStatusBG:  ui.DefaultSettings.ColorStatusBG,
		ColorStatusFG:  ui.DefaultSettings.ColorStatusFG,
		ColorStatusDim: ui.DefaultSettings.ColorStatusDim,
	}
	if opts.colorMark != "" {
		colors.ColorMarkFG = lipgloss.Color(opts.colorMark)
	}
	if opts.colorMuted != "" {
		colors.ColorMutFG = lipgloss.Color(opts.colorMuted)
	}
	if opts.colorHi != "" {
		colors.ColorHiFG = lipgloss.Color(opts.colorHi)
	}
	if opts.colorDim != "" {
		colors.ColorDimFG = lipgloss.Color(opts.colorDim)
	}
	if opts.colorStatusBG != "" {
		colors.ColorStatusBG = lipgloss.Color(opts.colorStatusBG)
	}
	if opts.colorStatusFG != "" {
		colors.ColorStatusFG = lipgloss.Color(opts.colorStatusFG)
	}
	if opts.colorStatusDim != "" {
		colors.ColorStatusDim = lipgloss.Color(opts.colorStatusDim)
	}

	m := ui.NewModel(cards, opts.noteRoot, loadOpts, rng)
	m.SetLoadDuration(loadDuration)
	m.SetLoadBreakdown(loadTimings.Walk, loadTimings.Sort)
	m.ApplyColors(colors)
	m.ApplyBindings(opts.bindings)
	if opts.pageStep > 0 {
		m.SetPageStep(opts.pageStep)
	}
	if opts.navChunkSize > 0 || opts.navJitter >= 0 {
		m.ApplyNavChunk(opts.navChunkSize, opts.navJitter)
	}
	layout := ui.Layout{
		StackVisibleCount: opts.stackVisible,
		StackOffsetX:      opts.stackOffsetX,
		StackOffsetY:      opts.stackOffsetY,
		ActiveLiftY:       opts.activeLiftY,
		MaxCursorDepth:    opts.maxCursorDepth,
	}
	if opts.borderCorner != "" {
		layout.BorderCorner = []rune(opts.borderCorner)[0]
	}
	if opts.borderH != "" {
		layout.BorderH = []rune(opts.borderH)[0]
	}
	if opts.borderV != "" {
		layout.BorderV = []rune(opts.borderV)[0]
	}
	m.ApplyLayout(layout)
	fc := ui.FileCreation{
		NewFileEditor:      opts.newFileEditor,
		ContinueCmd:        opts.continueNameCmd,
		BranchCmd:          opts.branchNameCmd,
		AutoSplitOnLink:    &opts.autoSplitOnLink,
		HighlightOverLimit: &opts.highlightOverLimit,
	}
	m.ApplyFileCreation(fc)
	m.SetTextWidth(opts.textWidth)
	m.EnableLiveReload(opts.liveReload)
	m.SetExternalEditMode(opts.externalEditMode)
	m.EnableDebugUI(opts.enableDebugUI)
	m.SetBoxes(boxConfigs)
	if *noCardLimitArg {
		m.EnableCardSizeLimit(false)
	}

	programOptions := []tea.ProgramOption{}
	if opts.useAlternateScreen {
		// Pre-clear the main screen buffer before entering alt-screen so that
		// the first external-editor launch doesn't flash old terminal history.
		fmt.Print("\033[2J\033[H")
		programOptions = append(programOptions, tea.WithAltScreen())
	}

	p := tea.NewProgram(m, programOptions...)
	runWithCrashLog(p, opts.crashLogPath)
}

func defaultCrashLogPath() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".thumbr", "crash.log")
	}
	return "thumbr-crash.log"
}

// runWithCrashLog runs the program and writes a crash log if a panic occurs.
func runWithCrashLog(p *tea.Program, path string) {
	if path == "" {
		path = defaultCrashLogPath()
	}
	defer func() {
		if r := recover(); r != nil {
			_ = os.MkdirAll(filepath.Dir(path), 0o755)
			msg := fmt.Sprintf("panic: %v\n%s", r, debug.Stack())
			_ = os.WriteFile(path, []byte(msg), 0o644)
			panic(r)
		}
	}()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
