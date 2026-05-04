package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# hello"), 0o644)
	return dir
}

func TestAppFocusSwitching(t *testing.T) {
	dir := setupTestDir(t)
	m := NewApp(dir, false, "monokai")

	if m.focus != focusTree {
		t.Error("expected initial focus on tree")
	}

	resized := m
	resized.width = 120
	resized.height = 40

	updated, _ := resized.Update(tea.KeyMsg{Type: tea.KeyTab})
	um := updated.(App)
	if um.focus != focusPreview {
		t.Errorf("expected focus on preview after Tab, got %d", um.focus)
	}

	updated2, _ := um.Update(tea.KeyMsg{Type: tea.KeyTab})
	um2 := updated2.(App)
	if um2.focus != focusTree {
		t.Errorf("expected focus back on tree after second Tab, got %d", um2.focus)
	}
}

func TestAppQuit(t *testing.T) {
	dir := setupTestDir(t)
	m := NewApp(dir, false, "monokai")
	m.width = 120
	m.height = 40

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Error("expected tea.QuitMsg")
	}
}
