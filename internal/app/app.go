package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lantzbuilds/xylem/internal/definition"
	"github.com/lantzbuilds/xylem/internal/finder"
	"github.com/lantzbuilds/xylem/internal/globalsearch"
	"github.com/lantzbuilds/xylem/internal/preview"
	"github.com/lantzbuilds/xylem/internal/statusbar"
	"github.com/lantzbuilds/xylem/internal/theme"
	itree "github.com/lantzbuilds/xylem/internal/tree"
	"github.com/lantzbuilds/xylem/internal/watcher"
)

type flashTickMsg struct{}
type fileChangedMsg struct{}

type defResultMsg struct {
	symbol  string
	results []definition.Result
}

type editorFinishedMsg struct {
	path string
	err  error
}

const (
	focusTree         = 0
	focusPreview      = 1
	focusFinder       = 2
	focusGlobalSearch = 3
)

const treePct = 0.30

type App struct {
	tree         itree.Model
	preview      preview.Model
	finder       finder.Model
	globalSearch globalsearch.Model
	statusbar    statusbar.Model
	theme     *theme.Manager
	focus     int
	width     int
	height    int
	rootPath    string
	showHelp    bool
	fullScreen  bool
	flashMsg    string
	flashTicks  int
	searchMode  bool
	searchBuf   string
	watcher    *watcher.Watcher
	gotoDefMode bool
	gotoDefBuf  string
	jumpStack   []jumpLocation
}

type jumpLocation struct {
	file   string
	offset int
}

