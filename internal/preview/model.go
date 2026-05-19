package preview

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"strings"

	"path/filepath"

	"github.com/atotto/clipboard"
	"charm.land/bubbles/v2/viewport"
	"github.com/charmbracelet/glamour"
	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"
	lpdf "github.com/ledongthuc/pdf"
	"github.com/lantzbuilds/xylem/internal/search"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/ansi"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
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
	renderedMode    bool
	isMarkdown      bool
	isImage         bool
	searchQuery     string
	searchMatches   []search.Match
	searchIndex     int
}

func NewModel(width, height int, theme string) Model {
	vp := viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
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
	m.isImage = false
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

	if isImageFile(path) {
		content, lang, err := renderImage(path, m.width, m.height)
		if err != nil {
			m.errMsg = fmt.Sprintf("image error: %v", err)
			m.viewport.SetContent(m.errMsg)
			return m
		}
		m.isImage = true
		m.rawContent = content
		m.language = lang
		m.lineCount = strings.Count(content, "\n") + 1
		m.isMarkdown = false
		m.renderedMode = false
		m.viewport.SetContent(content)
		m.viewport.GotoTop()
		return m
	}

	if isPDF(path) {
		text, pages, err := extractPDFText(path)
		if err != nil {
			m.errMsg = fmt.Sprintf("pdf error: %v", err)
			m.viewport.SetContent(m.errMsg)
			return m
		}
		m.rawContent = text
		m.language = "PDF"
		m.lineCount = strings.Count(text, "\n") + 1
		m.isMarkdown = false
		m.renderedMode = false
		m.viewport.SetContent(text)
		m.viewport.GotoTop()
		_ = pages
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
	m.isMarkdown = isMarkdownFile(path)
	m.renderedMode = m.isMarkdown

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

	if m.renderedMode && m.isMarkdown {
		return m.renderMarkdown()
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

func (m Model) renderMarkdown() string {
	style := glamourStyle(m.theme)
	width := m.width
	if width < 20 {
		width = 80
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath(style),
		glamour.WithWordWrap(width-2),
	)
	if err != nil {
		highlighted, _ := Highlight(m.rawContent, m.filePath, m.theme)
		return highlighted
	}

	rendered, err := renderer.Render(m.rawContent)
	if err != nil {
		highlighted, _ := Highlight(m.rawContent, m.filePath, m.theme)
		return highlighted
	}

	return strings.TrimRight(rendered, "\n")
}

func glamourStyle(theme string) string {
	switch theme {
	case "dracula":
		return "dracula"
	case "github":
		return "light"
	case "solarized-light", "catppuccin-latte":
		return "light"
	default:
		return "dark"
	}
}

func isMarkdownFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown") ||
		strings.HasSuffix(lower, ".mkd") || strings.HasSuffix(lower, ".mdx")
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

	rawLines := strings.Split(m.rawContent, "\n")
	lines := strings.Split(content, "\n")
	for i := range lines {
		if matchLines[i] && i < len(rawLines) {
			lines[i] = search.HighlightLine(rawLines[i], m.searchQuery, i == currentLine)
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

func (m Model) WordWrapEnabled() bool  { return m.wordWrap }
func (m Model) RenderedMode() bool     { return m.renderedMode }
func (m Model) IsMarkdown() bool       { return m.isMarkdown }

func (m Model) ToggleRenderedMode() Model {
	if !m.isMarkdown {
		return m
	}
	m.renderedMode = !m.renderedMode
	if m.filePath != "" && m.errMsg == "" && m.rawContent != "" {
		m.viewport.SetContent(m.renderContent())
	}
	return m
}

func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h)
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

func (m Model) IsImage() bool { return m.isImage }

func (m Model) CopyFile() error {
	if m.rawContent == "" {
		return fmt.Errorf("no file loaded")
	}
	return clipboard.WriteAll(m.rawContent)
}

func (m Model) CopyVisible() error {
	lines := strings.Split(m.rawContent, "\n")
	top := m.viewport.YOffset()
	bottom := top + m.viewport.Height()
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
	center := line - m.viewport.Height()/2
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

func isPDF(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".pdf"
}

func extractPDFText(path string) (string, int, error) {
	f, r, err := lpdf.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	totalPages := r.NumPage()
	var b strings.Builder
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	for i := 1; i <= totalPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			b.WriteString(fmt.Sprintf("[page %d: text extraction failed]\n", i))
			continue
		}

		header := dimStyle.Render(fmt.Sprintf("── page %d/%d ──", i, totalPages))
		b.WriteString(header + "\n")
		b.WriteString(strings.TrimRight(text, "\n"))
		b.WriteString("\n\n")
	}

	return strings.TrimRight(b.String(), "\n"), totalPages, nil
}

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".bmp": true, ".webp": true, ".svg": true,
}

func isImageFile(path string) bool {
	return imageExts[strings.ToLower(filepath.Ext(path))]
}

func renderImage(path string, width, height int) (string, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		return imageMetadata(path, nil, "", err)
	}

	return imageMetadata(path, img, format, nil)
}

func imageMetadata(path string, img image.Image, format string, decodeErr error) (string, string, error) {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	var b strings.Builder

	b.WriteString(headerStyle.Render("── image ──") + "\n\n")
	b.WriteString(fmt.Sprintf("  File:   %s\n", filepath.Base(path)))

	info, _ := os.Stat(path)
	if info != nil {
		size := info.Size()
		sizeStr := fmt.Sprintf("%d B", size)
		if size > 1024*1024 {
			sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
		} else if size > 1024 {
			sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
		}
		b.WriteString(fmt.Sprintf("  Size:   %s\n", sizeStr))
	}

	if decodeErr != nil {
		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		b.WriteString(fmt.Sprintf("  Format: %s\n", ext))
		b.WriteString(dimStyle.Render("\n  (decode failed: "+decodeErr.Error()+")") + "\n")
	} else if img != nil {
		bounds := img.Bounds()
		b.WriteString(fmt.Sprintf("  Format: %s\n", format))
		b.WriteString(fmt.Sprintf("  Dimensions: %dx%d\n", bounds.Dx(), bounds.Dy()))
		b.WriteString(fmt.Sprintf("  Color model: %T\n", img.At(0, 0)))
	}

	b.WriteString(dimStyle.Render("\n  press 'o' to open in native viewer") + "\n")

	lang := format
	if lang == "" {
		lang = strings.TrimPrefix(filepath.Ext(path), ".")
	}
	return b.String(), lang, nil
}
