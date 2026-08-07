package tui

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// pendingAction is retained for backward compatibility with Run()'s signature.
// No in-TUI flow sets it anymore: every operation now completes inside the TUI.
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

	quit         bool
	action       *pendingAction
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

// ── Action Handling ───────────────────────────────────────────────────────────

func (a *App) handleAction(msg core.ActionResultMsg) (tea.Model, tea.Cmd) {
	switch msg.Kind {
	case "quit":
		a.quit = true
		return a, tea.Quit

	case "register", "register-temp":
		title := "Register New Identity"
		help := "Enter profile name and email address"
		if msg.Kind == "register-temp" {
			title = "Create Temporary Profile"
			help = "Profile is deleted automatically when you switch away or log out"
		}
		return a, pushCmd(screens.NewForm(title, help, msg.Kind, []screens.FormInput{
			{Label: "Profile Name:", Placeholder: "e.g. work"},
			{Label: "Email Address:", Placeholder: "e.g. you@company.com"},
		}, a.theme))

	case "switch":
		if needsPassphraseForSwitch(a.store, msg.Name) {
			return a, pushCmd(screens.NewForm("Unlock SSH Key", fmt.Sprintf("Enter the passphrase for identity %q", msg.Name), "switch-pass:"+msg.Name, []screens.FormInput{
				{Label: "Passphrase:", IsPassword: true},
			}, a.theme))
		}
		return a, a.runTaskCmd("switch", msg.Name, func() (opResult, error) {
			return opSwitch(a.store, msg.Name, "")
		})

	case "rename":
		return a, pushCmd(screens.NewForm("Rename Identity", "Enter new profile name for "+msg.Name, "rename:"+msg.Name, []screens.FormInput{
			{Label: "New Name:", Value: msg.Name},
		}, a.theme))

	case "email":
		u := a.store.FindUser(msg.Name)
		currentEmail := ""
		if u != nil {
			currentEmail = u.Email
		}
		return a, pushCmd(screens.NewForm("Change Email", "Enter new email address for "+msg.Name, "email:"+msg.Name, []screens.FormInput{
			{Label: "New Email:", Value: currentEmail},
		}, a.theme))

	case "toggle-sign":
		return a.handleToggleSign(msg.Name)

	case "pubkey":
		res, err := opPubkey(a.store, msg.Name)
		if err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 4*time.Second)
		}
		return a, pushCmd(screens.NewReport("Public Key", res.detail, a.theme))

	case "pubkey-push":
		return a, pushCmd(screens.NewOptions(
			"Publish SSH Key to Platform",
			core.OptionsHelp(),
			"push-platform",
			[]screens.Option{
				{Label: "GitHub", Key: "github"},
				{Label: "GitLab", Key: "gitlab"},
				{Label: "Bitbucket", Key: "bitbucket"},
				{Label: "Cancel", Key: ""},
			},
			a.theme,
		))

	case "bind":
		user := a.store.FindUser(msg.Name)
		if user == nil {
			return a, core.ShowToastCmd("identity not found", theme.ToastStyleError, 3*time.Second)
		}
		return a, pushCmd(screens.NewOptions(
			"SSH Key Setup: "+msg.Name,
			core.OptionsHelp(),
			fmt.Sprintf("ssh-setup:%s||bind", msg.Name),
			[]screens.Option{
				{Label: "Generate new key automatically (recommended)", Key: "generate"},
				{Label: "Use existing key (provide path)", Key: "existing"},
				{Label: "Cancel", Key: ""},
			},
			a.theme,
		))

	case "check-ssh":
		return a, a.runTaskCmd("check-ssh", msg.Name, func() (opResult, error) {
			return opCheckSSH(a.store, msg.Name)
		})

	case "unbind":
		return a, pushCmd(screens.NewConfirm(
			fmt.Sprintf("Remove SSH key binding from %q? (file not deleted)", msg.Name),
			"unbind:"+msg.Name,
			a.theme,
		))

	case "rekey":
		return a, pushCmd(screens.NewConfirm(
			fmt.Sprintf("Rotate SSH key for %q? WARNING: Replaces key pair; requires re-uploading public key.", msg.Name),
			"rekey:"+msg.Name,
			a.theme,
		))

	case "passphrase":
		return a, pushCmd(screens.NewPassphraseMenu(a.store, msg.Name, a.theme))

	case "passphrase-set":
		u := a.store.FindUser(msg.Name)
		if u == nil || u.SSHKey == "" {
			return a, core.ShowToastCmd("No SSH key bound to this identity", theme.ToastStyleError, 3*time.Second)
		}
		protected, _ := isSSHKeyPassphraseProtected(u.SSHKey)
		if protected {
			return a, pushCmd(screens.NewForm("Change Passphrase", "Enter current and new passphrase for "+msg.Name, "passphrase-set-protected:"+msg.Name, []screens.FormInput{
				{Label: "Current Passphrase:", IsPassword: true},
				{Label: "New Passphrase:", IsPassword: true},
				{Label: "Confirm New Passphrase:", IsPassword: true},
			}, a.theme))
		}
		return a, pushCmd(screens.NewForm("Set Passphrase", "Enter a new passphrase for "+msg.Name, "passphrase-set:"+msg.Name, []screens.FormInput{
			{Label: "New Passphrase:", IsPassword: true},
			{Label: "Confirm New Passphrase:", IsPassword: true},
		}, a.theme))

	case "passphrase-remove":
		if a.store.Current != msg.Name {
			return a, core.ShowToastCmd(fmt.Sprintf("Must switch to profile %q to remove its passphrase", msg.Name), theme.ToastStyleError, 4*time.Second)
		}
		return a, pushCmd(screens.NewForm("Remove Passphrase", fmt.Sprintf("Enter the current passphrase for %q to confirm removal", msg.Name), "passphrase-remove:"+msg.Name, []screens.FormInput{
			{Label: "Current Passphrase:", IsPassword: true},
		}, a.theme))

	case "passphrase-verify":
		return a, pushCmd(screens.NewForm("Verify Passphrase", "Enter the passphrase to test", "passphrase-verify:"+msg.Name, []screens.FormInput{
			{Label: "Passphrase:", IsPassword: true},
		}, a.theme))

	case "bind-path":
		return a, pushCmd(screens.NewForm("Bind Directory", "Directory path to bind to "+msg.Name, "bind-path:"+msg.Name, []screens.FormInput{
			{Label: "Path:", Placeholder: "e.g. ~/work"},
		}, a.theme))

	case "unbind-path":
		u := a.store.FindUser(msg.Name)
		if u == nil || len(u.BindPaths) == 0 {
			return a, core.ShowToastCmd("No paths bound to this identity", theme.ToastStyleInfo, 3*time.Second)
		}
		if len(u.BindPaths) == 1 {
			path := u.BindPaths[0]
			return a, pushCmd(screens.NewConfirm(
				fmt.Sprintf("Unbind directory %q?", path),
				fmt.Sprintf("unbind-path-confirm:%s|%s", msg.Name, path),
				a.theme,
			))
		}
		var opts []screens.Option
		for _, p := range u.BindPaths {
			opts = append(opts, screens.Option{Label: p, Key: p})
		}
		opts = append(opts, screens.Option{Label: "Cancel", Key: ""})
		return a, pushCmd(screens.NewOptions(
			"Select directory to unbind",
			core.OptionsHelp(),
			"unbind-path:"+msg.Name,
			opts,
			a.theme,
		))

	case "export":
		names := []string{msg.Name}
		return a, a.pushExportForm(names)

	case "import-export":
		return a, pushCmd(screens.NewImportExport(a.store, a.theme))

	case "export-current":
		if a.store.Current == "" {
			return a, core.ShowToastCmd("No active identity — switch to one first", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.pushExportForm([]string{a.store.Current})

	case "export-all":
		return a, a.pushExportForm(nil)

	case "import":
		return a, pushCmd(screens.NewForm("Import Bundle", "Path to the encrypted bundle file (.bundle)", "import:path", []screens.FormInput{
			{Label: "Bundle Path:", Placeholder: "e.g. ~/git-user-export-2026-01-01.bundle"},
		}, a.theme))

	case "import-original":
		name := git.CurrentName()
		email := git.CurrentEmail()
		if a.store.Original != nil {
			if a.store.Original.Name != "" {
				name = a.store.Original.Name
			}
			if a.store.Original.Email != "" {
				email = a.store.Original.Email
			}
		}
		return a, pushCmd(screens.NewForm("Import Original Identity", "Import your existing ~/.gitconfig identity (you pick the name)", "import-original", []screens.FormInput{
			{Label: "Profile Name:", Value: name, Placeholder: "e.g. original"},
			{Label: "Email Address:", Value: email, Placeholder: "e.g. you@example.com"},
		}, a.theme))

	case "remove":
		return a, pushCmd(screens.NewConfirm(
			fmt.Sprintf("Remove identity %q? This cannot be undone.", msg.Name),
			"remove:"+msg.Name,
			a.theme,
		))

	case "logout":
		return a, a.runTaskCmd("logout", "", func() (opResult, error) {
			return opLogout(a.store)
		})

	case "fix-remote":
		return a, a.runTaskCmd("fix-remote", "", func() (opResult, error) {
			return opFixRemote()
		})

	case "security":
		return a, a.runTaskCmd("security", "", func() (opResult, error) {
			return opSecurity(a.store)
		})

	case "doctor":
		return a, a.runTaskCmd("doctor", "", func() (opResult, error) {
			return opDoctor(a.store)
		})

	case "update":
		return a, core.ShowToastCmd("Updating replaces the running binary — run 'git-user update' in your terminal", theme.ToastStyleInfo, 5*time.Second)
	}

	return a, nil
}

