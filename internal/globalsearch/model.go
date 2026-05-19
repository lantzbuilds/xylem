package globalsearch

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/lantzbuilds/xylem/internal/search"
)

type ResultSelectedMsg struct {
	File  string
	Line  int
	Query string
}

type SearchDoneMsg struct {
	Results []search.GlobalResult
}

type Model struct {
	query      string
	results    []search.GlobalResult
	flatItems  []flatItem
	cursor     int
	active     bool
	searching  bool
	width      int
	height     int
	rootPath   string
	scrollOff  int
}

type flatItem struct {
	file    string
	match   search.Match
	isFile  bool
}

func New(width, height int, rootPath string) Model {
	return Model{width: width, height: height, rootPath: rootPath}
}

func (m Model) Active() bool    { return m.active }
func (m Model) Searching() bool { return m.searching }

func (m Model) Open() Model {
	m.active = true
	m.query = ""
	m.results = nil
	m.flatItems = nil
	m.cursor = 0
	m.scrollOff = 0
	m.searching = false
	return m
}

func (m Model) Close() Model {
	m.active = false
	m.query = ""
	m.results = nil
	m.flatItems = nil
	m.cursor = 0
	m.scrollOff = 0
	m.searching = false
	return m
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}

func (m Model) SetRootPath(root string) Model {
	m.rootPath = root
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case SearchDoneMsg:
		m.searching = false
		m.results = msg.Results
		m.flatItems = m.flatten()
		m.cursor = 0
		m.scrollOff = 0
		return m, nil

	case tea.KeyPressMsg:
		if m.searching {
			return m, nil
		}

		switch msg.String() {
		case "escape", "esc":
			m.active = false
			return m, nil
		case "enter":
			if len(m.flatItems) > 0 && m.cursor < len(m.flatItems) {
				item := m.flatItems[m.cursor]
				m.active = false
				query := m.query
				return m, func() tea.Msg {
					return ResultSelectedMsg{
						File:  item.file,
						Line:  item.match.Line,
						Query: query,
					}
				}
			}
			if m.query != "" && m.results == nil {
				m.searching = true
				q := m.query
				root := m.rootPath
				return m, func() tea.Msg {
					results := search.SearchGlobal(root, q)
					return SearchDoneMsg{Results: results}
				}
			}
			return m, nil
		case "backspace":
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.results = nil
				m.flatItems = nil
				m.cursor = 0
				m.scrollOff = 0
			}
			return m, nil
		case "up":
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
			return m, nil
		case "down":
			if m.cursor < len(m.flatItems)-1 {
				m.cursor++
				m.ensureVisible()
			}
			return m, nil
		default:
			if msg.Text != "" {
				m.query += msg.Text
				m.results = nil
				m.flatItems = nil
				m.cursor = 0
				m.scrollOff = 0
			}
			return m, nil
		}
	}

	return m, nil
}

func (m Model) flatten() []flatItem {
	var items []flatItem
	for _, r := range m.results {
		for _, match := range r.Matches {
			items = append(items, flatItem{
				file:  r.File,
				match: match,
			})
		}
	}
	return items
}

func (m *Model) ensureVisible() {
	maxVisible := m.maxVisible()
	if m.cursor < m.scrollOff {
		m.scrollOff = m.cursor
	}
	if m.cursor >= m.scrollOff+maxVisible {
		m.scrollOff = m.cursor - maxVisible + 1
	}
}

func (m Model) maxVisible() int {
	v := m.height/2 - 4
	if v < 5 {
		v = 5
	}
	if v > 20 {
		v = 20
	}
	return v
}

func (m Model) View() string {
	if !m.active {
		return ""
	}

	boxW := m.width / 2
	if boxW < 50 {
		boxW = 50
	}
	if boxW > m.width-4 {
		boxW = m.width - 4
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(boxW).
		Padding(0, 1)

	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("110"))
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))

	var lines []string

	prompt := promptStyle.Render("/ ") + m.query + "█"
	lines = append(lines, prompt)
	lines = append(lines, strings.Repeat("─", boxW-4))

	if m.searching {
		engine := search.SearchEngine()
		lines = append(lines, dim.Render(fmt.Sprintf(" searching (%s)...", engine)))
	} else if m.results != nil && len(m.flatItems) == 0 {
		lines = append(lines, dim.Render(" no matches"))
	} else if len(m.flatItems) > 0 {
		maxVisible := m.maxVisible()
		end := m.scrollOff + maxVisible
		if end > len(m.flatItems) {
			end = len(m.flatItems)
		}

		prevFile := ""
		for i := m.scrollOff; i < end; i++ {
			item := m.flatItems[i]

			if item.file != prevFile {
				lines = append(lines, fileStyle.Render(" "+item.file))
				prevFile = item.file
			}

			lineNum := lineNumStyle.Render(fmt.Sprintf("  %d:", item.match.Line+1))
			text := truncate(strings.TrimSpace(item.match.Text), boxW-10)

			entry := lineNum + " " + text
			if i == m.cursor {
				entry = cursorStyle.Render(fmt.Sprintf("  %d: %s", item.match.Line+1, text))
			}
			lines = append(lines, entry)
		}

		totalMatches := len(m.flatItems)
		totalFiles := len(m.results)
		lines = append(lines, strings.Repeat("─", boxW-4))
		lines = append(lines, dim.Render(fmt.Sprintf(" %d match(es) in %d file(s)", totalMatches, totalFiles)))
	} else if m.query != "" {
		lines = append(lines, dim.Render(" press Enter to search"))
	}

	content := strings.Join(lines, "\n")
	return border.Render(content)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