func NewApp(path string, noLines bool, themeName string, version string) App {
	tm := theme.New()
	tm.Load()
	if themeName != "" {
		tm.Set(themeName)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	a := App{
		rootPath: absPath,
		theme:    tm,
		focus:    focusTree,
	}

	a.tree = itree.NewModel(absPath, 40, 20)
	a.preview = preview.NewModel(80, 20, tm.Current())
	if noLines {
		a.preview = a.preview.ToggleLineNumbers()
	}
	a.statusbar = statusbar.New(120, filepath.Base(absPath), version).SetTheme(tm.Current())
	a.finder = finder.New(nil, 120, 20)
	a.globalSearch = globalsearch.New(120, 40, absPath)

	if w, err := watcher.New(absPath); err == nil {
		a.watcher = w
	}

	return a
}

func (a App) Init() tea.Cmd {
	cmds := []tea.Cmd{a.tree.Init()}
	if a.watcher != nil {
		cmds = append(cmds, a.waitForChange())
	}
	return tea.Batch(cmds...)
}

func (a App) waitForChange() tea.Cmd {
	return func() tea.Msg {
		if a.watcher == nil {
			return nil
		}
		_, ok := <-a.watcher.Events
		if !ok {
			return nil
		}
		return fileChangedMsg{}
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editorFinishedMsg:
		if msg.path != "" {
			a.preview = a.preview.LoadFile(msg.path)
		}
		return a, nil

	case defResultMsg:
		return a.handleDefResult(msg)

	case fileChangedMsg:
		a.tree = a.tree.Refresh()
		if path := a.preview.FilePath(); path != "" {
			a.preview = a.preview.LoadFile(path)
		}
		return a, a.waitForChange()

	case flashTickMsg:
		a.flashTicks--
		if a.flashTicks <= 0 {
			a.flashMsg = ""
			return a, nil
		}
		return a, tea.Tick(time.Second, func(_ time.Time) tea.Msg { return flashTickMsg{} })

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a.resize(), nil

	case tea.KeyPressMsg:
		// Search input mode — capture all keystrokes
		if a.searchMode {
			switch msg.String() {
			case "escape", "esc":
				a.searchMode = false
				a.searchBuf = ""
				return a, nil
			case "enter":
				query := a.searchBuf
				a.searchMode = false
				a.searchBuf = ""
				if query != "" {
					a.preview = a.preview.Search(query)
					count := a.preview.SearchMatchCount()
					if count == 0 {
						a, cmd := a.flash("no matches")
						return a, cmd
					}
					a, cmd := a.flash(fmt.Sprintf("%d match(es)", count))
					return a, cmd
				}
				return a, nil
			case "backspace":
				if len(a.searchBuf) > 0 {
					a.searchBuf = a.searchBuf[:len(a.searchBuf)-1]
				}
				return a, nil
			default:
				if msg.Text != "" {
					a.searchBuf += msg.Text
				}
				return a, nil
			}
		}

		// Go-to-definition input mode
		if a.gotoDefMode {
			switch msg.String() {
			case "escape", "esc":
				a.gotoDefMode = false
				a.gotoDefBuf = ""
				return a, nil
			case "enter":
				symbol := a.gotoDefBuf
				a.gotoDefMode = false
				a.gotoDefBuf = ""
				if symbol != "" {
					return a.gotoDef(symbol)
				}
				return a, nil
			case "backspace":
				if len(a.gotoDefBuf) > 0 {
					a.gotoDefBuf = a.gotoDefBuf[:len(a.gotoDefBuf)-1]
				}
				return a, nil
			default:
				if msg.Text != "" {
					a.gotoDefBuf += msg.Text
				}
				return a, nil
			}
		}

		// Route to global search when active
		if a.focus == focusGlobalSearch {
			updated, cmd := a.globalSearch.Update(msg)
			a.globalSearch = updated
			if !a.globalSearch.Active() {
				a.focus = focusTree
			}
			return a, cmd
		}

		// Route to finder when active
		if a.focus == focusFinder {
			updated, cmd := a.finder.Update(msg)
			a.finder = updated
			if !a.finder.Active() {
				a.focus = focusTree
			}
			return a, cmd
		}

		if a.showHelp {
			switch msg.String() {
			case "?", "esc", "q", "ctrl+c":
				if msg.String() == "q" || msg.String() == "ctrl+c" {
					if a.watcher != nil {
						a.watcher.Close()
					}
					return a, tea.Quit
				}
				a.showHelp = false
				return a, nil
			}
			return a, nil
		}

		// When search results are active, n/N navigate matches
		if a.preview.SearchMatchCount() > 0 {
			switch msg.String() {
			case "n":
				a.preview = a.preview.NextMatch()
				idx := a.preview.SearchIndex() + 1
				total := a.preview.SearchMatchCount()
				a, cmd := a.flash(fmt.Sprintf("match %d/%d", idx, total))
				return a, cmd
			case "N":
				a.preview = a.preview.PrevMatch()
				idx := a.preview.SearchIndex() + 1
				total := a.preview.SearchMatchCount()
				a, cmd := a.flash(fmt.Sprintf("match %d/%d", idx, total))
				return a, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			if a.watcher != nil {
				a.watcher.Close()
			}
			return a, tea.Quit
		case "tab":
			if a.focus == focusTree {
				a.focus = focusPreview
			} else {
				a.focus = focusTree
			}
			return a, nil
		case "#":
			a.preview = a.preview.ToggleLineNumbers()
			return a, nil
		case "w":
			a.preview = a.preview.ToggleWordWrap()
			return a, nil
		case "t":
			name := a.theme.Next()
			a.theme.Save()
			a.preview = a.preview.SetTheme(name)
			a.statusbar = a.statusbar.SetTheme(name)
			return a, nil
		case "/":
			if a.focus == focusPreview {
				a.searchMode = true
				a.searchBuf = ""
				a.preview = a.preview.ClearSearch()
				return a, nil
			}
			a.globalSearch = a.globalSearch.SetSize(a.width, a.height)
			a.globalSearch = a.globalSearch.Open()
			a.focus = focusGlobalSearch
			return a, nil
		case "ctrl+p":
			files := a.tree.FilePaths()
			a.finder = a.finder.SetFiles(files)
			a.finder = a.finder.SetSize(a.width, a.height)
			a.finder = a.finder.Open()
			a.focus = focusFinder
			return a, nil
		case "?":
			a.showHelp = !a.showHelp
			return a, nil
		case "r":
			updated, cmd := a.tree.Update(msg)
			a.tree = updated.(itree.Model)
			if path := a.preview.FilePath(); path != "" {
				a.preview = a.preview.LoadFile(path)
			}
			return a, cmd
		}

		if a.focus == focusTree {
			switch msg.String() {
			case "enter":
				path := a.tree.SelectedPath()
				info, err := os.Stat(path)
				if err == nil && !info.IsDir() {
					a.fullScreen = true
					a.focus = focusPreview
					a.preview = a.preview.SetSize(a.width, a.height-1)
					return a, nil
				}
			}
			updated, cmd := a.tree.Update(msg)
			a.tree = updated.(itree.Model)
			return a, cmd
		}
		if a.focus == focusPreview {
			switch msg.String() {
			case "esc":
				if a.preview.SearchMatchCount() > 0 {
					a.preview = a.preview.ClearSearch()
					return a, nil
				}
				if a.fullScreen {
					a.fullScreen = false
					a.focus = focusTree
					a = a.resize()
					return a, nil
				}
			case "g":
				a.preview = a.preview.GotoTop()
				return a, nil
			case "G":
				a.preview = a.preview.GotoBottom()
				return a, nil
			case "m":
				if a.preview.IsMarkdown() {
					a.preview = a.preview.ToggleRenderedMode()
					mode := "source"
					if a.preview.RenderedMode() {
						mode = "rendered"
					}
					a, cmd := a.flash("markdown: " + mode)
					return a, cmd
				}
			case "o":
				if path := a.preview.FilePath(); path != "" {
					cmd := openNative(path)
					if cmd != nil {
						if err := cmd.Start(); err != nil {
							a, c := a.flash("open failed: " + err.Error())
							return a, c
						}
					}
					a, c := a.flash("opened: " + filepath.Base(path))
					return a, c
				}
			case "x":
				if a.preview.IsMarkdown() {
					outPath, err := a.preview.ExportHTML()
					if err != nil {
						a, c := a.flash("export failed: " + err.Error())
						return a, c
					}
					a, c := a.flash("exported: " + filepath.Base(outPath))
					return a, c
				}
			case "e":
				if path := a.preview.FilePath(); path != "" {
					editor := os.Getenv("EDITOR")
					if editor == "" {
						editor = "nano"
					}
					c := tea.ExecProcess(exec.Command(editor, path), func(err error) tea.Msg {
						return editorFinishedMsg{path: path, err: err}
					})
					return a, c
				}
			case "y":
				if err := a.preview.CopyFile(); err != nil {
					a, cmd := a.flash("copy failed: " + err.Error())
					return a, cmd
				}
				a, cmd := a.flash("copied file to clipboard")
				return a, cmd
			case "Y":
				if err := a.preview.CopyVisible(); err != nil {
					a, cmd := a.flash("copy failed: " + err.Error())
					return a, cmd
				}
				a, cmd := a.flash("copied visible lines to clipboard")
				return a, cmd
			case "ctrl+]":
				a.gotoDefMode = true
				a.gotoDefBuf = ""
				return a, nil
			case "ctrl+o":
				return a.jumpBack()
			}
			updated, cmd := a.preview.Update(msg)
			a.preview = updated
			return a, cmd
		}

	case globalsearch.ResultSelectedMsg:
		a.focus = focusPreview
		fullPath := filepath.Join(a.rootPath, msg.File)
		a.preview = a.preview.LoadFile(fullPath)
		a.tree = a.tree.NavigateTo(fullPath)
		rel, _ := filepath.Rel(a.rootPath, fullPath)
		a.statusbar = a.statusbar.SetFile(
			rel,
			a.preview.Language(),
			a.preview.LineCount(),
		)
		if msg.Query != "" {
			a.preview = a.preview.Search(msg.Query)
		}
		return a, nil

	case globalsearch.SearchDoneMsg:
		updated, cmd := a.globalSearch.Update(msg)
		a.globalSearch = updated
		return a, cmd

	case finder.FileFoundMsg:
		a.focus = focusTree
		fullPath := filepath.Join(a.rootPath, msg.Path)
		a.preview = a.preview.LoadFile(fullPath)
		a.tree = a.tree.NavigateTo(fullPath)
		rel, _ := filepath.Rel(a.rootPath, fullPath)
		a.statusbar = a.statusbar.SetFile(
			rel,
			a.preview.Language(),
			a.preview.LineCount(),
		)
		return a, nil

	case itree.FileSelectedMsg:
		if !msg.IsDir {
			a.preview = a.preview.LoadFile(msg.Path)
			rel, _ := filepath.Rel(a.rootPath, msg.Path)
			a.statusbar = a.statusbar.SetFile(
				rel,
				a.preview.Language(),
				a.preview.LineCount(),
			)
		}
		return a, nil

	case tea.MouseMsg:
		mouse := msg.Mouse()
		treeW := int(float64(a.width) * treePct)
		if mouse.X < treeW {
			a.focus = focusTree
			updated, cmd := a.tree.Update(msg)
			a.tree = updated.(itree.Model)
			return a, cmd
		} else {
			a.focus = focusPreview
			updated, cmd := a.preview.Update(msg)
			a.preview = updated
			return a, cmd
		}
	}

	return a, nil
}

func (a App) buildTabBar(treeW, previewW int) string {
	activeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Padding(0, 1)
	inactiveStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("245")).
		Padding(0, 1)

	treeLabel := inactiveStyle.Render("tree")
	previewLabel := inactiveStyle.Render("preview")
	if a.focus == focusTree {
		treeLabel = activeStyle.Render("tree")
	} else {
		previewLabel = activeStyle.Render("preview")
	}

	bgStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("236"))

	treeGap := treeW - lipgloss.Width(treeLabel)
	if treeGap < 0 {
		treeGap = 0
	}
	previewGap := previewW - lipgloss.Width(previewLabel)
	if previewGap < 0 {
		previewGap = 0
	}

	return treeLabel + bgStyle.Render(strings.Repeat(" ", treeGap)) +
		" " +
		previewLabel + bgStyle.Render(strings.Repeat(" ", previewGap))
}

