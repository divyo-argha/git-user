package tui

import (
	"fmt"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// pendingAction captures what to do after the TUI exits (for ops that need raw terminal).
type pendingAction struct {
	kind string
	name string
	arg  string
}

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

	// For actions that must run outside the TUI
	quit   bool
	action *pendingAction
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

	case core.ActionResultMsg:
		return a.handleAction(msg)

	case tea.KeyMsg:
		if msg.String() == "ctrl+p" {
			return a, a.pushScreen(screens.NewCommandPalette(a.theme))
		}
		if msg.String() == "?" {
			if _, isPalette := a.activeScreen().(*screens.CommandPalette); !isPalette {
				if _, isHelp := a.activeScreen().(*screens.HelpModal); !isHelp {
					return a, a.pushScreen(screens.NewHelpModal(a.theme))
				}
				}
		}
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

// ── Action Handling ───────────────────────────────────────────────────────────

func (a *App) handleAction(msg core.ActionResultMsg) (tea.Model, tea.Cmd) {
	switch msg.Kind {
	case "quit":
		a.quit = true
		return a, tea.Quit

	case "help":
		return a, a.pushScreen(screens.NewHelpModal(a.theme))

	case "register":
		return a, func() tea.Msg {
			return core.ScreenPushMsg{Screen: screens.NewForm(
				"Register New Identity",
				"Enter profile name and email address",
				"register",
				[]screens.FormInput{
					{Label: "Profile Name:", Placeholder: "e.g. work"},
					{Label: "Email Address:", Placeholder: "e.g. you@company.com"},
				},
				a.theme,
			)}
		}

	case "switch":
		a.action = &pendingAction{kind: "switch", name: msg.Name}
		return a, tea.Quit

	case "rename":
		return a, func() tea.Msg {
			return core.ScreenPushMsg{Screen: screens.NewForm(
				"Rename Identity",
				"Enter new profile name for "+msg.Name,
				"rename:"+msg.Name,
				[]screens.FormInput{
					{Label: "New Name:", Value: msg.Name},
				},
				a.theme,
			)}
		}

	case "email":
		u := a.store.FindUser(msg.Name)
		currentEmail := ""
		if u != nil {
			currentEmail = u.Email
		}
		return a, func() tea.Msg {
			return core.ScreenPushMsg{Screen: screens.NewForm(
				"Change Email",
				"Enter new email address for "+msg.Name,
				"email:"+msg.Name,
				[]screens.FormInput{
					{Label: "New Email:", Value: currentEmail},
				},
				a.theme,
			)}
		}

	case "pubkey":
		a.action = &pendingAction{kind: "pubkey", name: msg.Name}
		return a, tea.Quit

	case "pubkey-push":
		a.action = &pendingAction{kind: "pubkey-push", name: msg.Name}
		return a, tea.Quit

	case "bind":
		a.action = &pendingAction{kind: "bind", name: msg.Name}
		return a, tea.Quit

	case "check-ssh":
		a.action = &pendingAction{kind: "check-ssh", name: msg.Name}
		return a, tea.Quit

	case "unbind":
		return a, func() tea.Msg {
			return core.ScreenPushMsg{Screen: screens.NewConfirm(
				fmt.Sprintf("Remove SSH key binding from %q? (file not deleted)", msg.Name),
				"unbind:"+msg.Name,
				a.theme,
			)}
		}

	case "rekey":
		a.action = &pendingAction{kind: "rekey", name: msg.Name}
		return a, tea.Quit

	case "passphrase":
		a.action = &pendingAction{kind: "passphrase", name: msg.Name}
		return a, tea.Quit

	case "bind-path":
		a.action = &pendingAction{kind: "bind-path", name: msg.Name}
		return a, tea.Quit

	case "unbind-path":
		a.action = &pendingAction{kind: "unbind-path", name: msg.Name}
		return a, tea.Quit

	case "export":
		a.action = &pendingAction{kind: "export", name: msg.Name}
		return a, tea.Quit

	case "import-export":
		return a, func() tea.Msg {
			return core.ScreenPushMsg{Screen: screens.NewImportExport(a.store, a.theme)}
		}

	case "export-current":
		if a.store.Current == "" {
			return a, core.ShowToastCmd("No active identity — switch to one first", theme.ToastStyleError, 3*time.Second)
		}
		a.action = &pendingAction{kind: "export-current", name: a.store.Current}
		return a, tea.Quit

	case "export-all":
		a.action = &pendingAction{kind: "export-all"}
		return a, tea.Quit

	case "import":
		a.action = &pendingAction{kind: "import"}
		return a, tea.Quit

	case "import-original":
		a.action = &pendingAction{kind: "import-original"}
		return a, tea.Quit

	case "remove":
		return a, func() tea.Msg {
			return core.ScreenPushMsg{Screen: screens.NewConfirm(
				fmt.Sprintf("Remove identity %q? This cannot be undone.", msg.Name),
				"remove:"+msg.Name,
				a.theme,
			)}
		}

	case "logout":
		a.action = &pendingAction{kind: "logout"}
		return a, tea.Quit

	case "fix-remote":
		a.action = &pendingAction{kind: "fix-remote"}
		return a, tea.Quit

	case "security":
		a.action = &pendingAction{kind: "security"}
		return a, tea.Quit

	case "doctor":
		a.action = &pendingAction{kind: "doctor"}
		return a, tea.Quit

	case "update":
		a.action = &pendingAction{kind: "update"}
		return a, tea.Quit
	}

	return a, nil
}

func (a *App) handleConfirmResult(msg core.ConfirmResultMsg) (tea.Model, tea.Cmd) {
	if !msg.Confirmed {
		return a, core.ShowToastCmd("Cancelled", theme.ToastStyleInfo, 2*time.Second)
	}

	parts := strings.SplitN(msg.Context, ":", 2)
	if len(parts) != 2 {
		return a, nil
	}

	action := parts[0]
	name := parts[1]

	switch action {
	case "remove":
		a.action = &pendingAction{kind: "remove", name: name}
		return a, tea.Quit
	case "unbind":
		a.action = &pendingAction{kind: "unbind", name: name}
		return a, tea.Quit
	}

	return a, nil
}

func (a *App) handleFormResult(msg core.FormResultMsg) (tea.Model, tea.Cmd) {
	if len(msg.Values) == 0 {
		return a, nil
	}

	parts := strings.SplitN(msg.Context, ":", 2)
	action := parts[0]
	name := ""
	if len(parts) > 1 {
		name = parts[1]
	}

	switch action {
	case "register":
		if msg.Values[0] == "" || msg.Values[1] == "" {
			return a, core.ShowToastCmd("Profile name and email are required", theme.ToastStyleError, 3*time.Second)
		}
		a.action = &pendingAction{kind: "register", name: msg.Values[0], arg: msg.Values[1]}
		return a, tea.Quit

	case "rename":
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("New name cannot be empty", theme.ToastStyleError, 3*time.Second)
		}
		a.action = &pendingAction{kind: "rename", name: name, arg: msg.Values[0]}
		return a, tea.Quit

	case "email":
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("New email cannot be empty", theme.ToastStyleError, 3*time.Second)
		}
		a.action = &pendingAction{kind: "email", name: name, arg: msg.Values[0]}
		return a, tea.Quit
	}

	return a, nil
}

// ── Results ───────────────────────────────────────────────────────────────────

func (a *App) Quit() bool { return a.quit }

func (a *App) PendingAction() (kind, name, arg string) {
	if a.action == nil {
		return "", "", ""
	}
	return a.action.kind, a.action.name, a.action.arg
}
