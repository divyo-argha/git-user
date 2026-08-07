package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// App is the root tea.Model that coordinates all screens.
type App struct {
	store       *config.Store
	screenStack []core.Screen
	statusBar   components.StatusBar
	helpBar     components.HelpBar
	toast       components.Toast
	animFrame   uint64
	width       int
	height      int
	theme       theme.Theme

	quit          bool
	removeKeyPath string // SSH key path captured before removing an identity
}

func animateTickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return core.AnimTickMsg(t)
	})
}

// NewApp creates the root app model.
func NewApp(store *config.Store, initialScreen core.Screen) *App {
	th := theme.DefaultTheme()
	return &App{
		store:       store,
		screenStack: []core.Screen{initialScreen},
		statusBar:   components.NewStatusBar(store, th),
		helpBar:     components.NewHelpBar(th),
		toast:       components.NewToast(th),
		theme:       th,
	}
}

func (a *App) activeScreen() core.Screen {
	if len(a.screenStack) == 0 {
		return nil
	}
	return a.screenStack[len(a.screenStack)-1]
}

func (a *App) pushScreen(s core.Screen) tea.Cmd {
	a.screenStack = append(a.screenStack, s)
	a.helpBar.SetText(s.ShortHelp())
	return s.Init()
}

func (a *App) popScreen() {
	if len(a.screenStack) > 1 {
		a.screenStack = a.screenStack[:len(a.screenStack)-1]
		if s := a.activeScreen(); s != nil {
			a.helpBar.SetText(s.ShortHelp())
		}
	}
}

func pushCmd(s core.Screen) tea.Cmd {
	return func() tea.Msg { return core.ScreenPushMsg{Screen: s} }
}

// runTaskCmd runs an operation as a background command and reports its result.
func (a *App) runTaskCmd(kind, name string, fn func() (opResult, error)) tea.Cmd {
	return func() tea.Msg {
		res, err := fn()
		return core.TaskResultMsg{
			Kind:       kind,
			Name:       name,
			Success:    err == nil,
			Detail:     res.detail,
			ShowReport: res.showReport,
			Err:        err,
		}
	}
}

// ── tea.Model interface ───────────────────────────────────────────────────────

func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{
		core.CheckAgentCmd(),
		animateTickCmd(),
	}
	if s := a.activeScreen(); s != nil {
		cmds = append(cmds, s.Init())
		a.helpBar.SetText(s.ShortHelp())
	}
	return tea.Batch(cmds...)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case core.AnimTickMsg:
		a.animFrame++
		if s := a.activeScreen(); s != nil {
			newScreen, cmd := s.Update(msg)
			a.screenStack[len(a.screenStack)-1] = newScreen
			return a, tea.Batch(cmd, animateTickCmd())
		}
		return a, animateTickCmd()

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case core.AgentStatusMsg:
		a.statusBar.SetAgentStatus(msg.Connected, msg.KeyCount)
		return a, nil

	case core.StoreRefreshedMsg:
		if msg.Err == nil && msg.Store != nil {
			a.store = msg.Store
			a.statusBar.SetStore(msg.Store)
		}
		if s := a.activeScreen(); s != nil {
			newScreen, cmd := s.Update(msg)
			a.screenStack[len(a.screenStack)-1] = newScreen
			return a, cmd
		}
		return a, nil

	case core.ToastMsg:
		a.toast.Show(msg.Text, msg.Style)
		return a, core.ToastTimerCmd(msg.Duration)

	case core.ToastExpiredMsg:
		a.toast.Hide()
		return a, nil

	case core.ScreenPushMsg:
		cmd := a.pushScreen(msg.Screen)
		return a, cmd

	case core.ScreenPopMsg:
		a.popScreen()
		return a, core.RefreshStoreCmd()

	case core.ConfirmResultMsg:
		a.popScreen()
		return a.handleConfirmResult(msg)

	case core.FormResultMsg:
		a.popScreen()
		return a.handleFormResult(msg)

	case core.OptionResultMsg:
		a.popScreen()
		return a.handleOptionResult(msg)

	case core.TaskResultMsg:
		return a.handleTaskResult(msg)

	case core.ActionResultMsg:
		return a.handleAction(msg)

	case tea.KeyMsg:
		if s := a.activeScreen(); s != nil {
			newScreen, cmd := s.Update(msg)
			a.screenStack[len(a.screenStack)-1] = newScreen
			return a, cmd
		}
		return a, nil

	default:
		if s := a.activeScreen(); s != nil {
			newScreen, cmd := s.Update(msg)
			a.screenStack[len(a.screenStack)-1] = newScreen
			return a, cmd
		}
	}

	return a, nil
}

func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	sb.WriteString("\n")

	statusView := a.statusBar.View(a.width, a.height)
	sb.WriteString(statusView)
	sb.WriteString("\n")

	screenHeight := theme.ContentHeight(a.height)
	if s := a.activeScreen(); s != nil {
		sb.WriteString(s.View(a.width, screenHeight))
	}
	sb.WriteString("\n")

	if a.toast.IsVisible() {
		sb.WriteString(a.toast.View(a.width))
	}

	sb.WriteString("\n")

	sb.WriteString(a.helpBar.View(a.width))
	sb.WriteString("\n")

	return sb.String()
}

// ── Results ───────────────────────────────────────────────────────────────────

// Quit reports whether the user explicitly quit the TUI.
func (a *App) Quit() bool { return a.quit }
