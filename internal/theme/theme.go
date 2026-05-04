package theme

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

var themes = []string{
	"monokai",
	"dracula",
	"github",
	"solarized-dark",
	"solarized-light",
	"nord",
	"onedark",
	"gruvbox",
	"catppuccin-mocha",
	"catppuccin-latte",
	"catppuccin-frappe",
	"catppuccin-macchiato",
}

type Manager struct {
	index int
}

func New() *Manager {
	return &Manager{index: 0}
}

func (m *Manager) Current() string {
	return themes[m.index]
}

func (m *Manager) Next() string {
	m.index = (m.index + 1) % len(themes)
	return themes[m.index]
}

func (m *Manager) Set(name string) bool {
	for i, t := range themes {
		if t == name {
			m.index = i
			return true
		}
	}
	return false
}

func (m *Manager) Available() []string {
	return themes
}

func (m *Manager) BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))
}

func (m *Manager) CursorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230"))
}

type config struct {
	Theme string `json:"theme"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "xylem", "config.json")
}

func (m *Manager) Load() {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}
	m.Set(cfg.Theme)
}

func (m *Manager) Save() {
	cfg := config{Theme: m.Current()}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	path := configPath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, data, 0o644)
}

func (m *Manager) DimStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
}

func (m *Manager) StatusStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)
}
