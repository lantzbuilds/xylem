package search

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Match struct {
	Line int
	Col  int
	Text string
	File string
}

func SearchFile(content, query string) []Match {
	if query == "" {
		return nil
	}

	lower := strings.ToLower(query)
	lines := strings.Split(content, "\n")
	var matches []Match

	for i, line := range lines {
		col := strings.Index(strings.ToLower(line), lower)
		if col >= 0 {
			matches = append(matches, Match{
				Line: i,
				Col:  col,
				Text: line,
			})
		}
	}

	return matches
}

func HighlightLine(line, query string, isCurrent bool) string {
	if query == "" {
		return line
	}

	lower := strings.ToLower(line)
	lowerQ := strings.ToLower(query)
	idx := strings.Index(lower, lowerQ)
	if idx < 0 {
		return line
	}

	var style lipgloss.Style
	if isCurrent {
		style = lipgloss.NewStyle().
			Background(lipgloss.Color("214")).
			Foreground(lipgloss.Color("0")).
			Bold(true)
	} else {
		style = lipgloss.NewStyle().
			Background(lipgloss.Color("228")).
			Foreground(lipgloss.Color("0"))
	}

	var b strings.Builder
	for idx >= 0 {
		b.WriteString(line[:idx])
		b.WriteString(style.Render(line[idx : idx+len(query)]))
		line = line[idx+len(query):]
		lower = strings.ToLower(line)
		idx = strings.Index(lower, lowerQ)
	}
	b.WriteString(line)

	return b.String()
}
