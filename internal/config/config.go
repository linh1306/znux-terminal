package config

import (
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds the application configuration
type Config struct {
	Theme     ThemeConfig
	Keybinding KeybindingConfig
}

// ThemeConfig holds theme-related configuration
type ThemeConfig struct {
	SuggestionBg  string
	SuggestionFg  string
	SelectedBg    string
	SelectedFg    string
	Border        string
}

// KeybindingConfig holds keybinding configuration
type KeybindingConfig struct {
	AcceptSuggestion []string
	NextSuggestion   []string
	PrevSuggestion   []string
	ClosePopup       []string
}

// Load loads configuration from a TOML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Theme: ThemeConfig{
			SuggestionBg:  "black",
			SuggestionFg:  "white",
			SelectedBg:    "blue",
			SelectedFg:    "white",
			Border:        "white",
		},
		Keybinding: KeybindingConfig{
			AcceptSuggestion: []string{"enter", "tab", "right"},
			NextSuggestion:   []string{"down", "ctrl-n"},
			PrevSuggestion:   []string{"up", "ctrl-p"},
			ClosePopup:       []string{"escape", "ctrl-c"},
		},
	}
}

// Save saves configuration to a TOML file
func (c *Config) Save(path string) error {
	var buf strings.Builder
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(c); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(buf.String()), 0644)
}