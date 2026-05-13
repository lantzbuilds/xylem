# xylem

A terminal file browser with syntax highlighting. Navigate your codebase and preview any file — like `glow`, but for all file types.

## Install

```bash
# Homebrew
brew install lantzbuilds/tap/xylem

# Go
go install github.com/lantzbuilds/xylem@latest
```

Or download a binary from [Releases](https://github.com/lantzbuilds/xylem/releases).

## Usage

```bash
# Browse current directory
xylem

# Browse a specific path
xylem ./src

# Start with line numbers
xylem --lines

# Use a specific theme
xylem --theme dracula
```

## Key Bindings

| Key | Action |
|---|---|
| `Tab` | Switch focus (tree/preview) |
| `j/k` `↑/↓` | Navigate / scroll |
| `h/l` `←/→` | Collapse / expand directory |
| `Enter` | Expand dir / full-screen file |
| `Esc` | Back to split view / clear search |
| `n` | Toggle line numbers |
| `w` | Toggle word wrap |
| `t` | Cycle syntax theme |
| `r` | Refresh directory |
| `?` | Help |
| `q` | Quit |

### Tree (when focused)

| Key | Action |
|---|---|
| `/` | Fuzzy file finder |

### Preview (when focused)

| Key | Action |
|---|---|
| `Ctrl+u` / `Page Up` | Scroll up half-page |
| `Ctrl+d` / `Page Down` | Scroll down half-page |
| `g/G` | Jump to top / bottom |
| `/` | Search in file |
| `n/N` | Next / previous match |
| `Esc` | Clear search results |
| `y` | Copy file to clipboard |
| `Y` | Copy visible lines to clipboard |

## Themes

Cycle with `t`: monokai, dracula, github, solarized-dark, solarized-light, nord, onedark, gruvbox, catppuccin-mocha, catppuccin-latte, catppuccin-frappe, catppuccin-macchiato.

## Features

- Syntax highlighting for 500+ languages via [Chroma](https://github.com/alecthomas/chroma)
- Lazy directory loading
- Respects `.gitignore`
- Binary file detection
- Mouse support
- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)

## License

MIT
