// Package config loads the optional kubebar config file.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// maxSignal keeps waybar_signal inside the Linux real-time signal range,
// SIGRTMIN+1 to SIGRTMAX.
const maxSignal = 30

type Config struct {
	ProdPatterns []*regexp.Regexp
	MenuCommand  string
	// InputCommand prompts for free text. Empty means the menu itself
	// returns typed text that matches no item, as Walker and rofi do.
	InputCommand string
	WaybarSignal int
}

type file struct {
	ProdPatterns []string `toml:"prod_patterns"`
	MenuCommand  string   `toml:"menu_command"`
	InputCommand string   `toml:"input_command"`
	WaybarSignal int      `toml:"waybar_signal"`
}

func defaults() file {
	return file{
		ProdPatterns: []string{"prod"},
		MenuCommand:  "omarchy-menu-select",
		InputCommand: "omarchy-menu-input",
		WaybarSignal: 8,
	}
}

// Path returns $XDG_CONFIG_HOME/kubebar/config.toml, falling back to
// ~/.config when XDG_CONFIG_HOME is unset or relative.
func Path() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(dir) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate config dir: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "kubebar", "config.toml"), nil
}

// Load reads the config at path. A missing file yields the defaults, and
// keys left out of the file keep their defaults.
func Load(path string) (Config, error) {
	raw := defaults()
	md, err := toml.DecodeFile(path, &raw)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	if keys := md.Undecoded(); len(keys) > 0 {
		return Config{}, fmt.Errorf("read config %s: unknown keys %v", path, keys)
	}
	cfg, err := raw.parse()
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	return cfg, nil
}

func (f file) parse() (Config, error) {
	if strings.TrimSpace(f.MenuCommand) == "" {
		return Config{}, errors.New("menu_command is empty")
	}
	if f.WaybarSignal < 1 || f.WaybarSignal > maxSignal {
		return Config{}, fmt.Errorf("waybar_signal %d out of range 1-%d", f.WaybarSignal, maxSignal)
	}
	patterns := make([]*regexp.Regexp, 0, len(f.ProdPatterns))
	for _, p := range f.ProdPatterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return Config{}, fmt.Errorf("prod_patterns: %w", err)
		}
		patterns = append(patterns, re)
	}
	return Config{
		ProdPatterns: patterns,
		MenuCommand:  f.MenuCommand,
		InputCommand: strings.TrimSpace(f.InputCommand),
		WaybarSignal: f.WaybarSignal,
	}, nil
}

func (c Config) IsProd(context string) bool {
	for _, re := range c.ProdPatterns {
		if re.MatchString(context) {
			return true
		}
	}
	return false
}
