package tree

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FileSelectedMsg struct {
	Path  string
	IsDir bool
}

type Model struct {
	root      *Node
	rootPath  string
	ignore    IgnoreChecker
	cursor    int
	width     int
	height    int
	offset    int
	cursorSty lipgloss.Style
	dirSty    lipgloss.Style
	fileSty   lipgloss.Style
}

func NewModel(path string, width, height int) Model {
	root := NewNode(path, ".", true)
	root.Expanded = true

	gi, _ := NewGitIgnore(path)
	var ic IgnoreChecker
	if gi != nil {
		ic = gi
	}
	root.LoadChildren(ic)

	return Model{
		root:      root,
		rootPath:  path,
		ignore:    ic,
		width:     width,
		height:    height,
		cursorSty: lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")),
		dirSty:    lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true),
		fileSty:   lipgloss.NewStyle(),
	}
}

func (m Model) Init() tea.Cmd {
	if nodes := m.flatNodes(); len(nodes) > 0 {
		selected := nodes[0]
		if !selected.IsDir {
			return func() tea.Msg {
				return FileSelectedMsg{Path: selected.Path, IsDir: false}
			}
		}
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	flat := m.flatNodes()
	maxIdx := len(flat) - 1

	switch msg.String() {
	case "j", "down":
		if m.cursor < maxIdx {
			m.cursor++
			m.ensureVisible()
			return m, m.selectCmd()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.ensureVisible()
			return m, m.selectCmd()
		}
	case "l", "right":
		if node := flat[m.cursor]; node.IsDir && !node.Expanded {
			node.Expanded = true
			node.LoadChildren(m.ignore)
			return m, m.selectCmd()
		}
	case "h", "left":
		if node := flat[m.cursor]; node.IsDir && node.Expanded {
			node.Expanded = false
			node.Children = nil
			return m, m.selectCmd()
		}
	case "enter":
		if node := flat[m.cursor]; node.IsDir {
			node.Expanded = !node.Expanded
			if node.Expanded {
				node.LoadChildren(m.ignore)
			} else {
				node.Children = nil
			}
			return m, m.selectCmd()
		}
		return m, m.selectCmd()
	case "g":
		m.cursor = 0
		m.offset = 0
		return m, m.selectCmd()
	case "G":
		m.cursor = maxIdx
		m.ensureVisible()
		return m, m.selectCmd()
	}

	return m, nil
}

func (m Model) selectCmd() tea.Cmd {
	flat := m.flatNodes()
	if m.cursor >= len(flat) {
		return nil
	}
	node := flat[m.cursor]
	return func() tea.Msg {
		return FileSelectedMsg{Path: node.Path, IsDir: node.IsDir}
	}
}

func (m *Model) ensureVisible() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.height {
		m.offset = m.cursor - m.height + 1
	}
}

func (m Model) flatNodes() []*Node {
	if m.root == nil {
		return nil
	}
	all := m.root.Flatten()
	if len(all) > 0 {
		return all[1:]
	}
	return all
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}

func (m Model) View() string {
	flat := m.flatNodes()
	if len(flat) == 0 {
		return "(empty)"
	}

	var b strings.Builder
	end := m.offset + m.height
	if end > len(flat) {
		end = len(flat)
	}

	for i := m.offset; i < end; i++ {
		node := flat[i]
		depth := node.Depth(m.rootPath)
		indent := strings.Repeat("  ", depth)

		var icon string
		var line string
		if node.IsDir {
			if node.Expanded {
				icon = "▼ "
			} else {
				icon = "▶ "
			}
			line = fmt.Sprintf("%s%s%s", indent, icon, node.Name)
			line = m.dirSty.Render(line)
		} else {
			icon = "  "
			line = fmt.Sprintf("%s%s%s", indent, icon, node.Name)
			line = m.fileSty.Render(line)
		}

		if i == m.cursor {
			line = m.cursorSty.Width(m.width).Render(line)
		}

		b.WriteString(line)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func (m Model) SelectedPath() string {
	flat := m.flatNodes()
	if m.cursor < len(flat) {
		return flat[m.cursor].Path
	}
	return ""
}

func (m Model) NavigateTo(path string) Model {
	flat := m.flatNodes()
	for i, node := range flat {
		if node.Path == path {
			m.cursor = i
			m.ensureVisible()
			return m
		}
	}
	return m
}
