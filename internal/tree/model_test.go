package tree

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
