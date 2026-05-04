package app

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lantzbuilds/xylem/internal/preview"
	"github.com/lantzbuilds/xylem/internal/statusbar"
	"github.com/lantzbuilds/xylem/internal/theme"
	itree "github.com/lantzbuilds/xylem/internal/tree"
)

const (
	focusTree    = 0
	focusPreview = 1
	focusFinder  = 2
)

const treePct = 0.30

type App struct {
	tree      itree.Model
	preview   preview.Model
	statusbar statusbar.Model
	theme     *theme.Manager
	focus     int
	width     int
	height    int
	rootPath  string
}

func NewApp(path string, showLines bool, themeName string) App {
	tm := theme.New()
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
	a.statusbar = statusbar.New(120).SetTheme(tm.Current())

	return a
}

func (a App) Init() tea.Cmd {
	return a.tree.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a.resize(), nil

	case tea.KeyMsg:
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
		case "t":
			name := a.theme.Next()
			a.preview = a.preview.SetTheme(name)
			a.statusbar = a.statusbar.SetTheme(name)
			return a, nil
		}

		if a.focus == focusTree {
			updated, cmd := a.tree.Update(msg)
			a.tree = updated.(itree.Model)
			return a, cmd
		}
		if a.focus == focusPreview {
			updated, cmd := a.preview.Update(msg)
			a.preview = updated
			return a, cmd
		}

	case itree.FileSelectedMsg:
		if !msg.IsDir {
			a.preview = a.preview.LoadFile(msg.Path)
			a.statusbar = a.statusbar.SetFile(
				filepath.Base(msg.Path),
				a.preview.Language(),
				a.preview.LineCount(),
			)
		}
		return a, nil
	}

	return a, nil
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

	contentHeight := a.height - 1
	treeW := int(float64(a.width) * treePct)
	previewW := a.width - treeW - 1

	treeBorder := lipgloss.NewStyle().
		Width(treeW).
		Height(contentHeight).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("238"))

	previewStyle := lipgloss.NewStyle().
		Width(previewW).
		Height(contentHeight)

	treeView := treeBorder.Render(a.tree.View())
	previewView := previewStyle.Render(a.preview.View())

	content := lipgloss.JoinHorizontal(lipgloss.Top, treeView, previewView)
	return content + "\n" + a.statusbar.View()
}