func (a App) buildStatusLine() string {
	if a.searchMode {
		searchStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252"))
		promptStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("62")).
			Bold(true)
		line := promptStyle.Render(" / ") + searchStyle.Render(a.searchBuf+"█")
		gap := a.width - lipgloss.Width(line)
		if gap > 0 {
			line += searchStyle.Render(strings.Repeat(" ", gap))
		}
		return line
	}

	if a.gotoDefMode {
		searchStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252"))
		promptStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("114")).
			Bold(true)
		line := promptStyle.Render(" gd: ") + searchStyle.Render(a.gotoDefBuf+"█")
		gap := a.width - lipgloss.Width(line)
		if gap > 0 {
			line += searchStyle.Render(strings.Repeat(" ", gap))
		}
		return line
	}

	if a.flashMsg != "" {
		flashStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1)
		return flashStyle.Render(a.flashMsg) +
			strings.Repeat(" ", max(0, a.width-lipgloss.Width(a.flashMsg)-2))
	}

	return a.statusbar.View()
}

func openNative(path string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path)
	case "linux":
		return exec.Command("xdg-open", path)
	case "windows":
		return exec.Command("cmd", "/c", "start", path)
	}
	return nil
}

func (a App) gotoDef(symbol string) (App, tea.Cmd) {
	root := a.rootPath
	cmd := func() tea.Msg {
		results := definition.Search(root, symbol)
		return defResultMsg{symbol: symbol, results: results}
	}
	a.flashMsg = "searching: " + symbol + "..."
	a.flashTicks = 0
	return a, cmd
}

