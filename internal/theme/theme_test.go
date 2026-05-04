package theme

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestLoadSaveRoundTrip(t *testing.T) {
	path := configPath()

	existing, existErr := os.ReadFile(path)
	t.Cleanup(func() {
		if existErr == nil {
			os.WriteFile(path, existing, 0o644)
		} else {
			os.Remove(path)
		}
	})

	tm := New()
	tm.Set("dracula")
	tm.Save()

	tm2 := New()
	tm2.Load()
	if tm2.Current() != "dracula" {
		t.Errorf("expected dracula after load, got %s", tm2.Current())
	}
}

func TestLoadNonexistentConfig(t *testing.T) {
	path := configPath()

	existing, existErr := os.ReadFile(path)
	t.Cleanup(func() {
		if existErr == nil {
			os.WriteFile(path, existing, 0o644)
		} else {
			os.Remove(path)
		}
	})

	os.Remove(path)

	tm := New()
	tm.Load()
	if tm.Current() != "monokai" {
		t.Errorf("expected default monokai after missing config, got %s", tm.Current())
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	path := configPath()

	existing, existErr := os.ReadFile(path)
	t.Cleanup(func() {
		if existErr == nil {
			os.WriteFile(path, existing, 0o644)
		} else {
			os.Remove(path)
		}
	})

	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte("not valid json {{{"), 0o644)

	tm := New()
	tm.Load()
	if tm.Current() != "monokai" {
		t.Errorf("expected default monokai after invalid JSON, got %s", tm.Current())
	}
}

func TestAvailableContainsCatppuccin(t *testing.T) {
	tm := New()
	available := tm.Available()

	catppuccin := []string{
		"catppuccin-mocha",
		"catppuccin-latte",
		"catppuccin-frappe",
		"catppuccin-macchiato",
	}

	for _, want := range catppuccin {
		found := false
		for _, got := range available {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected Available() to contain %q", want)
		}
	}
}

func TestStyleHelpers(t *testing.T) {
	tm := New()

	if got := tm.BorderStyle().Render("test"); got == "" {
		t.Error("BorderStyle().Render() returned empty string")
	}
	if got := tm.CursorStyle().Render("test"); got == "" {
		t.Error("CursorStyle().Render() returned empty string")
	}
	if got := tm.DimStyle().Render("test"); got == "" {
		t.Error("DimStyle().Render() returned empty string")
	}
	if got := tm.StatusStyle().Render("test"); got == "" {
		t.Error("StatusStyle().Render() returned empty string")
	}
}
