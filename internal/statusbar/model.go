package statusbar

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type Model struct {
	width    int
	rootName string
	filename string
	language string
	lines    int
	theme    string
	focus    string
	version  string
}

func New(width int, rootName string, version string) Model {
	return Model{width: width, rootName: rootName, version: version}
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

func (m Model) SetFocus(focus string) Model {
	m.focus = focus
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
		display := m.filename
		if m.rootName != "" {
			display = m.rootName + "/" + m.filename
		}
		left = fmt.Sprintf(" %s │ %s │ %d lines", display, m.language, m.lines)
	} else if m.rootName != "" {
		left = fmt.Sprintf(" %s", m.rootName)
	} else {
		left = " xylem"
	}

	versionPart := ""
	if m.version != "" {
		versionPart = style.Render(m.version + " ")
	}
	themePart := ""
	if m.theme != "" {
		themePart = accent.Render(m.theme)
	}
	right := versionPart + themePart

	leftRendered := style.Render(left)
	gap := m.width - lipgloss.Width(leftRendered) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	padding := style.Render(strings.Repeat(" ", gap))

	return leftRendered + padding + right
}
