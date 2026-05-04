package preview

import (
	"strings"
	"testing"
)

func TestHighlightDetectsLanguage(t *testing.T) {
	result, lang := Highlight("package main\n\nfunc main() {}\n", "main.go", "monokai")
	if lang != "Go" {
		t.Errorf("expected language Go, got %s", lang)
	}
	if len(result) == 0 {
		t.Error("expected non-empty highlighted output")
	}
}

func TestHighlightContainsANSI(t *testing.T) {
	result, _ := Highlight("package main\n", "main.go", "monokai")
	if !strings.Contains(result, "\033[") {
		t.Error("expected ANSI escape codes in output")
	}
}

func TestHighlightUnknownExtension(t *testing.T) {
	result, lang := Highlight("some content", "file.xyz123", "monokai")
	if lang != "plaintext" {
		t.Errorf("expected plaintext fallback, got %s", lang)
	}
	if !strings.Contains(result, "some content") {
		t.Error("expected content preserved for unknown types")
	}
}

func TestHighlightDifferentThemes(t *testing.T) {
	r1, _ := Highlight("x = 1", "test.py", "monokai")
	r2, _ := Highlight("x = 1", "test.py", "dracula")
	if r1 == r2 {
		t.Error("expected different themes to produce different output")
	}
}
