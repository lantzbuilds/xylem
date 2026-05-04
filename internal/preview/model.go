package preview

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

const maxFileSize = 1024 * 1024

type Model struct {
	viewport        viewport.Model
	filePath        string
	language        string
	lineCount       int
	showLineNumbers bool
	theme           string
	width           int
	height          int
	errMsg          string
	rawContent      string
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

	highlighted, lang := Highlight(source, path, m.theme)
	m.language = lang

	content := m.applyLineNumbers(highlighted)
	m.viewport.SetContent(content)
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

func (m Model) SetTheme(theme string) Model {
	m.theme = theme
	if m.filePath != "" && m.errMsg == "" && m.rawContent != "" {
		highlighted, _ := Highlight(m.rawContent, m.filePath, m.theme)
		m.viewport.SetContent(m.applyLineNumbers(highlighted))
	}
	return m
}

func (m Model) ToggleLineNumbers() Model {
	m.showLineNumbers = !m.showLineNumbers
	if m.filePath != "" && m.errMsg == "" && m.rawContent != "" {
		highlighted, _ := Highlight(m.rawContent, m.filePath, m.theme)
		m.viewport.SetContent(m.applyLineNumbers(highlighted))
	}
	return m
}

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
