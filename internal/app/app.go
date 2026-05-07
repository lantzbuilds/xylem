package app

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lantzbuilds/xylem/internal/finder"
	"github.com/lantzbuilds/xylem/internal/preview"
	"github.com/lantzbuilds/xylem/internal/statusbar"
	"github.com/lantzbuilds/xylem/internal/theme"
	itree "github.com/lantzbuilds/xylem/internal/tree"
)

type flashTickMsg struct{}

const (
	focusTree    = 0
	focusPreview = 1
	focusFinder  = 2
)

const treePct = 0.30

type App struct {
	tree      itree.Model
	preview   preview.Model
	finder    finder.Model
	statusbar statusbar.Model
	theme     *theme.Manager
	focus     int
	width     int
	height    int
	rootPath    string
	showHelp    bool
	fullScreen  bool
	flashMsg    string
	flashTicks  int
}

func NewApp(path string, showLines bool, themeName string) App {
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
	if showLines {
		a.preview = a.preview.ToggleLineNumbers()
	}
	a.statusbar = statusbar.New(120, filepath.Base(absPath)).SetTheme(tm.Current())
	a.finder = finder.New(nil, 120, 20)

	return a
}

func (a App) Init() tea.Cmd {
	return a.tree.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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

	case tea.KeyMsg:
		// Route to finder first when active, so keystrokes aren't
		// intercepted by global shortcuts (e.g. typing 'q' or 't').
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
					return a, tea.Quit
				}
				a.showHelp = false
				return a, nil
			}
			return a, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "tab":
			if a.focus == focusTree {
				a.focus = focusPreview
			} else {
				a.focus = focusTree
			}
			return a, nil
		case "n":
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
			a.finder = a.finder.SetFiles(a.collectFiles())
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
			}
			updated, cmd := a.preview.Update(msg)
			a.preview = updated
			return a, cmd
		}

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
		treeW := int(float64(a.width) * treePct)
		if msg.X < treeW {
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

func (a App) flash(msg string) (App, tea.Cmd) {
	a.flashMsg = msg
	a.flashTicks = 3
	return a, tea.Tick(time.Second, func(_ time.Time) tea.Msg { return flashTickMsg{} })
}

func (a App) resize() App {
	contentHeight := a.height - 1
	treeW := int(float64(a.width) * treePct)
	previewW := a.width - treeW - 1

	a.tree = a.tree.SetSize(treeW, contentHeight)
	a.preview = a.preview.SetSize(previewW, contentHeight)
	a.statusbar = a.statusbar.SetWidth(a.width)

	return a
}

func (a App) View() string {
	if a.width == 0 {
		return "loading..."
	}

	if a.fullScreen {
		previewStyle := lipgloss.NewStyle().
			Width(a.width).
			Height(a.height - 1)
		fsStatus := a.statusbar.View()
		if a.flashMsg != "" {
			flashStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("62")).
				Bold(true).
				Padding(0, 1)
			fsStatus = flashStyle.Render(a.flashMsg) +
				strings.Repeat(" ", max(0, a.width-lipgloss.Width(a.flashMsg)-2))
		}
		return previewStyle.Render(a.preview.View()) + "\n" + fsStatus
	}

	contentHeight := a.height - 1
	treeW := int(float64(a.width) * treePct)
	previewW := a.width - treeW - 1

	var treeBorderColor, previewBorderColor string
	if a.focus == focusTree {
		treeBorderColor = "62"
		previewBorderColor = "238"
	} else {
		treeBorderColor = "238"
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

	treeView := treeBorder.Render(a.tree.View())
	previewView := previewStyle.Render(a.preview.View())

	focusName := "tree"
	if a.focus == focusPreview {
		focusName = "preview"
	}
	sb := a.statusbar.SetFocus(focusName)

	content := lipgloss.JoinHorizontal(lipgloss.Top, treeView, previewView)
	statusLine := sb.View()
	if a.flashMsg != "" {
		flashStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1)
		statusLine = flashStyle.Render(a.flashMsg) +
			strings.Repeat(" ", max(0, a.width-lipgloss.Width(a.flashMsg)-2))
	}
	result := content + "\n" + statusLine

	if a.showHelp {
		helpView := a.helpView()
		result = placeCentered(result, helpView, a.width, a.height)
	}

	if a.finder.Active() {
		finderView := a.finder.View()
		result = placeCentered(result, finderView, a.width, a.height)
	}

	return result
}

func (a App) helpView() string {
	help := `
  Key Bindings

  Global
  ──────────────────────
  Tab       Switch focus
  /         Fuzzy finder
  n         Line numbers
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
  g/G       Top / Bottom

  Preview
  ──────────────────────
  j/k ↑/↓       Scroll
  Ctrl+u/d      Half page
  PgUp/PgDn     Half page
  g/G           Top / Bottom
  y             Copy file to clipboard
  Y             Copy visible to clipboard
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

func placeCentered(bg, overlay string, width, height int) string {
	bgLines := strings.Split(bg, "\n")
	ovLines := strings.Split(overlay, "\n")

	startRow := (height - len(ovLines)) / 2
	startCol := (width - lipgloss.Width(overlay)) / 2
	if startCol < 0 {
		startCol = 0
	}

	for i, ovLine := range ovLines {
		row := startRow + i
		if row >= 0 && row < len(bgLines) {
			pad := ""
			if startCol > 0 {
				pad = strings.Repeat(" ", startCol)
			}
			bgLines[row] = pad + ovLine
		}
	}

	return strings.Join(bgLines, "\n")
}
