# Xylem Handoff — Remaining Work

Current version: **v0.16.0** | Bubble Tea v2 | chafa-go image rendering

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
| v0.16.0 | Gitignored files shown dimmed in tree, sorted to bottom |

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

### Next up: Git integration

**#12 Git diff: side-by-side commit comparison**
- New rendering mode reusing the split-pane layout (old | new)
- Shell out to `git diff` for unified diff, parse into side-by-side columns
- Highlight additions (green), deletions (red)
- Entry point: `d` key when in a git repo, show commit picker
- Could also support: working tree vs HEAD, staged vs unstaged
- Big scope — needs dedicated session

**Git blame annotations**
- Toggle with `b` key — inline gutter showing author, relative time, per line
- Shell out to `git blame --porcelain`, parse into per-line metadata
- Render as dimmed column between line numbers and code
- Complements go-to-definition: "who wrote this" alongside "where is it defined"

**Git status coloring in tree**
- Extends the `Ignored` flag pattern from v0.16.0 to richer git state
- Modified files (green), untracked-but-not-ignored (grey), staged (yellow)
- Derive from `git status --porcelain` at startup, store as node flags
- Natural pairing with blame and diff — the tree becomes git-aware end to end

### Differentiators to explore

**Zero-friction entry points**
- Open a GitHub repo URL directly (`xylem https://github.com/...`)
- Browse a tarball / zip without extracting
- These are things neovim needs plugins and config for

**Richer visual previews**
- Data file rendering: CSV/TSV as aligned columns, JSON/YAML with collapsible sections
- Diagram rendering: Mermaid/D2 to ASCII or sixel
- SVG support (needs rsvg-convert or similar)

### Parked

**#8 PDF rasterization via Kitty graphics protocol**
- Blocked by split-pane ANSI issue with lipgloss border rendering
- Text extraction (v0.6.0) + `o` to open in Preview covers most cases
- Revisit when lipgloss/BT supports graphics layers

**#5 Remote filesystem support**
- SSH, GitHub repos, S3 browsing
- Major architecture change, unclear demand
- Park until there's a concrete use case driving it

## Strategic positioning

Xylem is a **zero-config code browser**, not an editor. The value is instant, visual exploration of any codebase without setup. Features like `e` (open in $EDITOR) are escape hatches, not an attempt to become an editor.

**Where xylem wins over neovim/VS Code:**
- Works on first run — no config, no plugins, no LSP setup
- Visual previews (images, markdown, PDF) that editors don't do well natively
- Purpose-built for reading and navigating, not editing — lower cognitive load

**Where it should not compete:**
- Editing, refactoring, diagnostics — that's what `e` hands off to
- LSP-powered intelligence — regex definition search is "good enough" for browsing; tree-sitter/ctags would be incremental improvement, not worth the dependency
- Plugin ecosystem — xylem is a single binary, keep it that way

**Guiding principle:** if a feature makes sense for someone *reading* code, it belongs. If it only makes sense for someone *writing* code, it doesn't. Git diff and blame are reader features. Multi-cursor editing is not.

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
- **Git-aware tree** (`internal/tree/node.go`): `Node.Ignored` flag marks gitignored entries. Ignored nodes render with `ignoredSty` (color 240), sort to bottom of each directory. Children inherit parent's ignored state. Only `.git/` is fully excluded. This flag pattern extends to future git state coloring (modified, untracked, staged).
- **Build note**: use `go install .` to update the binary on `$GOPATH/bin`. `go build ./...` only verifies compilation.
