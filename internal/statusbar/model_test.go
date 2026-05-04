package statusbar

import (
	"strings"
	"testing"
)

func TestStatusBarShowsFileInfo(t *testing.T) {
	m := New(80)
	m = m.SetFile("main.go", "Go", 42)
	m = m.SetTheme("monokai")

	view := m.View()
	if !strings.Contains(view, "main.go") {
		t.Error("expected filename in status bar")
	}
	if !strings.Contains(view, "Go") {
		t.Error("expected language in status bar")
	}
	if !strings.Contains(view, "42") {
		t.Error("expected line count in status bar")
	}
	if !strings.Contains(view, "monokai") {
		t.Error("expected theme in status bar")
	}
}

func TestStatusBarEmpty(t *testing.T) {
	m := New(80)
	view := m.View()
	if len(view) == 0 {
		t.Error("expected non-empty status bar even with no file")
	}
}
