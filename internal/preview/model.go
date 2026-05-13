package preview

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/lantzbuilds/xylem/internal/search"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/ansi"
)

const maxFileSize = 1024 * 1024

type Model struct {
	viewport        viewport.Model
	filePath        string
	language        string
	lineCount       int
	showLineNumbers bool
	wordWrap        bool
	theme           string
	width           int
	height          int
	errMsg          string
	rawContent      string
	searchQuery     string
	searchMatches   []search.Match
	searchIndex     int
}

func NewModel(width, height int, theme string) Model {
	vp := viewport.New(width, height)
	return Model{
		viewport: vp,
		theme:    theme,
		width:    width,
		height:   height,
	}
}

func (m Model) LoadFile(path string) Model {
	m.filePath = path
	m.errMsg = ""
	m.rawContent = ""
	m.language = ""
	m.lineCount = 0
	m.searchQuery = ""
	m.searchMatches = nil
	m.searchIndex = 0

	info, err := os.Stat(path)
	if err != nil {
		if os.IsPermission(err) {
			m.errMsg = "permission denied"
		} else {
			m.errMsg = fmt.Sprintf("error: %v", err)
		}
		m.viewport.SetContent(m.errMsg)
		return m
	}

	if info.IsDir() {
		m.viewport.SetContent("")
		return m
	}

	if info.Size() > maxFileSize {
		m.errMsg = fmt.Sprintf("file too large for preview (%d bytes)", info.Size())
		m.viewport.SetContent(m.errMsg)
		return m
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsPermission(err) {
			m.errMsg = "permission denied"
		} else {
			m.errMsg = fmt.Sprintf("error: %v", err)
		}
		m.viewport.SetContent(m.errMsg)
		return m
	}

	if isBinary(data) {
		m.errMsg = fmt.Sprintf("binary file — %d bytes", len(data))
		m.viewport.SetContent(m.errMsg)
		return m
	}

	source := string(data)
	m.rawContent = source
	m.lineCount = strings.Count(source, "\n")
	if len(source) > 0 && !strings.HasSuffix(source, "\n") {
		m.lineCount++
	}

	_, lang := Highlight(source, path, m.theme)
	m.language = lang

	m.viewport.SetContent(m.renderContent())
	m.viewport.GotoTop()

	return m
}

