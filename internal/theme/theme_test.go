package theme

import "testing"

func TestDefaultTheme(t *testing.T) {
	tm := New()
	if tm.Current() != "monokai" {
		t.Errorf("expected default monokai, got %s", tm.Current())
	}
}

func TestCycleTheme(t *testing.T) {
	tm := New()
	first := tm.Current()
	tm.Next()
	second := tm.Current()
	if first == second {
		t.Error("expected theme to change after Next()")
	}
}

func TestCycleWraps(t *testing.T) {
	tm := New()
	count := len(tm.Available())
	for i := 0; i < count; i++ {
		tm.Next()
	}
	if tm.Current() != "monokai" {
		t.Errorf("expected wrap to monokai, got %s", tm.Current())
	}
}

func TestSetTheme(t *testing.T) {
	tm := New()
	ok := tm.Set("dracula")
	if !ok {
		t.Error("expected Set(dracula) to succeed")
	}
	if tm.Current() != "dracula" {
		t.Errorf("expected dracula, got %s", tm.Current())
	}
}

func TestSetInvalidTheme(t *testing.T) {
	tm := New()
	ok := tm.Set("nonexistent")
	if ok {
		t.Error("expected Set(nonexistent) to fail")
	}
	if tm.Current() != "monokai" {
		t.Error("theme should not change on invalid Set")
	}
}
