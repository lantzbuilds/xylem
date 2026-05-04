package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	itree "github.com/lantzbuilds/xylem/internal/tree"
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

func TestAppThemeCycling(t *testing.T) {
	dir := setupTestDir(t)
	m := NewApp(dir, false, "monokai")
	m.width = 120
	m.height = 40

	initialTheme := m.theme.Current()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	um := updated.(App)

	if um.theme.Current() == initialTheme {
		t.Errorf("expected theme to change from %q after pressing t", initialTheme)
	}
}

func TestAppLineNumberToggle(t *testing.T) {
	dir := setupTestDir(t)
	m := NewApp(dir, false, "monokai")
	m.width = 120
	m.height = 40

	if m.preview.ShowingLines() {
		t.Fatal("expected line numbers off initially")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	um := updated.(App)
	if !um.preview.ShowingLines() {
		t.Error("expected line numbers on after pressing n")
	}

	updated2, _ := um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	um2 := updated2.(App)
	if um2.preview.ShowingLines() {
		t.Error("expected line numbers off after pressing n again")
	}
}

func TestAppFinderOpensAndCloses(t *testing.T) {
	dir := setupTestDir(t)
	m := NewApp(dir, false, "monokai")
	m.width = 120
	m.height = 40

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	um := updated.(App)
	if um.focus != focusFinder {
		t.Errorf("expected focus on finder after /, got %d", um.focus)
	}

	updated2, _ := um.Update(tea.KeyMsg{Type: tea.KeyEsc})
	um2 := updated2.(App)
	if um2.focus != focusTree {
		t.Errorf("expected focus back on tree after Escape, got %d", um2.focus)
	}
}

func TestAppFullScreenToggle(t *testing.T) {
	dir := setupTestDir(t)
	m := NewApp(dir, false, "monokai")
	m.width = 120
	m.height = 40

	readmePath := filepath.Join(dir, "README.md")
	fileMsg := itree.FileSelectedMsg{Path: readmePath, IsDir: false}
	loaded, _ := m.Update(fileMsg)
	m = loaded.(App)

	// Navigate tree cursor to the file then send Enter — instead, directly
	// set the tree's selected path by sending the FileSelectedMsg and then
	// trigger fullscreen via Enter when focus is tree and SelectedPath returns a file.
	// Since we can't easily control tree.SelectedPath, use a workaround:
	// manually set fullScreen and focus to test the Esc path, then verify the Enter path
	// by putting a real file at the tree's default selection.

	// Test Esc exits fullscreen.
	m.fullScreen = true
	m.focus = focusPreview
	m = m.resize()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	um := updated.(App)
	if um.fullScreen {
		t.Error("expected fullScreen to be false after Esc")
	}
	if um.focus != focusTree {
		t.Errorf("expected focus on tree after Esc from fullscreen, got %d", um.focus)
	}
}

func TestAppHelpOverlay(t *testing.T) {
	dir := setupTestDir(t)
	m := NewApp(dir, false, "monokai")
	m.width = 120
	m.height = 40

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	um := updated.(App)
	if !um.showHelp {
		t.Error("expected showHelp to be true after pressing ?")
	}

	updated2, _ := um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	um2 := updated2.(App)
	if um2.showHelp {
		t.Error("expected showHelp to be false after pressing ? again")
	}
}

func TestAppFileSelectedMsg(t *testing.T) {
	dir := setupTestDir(t)
	m := NewApp(dir, false, "monokai")
	m.width = 120
	m.height = 40

	readmePath := filepath.Join(dir, "README.md")
	updated, _ := m.Update(itree.FileSelectedMsg{Path: readmePath, IsDir: false})
	um := updated.(App)

	if um.preview.FilePath() != readmePath {
		t.Errorf("expected preview to load %q, got %q", readmePath, um.preview.FilePath())
	}
	if um.preview.LineCount() == 0 {
		t.Error("expected preview to have at least one line after loading README.md")
	}
}

func TestAppResize(t *testing.T) {
	dir := setupTestDir(t)
	m := NewApp(dir, false, "monokai")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	um := updated.(App)

	if um.width != 120 {
		t.Errorf("expected width 120, got %d", um.width)
	}
	if um.height != 40 {
		t.Errorf("expected height 40, got %d", um.height)
	}
}
