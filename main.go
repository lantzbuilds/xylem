package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lantzbuilds/xylem/internal/app"
)

var version = "dev"

func main() {
	showLines := flag.Bool("lines", false, "Start with line numbers enabled")
	themeName := flag.String("theme", "monokai", "Set syntax theme")
	showVersion := flag.Bool("version", false, "Print version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `xylem - navigate and inspect files with syntax highlighting

Usage:
  xylem [flags] [path]

Flags:
  --lines          Start with line numbers enabled
  --theme <name>   Set syntax theme (default: monokai)
  --version        Print version and exit
  --help           Show this help message

Key Bindings:
  Tab       Switch focus (tree/preview)
  j/k       Navigate / scroll
  h/l       Collapse / expand directory
  /         Search in file (preview) / file finder (tree)
  n/N       Next / prev match (after search)
  n         Toggle line numbers (no active search)
  w         Toggle word wrap
  t         Cycle theme
  r         Refresh directory
  m         Toggle markdown rendered/source (preview)
  y         Copy file to clipboard (preview)
  Y         Copy visible to clipboard (preview)
  Esc       Clear search / exit fullscreen
  ?         Full help overlay
  q         Quit
`)
	}
	flag.Parse()

	var themeExplicit bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "theme" {
			themeExplicit = true
		}
	})

	if *showVersion {
		fmt.Printf("xylem %s\n", version)
		os.Exit(0)
	}

	path := "."
	if flag.NArg() > 0 {
		path = flag.Arg(0)
	}

	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "error: %s is not a directory\n", path)
		os.Exit(1)
	}

	themeArg := ""
	if themeExplicit {
		themeArg = *themeName
	}
	a := app.NewApp(path, *showLines, themeArg, version)
	p := tea.NewProgram(a, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
