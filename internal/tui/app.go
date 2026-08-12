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
	spinner     components.Spinner
	animFrame   uint64
	width       int
	height      int
	theme       theme.Theme

	quit          bool
	removeKeyPath string // SSH key path captured before removing an identity
	taskRunning   bool   // true while a background task (runTaskCmd) is in flight
	taskLabel     string // human-readable label shown next to the spinner
	refreshTick   uint64 // counts animation frames for the periodic store refresh
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
		spinner:     components.NewSpinner(th),
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
// It sets taskRunning = true before dispatching so the spinner is shown.
func (a *App) runTaskCmd(kind, name string, fn func() (opResult, error)) tea.Cmd {
	a.taskRunning = true
	a.taskLabel = titleForKind(kind)
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
		a.spinner.Init(),
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
		var cmds []tea.Cmd
		cmds = append(cmds, animateTickCmd())
		if s := a.activeScreen(); s != nil {
			newScreen, cmd := s.Update(msg)
			a.screenStack[len(a.screenStack)-1] = newScreen
			cmds = append(cmds, cmd)
		}
		// Periodically reload the store while idle. A long-running session that
		// never refreshes can otherwise resurrect deleted profiles or act on
		// stale state changed by another session or a manual edit.
		a.refreshTick++
		if a.refreshTick%60 == 0 && !a.taskRunning {
			cmds = append(cmds, core.RefreshStoreCmd(), core.CheckSyncStatusCmd(a.store))
		}
		return a, tea.Batch(cmds...)

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
			return a, tea.Batch(cmd, core.CheckSyncStatusCmd(a.store))
		}
		return a, core.CheckSyncStatusCmd(a.store)

	case core.SyncStatusMsg:
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
		// Task completed — clear the loading spinner.
		a.taskRunning = false
		a.taskLabel = ""
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
		var cmds []tea.Cmd
		// Always forward unrecognised messages to the spinner so its TickMsg
		// drives the animation while a background task is in flight.
		if a.taskRunning {
			newSpinner, spinCmd := a.spinner.Update(msg)
			a.spinner = newSpinner
			cmds = append(cmds, spinCmd)
		}
		if s := a.activeScreen(); s != nil {
			newScreen, cmd := s.Update(msg)
			a.screenStack[len(a.screenStack)-1] = newScreen
			cmds = append(cmds, cmd)
		}
		if len(cmds) > 0 {
			return a, tea.Batch(cmds...)
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

	// Show a spinner row while a background task is running.
	if a.taskRunning {
		spinView := a.spinner.View() + " " + a.theme.Dim().Render(a.taskLabel+"…")
		sb.WriteString("  ")
		sb.WriteString(spinView)
		sb.WriteString("\n")
	}

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
