package preview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewLoadsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	os.WriteFile(path, []byte("package main\n\nfunc main() {}\n"), 0o644)

	m := NewModel(60, 20, "monokai")
	m = m.LoadFile(path)

	if m.language != "Go" {
		t.Errorf("expected Go, got %s", m.language)
	}
	if m.lineCount != 3 {
		t.Errorf("expected 3 lines, got %d", m.lineCount)
	}
}

func TestPreviewBinaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.bin")
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}
	os.WriteFile(path, data, 0o644)

	m := NewModel(60, 20, "monokai")
	m = m.LoadFile(path)

	if !strings.Contains(m.View(), "binary file") {
		t.Error("expected binary file message in view")
	}
}

func TestPreviewLargeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")
	data := make([]byte, 1024*1024+1)
	for i := range data {
		data[i] = 'a'
	}
	os.WriteFile(path, data, 0o644)

	m := NewModel(60, 20, "monokai")
	m = m.LoadFile(path)

	if !strings.Contains(m.View(), "too large") {
		t.Error("expected too large message in view")
	}
}

func TestPreviewLineNumbers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("line one\nline two\nline three\n"), 0o644)

	m := NewModel(60, 20, "monokai")
	m = m.LoadFile(path)
	m.showLineNumbers = true

	view := m.View()
	if !strings.Contains(view, "1") {
		t.Error("expected line numbers in view")
	}
}

func TestPreviewPermissionDenied(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")
	os.WriteFile(path, []byte("secret"), 0o000)

	m := NewModel(60, 20, "monokai")
	m = m.LoadFile(path)

	if !strings.Contains(m.View(), "permission denied") {
		t.Error("expected permission denied message")
	}

	os.Chmod(path, 0o644)
}

func TestPreviewDirectoryShowsPlaceholder(t *testing.T) {
	m := NewModel(60, 20, "monokai")
	view := m.View()
	if len(view) == 0 {
		t.Error("expected placeholder content for empty preview")
	}
}
