package statusbar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	width    int
	filename string
	language string
	lines    int
	theme    string
}

func New(width int) Model {
	return Model{width: width}
}

func (m Model) SetFile(filename, language string, lines int) Model {
	m.filename = filename
	m.language = language
	m.lines = lines
	return m
}

func (m Model) SetTheme(theme string) Model {
	m.theme = theme
	return m
}

func (m Model) SetWidth(w int) Model {
	m.width = w
	return m
}

func (m Model) View() string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252"))

	accent := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	var left string
	if m.filename != "" {
		left = fmt.Sprintf(" %s │ %s │ %d lines", m.filename, m.language, m.lines)
	} else {
		left = " xylem"
	}

	right := ""
	if m.theme != "" {
		right = accent.Render(m.theme)
	}

	leftRendered := style.Render(left)
	gap := m.width - lipgloss.Width(leftRendered) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	padding := style.Render(strings.Repeat(" ", gap))

	return leftRendered + padding + right
}