func (a App) handleDefResult(msg defResultMsg) (App, tea.Cmd) {
	if len(msg.results) == 0 {
		return a.flash("no definition found: " + msg.symbol)
	}

	a.jumpStack = append(a.jumpStack, jumpLocation{
		file:   a.preview.FilePath(),
		offset: a.preview.ScrollOffset(),
	})

	if len(msg.results) == 1 {
		return a.jumpTo(msg.results[0], msg.symbol)
	}

	var globalResults []globalsearch.FlatResult
	for _, r := range msg.results {
		globalResults = append(globalResults, globalsearch.FlatResult{
			File: r.File,
			Line: r.Line,
			Text: r.Text,
		})
	}
	a.globalSearch = a.globalSearch.SetSize(a.width, a.height)
	a.globalSearch = a.globalSearch.OpenWithResults(msg.symbol, globalResults)
	a.focus = focusGlobalSearch
	return a, nil
}

func (a App) jumpTo(r definition.Result, symbol string) (App, tea.Cmd) {
	fullPath := filepath.Join(a.rootPath, r.File)
	a.preview = a.preview.LoadFile(fullPath)
	a.preview = a.preview.Search(symbol)
	a.preview = a.preview.GotoLine(r.Line)
	a.tree = a.tree.NavigateTo(fullPath)
	a.statusbar = a.statusbar.SetFile(
		r.File,
		a.preview.Language(),
		a.preview.LineCount(),
	)
	return a.flash(fmt.Sprintf("→ %s:%d", r.File, r.Line+1))
}

func (a App) jumpBack() (App, tea.Cmd) {
	if len(a.jumpStack) == 0 {
		return a.flash("no jump history")
	}
	loc := a.jumpStack[len(a.jumpStack)-1]
	a.jumpStack = a.jumpStack[:len(a.jumpStack)-1]

	if loc.file != "" {
		a.preview = a.preview.LoadFile(loc.file)
		a.preview = a.preview.GotoLine(loc.offset)
		a.tree = a.tree.NavigateTo(loc.file)
		rel, _ := filepath.Rel(a.rootPath, loc.file)
		a.statusbar = a.statusbar.SetFile(
			rel,
			a.preview.Language(),
			a.preview.LineCount(),
		)
	}
	return a.flash("← back")
}