func (a *App) pushExportForm(names []string) tea.Cmd {
	context := "export-all"
	if names != nil {
		context = "export:" + strings.Join(names, ",")
	}
	return pushCmd(screens.NewForm("Export Identities", "Encrypt identities into a bundle file", context, []screens.FormInput{
		{Label: "Encryption Passphrase:", IsPassword: true},
		{Label: "Confirm Passphrase:", IsPassword: true},
	}, a.theme))
}

func (a *App) handleToggleSign(name string) (tea.Model, tea.Cmd) {
	user := a.store.FindUser(name)
	if user != nil {
		if !user.SignDisabled && user.SignKey != "" {
			a.store.ToggleSigning(user.Name, true)
			if a.store.Current == user.Name {
				git.RemoveSigningConfig()
			}
		} else {
			if user.SSHKey != "" {
				a.store.SetSigningKey(user.Name, user.SSHKey, "ssh")
				if a.store.Current == user.Name {
					_ = git.ConfigureSigning(user.SSHKey, "ssh")
				}
			} else {
				a.store.ToggleSigning(user.Name, !user.SignDisabled)
				if a.store.Current == user.Name {
					if !user.SignDisabled {
						git.RemoveSigningConfig()
					}
				}
			}
		}
		config.Save(a.store)
	}
	return a, core.RefreshStoreCmd()
}