func (m Model) applyLineNumbers(content string) string {
	if !m.showLineNumbers {
		return content
	}

	lines := strings.Split(content, "\n")
	gutterWidth := len(fmt.Sprintf("%d", len(lines)))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var b strings.Builder
	for i, line := range lines {
		num := fmt.Sprintf("%*d", gutterWidth, i+1)
		b.WriteString(dimStyle.Render(num))
		b.WriteString(" │ ")
		b.WriteString(line)
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m Model) renderContent() string {
	if m.rawContent == "" {
		return ""
	}
	highlighted, _ := Highlight(m.rawContent, m.filePath, m.theme)

	if m.searchQuery != "" && len(m.searchMatches) > 0 {
		highlighted = m.applySearchHighlight(highlighted)
	}

	content := m.applyLineNumbers(highlighted)
	if m.wordWrap {
		content = m.wrapLines(content)
	}
	return content
}

func (m Model) applySearchHighlight(content string) string {
	matchLines := make(map[int]bool)
	currentLine := -1
	for i, match := range m.searchMatches {
		matchLines[match.Line] = true
		if i == m.searchIndex {
			currentLine = match.Line
		}
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if matchLines[i] {
			lines[i] = search.HighlightLine(line, m.searchQuery, i == currentLine)
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) wrapLines(content string) string {
	wrapWidth := m.width
	if m.showLineNumbers {
		gutterWidth := len(fmt.Sprintf("%d", m.lineCount)) + 3
		wrapWidth -= gutterWidth
	}
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	lines := strings.Split(content, "\n")
	var b strings.Builder
	for i, line := range lines {
		lineWidth := ansi.PrintableRuneWidth(line)
		if lineWidth > wrapWidth {
			wrapped := wordwrap.String(line, wrapWidth)
			b.WriteString(wrapped)
		} else {
			b.WriteString(line)
		}
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m Model) SetTheme(theme string) Model {
	m.theme = theme
	if m.filePath != "" && m.errMsg == "" && m.rawContent != "" {
		m.viewport.SetContent(m.renderContent())
	}
	return m
}

func (m Model) ToggleLineNumbers() Model {
	m.showLineNumbers = !m.showLineNumbers
	if m.filePath != "" && m.errMsg == "" && m.rawContent != "" {
		m.viewport.SetContent(m.renderContent())
	}
	return m
}

func (m Model) ToggleWordWrap() Model {
	m.wordWrap = !m.wordWrap
	if m.filePath != "" && m.errMsg == "" && m.rawContent != "" {
		m.viewport.SetContent(m.renderContent())
	}
	return m
}

func (m Model) WordWrapEnabled() bool { return m.wordWrap }

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) GotoTop() Model {
	m.viewport.GotoTop()
	return m
}

func (m Model) GotoBottom() Model {
	m.viewport.GotoBottom()
	return m
}

func (m Model) View() string {
	if m.errMsg != "" {
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		return dim.Render(m.errMsg)
	}
	if m.filePath == "" {
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		return dim.Render("select a file to preview")
	}
	return m.viewport.View()
}

func (m Model) CopyFile() error {
	if m.rawContent == "" {
		return fmt.Errorf("no file loaded")
	}
	return clipboard.WriteAll(m.rawContent)
}

func (m Model) CopyVisible() error {
	lines := strings.Split(m.rawContent, "\n")
	top := m.viewport.YOffset
	bottom := top + m.viewport.Height
	if bottom > len(lines) {
		bottom = len(lines)
	}
	if top >= len(lines) {
		return fmt.Errorf("no visible content")
	}
	return clipboard.WriteAll(strings.Join(lines[top:bottom], "\n"))
}

func (m Model) Search(query string) Model {
	m.searchQuery = query
	m.searchMatches = search.SearchFile(m.rawContent, query)
	m.searchIndex = 0
	if len(m.searchMatches) > 0 {
		m.viewport.SetContent(m.renderContent())
		m = m.scrollToCurrentMatch()
	} else {
		m.viewport.SetContent(m.renderContent())
	}
	return m
}

func (m Model) NextMatch() Model {
	if len(m.searchMatches) == 0 {
		return m
	}
	m.searchIndex = (m.searchIndex + 1) % len(m.searchMatches)
	m.viewport.SetContent(m.renderContent())
	return m.scrollToCurrentMatch()
}

func (m Model) PrevMatch() Model {
	if len(m.searchMatches) == 0 {
		return m
	}
	m.searchIndex--
	if m.searchIndex < 0 {
		m.searchIndex = len(m.searchMatches) - 1
	}
	m.viewport.SetContent(m.renderContent())
	return m.scrollToCurrentMatch()
}

func (m Model) ClearSearch() Model {
	m.searchQuery = ""
	m.searchMatches = nil
	m.searchIndex = 0
	if m.rawContent != "" {
		m.viewport.SetContent(m.renderContent())
	}
	return m
}

func (m Model) scrollToCurrentMatch() Model {
	if len(m.searchMatches) == 0 {
		return m
	}
	line := m.searchMatches[m.searchIndex].Line
	center := line - m.viewport.Height/2
	if center < 0 {
		center = 0
	}
	m.viewport.SetYOffset(center)
	return m
}

func (m Model) SearchQuery() string      { return m.searchQuery }
func (m Model) SearchMatchCount() int    { return len(m.searchMatches) }
func (m Model) SearchIndex() int         { return m.searchIndex }

func (m Model) FilePath() string   { return m.filePath }
func (m Model) Language() string   { return m.language }
func (m Model) LineCount() int     { return m.lineCount }
func (m Model) ShowingLines() bool { return m.showLineNumbers }

func isBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}
	contentType := http.DetectContentType(sample)
	return !strings.HasPrefix(contentType, "text/")
}
