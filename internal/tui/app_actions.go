package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"strings"
	"time"
)

// ── Action Handling ───────────────────────────────────────────────────────────

func (a *App) handleAction(msg core.ActionResultMsg) (tea.Model, tea.Cmd) {
	switch msg.Kind {
	case "quit-confirm":
		// 'q' from the dashboard: show a confirmation dialog before quitting.
		return a, pushCmd(screens.NewConfirm(
			"Quit git-user?",
			"quit-confirmed",
			a.theme,
		))

	case "quit":
		a.quit = true
		return a, tea.Quit

	case "firstrun-skip":
		// Mark the prompt as shown so it never appears again, then return to
		// the dashboard.
		a.store.ImportPrompted = true
		_ = config.Save(a.store)
		return a, core.RefreshStoreCmd()

	case "firstrun-import":
		// Reuse the standard import-original flow (name + email form prefilled
		// with the detected original identity).
		name := ""
		email := ""
		if a.store.Original != nil {
			name, email = a.store.Original.Name, a.store.Original.Email
		}
		if name == "" || email == "" {
			name = git.CurrentName()
			email = git.CurrentEmail()
		}
		if email == "" {
			email = name
			name = ""
		}
		return a, pushCmd(screens.NewForm("Import Original Identity", "Import your existing ~/.gitconfig identity (you pick the name)", "import-original", []screens.FormInput{
			{Label: "Profile Name:", Value: name, Placeholder: "e.g. original"},
			{Label: "Email Address:", Value: email, Placeholder: "e.g. you@example.com"},
		}, a.theme))

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
			return a, a.switchPassphraseFormCmd(msg.Name)
		}
		return a, a.runTaskCmd("switch", msg.Name, func() (opResult, error) {
			return opSwitch(a.store, msg.Name, "")
		})

	case "fix-sync":
		// Re-apply the active identity when the git config drifted (e.g. a
		// manual edit or another session left user.name/user.email out of
		// sync). Only the currently active profile is re-applied — nothing is
		// created, deleted, or switched.
		if a.store == nil || a.store.Current == "" {
			return a, core.ShowToastCmd("No active identity to re-apply", theme.ToastStyleError, 3*time.Second)
		}
		name := a.store.Current
		return a, a.runTaskCmd("switch", name, func() (opResult, error) {
			return opSwitch(a.store, name, "")
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
		return a, a.runTaskCmd("pubkey", msg.Name, func() (opResult, error) {
			return opPubkey(a.store, msg.Name)
		})

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
			return opCheckSSH(a.store, msg.Name, "")
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

	case "unbind-path-confirm":
		// msg.Name = "profileName|path" (sent from Detail's inactive path list).
		// Push a Confirm dialog with the right context for app_confirm.go.
		parts := strings.SplitN(msg.Name, "|", 2)
		name := parts[0]
		path := ""
		if len(parts) > 1 {
			path = parts[1]
		}
		return a, pushCmd(screens.NewConfirm(
			fmt.Sprintf("Unbind directory %q from %q?", path, name),
			fmt.Sprintf("unbind-path-confirm:%s|%s", name, path),
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

	case "doctor", "security":
		return a, a.runTaskCmd("doctor", "", func() (opResult, error) {
			return opDoctor(a.store)
		})

	case "refresh":
		return a, a.runTaskCmd("refresh", "", func() (opResult, error) {
			return opRefresh(a.store)
		})

	case "log":
		return a, a.runTaskCmd("log", "", func() (opResult, error) {
			return opLog()
		})

	case "stats":
		if !git.IsInRepo() {
			return a, core.ShowToastCmd("not in a git repository — run Stats inside a repository", theme.ToastStyleError, 4*time.Second)
		}
		return a, pushCmd(screens.NewStatsScreen(a.store, a.theme))

	case "clone":
		return a, pushCmd(screens.NewForm("Clone Repository", "Clone a repository and configure the local identity", "clone", []screens.FormInput{
			{Label: "Repository URL:", Placeholder: "git@github.com:user/repo.git"},
			{Label: "Destination Dir:", Placeholder: "Optional, defaults to repo name"},
		}, a.theme))

	case "clone-identity":
		return a.handleCloneIdentity(msg.Name)

	case "hook":
		return a, pushCmd(screens.NewOptions(
			"Git Hooks",
			core.OptionsHelp(),
			"hook",
			[]screens.Option{
				{Label: "Install pre-commit hook", Key: "install"},
				{Label: "Uninstall pre-commit hook", Key: "uninstall"},
				{Label: "Check identity (used by hook)", Key: "check"},
				{Label: "Cancel", Key: ""},
			},
			a.theme,
		))

	case "sync":
		if a.store.Sync != nil && a.store.Sync.RepoURL != "" {
			return a, pushCmd(screens.NewForm("Sync Identities", "Enter the sync passphrase to decrypt the bundle", "sync-pass", []screens.FormInput{
				{Label: "Passphrase:", IsPassword: true},
			}, a.theme))
		}
		return a, pushCmd(screens.NewForm("Configure Sync", "Keep identities synchronized across devices using a private git repository", "sync-setup", []screens.FormInput{
			{Label: "Repository URL:", Placeholder: "git@github.com:user/sync.git"},
			{Label: "Passphrase:", IsPassword: true},
			{Label: "Confirm Passphrase:", IsPassword: true},
		}, a.theme))

	case "config":
		return a.handleConfigAction(msg.Name)

	case "update":
		return a, a.runTaskCmd("update", "", func() (opResult, error) {
			return opUpdate()
		})

	case "uninstall":
		return a, pushCmd(screens.NewConfirm(
			"Uninstall git-user? This restores your original git identity, removes git-user's config and stored passphrases, and deletes its data directory. SSH keys and the binary are kept — this cannot be undone.",
			"uninstall-confirmed",
			a.theme,
		))
	}

	return a, nil
}

// handleCloneIdentity picks which identity to use for a clone.
func (a *App) handleCloneIdentity(name string) (tea.Model, tea.Cmd) {
	parts := strings.SplitN(name, "|", 2)
	repoURL := parts[0]
	destDir := ""
	if len(parts) > 1 {
		destDir = parts[1]
	}
	if len(a.store.Users) == 0 {
		return a, core.ShowToastCmd("No registered identities — register one first", theme.ToastStyleError, 3*time.Second)
	}
	var opts []screens.Option
	for _, u := range a.store.Users {
		opts = append(opts, screens.Option{Label: fmt.Sprintf("%s (%s)", u.Name, u.Email), Key: u.Name})
	}
	opts = append(opts, screens.Option{Label: "Cancel", Key: ""})
	return a, pushCmd(screens.NewOptions(
		"Select identity for this repository",
		core.OptionsHelp(),
		fmt.Sprintf("clone-identity:%s|%s", repoURL, destDir),
		opts,
		a.theme,
	))
}

// handleConfigAction shows the custom config management menu for an identity.
func (a *App) handleConfigAction(name string) (tea.Model, tea.Cmd) {
	if a.store.FindUser(name) == nil {
		return a, core.ShowToastCmd("identity not found", theme.ToastStyleError, 3*time.Second)
	}
	return a, pushCmd(screens.NewOptions(
		fmt.Sprintf("Custom Git Config: %s", name),
		core.OptionsHelp(),
		"config-action:"+name,
		[]screens.Option{
			{Label: "List config keys", Key: "list"},
			{Label: "Set a config key", Key: "set"},
			{Label: "Unset a config key", Key: "unset"},
			{Label: "Cancel", Key: ""},
		},
		a.theme,
	))
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

// switchPassphraseFormCmd prompts for the SSH key passphrase during a switch.
// The passphrase is asked entirely inside the TUI.
func (a *App) switchPassphraseFormCmd(name string) tea.Cmd {
	return pushCmd(screens.NewForm("Enter Passphrase", "", "switch-pass:"+name, []screens.FormInput{
		{Label: "Enter Passphrase:", IsPassword: true},
	}, a.theme))
}

// checkSSHPassphraseFormCmd asks for the SSH key passphrase in-app before
// running the SSH connection check.
func (a *App) checkSSHPassphraseFormCmd(name string) tea.Cmd {
	return pushCmd(screens.NewForm("Check SSH Connection", "", "check-ssh-pass:"+name, []screens.FormInput{
		{Label: "Enter Passphrase:", IsPassword: true},
	}, a.theme))
}

func (a *App) handleToggleSign(name string) (tea.Model, tea.Cmd) {
	user := a.store.FindUser(name)
	if user == nil {
		return a, core.RefreshStoreCmd()
	}

	var warnings []string
	nowEnabled := user.SignDisabled || user.SignKey == ""

	if !user.SignDisabled && user.SignKey != "" {
		if err := a.store.ToggleSigning(user.Name, true); err != nil {
			warnings = append(warnings, err.Error())
		}
		if a.store.Current == user.Name {
			git.RemoveSigningConfig()
		}
	} else if user.SSHKey != "" {
		if err := a.store.SetSigningKey(user.Name, user.SSHKey, "ssh"); err != nil {
			warnings = append(warnings, err.Error())
		}
		if a.store.Current == user.Name {
			if err := git.ConfigureSigning(user.SSHKey, "ssh"); err != nil {
				warnings = append(warnings, fmt.Sprintf("applying signing config: %v", err))
			}
		}
	} else {
		if err := a.store.ToggleSigning(user.Name, !user.SignDisabled); err != nil {
			warnings = append(warnings, err.Error())
		}
		if a.store.Current == user.Name && !user.SignDisabled {
			git.RemoveSigningConfig()
		}
	}

	if err := config.Save(a.store); err != nil {
		warnings = append(warnings, fmt.Sprintf("saving config: %v", err))
	}

	cmds := []tea.Cmd{core.RefreshStoreCmd()}
	if len(warnings) > 0 {
		cmds = append(cmds, core.ShowToastCmd("⚠ "+strings.Join(warnings, "; "), theme.ToastStyleError, 5*time.Second))
	} else {
		status := "disabled"
		if nowEnabled {
			status = "enabled"
		}
		cmds = append(cmds, core.ShowToastCmd(fmt.Sprintf("Commit signing %s for %q", status, name), theme.ToastStyleSuccess, 3*time.Second))
	}
	return a, tea.Batch(cmds...)
}