// ── Options Results ───────────────────────────────────────────────────────────

func (a *App) handleOptionResult(msg core.OptionResultMsg) (tea.Model, tea.Cmd) {
	if msg.Choice == "" {
		return a, core.ShowToastCmd("Cancelled", theme.ToastStyleInfo, 2*time.Second)
	}

	parts := strings.SplitN(msg.Context, ":", 2)
	action := parts[0]
	data := ""
	if len(parts) > 1 {
		data = parts[1]
	}

	switch action {
	case "ssh-setup":
		fields := strings.Split(data, "|") // name|email|mode
		name := fields[0]
		email := ""
		mode := "register"
		if len(fields) > 1 {
			email = fields[1]
		}
		if len(fields) > 2 {
			mode = fields[2]
		}
		switch msg.Choice {
		case "generate":
			return a, pushCmd(screens.NewForm("SSH Key Passphrase", "Optional: protect the new key (leave empty for no passphrase)", fmt.Sprintf("ssh-passphrase:%s|%s|%s|%s", name, email, mode, msg.Choice), []screens.FormInput{
				{Label: "New Passphrase:", IsPassword: true},
				{Label: "Confirm Passphrase:", IsPassword: true},
			}, a.theme))
		case "existing":
			return a, pushCmd(screens.NewForm("Existing SSH Key", "Path to your SSH private key", fmt.Sprintf("ssh-keypath:%s|%s|%s", name, email, mode), []screens.FormInput{
				{Label: "Key Path:", Placeholder: "e.g. ~/.ssh/id_ed25519"},
			}, a.theme))
		default:
			// skip
			return a, a.runTaskCmd("attach-key", name, func() (opResult, error) {
				return opAttachKey(a.store, name, email, mode, "skip", "", "", false)
			})
		}

	case "push-platform":
		platform := msg.Choice
		return a, a.runTaskCmd("pubkey-push", platform, func() (opResult, error) {
			return opPushKey(a.store, platform)
		})

	case "unbind-path":
		name := data
		return a, a.runTaskCmd("unbind-path", name, func() (opResult, error) {
			err := opUnbindPath(a.store, name, msg.Choice)
			return opResult{detail: fmt.Sprintf("Unbound directory %q", msg.Choice)}, err
		})
	}

	return a, nil
}