func (a App) flash(msg string) (App, tea.Cmd) {
	a.flashMsg = msg
	a.flashTicks = 3
	return a, tea.Tick(time.Second, func(_ time.Time) tea.Msg { return flashTickMsg{} })
}

func (a App) resize() App {
	contentHeight := a.height - 2
	treeW := int(float64(a.width) * treePct)
	previewW := a.width - treeW - 1

	a.tree = a.tree.SetSize(treeW, contentHeight)
	a.preview = a.preview.SetSize(previewW, contentHeight)
	a.statusbar = a.statusbar.SetWidth(a.width)

	return a
}

func (a App) View() tea.View {
	v := tea.View{
		AltScreen: true,
		MouseMode: tea.MouseModeCellMotion,
	}

	if a.width == 0 {
		v.Content = "loading..."
		return v
	}

	if a.fullScreen {
		previewStyle := lipgloss.NewStyle().
			Width(a.width).
			Height(a.height - 1)
		fsStatus := a.buildStatusLine()
		v.Content = previewStyle.Render(a.preview.View()) + "\n" + fsStatus
		return v
	}

	contentHeight := a.height - 2
	treeW := int(float64(a.width) * treePct)
	previewW := a.width - treeW - 1

	treeBorderColor := "238"
	previewBorderColor := "238"
	if a.focus == focusTree {
		treeBorderColor = "62"
	} else {
		previewBorderColor = "62"
	}

	treeBorder := lipgloss.NewStyle().
		Width(treeW).
		Height(contentHeight).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(treeBorderColor))

	previewStyle := lipgloss.NewStyle().
		Width(previewW).
		Height(contentHeight).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(previewBorderColor))

	treeView := clipLines(treeBorder.Render(a.tree.ViewString()), contentHeight)
	previewView := clipLines(previewStyle.Render(a.preview.View()), contentHeight)

	tabBar := a.buildTabBar(treeW, previewW)
	content := lipgloss.JoinHorizontal(lipgloss.Top, treeView, previewView)
	statusLine := a.buildStatusLine()
	result := tabBar + "\n" + content + "\n" + statusLine

	if a.showHelp {
		helpView := a.helpView()
		result = placeCentered(result, helpView, a.width, a.height)
	}

	if a.finder.Active() {
		finderView := a.finder.View()
		result = placeCentered(result, finderView, a.width, a.height)
	}

	if a.globalSearch.Active() {
		gsView := a.globalSearch.View()
		result = placeCentered(result, gsView, a.width, a.height)
	}

	v.Content = result
	return v
}

func (a App) helpView() string {
	help := `
  Key Bindings

  Global
  ──────────────────────
  Tab       Switch focus
  Ctrl+p    Find file by name
  #         Line numbers
  w         Word wrap
  t         Cycle theme
  r         Refresh
  ?         This help
  q         Quit

  Tree
  ──────────────────────
  j/k ↑/↓   Navigate
  h/l ←/→   Collapse/Expand
  Enter     Toggle dir / view file
  /         Search all files
  g/G       Top / Bottom

  Preview
  ──────────────────────
  j/k ↑/↓       Scroll
  Ctrl+u/d      Half page
  PgUp/PgDn     Half page
  g/G           Top / Bottom
  /             Search in file
  n/N           Next / prev match
  Esc           Clear search
  m             Toggle markdown rendered/source
  e             Edit in $EDITOR
  x             Export markdown to HTML
  o             Open in native viewer
  y             Copy file to clipboard
  Y             Copy visible to clipboard
  Ctrl+]        Go to definition
  Ctrl+o        Jump back
`
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)

	return style.Render(help)
}

func (a App) collectFiles() []string {
	var files []string
	filepath.Walk(a.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		name := info.Name()
		if info.IsDir() {
			if name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(a.rootPath, path)
		files = append(files, rel)
		return nil
	})
	return files
}

func clipLines(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	for len(lines) < maxLines {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func placeCentered(bg, overlay string, width, height int) string {
	bgLines := strings.Split(bg, "\n")
	ovLines := strings.Split(overlay, "\n")

	ovWidth := lipgloss.Width(overlay)
	startRow := (height - len(ovLines)) / 2
	startCol := (width - ovWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	for i, ovLine := range ovLines {
		row := startRow + i
		if row >= 0 && row < len(bgLines) {
			leftPad := strings.Repeat(" ", startCol)
			rightPad := strings.Repeat(" ", max(0, width-startCol-ovWidth))
			bgLines[row] = leftPad + ovLine + rightPad
		}
	}

	return strings.Join(bgLines, "\n")
}
