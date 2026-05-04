package finder

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFinderFilters(t *testing.T) {
	files := []string{"main.go", "README.md", "internal/app/app.go", "go.mod"}
	m := New(files, 60, 20)
	m.active = true

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if len(m.matches) == 0 {
		t.Error("expected matches for 'main'")
	}
	found := false
	for _, match := range m.matches {
		if match == "main.go" {
			found = true
		}
	}
	if !found {
		t.Error("expected main.go in matches")
	}
}

func TestFinderEscapeDismisses(t *testing.T) {
	m := New([]string{"a.go"}, 60, 20)
	m.active = true

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.active {
		t.Error("expected finder to be dismissed")
	}
}

func TestFinderEnterSelectsResult(t *testing.T) {
	files := []string{"main.go", "README.md"}
	m := New(files, 60, 20)
	m.active = true
	m.matches = files
	m.cursor = 0

	var selectedMsg *FileFoundMsg
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		msg := cmd()
		if fm, ok := msg.(FileFoundMsg); ok {
			selectedMsg = &fm
		}
	}
	if selectedMsg == nil {
		t.Fatal("expected FileFoundMsg after Enter")
	}
	if selectedMsg.Path != "main.go" {
		t.Errorf("expected main.go, got %s", selectedMsg.Path)
	}
}

func TestFinderBackspace(t *testing.T) {
	m := New([]string{"main.go"}, 60, 20)
	m.active = true

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if m.query != "ab" {
		t.Errorf("expected query 'ab', got '%s'", m.query)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.query != "a" {
		t.Errorf("expected query 'a' after backspace, got '%s'", m.query)
	}
}