// ── Task Results ──────────────────────────────────────────────────────────────

func (a *App) handleTaskResult(msg core.TaskResultMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if msg.Err != nil {
		a.removeKeyPath = ""
		switch {
		case errors.Is(msg.Err, ErrNeedsPassphrase):
			cmds = append(cmds, pushCmd(screens.NewForm("Unlock SSH Key", fmt.Sprintf("Enter the passphrase for identity %q", msg.Name), "switch-pass:"+msg.Name, []screens.FormInput{
				{Label: "Passphrase:", IsPassword: true},
			}, a.theme)))
		case errors.Is(msg.Err, ErrNeedsCredential):
			cmds = append(cmds, a.credentialFormCmd(msg.Name))
		default:
			cmds = append(cmds, core.ShowToastCmd(msg.Err.Error(), theme.ToastStyleError, 5*time.Second))
		}
		cmds = append(cmds, core.RefreshStoreCmd())
		return a, tea.Batch(cmds...)
	}

	switch msg.Kind {
	case "remove":
		cmds = append(cmds, core.ShowToastCmd(fmt.Sprintf("Removed identity %q", msg.Name), theme.ToastStyleSuccess, 3*time.Second))
		if a.removeKeyPath != "" {
			keyPath := a.removeKeyPath
			a.removeKeyPath = ""
			cmds = append(cmds, pushCmd(screens.NewConfirm(
				fmt.Sprintf("Delete SSH key file %s?", keyPath),
				"delete-key:"+keyPath,
				a.theme,
			)))
		}
	case "switch":
		first := firstLine(msg.Detail)
		if first == "" {
			first = fmt.Sprintf("Switched to %q", msg.Name)
		}
		cmds = append(cmds, core.ShowToastCmd(first, theme.ToastStyleSuccess, 3*time.Second))
		if git.HasHTTPSRemotes() {
			cmds = append(cmds, pushCmd(screens.NewConfirm(
				"This repo uses HTTPS remotes. Convert to SSH for passwordless push?",
				"switch-https",
				a.theme,
			)))
		}
	case "logout":
		cmds = append(cmds, core.ShowToastCmd(firstLine(msg.Detail), theme.ToastStyleSuccess, 3*time.Second))
	default:
		if msg.ShowReport {
			cmds = append(cmds, pushCmd(screens.NewReport(titleForKind(msg.Kind), msg.Detail, a.theme)))
		} else {
			cmds = append(cmds, core.ShowToastCmd(firstLine(msg.Detail), theme.ToastStyleSuccess, 3*time.Second))
		}
	}

	cmds = append(cmds, core.RefreshStoreCmd())
	return a, tea.Batch(cmds...)
}

func (a *App) credentialFormCmd(platform string) tea.Cmd {
	switch platform {
	case "github":
		return pushCmd(screens.NewForm("GitHub Personal Access Token", "Token requires the 'write:public_key' scope", "push-cred:github", []screens.FormInput{
			{Label: "GitHub Token:", IsPassword: true},
		}, a.theme))
	case "gitlab":
		return pushCmd(screens.NewForm("GitLab Personal Access Token", "Token requires the 'api' or 'write_repository' scope", "push-cred:gitlab", []screens.FormInput{
			{Label: "GitLab Token:", IsPassword: true},
		}, a.theme))
	case "bitbucket":
		return pushCmd(screens.NewForm("Bitbucket Credentials", "Enter your Bitbucket username and an App Password with 'ssh:write' scope", "push-cred:bitbucket", []screens.FormInput{
			{Label: "Username:", Placeholder: "e.g. bobdylan"},
			{Label: "App Password:", IsPassword: true},
		}, a.theme))
	}
	return core.ShowToastCmd("unsupported platform", theme.ToastStyleError, 3*time.Second)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx != -1 {
		return s[:idx]
	}
	return s
}

