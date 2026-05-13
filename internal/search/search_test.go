package search

import (
	"strings"
	"testing"
)

func TestSearchFileBasic(t *testing.T) {
	content := "hello world\nfoo bar\nhello again"
	matches := SearchFile(content, "hello")

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if matches[0].Line != 0 || matches[0].Col != 0 {
		t.Errorf("first match: expected line 0 col 0, got line %d col %d", matches[0].Line, matches[0].Col)
	}
	if matches[1].Line != 2 {
		t.Errorf("second match: expected line 2, got %d", matches[1].Line)
	}
}

func TestSearchFileCaseInsensitive(t *testing.T) {
	content := "Hello World\nHELLO again"
	matches := SearchFile(content, "hello")

	if len(matches) != 2 {
		t.Fatalf("expected 2 case-insensitive matches, got %d", len(matches))
	}
}

func TestSearchFileEmpty(t *testing.T) {
	matches := SearchFile("some content", "")
	if matches != nil {
		t.Error("expected nil for empty query")
	}
}

func TestSearchFileNoMatch(t *testing.T) {
	matches := SearchFile("hello world", "xyz")
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestHighlightLineMatch(t *testing.T) {
	result := HighlightLine("hello world", "world", false)
	if !strings.Contains(result, "world") {
		t.Error("expected highlighted result to contain the match text")
	}
	if !strings.Contains(result, "hello") {
		t.Error("expected non-matched prefix to be preserved")
	}
}

func TestHighlightLineNoMatch(t *testing.T) {
	result := HighlightLine("hello world", "xyz", false)
	if result != "hello world" {
		t.Error("expected unchanged line when no match")
	}
}

func TestHighlightLineMultipleMatches(t *testing.T) {
	result := HighlightLine("foo bar foo baz foo", "foo", false)
	if strings.Count(result, "foo") < 3 {
		t.Error("expected all occurrences to be present")
	}
}
