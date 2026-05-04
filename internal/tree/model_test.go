package tree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0o644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0o644)
	return dir
}

func TestTreeModelInit(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	if m.root == nil {
		t.Fatal("expected root node")
	}
	if !m.root.Expanded {
		t.Error("root should start expanded")
	}
	if len(m.root.Children) == 0 {
		t.Error("root should have children after init")
	}
}

func TestTreeCursorDown(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	initial := m.cursor
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	um := updated.(Model)
	if um.cursor != initial+1 {
		t.Error("expected cursor to move down")
	}
}

func TestTreeCursorUp(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	um := updated.(Model)
	if um.cursor != 0 {
		t.Error("expected cursor back at 0")
	}
}

func TestTreeCursorDoesNotGoBelowZero(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	um := updated.(Model)
	if um.cursor != 0 {
		t.Error("cursor should not go below 0")
	}
}

func TestTreeExpandCollapse(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	flat := m.flatNodes()
	var srcIdx int
	for i, n := range flat {
		if n.Name == "src" {
			srcIdx = i
			break
		}
	}

	model := m
	model.cursor = srcIdx
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	um := updated.(Model)
	expandedFlat := um.flatNodes()
	if len(expandedFlat) <= len(flat) {
		t.Error("expected more nodes after expanding src/")
	}

	updated2, _ := um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	um2 := updated2.(Model)
	collapsedFlat := um2.flatNodes()
	if len(collapsedFlat) != len(flat) {
		t.Error("expected same node count after collapsing src/")
	}
}

func setupNestedTestDir(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	deepDir := filepath.Join(dir, "src", "internal", "deep")
	os.MkdirAll(deepDir, 0o755)
	filePath := filepath.Join(deepDir, "file.go")
	os.WriteFile(filePath, []byte("package deep"), 0o644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0o644)
	return dir, filePath
}

func TestTreeNavigateToNestedFile(t *testing.T) {
	dir, filePath := setupNestedTestDir(t)
	m := NewModel(dir, 80, 40)

	initialFlat := m.flatNodes()
	for _, n := range initialFlat {
		if n.Path == filePath {
			t.Fatal("file.go should not be in flatNodes before NavigateTo (src is collapsed)")
		}
	}

	m = m.NavigateTo(filePath)

	flat := m.flatNodes()
	found := false
	for _, n := range flat {
		if n.Path == filePath {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected file.go to be in flatNodes after NavigateTo")
	}
	if flat[m.cursor].Path != filePath {
		t.Errorf("expected cursor to point to file.go, got %s", flat[m.cursor].Path)
	}
}

func TestTreeNavigateToNonexistent(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	initialCursor := m.cursor
	m = m.NavigateTo(filepath.Join(dir, "does", "not", "exist.go"))

	if m.cursor != initialCursor {
		t.Errorf("cursor should not change for nonexistent path, got %d", m.cursor)
	}
}

func TestTreeJumpToTop(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	// Move cursor down a few times
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Jump to top with 'g'
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	um := updated.(Model)

	if um.cursor != 0 {
		t.Errorf("expected cursor at 0 after 'g', got %d", um.cursor)
	}
}

func TestTreeJumpToBottom(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	um := updated.(Model)

	flat := um.flatNodes()
	expected := len(flat) - 1
	if um.cursor != expected {
		t.Errorf("expected cursor at %d after 'G', got %d", expected, um.cursor)
	}
}

func TestTreeEnterOnDirectory(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	flat := m.flatNodes()
	var srcIdx int
	for i, n := range flat {
		if n.Name == "src" {
			srcIdx = i
			break
		}
	}
	m.cursor = srcIdx

	// Enter expands
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)
	expandedFlat := um.flatNodes()
	if len(expandedFlat) <= len(flat) {
		t.Error("expected more nodes after Enter on directory (expand)")
	}

	// Enter again collapses
	updated2, _ := um.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um2 := updated2.(Model)
	collapsedFlat := um2.flatNodes()
	if len(collapsedFlat) != len(flat) {
		t.Error("expected original node count after second Enter (collapse)")
	}
}

func TestTreeViewRendersContent(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 80, 40)

	view := m.View()

	if !strings.Contains(view, "src") {
		t.Error("expected View() to contain 'src'")
	}
	if !strings.Contains(view, "README.md") {
		t.Error("expected View() to contain 'README.md'")
	}
	if !strings.Contains(view, "go.mod") {
		t.Error("expected View() to contain 'go.mod'")
	}
}

func TestTreeSelectedPath(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	flat := m.flatNodes()
	if len(flat) == 0 {
		t.Fatal("expected at least one node")
	}

	if m.SelectedPath() != flat[0].Path {
		t.Errorf("expected SelectedPath() to return %s, got %s", flat[0].Path, m.SelectedPath())
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	um := updated.(Model)

	if um.SelectedPath() != flat[1].Path {
		t.Errorf("expected SelectedPath() to return %s after moving down, got %s", flat[1].Path, um.SelectedPath())
	}
}

func TestTreeFlatNodesSkipsRoot(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	flat := m.flatNodes()
	for _, n := range flat {
		if n.Name == "." {
			t.Error("flatNodes should not include the root node (name '.')")
		}
	}
}

func TestTreeRefreshPicksUpNewFiles(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	initialCount := len(m.flatNodes())

	os.WriteFile(filepath.Join(dir, "new_file.txt"), []byte("hello"), 0o644)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	um := updated.(Model)

	newCount := len(um.flatNodes())
	if newCount != initialCount+1 {
		t.Errorf("expected %d nodes after refresh, got %d", initialCount+1, newCount)
	}

	found := false
	for _, n := range um.flatNodes() {
		if n.Name == "new_file.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected new_file.txt to appear after refresh")
	}
}

func TestTreeRefreshPicksUpDeletedFiles(t *testing.T) {
	dir := setupTestDir(t)
	m := NewModel(dir, 40, 20)

	initialCount := len(m.flatNodes())

	os.Remove(filepath.Join(dir, "README.md"))

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	um := updated.(Model)

	newCount := len(um.flatNodes())
	if newCount != initialCount-1 {
		t.Errorf("expected %d nodes after refresh, got %d", initialCount-1, newCount)
	}
}
