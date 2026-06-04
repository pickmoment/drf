package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type SortBy string

const (
	SortByName      SortBy = "name"
	SortBySize      SortBy = "size"
	SortByModified  SortBy = "modified"
	SortByExtension SortBy = "extension"
)

type ImageProtocol string

const (
	ImageProtocolAuto   ImageProtocol = "auto"
	ImageProtocolKitty  ImageProtocol = "kitty"
	ImageProtocolITerm2 ImageProtocol = "iterm2"
	ImageProtocolSixel  ImageProtocol = "sixel"
	ImageProtocolBraille ImageProtocol = "braille"
)

type OpenerConfig struct {
	Name     string   `toml:"name"`
	Command  string   `toml:"command"`
	Args     []string `toml:"args"`
	Terminal bool     `toml:"terminal"`
}

type GeneralConfig struct {
	StartDir       string `toml:"start_dir"`
	ShowHidden     bool   `toml:"show_hidden"`
	SortBy         SortBy `toml:"sort_by"`
	SortDescending bool   `toml:"sort_descending"`
	OnboardingDone bool   `toml:"onboarding_done"`
}

type UiConfig struct {
	Theme              string `toml:"theme"`
	ShowBookmarksPanel bool   `toml:"show_bookmarks_panel"`
	ShowPreviewPanel   bool   `toml:"show_preview_panel"`
	ShowHintBar        bool   `toml:"show_hint_bar"`
	ShowIcons          bool   `toml:"show_icons"`
}

type KeymapConfig struct {
	VimKeys bool `toml:"vim_keys"`
}

type PreviewConfig struct {
	MarkdownRender bool          `toml:"markdown_render"`
	ImageProtocol  ImageProtocol `toml:"image_protocol"`
	SyntaxTheme    string        `toml:"syntax_theme"`
	MaxFileSize    int64         `toml:"max_file_size"`
	Wrap           bool          `toml:"wrap"`
}

type Config struct {
	General   GeneralConfig  `toml:"general"`
	Ui        UiConfig       `toml:"ui"`
	Keymap    KeymapConfig   `toml:"keymap"`
	Preview   PreviewConfig  `toml:"preview"`
	Bookmarks []string       `toml:"bookmarks"`
	Openers   []OpenerConfig `toml:"openers"`
}

func defaultOpeners() []OpenerConfig {
	return []OpenerConfig{
		{Name: "VS Code", Command: "code", Args: []string{}, Terminal: false},
	}
}

func Default() Config {
	homeDir, _ := os.UserHomeDir()
	return Config{
		General: GeneralConfig{
			StartDir:       homeDir,
			ShowHidden:     false,
			SortBy:         SortByName,
			SortDescending: false,
			OnboardingDone: false,
		},
		Ui: UiConfig{
			Theme:              "default",
			ShowBookmarksPanel: true,
			ShowPreviewPanel:   true,
			ShowHintBar:        true,
			ShowIcons:          true,
		},
		Keymap: KeymapConfig{
			VimKeys: true,
		},
		Preview: PreviewConfig{
			MarkdownRender: true,
			ImageProtocol:  ImageProtocolAuto,
			SyntaxTheme:    "monokai",
			MaxFileSize:    10 * 1024 * 1024,
			Wrap:           false,
		},
		Bookmarks: []string{},
		Openers:   defaultOpeners(),
	}
}

func ConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "drf", "config.toml")
}

func Load() (Config, error) {
	cfg := Default()
	path := ConfigPath()
	if path == "" {
		return cfg, nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return Default(), nil
	}
	if len(cfg.Openers) == 0 {
		cfg.Openers = defaultOpeners()
	}
	return cfg, nil
}

func (c *Config) Save() error {
	path := ConfigPath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}