func titleForKind(kind string) string {
	switch kind {
	case "register", "register-temp", "attach-key":
		return "Identity Created"
	case "bind":
		return "SSH Key Configured"
	case "rekey":
		return "SSH Key Rotated"
	case "export", "export-current", "export-all":
		return "Export"
	case "import":
		return "Import"
	case "import-original":
		return "Original Identity Imported"
	case "check-ssh":
		return "SSH Connection Check"
	case "security":
		return "Security Audit"
	case "doctor":
		return "Diagnostics"
	case "fix-remote":
		return "Remote Conversion"
	case "pubkey-push":
		return "Publish SSH Key"
	}
	return "Result"
}

// ── Confirm Results ───────────────────────────────────────────────────────────

func (a *App) handleConfirmResult(msg core.ConfirmResultMsg) (tea.Model, tea.Cmd) {
	if !msg.Confirmed {
		return a, core.ShowToastCmd("Cancelled", theme.ToastStyleInfo, 2*time.Second)
	}

	parts := strings.SplitN(msg.Context, ":", 2)
	action := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	}

	switch action {
	case "remove":
		keyPath := ""
		if u := a.store.FindUser(rest); u != nil {
			keyPath = u.SSHKey
		}
		a.removeKeyPath = keyPath
		return a, a.runTaskCmd("remove", rest, func() (opResult, error) {
			_, err := opRemove(a.store, rest)
			return opResult{}, err
		})

	case "delete-key":
		keyPath := rest
		return a, a.runTaskCmd("delete-key", "", func() (opResult, error) {
			opDeleteKeyFiles(keyPath)
			return opResult{detail: "SSH key files deleted"}, nil
		})

	case "unbind":
		return a, a.runTaskCmd("unbind", rest, func() (opResult, error) {
			err := opUnbind(a.store, rest)
			return opResult{detail: fmt.Sprintf("SSH key binding removed from %q", rest)}, err
		})

	case "rekey":
		return a, pushCmd(screens.NewForm("New Key Passphrase", "Optional: protect the new key (leave empty for no passphrase)", "rekey-pass:"+rest, []screens.FormInput{
			{Label: "New Passphrase:", IsPassword: true},
			{Label: "Confirm Passphrase:", IsPassword: true},
		}, a.theme))

	case "unbind-path-confirm":
		fields := strings.SplitN(rest, "|", 2)
		name := fields[0]
		path := ""
		if len(fields) > 1 {
			path = fields[1]
		}
		return a, a.runTaskCmd("unbind-path", name, func() (opResult, error) {
			err := opUnbindPath(a.store, name, path)
			return opResult{detail: fmt.Sprintf("Unbound directory %q", path)}, err
		})

	case "ssh-sign":
		fields := strings.Split(rest, "|") // name|email|mode|choice|pass|keyPath
		name := ""
		email := ""
		mode := "register"
		choice := ""
		pass := ""
		keyPath := ""
		if len(fields) > 0 {
			name = fields[0]
		}
		if len(fields) > 1 {
			email = fields[1]
		}
		if len(fields) > 2 {
			mode = fields[2]
		}
		if len(fields) > 3 {
			choice = fields[3]
		}
		if len(fields) > 4 {
			pass = fields[4]
		}
		if len(fields) > 5 {
			keyPath = fields[5]
		}
		return a, a.runTaskCmd("attach-key", name, func() (opResult, error) {
			return opAttachKey(a.store, name, email, mode, choice, pass, keyPath, msg.Confirmed)
		})

	case "switch-https":
		return a, a.runTaskCmd("fix-remote", "", func() (opResult, error) {
			return opFixRemote()
		})
	}

	return a, nil
}

// ── Form Results ──────────────────────────────────────────────────────────────

