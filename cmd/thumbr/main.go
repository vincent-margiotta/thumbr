package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"thumbr/internal/notes"
	"thumbr/internal/ui"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

type config struct {
	NoteRoot         string   `json:"noteRoot" yaml:"noteRoot" toml:"noteRoot"`
	RandomSeed       *int64   `json:"randomSeed" yaml:"randomSeed" toml:"randomSeed"`
	AltScreen        *bool    `json:"altScreen" yaml:"altScreen" toml:"altScreen"`
	IncludeExts      []string `json:"includeExts" yaml:"includeExts" toml:"includeExts"`
	IgnoreGlobs      []string `json:"ignoreGlobs" yaml:"ignoreGlobs" toml:"ignoreGlobs"`
	ColorMark        string   `json:"colorMark" yaml:"colorMark" toml:"colorMark"`
	ColorMuted       string   `json:"colorMuted" yaml:"colorMuted" toml:"colorMuted"`
	ColorHi          string   `json:"colorHi" yaml:"colorHi" toml:"colorHi"`
	ColorDim         string   `json:"colorDim" yaml:"colorDim" toml:"colorDim"`
	ColorStatusBG    string   `json:"colorStatusBG" yaml:"colorStatusBG" toml:"colorStatusBG"`
	ColorStatusFG    string   `json:"colorStatusFG" yaml:"colorStatusFG" toml:"colorStatusFG"`
	ColorStatusDim   string   `json:"colorStatusDim" yaml:"colorStatusDim" toml:"colorStatusDim"`
	PageStep         int      `json:"pageStep" yaml:"pageStep" toml:"pageStep"`
	NavAccelMs       int      `json:"navAccelMs" yaml:"navAccelMs" toml:"navAccelMs"`
	NavMaxStep       int      `json:"navMaxStep" yaml:"navMaxStep" toml:"navMaxStep"`
	StackVisible     int      `json:"stackVisible" yaml:"stackVisible" toml:"stackVisible"`
	StackOffsetX     int      `json:"stackOffsetX" yaml:"stackOffsetX" toml:"stackOffsetX"`
	StackOffsetY     int      `json:"stackOffsetY" yaml:"stackOffsetY" toml:"stackOffsetY"`
	CardWidthFrac    float64  `json:"cardWidthFrac" yaml:"cardWidthFrac" toml:"cardWidthFrac"`
	CardHeightFrac   float64  `json:"cardHeightFrac" yaml:"cardHeightFrac" toml:"cardHeightFrac"`
	ActiveLiftY      int      `json:"activeLiftY" yaml:"activeLiftY" toml:"activeLiftY"`
	StickyOverlayNav *bool    `json:"stickyOverlayNav" yaml:"stickyOverlayNav" toml:"stickyOverlayNav"`
	BindUp           []string `json:"bindUp" yaml:"bindUp" toml:"bindUp"`
	BindDown         []string `json:"bindDown" yaml:"bindDown" toml:"bindDown"`
	BindRandom       []string `json:"bindRandom" yaml:"bindRandom" toml:"bindRandom"`
	BindOverlay      []string `json:"bindOverlay" yaml:"bindOverlay" toml:"bindOverlay"`
	BindMark         []string `json:"bindMark" yaml:"bindMark" toml:"bindMark"`
	BindFilter       []string `json:"bindFilter" yaml:"bindFilter" toml:"bindFilter"`
	BindHelp         []string `json:"bindHelp" yaml:"bindHelp" toml:"bindHelp"`
	BindQuit         []string `json:"bindQuit" yaml:"bindQuit" toml:"bindQuit"`
	BindPageNext     []string `json:"bindPageNext" yaml:"bindPageNext" toml:"bindPageNext"`
	BindPagePrev     []string `json:"bindPagePrev" yaml:"bindPagePrev" toml:"bindPagePrev"`
	BindOverlayUp    []string `json:"bindOverlayUp" yaml:"bindOverlayUp" toml:"bindOverlayUp"`
	BindOverlayDown  []string `json:"bindOverlayDown" yaml:"bindOverlayDown" toml:"bindOverlayDown"`
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

func main() {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] [path]\n\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		flagSet.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), "\nPath defaults to the current directory if omitted.")
	}

	var (
		configPath          string
		showVersion         bool
		randomSeedArg       int64Flag
		altScreenArg        bool
		noAltScreen         bool
		includeExtsArg      string
		ignoreGlobsArg      string
		colorMarkArg        string
		colorMutedArg       string
		colorHiArg          string
		colorDimArg         string
		colorStatusBGArg    string
		colorStatusFGArg    string
		colorStatusDimArg   string
		pageStepArg         int
		cfgNavAccelMs       int
		cfgNavMaxStep       int
		stackVisibleArg     int
		stackOffsetXArg     int
		stackOffsetYArg     int
		cardWidthFracArg    float64
		cardHeightFracArg   float64
		activeLiftYArg      int
		stickyOverlayNavArg bool
	)

	flagSet.StringVar(&configPath, "config", "", "path to optional JSON/YAML/TOML config file (fields: noteRoot, randomSeed, altScreen, includeExts, ignoreGlobs)")
	flagSet.BoolVar(&showVersion, "version", false, "print version and exit")
	flagSet.BoolVar(&showVersion, "v", false, "print version and exit (shorthand)")
	flagSet.Var(&randomSeedArg, "random-seed", "set RNG seed for reproducible random jumps (int64)")
	flagSet.BoolVar(&altScreenArg, "alt-screen", true, "use alternate screen buffer; set to false to stay on the main screen")
	flagSet.BoolVar(&noAltScreen, "no-alt-screen", false, "run without the alternate screen buffer (same as --alt-screen=false)")
	flagSet.StringVar(&includeExtsArg, "include-exts", "", "comma-separated list of file extensions to include (e.g., .md,.txt)")
	flagSet.StringVar(&ignoreGlobsArg, "ignore", "", "comma-separated list of glob patterns to skip (matched on full path and basename)")
	flagSet.StringVar(&colorMarkArg, "color-mark", "", "hex color for marked indicator (e.g., #6CCB5F)")
	flagSet.StringVar(&colorMutedArg, "color-muted", "", "hex color for muted/unmarked headers when marks exist (e.g., #444444)")
	flagSet.StringVar(&colorHiArg, "color-hi", "", "hex color for highlight accents")
	flagSet.StringVar(&colorDimArg, "color-dim", "", "hex color for dimmed accents")
	flagSet.StringVar(&colorStatusBGArg, "color-status-bg", "", "hex color for status bar background")
	flagSet.StringVar(&colorStatusFGArg, "color-status-fg", "", "hex color for status bar foreground")
	flagSet.StringVar(&colorStatusDimArg, "color-status-dim", "", "hex color for status bar muted text")
	flagSet.IntVar(&pageStepArg, "page-step", 0, "override overlay page step (lines); defaults to half the overlay body height")
	flagSet.IntVar(&cfgNavAccelMs, "nav-accel-ms", 0, "navigation acceleration window in milliseconds (0 to use default)")
	flagSet.IntVar(&cfgNavMaxStep, "nav-max-step", 0, "maximum navigation step when accelerating (0 to use default)")
	flagSet.IntVar(&stackVisibleArg, "stack-visible", 0, "number of cards visible in the stack (0 to use default)")
	flagSet.IntVar(&stackOffsetXArg, "stack-offset-x", 0, "horizontal offset between stacked cards (0 to use default)")
	flagSet.IntVar(&stackOffsetYArg, "stack-offset-y", 0, "vertical offset between stacked cards (0 to use default)")
	flagSet.Float64Var(&cardWidthFracArg, "card-width-frac", 0, "card width fraction of viewport (0 to use default)")
	flagSet.Float64Var(&cardHeightFracArg, "card-height-frac", 0, "card height fraction of viewport (0 to use default)")
	flagSet.IntVar(&activeLiftYArg, "active-lift-y", 0, "active card vertical lift (0 to use default)")
	flagSet.BoolVar(&stickyOverlayNavArg, "sticky-overlay-nav", ui.DefaultSettings.StickyOverlayNav, "allow overlay to retain nav keys without exiting")
	// Keybinding overrides (comma-separated lists)
	var (
		bindUpArg        string
		bindDownArg      string
		bindRandomArg    string
		bindOverlayArg   string
		bindMarkArg      string
		bindFilterArg    string
		bindHelpArg      string
		bindQuitArg      string
		bindPageNextArg  string
		bindPagePrevArg  string
		bindOverlayUpArg string
		bindOverlayDnArg string
	)
	flagSet.StringVar(&bindUpArg, "bind-up", "", "comma-separated keys for up navigation")
	flagSet.StringVar(&bindDownArg, "bind-down", "", "comma-separated keys for down navigation")
	flagSet.StringVar(&bindRandomArg, "bind-random", "", "comma-separated keys for random jump")
	flagSet.StringVar(&bindOverlayArg, "bind-overlay", "", "comma-separated keys to toggle overlay")
	flagSet.StringVar(&bindMarkArg, "bind-mark", "", "comma-separated keys to mark/unmark")
	flagSet.StringVar(&bindFilterArg, "bind-filter", "", "comma-separated keys to toggle marked filter")
	flagSet.StringVar(&bindHelpArg, "bind-help", "", "comma-separated keys to toggle help")
	flagSet.StringVar(&bindQuitArg, "bind-quit", "", "comma-separated keys to quit/back")
	flagSet.StringVar(&bindPageNextArg, "bind-page-next", "", "comma-separated keys for next page")
	flagSet.StringVar(&bindPagePrevArg, "bind-page-prev", "", "comma-separated keys for previous page")
	flagSet.StringVar(&bindOverlayUpArg, "bind-overlay-up", "", "comma-separated keys to scroll overlay up")
	flagSet.StringVar(&bindOverlayDnArg, "bind-overlay-down", "", "comma-separated keys to scroll overlay down")

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
		includeExts        []string
		ignoreGlobs        []string
		colorMark          string
		colorMuted         string
		colorHi            string
		colorDim           string
		colorStatusBG      string
		colorStatusFG      string
		colorStatusDim     string
		pageStep           int
		navAccelMs         int
		navMaxStep         int
		bindings           ui.KeyBindings
		stackVisible       int
		stackOffsetX       int
		stackOffsetY       int
		cardWidthFrac      float64
		cardHeightFrac     float64
		activeLiftY        int
		stickyOverlayNav   bool
	}{
		noteRoot:           ".",
		randomSeed:         time.Now().UnixNano(),
		useAlternateScreen: true,
		includeExts:        nil,
		ignoreGlobs:        nil,
		colorMark:          "",
		colorMuted:         "",
		colorHi:            "",
		colorDim:           "",
		colorStatusBG:      "",
		colorStatusFG:      "",
		colorStatusDim:     "",
		pageStep:           0,
		navAccelMs:         0,
		navMaxStep:         0,
		bindings:           ui.DefaultBindings(),
		stackVisible:       0,
		stackOffsetX:       0,
		stackOffsetY:       0,
		cardWidthFrac:      0,
		cardHeightFrac:     0,
		activeLiftY:        0,
		stickyOverlayNav:   ui.DefaultSettings.StickyOverlayNav,
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
		if len(cfg.IncludeExts) > 0 {
			opts.includeExts = cfg.IncludeExts
		}
		if len(cfg.IgnoreGlobs) > 0 {
			opts.ignoreGlobs = cfg.IgnoreGlobs
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
		if cfg.NavAccelMs > 0 {
			opts.navAccelMs = cfg.NavAccelMs
		}
		if cfg.NavMaxStep > 0 {
			opts.navMaxStep = cfg.NavMaxStep
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
		if cfg.CardWidthFrac > 0 {
			opts.cardWidthFrac = cfg.CardWidthFrac
		}
		if cfg.CardHeightFrac > 0 {
			opts.cardHeightFrac = cfg.CardHeightFrac
		}
		if cfg.ActiveLiftY != 0 {
			opts.activeLiftY = cfg.ActiveLiftY
		}
		if cfg.StickyOverlayNav != nil {
			opts.stickyOverlayNav = *cfg.StickyOverlayNav
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
		mergeBinding(&opts.bindings.Mark, cfg.BindMark)
		mergeBinding(&opts.bindings.Filter, cfg.BindFilter)
		mergeBinding(&opts.bindings.Help, cfg.BindHelp)
		mergeBinding(&opts.bindings.Quit, cfg.BindQuit)
		mergeBinding(&opts.bindings.PageNext, cfg.BindPageNext)
		mergeBinding(&opts.bindings.PagePrev, cfg.BindPagePrev)
		mergeBinding(&opts.bindings.OverlayUp, cfg.BindOverlayUp)
		mergeBinding(&opts.bindings.OverlayDown, cfg.BindOverlayDown)
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
	if includeExtsArg != "" {
		parts := strings.Split(includeExtsArg, ",")
		var exts []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if !strings.HasPrefix(p, ".") {
				p = "." + p
			}
			exts = append(exts, p)
		}
		opts.includeExts = exts
	}
	if ignoreGlobsArg != "" {
		parts := strings.Split(ignoreGlobsArg, ",")
		var globs []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			globs = append(globs, p)
		}
		opts.ignoreGlobs = globs
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
	if pageStepArg > 0 {
		opts.pageStep = pageStepArg
	}
	if cfgNavAccelMs > 0 {
		opts.navAccelMs = cfgNavAccelMs
	}
	if cfgNavMaxStep > 0 {
		opts.navMaxStep = cfgNavMaxStep
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
	if cardWidthFracArg > 0 {
		opts.cardWidthFrac = cardWidthFracArg
	}
	if cardHeightFracArg > 0 {
		opts.cardHeightFrac = cardHeightFracArg
	}
	if activeLiftYArg != 0 {
		opts.activeLiftY = activeLiftYArg
	}
	opts.stickyOverlayNav = stickyOverlayNavArg
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

	// Positional path overrides everything else.
	if args := flagSet.Args(); len(args) > 0 {
		opts.noteRoot = args[0]
	}

	// Seed RNG once for the whole process (random card jumps).
	rng := rand.New(rand.NewSource(opts.randomSeed))

	loadOpts := notes.LoadOptions{
		IncludeExts: opts.includeExts,
		IgnoreGlobs: opts.ignoreGlobs,
	}

	cards, err := notes.LoadCardsFromDir(opts.noteRoot, loadOpts)
	if err != nil {
		log.Printf("error loading cards from %s: %v", opts.noteRoot, err)
	}

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

	m := ui.NewModel(cards, opts.noteRoot, rng)
	m.ApplyColors(colors)
	m.ApplyBindings(opts.bindings)
	if opts.pageStep > 0 {
		m.SetPageStep(opts.pageStep)
	}
	if opts.navAccelMs > 0 || opts.navMaxStep > 0 {
		m.ApplyNav(opts.navAccelMs, opts.navMaxStep)
	}
	layout := ui.Layout{
		StackVisibleCount: opts.stackVisible,
		StackOffsetX:      opts.stackOffsetX,
		StackOffsetY:      opts.stackOffsetY,
		CardWidthFrac:     opts.cardWidthFrac,
		CardHeightFrac:    opts.cardHeightFrac,
		ActiveLiftY:       opts.activeLiftY,
		StickyOverlayNav:  &opts.stickyOverlayNav,
	}
	m.ApplyLayout(layout)

	programOptions := []tea.ProgramOption{}
	if opts.useAlternateScreen {
		programOptions = append(programOptions, tea.WithAltScreen())
	}

	p := tea.NewProgram(m, programOptions...)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
