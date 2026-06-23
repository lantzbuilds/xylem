# Xylem Handoff — Remaining Work

Current version: **v0.15.0** | Bubble Tea v2 | chafa-go image rendering

## What shipped this cycle

| Version | Feature |
|---|---|
| v0.2.0 | Clipboard copy (y/Y) |
| v0.3.0 | Version in status bar |
| v0.4.0 | In-file + global search (hybrid rg/Go) |
| v0.5.0 | Glamour markdown rendering |
| v0.6.0 | PDF text extraction |
| v0.6.1 | Version from Go build info |
| v0.7.0 | Image metadata, open in native viewer (o) |
| v0.8.0 | Bubble Tea v2 migration |
| v0.9.0 | Search highlighting fix, line numbers default, # toggle |
| v0.10.0 | chafa-go image rendering, search hang fix |
| v0.11.0 | Edit in $EDITOR (e), focus indicator, MD→HTML export (x) |
| v0.11.1 | Tab bar focus indicator, styled HTML export |
| v0.11.2 | Status bar fix, colored border divider restored |
| v0.11.3 | Tab bar moved to top |
| v0.11.4 | Layout clipping fix, full-height border, CSS custom properties |
| v0.12.0 | Fuzzy file finder (Ctrl+p), finder panel rendering fix |
| v0.13.0 | Auto-refresh file watcher via fsnotify (closes #4) |
| v0.14.0 | Go to definition (Ctrl+]) + jump back (Ctrl+o) (closes #13) |
| v0.15.0 | Stable release: async def search, symbol highlighting, watcher fixes |

## Open issues — prioritized

### Tier 1: Completed

**#4 File system watcher for auto-refresh** — shipped v0.13.0
- `internal/watcher` package wraps fsnotify with 200ms debounce
- Recursive directory watching, skips `.git`/`node_modules`/`vendor`/`dist`
- New directories added to watch list dynamically on create
- Search highlighting preserved across watcher-triggered refreshes

**#13 Go to source (function navigation)** — shipped v0.14.0
- `Ctrl+]` opens symbol input prompt, async regex search across codebase
- Supports Go (`func`/`type`/`var`/`const`), Python (`def`/`class`), JS/TS (`function`/`const`/`let`/`class`/`interface`/`type`/`enum`), Rust (`fn`/`struct`/`enum`/`trait`)
- Single result jumps directly with symbol highlighted; multiple results show picker
- `Ctrl+o` pops jump stack to return to previous location
- `internal/definition` package handles pattern matching

### Tier 2: Large scope, high differentiator

**#12 Git diff: side-by-side commit comparison**
- New rendering mode reusing the split-pane layout (old | new)
- Shell out to `git diff` for unified diff, parse into side-by-side columns
- Highlight additions (green), deletions (red)
- Entry point: `d` key when in a git repo, show commit picker
- Could also support: working tree vs HEAD, staged vs unstaged
- Big scope — needs dedicated session

### Tier 3: Blocked or niche

**#8 PDF rasterization via Kitty graphics protocol**
- Blocked by same split-pane ANSI issue as image rendering
- Text extraction (v0.6.0) is the functional workaround
- Would need: Kitty protocol escape sequences that don't conflict with lipgloss border rendering
- Options: shell out to `pdftoppm`, or wait for lipgloss/BT to support graphics layers
- Low urgency — text extraction + `o` to open in Preview covers most cases

**#5 Remote filesystem support**
- SSH, GitHub repos, S3 browsing
- Needs pluggable filesystem abstraction layer
- Major architecture change, unclear demand
- Park until there's a concrete use case driving it

## Known limitations

- **Image split-pane banding**: chafa output interacts with lipgloss border styling, causing horizontal stripes on some images in split view. Fullscreen (Enter) renders cleanly. Root cause: lipgloss wraps each line in ANSI reset/style sequences that interrupt chafa's color state.
- **PDF text extraction quality**: varies by PDF structure. Simple single-column PDFs are readable; multi-column and image-based PDFs produce garbled or no text. Rasterization (#8) would fix this.
- **SVG not supported**: listed in image extensions but Go's `image` package can't decode SVG. Would need rsvg-convert or similar.

## Architecture notes

- **Layout clipping**: `clipLines()` in app.go is the safety valve — clips each pane to exact height after lipgloss rendering. Don't rely on lipgloss `Height()` alone.
- **Search engine**: `HasRipgrep()` caches the result of `exec.LookPath("rg")` at startup. Global search shells out to rg with 10s timeout, falls back to Go `filepath.Walk`.
- **Image rendering**: `chafa.Load()` decodes image, pre-scales large images to ~640px max dimension, then `CanvasDrawAllPixels` renders as Unicode symbols.
- **Markdown mode**: `renderedMode` flag auto-switches to source when search is active, restores on clear.
- **Key routing priority**: searchMode → gotoDefMode → globalSearch/finder (focus-based) → search match navigation → global keys → focus-specific keys.
- **File watcher** (`internal/watcher`): goroutine listens to fsnotify, debounces 200ms, sends `fileChangedMsg` via `tea.Cmd` channel. Watcher initialized in `NewApp()`, not `Init()` — pointer must be set before value-receiver copies diverge.
- **Definition search** (`internal/definition`): regex patterns compiled per-search with `fmt.Sprintf` to inject the escaped symbol. Runs async via `tea.Cmd` to avoid blocking the TUI on large codebases. Results flow back as `defResultMsg`.
- **Jump stack**: `[]jumpLocation` in App stores (file, scrollOffset) pairs. Pushed on `Ctrl+]` jump, popped on `Ctrl+o`.
- **Fuzzy finder** (`internal/finder`): uses `sahilm/fuzzy` (fzf-style scoring). `Ctrl+p` collects file paths via `tree.FilePaths()` which walks the filesystem independently of tree expand state.
- **Build note**: use `go install .` to update the binary on `$GOPATH/bin`. `go build ./...` only verifies compilation.