func (a *App) handleFormResult(msg core.FormResultMsg) (tea.Model, tea.Cmd) {
	if len(msg.Values) == 0 {
		return a, nil
	}

	parts := strings.SplitN(msg.Context, ":", 2)
	action := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	}

	switch action {
	case "register", "register-temp":
		if msg.Values[0] == "" || msg.Values[1] == "" {
			return a, core.ShowToastCmd("Profile name and email are required", theme.ToastStyleError, 3*time.Second)
		}
		if !isValidEmail(msg.Values[1]) {
			return a, core.ShowToastCmd("Invalid email format", theme.ToastStyleError, 3*time.Second)
		}
		if a.store.IsNameTaken(msg.Values[0]) {
			return a, core.ShowToastCmd(fmt.Sprintf("identity %q already exists", msg.Values[0]), theme.ToastStyleError, 3*time.Second)
		}
		if a.store.IsEmailTaken(msg.Values[1]) {
			return a, core.ShowToastCmd("Email already in use — each identity must have a unique email", theme.ToastStyleError, 3*time.Second)
		}
		return a, pushCmd(screens.NewOptions(
			"SSH Key Setup",
			core.OptionsHelp(),
			fmt.Sprintf("ssh-setup:%s|%s|%s", msg.Values[0], msg.Values[1], action),
			[]screens.Option{
				{Label: "Generate new key automatically (recommended)", Key: "generate"},
				{Label: "Use existing key (provide path)", Key: "existing"},
				{Label: "Skip for now (set up later)", Key: "skip"},
			},
			a.theme,
		))

	case "rename":
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("New name cannot be empty", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("rename", rest, func() (opResult, error) {
			err := opRename(a.store, rest, msg.Values[0])
			return opResult{detail: fmt.Sprintf("Renamed %q → %q", rest, msg.Values[0])}, err
		})

	case "email":
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("New email cannot be empty", theme.ToastStyleError, 3*time.Second)
		}
		if !isValidEmail(msg.Values[0]) {
			return a, core.ShowToastCmd("Invalid email format", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("email", rest, func() (opResult, error) {
			err := opChangeEmail(a.store, rest, msg.Values[0])
			return opResult{detail: fmt.Sprintf("Updated %q → email is now %s", rest, msg.Values[0])}, err
		})

	case "ssh-passphrase":
		fields := strings.Split(rest, "|")
		name, email, mode, choice := field(fields, 0), field(fields, 1), field(fields, 2), field(fields, 3)
		newPass := msg.Values[0]
		if newPass != msg.Values[1] {
			return a, core.ShowToastCmd("Passphrases do not match", theme.ToastStyleError, 3*time.Second)
		}
		return a, pushCmd(screens.NewConfirm(
			"Would you like to sign your Git commits automatically using this identity's SSH key?",
			fmt.Sprintf("ssh-sign:%s|%s|%s|%s|%s|", name, email, mode, choice, newPass),
			a.theme,
		))

	case "ssh-keypath":
		fields := strings.Split(rest, "|")
		name, email, mode := field(fields, 0), field(fields, 1), field(fields, 2)
		keyPath := expandPath(msg.Values[0])
		if keyPath == "" {
			return a, core.ShowToastCmd("No key path provided", theme.ToastStyleError, 3*time.Second)
		}
		if _, err := os.Stat(keyPath); err != nil {
			return a, core.ShowToastCmd(fmt.Sprintf("Key file not found: %s", msg.Values[0]), theme.ToastStyleError, 3*time.Second)
		}
		return a, pushCmd(screens.NewConfirm(
			"Would you like to sign your Git commits automatically using this identity's SSH key?",
			fmt.Sprintf("ssh-sign:%s|%s|%s|existing||%s", name, email, mode, keyPath),
			a.theme,
		))

	case "bind-path":
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("Path cannot be empty", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("bind-path", rest, func() (opResult, error) {
			err := opBindPath(a.store, rest, msg.Values[0])
			return opResult{detail: fmt.Sprintf("Bound directory %q to %q", msg.Values[0], rest)}, err
		})

	case "export", "export-all":
		pass := msg.Values[0]
		if pass == "" {
			return a, core.ShowToastCmd("Passphrase must not be empty", theme.ToastStyleError, 3*time.Second)
		}
		if pass != msg.Values[1] {
			return a, core.ShowToastCmd("Passphrases do not match", theme.ToastStyleError, 3*time.Second)
		}
		var names []string
		all := false
		if action == "export-all" {
			all = true
		} else if rest != "" {
			names = strings.Split(rest, ",")
		}
		return a, a.runTaskCmd("export", "", func() (opResult, error) {
			return opExport(a.store, names, all, pass)
		})

	case "import:path":
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("Bundle path cannot be empty", theme.ToastStyleError, 3*time.Second)
		}
		return a, pushCmd(screens.NewForm("Import Bundle", "Enter the passphrase for the bundle", "import-pass:"+msg.Values[0], []screens.FormInput{
			{Label: "Passphrase:", IsPassword: true},
		}, a.theme))

	case "import-pass":
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("Passphrase must not be empty", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("import", "", func() (opResult, error) {
			return opImport(a.store, rest, msg.Values[0], false)
		})

	case "import-original":
		name := msg.Values[0]
		email := msg.Values[1]
		if name == "" {
			return a, core.ShowToastCmd("Profile name is required", theme.ToastStyleError, 3*time.Second)
		}
		if email == "" {
			return a, core.ShowToastCmd("Email is required", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("import-original", "", func() (opResult, error) {
			return opImportOriginal(a.store, name, email)
		})

	case "push-cred:github", "push-cred:gitlab":
		platform := strings.TrimPrefix(action, "push-cred:")
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("Token required to interact with the API", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("pubkey-push", platform, func() (opResult, error) {
			return opPushKeyWithCredential(a.store, platform, "", msg.Values[0])
		})

	case "push-cred:bitbucket":
		if msg.Values[0] == "" || msg.Values[1] == "" {
			return a, core.ShowToastCmd("Username and App Password are required", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("pubkey-push", "bitbucket", func() (opResult, error) {
			return opPushKeyWithCredential(a.store, "bitbucket", msg.Values[0], msg.Values[1])
		})

	case "passphrase-set-protected":
		if msg.Values[1] != msg.Values[2] {
			return a, core.ShowToastCmd("New passphrases do not match", theme.ToastStyleError, 3*time.Second)
		}
		if msg.Values[1] == "" {
			return a, core.ShowToastCmd("Passphrase must not be empty", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("passphrase-set", rest, func() (opResult, error) {
			err := opPassphraseSet(a.store, rest, msg.Values[0], msg.Values[1])
			return opResult{detail: fmt.Sprintf("Passphrase changed for %q", rest)}, err
		})

	case "passphrase-set":
		if msg.Values[0] != msg.Values[1] {
			return a, core.ShowToastCmd("Passphrases do not match", theme.ToastStyleError, 3*time.Second)
		}
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("Passphrase must not be empty", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("passphrase-set", rest, func() (opResult, error) {
			err := opPassphraseSet(a.store, rest, "", msg.Values[0])
			return opResult{detail: fmt.Sprintf("Passphrase added for %q", rest)}, err
		})

	case "passphrase-verify":
		return a, a.runTaskCmd("passphrase-verify", rest, func() (opResult, error) {
			err := opPassphraseVerify(a.store, rest, msg.Values[0])
			return opResult{detail: fmt.Sprintf("Passphrase for %q verified successfully", rest)}, err
		})

	case "passphrase-remove":
		return a, a.runTaskCmd("passphrase-remove", rest, func() (opResult, error) {
			err := opPassphraseRemove(a.store, rest, msg.Values[0])
			return opResult{detail: fmt.Sprintf("Passphrase security removed for %q", rest)}, err
		})

	case "rekey-pass":
		if msg.Values[0] != msg.Values[1] {
			return a, core.ShowToastCmd("Passphrases do not match", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("rekey", rest, func() (opResult, error) {
			return opRekey(a.store, rest, msg.Values[0])
		})

	case "switch-pass":
		return a, a.runTaskCmd("switch", rest, func() (opResult, error) {
			return opSwitch(a.store, rest, msg.Values[0])
		})
	}

	return a, nil
}

func field(fields []string, i int) string {
	if i < len(fields) {
		return fields[i]
	}
	return ""
}

// ── Results ───────────────────────────────────────────────────────────────────

func (a *App) Quit() bool { return a.quit }

func (a *App) PendingAction() (kind, name, arg string) {
	if a.action == nil {
		return "", "", ""
	}
	return a.action.kind, a.action.name, a.action.arg
}
