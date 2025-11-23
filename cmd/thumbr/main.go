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
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"thumbr/internal/notes"
	"thumbr/internal/ui"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

type config struct {
	NoteRoot    string   `json:"noteRoot" yaml:"noteRoot" toml:"noteRoot"`
	RandomSeed  *int64   `json:"randomSeed" yaml:"randomSeed" toml:"randomSeed"`
	AltScreen   *bool    `json:"altScreen" yaml:"altScreen" toml:"altScreen"`
	IncludeExts []string `json:"includeExts" yaml:"includeExts" toml:"includeExts"`
	IgnoreGlobs []string `json:"ignoreGlobs" yaml:"ignoreGlobs" toml:"ignoreGlobs"`
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
		configPath     string
		showVersion    bool
		randomSeedArg  int64Flag
		altScreenArg   bool
		noAltScreen    bool
		includeExtsArg string
		ignoreGlobsArg string
	)

	flagSet.StringVar(&configPath, "config", "", "path to optional JSON/YAML/TOML config file (fields: noteRoot, randomSeed, altScreen, includeExts, ignoreGlobs)")
	flagSet.BoolVar(&showVersion, "version", false, "print version and exit")
	flagSet.BoolVar(&showVersion, "v", false, "print version and exit (shorthand)")
	flagSet.Var(&randomSeedArg, "random-seed", "set RNG seed for reproducible random jumps (int64)")
	flagSet.BoolVar(&altScreenArg, "alt-screen", true, "use alternate screen buffer; set to false to stay on the main screen")
	flagSet.BoolVar(&noAltScreen, "no-alt-screen", false, "run without the alternate screen buffer (same as --alt-screen=false)")
	flagSet.StringVar(&includeExtsArg, "include-exts", "", "comma-separated list of file extensions to include (e.g., .md,.txt)")
	flagSet.StringVar(&ignoreGlobsArg, "ignore", "", "comma-separated list of glob patterns to skip (matched on full path and basename)")

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
	}{
		noteRoot:           ".",
		randomSeed:         time.Now().UnixNano(),
		useAlternateScreen: true,
		includeExts:        nil,
		ignoreGlobs:        nil,
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

	m := ui.NewModel(cards, opts.noteRoot, rng)

	programOptions := []tea.ProgramOption{}
	if opts.useAlternateScreen {
		programOptions = append(programOptions, tea.WithAltScreen())
	}

	p := tea.NewProgram(m, programOptions...)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
