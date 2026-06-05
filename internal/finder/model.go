package finder

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
)

type FileFoundMsg struct {
	Path string
}

type Model struct {
	allFiles []string
	matches  []string
	query    string
	cursor   int
	active   bool
	width    int
	height   int
}

func New(files []string, width, height int) Model {
	return Model{
		allFiles: files,
		matches:  files,
		width:    width,
		height:   height,
	}
}

func (m Model) Active() bool { return m.active }

func (m Model) Open() Model {
	m.active = true
	m.query = ""
	m.cursor = 0
	m.matches = m.allFiles
	return m
}

func (m Model) Close() Model {
	m.active = false
	m.query = ""
	m.cursor = 0
	return m
}

func (m Model) SetFiles(files []string) Model {
	m.allFiles = files
	return m
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "escape", "esc":
			m.active = false
			return m, nil
		case "enter":
			if len(m.matches) > 0 && m.cursor < len(m.matches) {
				path := m.matches[m.cursor]
				m.active = false
				return m, func() tea.Msg {
					return FileFoundMsg{Path: path}
				}
			}
			return m, nil
		case "backspace":
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.filter()
			}
			return m, nil
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down":
			if m.cursor < len(m.matches)-1 {
				m.cursor++
			}
			return m, nil
		default:
			if msg.Text != "" {
				m.query += msg.Text
				m.filter()
			}
			return m, nil
		}
	}

	return m, nil
}

func (m *Model) filter() {
	m.cursor = 0
	if m.query == "" {
		m.matches = m.allFiles
		return
	}

	results := fuzzy.Find(m.query, m.allFiles)
	m.matches = make([]string, len(results))
	for i, r := range results {
		m.matches[i] = r.Str
	}
}

func (m Model) View() string {
	if !m.active {
		return ""
	}

	boxW := m.width / 2
	if boxW < 40 {
		boxW = 40
	}
	if boxW > m.width-4 {
		boxW = m.width - 4
	}
	maxResults := 10

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(boxW).
		Padding(0, 1)

	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render("/ ")
	input := prompt + m.query + "█"

	var lines []string
	lines = append(lines, input)
	lines = append(lines, strings.Repeat("─", boxW-4))

	end := len(m.matches)
	if end > maxResults {
		end = maxResults
	}

	cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	for i := 0; i < end; i++ {
		entry := m.matches[i]
		if i == m.cursor {
			lines = append(lines, cursorStyle.Render(fmt.Sprintf(" %s ", entry)))
		} else {
			lines = append(lines, dim.Render(fmt.Sprintf(" %s", entry)))
		}
	}

	if len(m.matches) > maxResults {
		lines = append(lines, dim.Render(fmt.Sprintf(" ... and %d more", len(m.matches)-maxResults)))
	}

	if len(m.matches) == 0 {
		lines = append(lines, dim.Render(" no matches"))
	}

	content := strings.Join(lines, "\n")
	return border.Render(content)
}
