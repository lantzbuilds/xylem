package tree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitIgnoreFromFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\nbuild/\n"), 0o644)

	gi, err := NewGitIgnore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gi.MatchesPath("output.log") {
		t.Error("expected *.log to match")
	}
	if !gi.MatchesPath("build/main") {
		t.Error("expected build/ to match")
	}
	if gi.MatchesPath("main.go") {
		t.Error("main.go should not match")
	}
}

func TestGitIgnoreFallback(t *testing.T) {
	dir := t.TempDir()

	gi, err := NewGitIgnore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gi != nil {
		t.Error("expected nil when no .gitignore exists")
	}
}

func TestLoadChildrenWithGitIgnore(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("ignored/\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, "ignored"), 0o755)
	os.MkdirAll(filepath.Join(dir, "kept"), 0o755)

	gi, _ := NewGitIgnore(dir)
	n := NewNode(dir, filepath.Base(dir), true)
	n.LoadChildren(gi)

	var foundIgnored, foundKept bool
	for _, child := range n.Children {
		if child.Name == "ignored" {
			foundIgnored = true
			if !child.Ignored {
				t.Error("gitignored directory should be marked Ignored")
			}
		}
		if child.Name == "kept" {
			foundKept = true
			if child.Ignored {
				t.Error("non-ignored directory should not be marked Ignored")
			}
		}
	}
	if !foundIgnored {
		t.Error("ignored directory should be present in children")
	}
	if !foundKept {
		t.Error("kept directory should be present in children")
	}
}
