package tree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewNode(t *testing.T) {
	n := NewNode("/tmp/test", "test", true)
	if n.Path != "/tmp/test" {
		t.Errorf("expected path /tmp/test, got %s", n.Path)
	}
	if n.Name != "test" {
		t.Errorf("expected name test, got %s", n.Name)
	}
	if !n.IsDir {
		t.Error("expected IsDir true")
	}
	if n.Expanded {
		t.Error("expected Expanded false by default")
	}
}

func TestLoadChildren(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "subdir"), 0o755)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0o644)
	os.WriteFile(filepath.Join(dir, "alpha.go"), []byte("package main"), 0o644)

	n := NewNode(dir, filepath.Base(dir), true)
	err := n.LoadChildren(nil)
	if err != nil {
		t.Fatalf("LoadChildren error: %v", err)
	}
	if len(n.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(n.Children))
	}
}

func TestLoadChildrenSortOrder(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "zebra"), 0o755)
	os.MkdirAll(filepath.Join(dir, "alpha"), 0o755)
	os.WriteFile(filepath.Join(dir, "middle.txt"), []byte(""), 0o644)

	n := NewNode(dir, filepath.Base(dir), true)
	n.LoadChildren(nil)

	if n.Children[0].Name != "alpha" {
		t.Errorf("expected first child 'alpha' (dir), got %s", n.Children[0].Name)
	}
	if n.Children[1].Name != "zebra" {
		t.Errorf("expected second child 'zebra' (dir), got %s", n.Children[1].Name)
	}
	if n.Children[2].Name != "middle.txt" {
		t.Errorf("expected third child 'middle.txt' (file), got %s", n.Children[2].Name)
	}
}

func TestLoadChildrenSkipsDotGit(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)

	n := NewNode(dir, filepath.Base(dir), true)
	n.LoadChildren(nil)

	if len(n.Children) != 1 {
		t.Fatalf("expected 1 child (no .git), got %d", len(n.Children))
	}
	if n.Children[0].Name != "main.go" {
		t.Errorf("expected main.go, got %s", n.Children[0].Name)
	}
}
