package theme

import (
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
