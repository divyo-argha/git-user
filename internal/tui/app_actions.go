package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/promptops"
	"github.com/divyo-argha/git-user/internal/shellinit"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"github.com/divyo-argha/git-user/internal/validate"
)

// sshKeyPickManual is the sentinel Option.Key used by the existing-key
// picker's "enter a path manually" fallback entry.
const sshKeyPickManual = "__manual__"

// shellIntegrationSnippet is background reading shown on a Report screen
// (which has clipboard-copy built in) behind the "How this works" entry of
// the multi-account picker — the picker itself (multiAccountMenuCmd) is the
// primary, actionable flow; this is just the explanation for anyone who
// wants it.
const shellIntegrationSnippet = `WORKING WITH TWO (OR MORE) ACCOUNTS AT ONCE

Pick "Open <name> in a new terminal window" from this menu (or from an
identity's own screen) to spawn a separate, detached terminal window running
an isolated shell for that identity — its own commit author/committer
name+email and its own SSH key, set only as environment variables for that
one shell process. It never edits ~/.gitconfig, so a second window opened
the same way for a different identity works completely independently, at
the same time. Type 'exit' in a window to leave it.

Prefer the command line? The same thing, done by hand, in any terminal:
  git-user shell <name>

────────────────────────────────────────────────────────────────────────

OPTIONAL: "Install shell shortcut" (also in this menu) adds one line to your
shell config so 'git-user switch --session <name>' / 'git-user env <name>'
can activate an identity in your CURRENT shell — no new window — with a
single command, instead of needing 'eval "$(git-user env <name>)"' by hand.
It only ever appends that one line to your rc file; installing it again is a
no-op if it's already there.`

// manualShellWindowInstructions is shown when openNewTerminalWindow could not
// find a way to spawn a new terminal window automatically (headless session,
// unrecognized terminal emulator, unsupported OS, etc.) — it falls back to
// the exact manual command, which needs no shell integration or setup at all.
func manualShellWindowInstructions(name string, cause error) string {
	return fmt.Sprintf(`Couldn't open a new terminal window automatically:
  %v

Open another terminal window yourself and run:

  git-user shell %s

That starts an isolated shell scoped to %q — its own commit author/committer
identity and its own SSH key, set only as environment variables for that one
shell process. It never touches ~/.gitconfig, so this works alongside a
different identity's isolated shell (or the globally-active one) running in
another window at the same time. Type 'exit' to leave it.`, cause, name, name)
}

// multiAccountMenuCmd builds the interactive "work with multiple accounts"
// picker: one entry per registered identity to open it in a brand-new
// terminal window right now (the actual answer to "how do I use two
// accounts at once"), plus the optional current-shell install and a
// read-only explanation for anyone who wants the details.
func (a *App) multiAccountMenuCmd() tea.Cmd {
	var opts []screens.Option
	for _, u := range a.store.Users {
		opts = append(opts, screens.Option{
			Label: fmt.Sprintf("%s Open %q in a new terminal window", theme.IconWindow, u.Name),
			Key:   "open:" + u.Name,
		})
	}
	if len(a.store.Users) == 0 {
		opts = append(opts, screens.Option{Label: "(no identities registered yet)", Key: ""})
	}
	sh := shellinit.Detect("")
	opts = append(opts,
		screens.Option{Label: "❯ Terminal prompt indicator (Fish, Zsh, Bash, etc.)", Key: "prompt-menu"},
		screens.Option{Label: fmt.Sprintf("⌘ Install optional shell shortcut (%s)", shellLabel(sh)), Key: "install"},
		screens.Option{Label: "ℹ How this works", Key: "info"},
		screens.Option{Label: "Cancel", Key: ""},
	)
	return pushCmd(screens.NewOptions(
		"Work With Multiple Accounts",
		core.OptionsHelp(),
		"multi-account",
		opts,
		a.theme,
	))
}

const terminalIntegrationGuideText = `TERMINAL PROMPT INTEGRATION GUIDE

Display your active Git profile name and status indicator directly inside your
terminal shell prompt across Fish, Zsh, Bash, PowerShell, Nushell, and Starship!

────────────────────────────────────────────────────────────────────────
SUPPORTED SHELLS & CONFIGURATION FILES:

  • Fish Shell:
    Configuration is installed automatically to:
      ~/.config/fish/conf.d/git_user_prompt.fish
    Fish loads this automatically on startup. If you use a custom theme
    (like Tide, Hydro, or Starship), your existing right prompt is preserved.

  • Zsh / Oh My Zsh:
    Configuration is appended to:
      ~/.zshrc
    Uses dynamic precmd hooks and PROMPT_SUBST with RPROMPT support.
    Run 'source ~/.zshrc' after installation to activate.
    Powerlevel10k users: Add a custom 'gituser' prompt segment in ~/.p10k.zsh.

  • Bash:
    Configuration is appended to:
      ~/.bashrc
    Uses PROMPT_COMMAND with ANSI escape guards (\001 and \002) to guarantee
    zero cursor jumps or line-wrapping bugs on long commands.
    Run 'source ~/.bashrc' after installation to activate.

  • Starship Prompt:
    Configuration is added to:
      ~/.config/starship.toml
    Adds a custom [custom.gituser] module running 'git-user prompt'.

  • PowerShell (Windows & Unix):
    Installed into your $PROFILE script:
      Windows: ~/Documents/PowerShell/Microsoft.PowerShell_profile.ps1
      Linux/macOS: ~/.config/powershell/Microsoft.PowerShell_profile.ps1
    Wraps the prompt function while preserving existing prompt customizations.

  • Nushell:
    Installed to:
      Linux/macOS: ~/.config/nushell/env.nu
      Windows: %APPDATA%\nushell\env.nu
    Uses modern $env.PROMPT_COMMAND_RIGHT closures with error tolerance.

────────────────────────────────────────────────────────────────────────
CUSTOMIZING ICONS & GLYPHS:

  • Default:  <profile> (requires a Nerd Font installed in your terminal).
  • Fallback: On virtual consoles or dumb terminals (TERM=linux or dumb),
    icons automatically switch to plain ASCII 'git:<profile>'.
  • Custom: Set environment variable:
      export GIT_USER_PROMPT_ICON="🚀 "
    or configure your preferred icon directly in the TUI prompt settings.

────────────────────────────────────────────────────────────────────────
COMMAND-LINE UTILITIES:

  • Show active profile:
      git-user prompt
  • Show with icon:
      git-user prompt --icon
  • Always show even outside git repos:
      git-user prompt --always
  • Interactive installer from terminal:
      git-user prompt install
  • Temporary session switch in current terminal:
      gu switch -s <name>
`

func (a *App) promptIntegrationMenuCmd() tea.Cmd {
	activeTarget := promptops.DetectActiveTarget()
	activeInfo := promptops.CheckTarget(activeTarget)

	installedStatus := " [Not Installed]"
	if activeInfo.Installed {
		installedStatus = " [Installed]"
	}

	opts := []screens.Option{
		{Label: "✦ View Status & Live Preview", Key: "status"},
		{Label: fmt.Sprintf("▶ Install for Active Shell (%s)%s", activeInfo.Name, installedStatus), Key: "install-active:" + string(activeTarget)},
		{Label: "↓ Choose Shell to Install...", Key: "install-pick"},
		{Label: "⚙ Configure Icon & Appearance", Key: "config-appearance"},
		{Label: "✖ Uninstall from Shell...", Key: "uninstall-pick"},
		{Label: "ℹ Integration Guide & Tips", Key: "help"},
		{Label: "Cancel", Key: ""},
	}

	return pushCmd(screens.NewOptions(
		"Terminal Prompt Indicator",
		core.OptionsHelp(),
		"prompt-menu",
		opts,
		a.theme,
	))
}

func (a *App) promptInstallPickCmd() tea.Cmd {
	targets := promptops.CheckAllTargets()
	var opts []screens.Option
	for _, info := range targets {
		badge := ""
		if info.Installed {
			badge = " (installed)"
		}
		activeTag := ""
		if info.IsActive {
			activeTag = " ★ active"
		}
		opts = append(opts, screens.Option{
			Label: fmt.Sprintf("%s%s%s", info.Name, activeTag, badge),
			Key:   "install:" + string(info.Target),
		})
	}
	opts = append(opts, screens.Option{Label: "Cancel", Key: ""})
	return pushCmd(screens.NewOptions(
		"Select Shell For Prompt Integration",
		core.OptionsHelp(),
		"prompt-install-pick",
		opts,
		a.theme,
	))
}

func (a *App) promptUninstallPickCmd() tea.Cmd {
	activeTarget := promptops.DetectActiveTarget()
	activeName := promptops.TargetName(activeTarget)

	opts := []screens.Option{
		{Label: fmt.Sprintf("Uninstall from active shell (%s)", activeName), Key: "uninstall:" + string(activeTarget)},
		{Label: "Uninstall from ALL shells", Key: "uninstall:all"},
		{Label: "Cancel", Key: ""},
	}
	return pushCmd(screens.NewOptions(
		"Uninstall Prompt Integration",
		core.OptionsHelp(),
		"prompt-uninstall-pick",
		opts,
		a.theme,
	))
}

func (a *App) promptConfigPickCmd() tea.Cmd {
	alwaysState := "OFF (Git repos only)"
	if a.store != nil && a.store.Prompt != nil && a.store.Prompt.Always {
		alwaysState = "ON (Always visible)"
	}
	plainState := "OFF (includes badges)"
	if a.store != nil && a.store.Prompt != nil && a.store.Prompt.Plain {
		plainState = "ON (name only)"
	}
	opts := []screens.Option{
		{Label: "Nerd Font Icon ( ) — Default", Key: "icon:nerd"},
		{Label: "Plain ASCII Tag (git: ) — No special font required", Key: "icon:plain"},
		{Label: "Custom Icon / Emoji — Enter custom text", Key: "icon:custom"},
		{Label: "No Icon — Profile name only", Key: "icon:none"},
		{Label: fmt.Sprintf("Toggle Always Show (current: %s)", alwaysState), Key: "toggle-always"},
		{Label: fmt.Sprintf("Toggle Plain Format (current: %s)", plainState), Key: "toggle-plain"},
		{Label: "Cancel", Key: ""},
	}
	return pushCmd(screens.NewOptions(
		"Prompt Appearance & Options",
		core.OptionsHelp(),
		"prompt-config-pick",
		opts,
		a.theme,
	))
}

// shellLabel names the shell shellinit.Detect resolved to, for display in
// the picker entry.
func shellLabel(sh shellinit.Shell) string {
	switch sh {
	case shellinit.Fish:
		return "fish"
	case shellinit.PowerShell:
		return "PowerShell"
	case shellinit.Cmd:
		return "cmd.exe"
	default:
		return "bash/zsh"
	}
}

// installConfirmQuestion names the exact file(s) an "install shell shortcut"
// confirmation is about to append one line to, so confirming isn't a leap of
// faith about what gets touched.
func installConfirmQuestion(sh shellinit.Shell) string {
	switch sh {
	case shellinit.Fish:
		return "Add the git-user shell shortcut to ~/.config/fish/config.fish?"
	case shellinit.PowerShell:
		return "Add the git-user shell shortcut to your PowerShell $PROFILE?"
	case shellinit.Cmd:
		return "Create the gu.cmd batch helper in your user profile folder (%USERPROFILE%\\gu.cmd)?"
	default:
		return "Add the git-user shell shortcut to ~/.zshrc / ~/.bashrc?"
	}
}

// ── Action Handling ───────────────────────────────────────────────────────────

func (a *App) handleAction(msg core.ActionResultMsg) (tea.Model, tea.Cmd) {
	if strings.HasPrefix(msg.Kind, "signer-remove:") {
		principal := strings.TrimPrefix(msg.Kind, "signer-remove:")
		return a, pushCmd(screens.NewConfirm(
			fmt.Sprintf("Remove %q from .allowed-signers?", principal),
			"policy-signer-remove:"+principal,
			a.theme,
		))
	}

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
		// the dashboard. FirstRun is a one-shot onboarding gate, not a hub
		// screen — it must be popped here or it lingers on the stack forever
		// (ActionResultMsg, unlike Confirm/Form/Option results, never
		// auto-pops its sender since hub screens like Dashboard/Detail rely
		// on staying put).
		a.popScreen()
		a.store.ImportPrompted = true
		_ = config.Save(a.store)
		return a, core.RefreshStoreCmd()

	case "firstrun-import":
		// Reuse the standard import-original flow (name + email form prefilled
		// with the detected original identity). Pop FirstRun first so the form
		// replaces it instead of stacking on top of it — see the popScreen note
		// in "firstrun-skip" above.
		a.popScreen()
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
			{Label: "Profile Name:", Value: name, Placeholder: "e.g. original", Validate: validate.IdentityName},
			{Label: "Email Address:", Value: email, Placeholder: "e.g. you@example.com", Validate: validate.Email},
		}, a.theme))

	case "register", "register-temp":
		return a, a.registerFormCmd(msg.Kind, "", "")

	case "switch":
		if needsPassphraseForSwitch(a.store, msg.Name) {
			return a, a.switchPassphraseFormCmd(msg.Name)
		}
		return a, a.runTaskCmd("switch", msg.Name, func() (opResult, error) {
			return opSwitch(a.store, msg.Name, "")
		})

	case "switch-session":
		return a, a.runTaskCmd("switch-session", msg.Name, func() (opResult, error) {
			return opSwitchSession(a.store, msg.Name)
		})

	case "shell-session":
		user := a.store.FindUser(msg.Name)
		if user == nil {
			return a, core.ShowToastCmd("identity not found", theme.ToastStyleError, 3*time.Second)
		}
		if needsPassphraseForSwitch(a.store, msg.Name) {
			return a, a.shellPassphraseFormCmd(msg.Name)
		}
		return a, openIdentityShellCmd(msg.Name, user)

	case "shell-window":
		if err := openNewTerminalWindow(msg.Name); err != nil {
			return a, pushCmd(screens.NewReport("Open Side-by-Side Terminal", manualShellWindowInstructions(msg.Name, err), a.theme))
		}
		return a, core.ShowToastCmd(fmt.Sprintf("Opened a new terminal window for %q — safe to use alongside this one", msg.Name), theme.ToastStyleSuccess, 3*time.Second)

	case "prompt-integration":
		return a, a.promptIntegrationMenuCmd()

	case "shell-integration":
		return a, a.multiAccountMenuCmd()

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
			{Label: "New Name:", Value: msg.Name, Validate: validate.IdentityName},
		}, a.theme))

	case "email":
		u := a.store.FindUser(msg.Name)
		currentEmail := ""
		if u != nil {
			currentEmail = u.Email
		}
		return a, pushCmd(screens.NewForm("Change Email", "Enter new email address for "+msg.Name, "email:"+msg.Name, []screens.FormInput{
			{Label: "New Email:", Value: currentEmail, Validate: validate.Email},
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
			fmt.Sprintf("ssh-setup:%s|%s|bind", msg.Name, user.Email),
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

	case "delete-backup":
		return a, pushCmd(screens.NewConfirm(
			fmt.Sprintf("Securely delete the old (pre-rotation) key backup for %q? Only do this once you've confirmed the current key works.", msg.Name),
			"delete-backup:"+msg.Name,
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
			return a, a.passphraseSetProtectedFormCmd(msg.Name)
		}
		return a, a.passphraseSetFormCmd(msg.Name)

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
			{Label: "Path:", Placeholder: "e.g. ~/work", Validate: func(p string) error { return validate.BindPath(p, true) }},
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
		return a, a.importPathFormCmd("")

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
			{Label: "Profile Name:", Value: name, Placeholder: "e.g. original", Validate: validate.IdentityName},
			{Label: "Email Address:", Value: email, Placeholder: "e.g. you@example.com", Validate: validate.Email},
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
		return a, pushCmd(screens.NewHealth(a.store, a.theme))

	case "policy":
		if !git.IsInRepo() {
			return a, core.ShowToastCmd("not in a git repository — repository policy applies inside a repo", theme.ToastStyleError, 4*time.Second)
		}
		return a, pushCmd(screens.NewPolicyScreen(a.store, a.theme))

	case "signers":
		if !git.IsInRepo() {
			return a, core.ShowToastCmd("not in a git repository", theme.ToastStyleError, 4*time.Second)
		}
		return a, pushCmd(screens.NewSignersScreen(a.store, a.theme))

	case "verify":
		if !git.IsInRepo() {
			return a, core.ShowToastCmd("not in a git repository — run Verify inside a repository", theme.ToastStyleError, 4*time.Second)
		}
		return a, pushCmd(screens.NewVerifyScreen(a.store, a.theme))

	case "verify-set-range":
		return a, pushCmd(screens.NewForm("Verify Range", "Commit range to check (blank = last 50 commits)", "verify-range", []screens.FormInput{
			{Label: "Range:", Placeholder: "e.g. origin/main..HEAD"},
		}, a.theme).Skippable())

	case "policy-edit":
		return a, pushCmd(screens.NewConfirm(
			"Require commits to be signed in this repository?",
			"policy-require-signing",
			a.theme,
		))

	case "policy-hook-install":
		return a, a.runTaskCmd("hook", "install", func() (opResult, error) {
			return opHook("install")
		})

	case "signers-add-identity":
		var opts []screens.Option
		opts = append(opts, screens.Option{Label: fmt.Sprintf("(active identity: %s)", a.store.Current), Key: "__active__"})
		for _, u := range a.store.Users {
			opts = append(opts, screens.Option{Label: u.Name, Key: u.Name})
		}
		opts = append(opts, screens.Option{Label: "Cancel", Key: ""})
		return a, pushCmd(screens.NewOptions(
			"Add Signer From Identity",
			core.OptionsHelp(),
			"policy-signer-identity",
			opts,
			a.theme,
		))

	case "signers-add-email":
		return a, pushCmd(screens.NewForm("Add Signer", "Add a contributor who isn't a local git-user identity", "policy-signer-email", []screens.FormInput{
			{Label: "Email:"},
			{Label: "Public key file path:", Placeholder: "e.g. ~/.ssh/id_ed25519.pub"},
		}, a.theme))

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
			{Label: "Repository URL:", Placeholder: "git@github.com:user/repo.git", Validate: validate.RepoURL},
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
		return a, a.syncSetupFormCmd("")

	case "config":
		return a.handleConfigAction(msg.Name)

	case "token":
		return a.handleTokenAction(msg.Name)

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

// handleTokenAction opens the HTTPS-token management menu for an identity —
// the TUI counterpart of `git-user token <name>`. "Remove" is only offered
// when a token is actually stored, mirroring how the SSH-key section of the
// detail screen only shows key-dependent actions once a key exists.
func (a *App) handleTokenAction(name string) (tea.Model, tea.Cmd) {
	if a.store.FindUser(name) == nil {
		return a, core.ShowToastCmd("identity not found", theme.ToastStyleError, 3*time.Second)
	}
	opts := []screens.Option{{Label: "Set/update token", Key: "set"}}
	if keyring.HasHTTPSToken(name) {
		opts = append(opts, screens.Option{Label: "Remove token", Key: "remove"})
	}
	opts = append(opts, screens.Option{Label: "Cancel", Key: ""})
	return a, pushCmd(screens.NewOptions(
		fmt.Sprintf("HTTPS Token: %s", name),
		core.OptionsHelp(),
		"token-action:"+name,
		opts,
		a.theme,
	))
}

func (a *App) pushExportForm(names []string) tea.Cmd {
	context := "export-all"
	if names != nil {
		context = "export:" + strings.Join(names, ",")
	}
	return pushCmd(screens.NewForm("Export Identities", "Encrypt identities into a bundle file", context, []screens.FormInput{
		{Label: "Encryption Passphrase:", IsPassword: true, Validate: func(p string) error { return validate.Passphrase(p, 8) }},
		{Label: "Confirm Passphrase:", IsPassword: true, Validate: func(p string) error { return validate.Passphrase(p, 8) }},
	}, a.theme))
}

// registerFormCmd builds the name/email registration form, shared by the
// initial "register"/"register-temp" action and by handleFormResult when a
// submitted name or email is rejected (taken, invalid) — reopening it
// prefilled with what was already typed instead of discarding it back to the
// dashboard, which used to force restarting the whole registration.
func (a *App) registerFormCmd(kind, name, email string) tea.Cmd {
	title := "Register New Identity"
	help := "Enter profile name and email address"
	if kind == "register-temp" {
		title = "Create Temporary Profile"
		help = "Profile is deleted automatically when you switch away or log out"
	}
	return pushCmd(screens.NewForm(title, help, kind, []screens.FormInput{
		{Label: "Profile Name:", Value: name, Placeholder: "e.g. work", Validate: validate.IdentityName},
		{Label: "Email Address:", Value: email, Placeholder: "e.g. you@company.com", Validate: validate.Email},
	}, a.theme))
}

// sshPassphraseFormCmd prompts for a new key's passphrase during key
// generation (register/bind chains). Reused on submit (mismatched
// confirmation) to reopen the same step instead of losing the profile
// name/email/key-choice/filename already collected earlier in the chain.
func (a *App) sshPassphraseFormCmd(name, email, mode, choice, keyPath string) tea.Cmd {
	return pushCmd(screens.NewForm("SSH Key Passphrase", "Optional: protect the new key (leave empty or press Esc to skip)", fmt.Sprintf("ssh-passphrase:%s|%s|%s|%s|%s", name, email, mode, choice, keyPath), []screens.FormInput{
		{Label: "New Passphrase:", IsPassword: true, Hint: validate.PassphraseHintLine},
		{Label: "Confirm Passphrase:", IsPassword: true},
	}, a.theme).Skippable())
}

// sshKeyNameFormCmd prompts for the filename of the key about to be
// generated (stored inside the managed SSH directory). It's pre-filled with
// a suggestion that doesn't collide with anything already on disk, but the
// user is free to type over it — the field validates live and again at
// submit, so a name that turns out to collide is caught immediately instead
// of the generator silently reusing whatever file happened to already be
// there (the previous behavior when register/rekey always forced the fixed
// git_<name> filename).
func (a *App) sshKeyNameFormCmd(name, email, mode, prefill string) tea.Cmd {
	suggestion := prefill
	if suggestion == "" {
		var err error
		suggestion, err = config.SuggestSSHKeyFilename(name)
		if err != nil {
			suggestion = "git_" + name
		}
	}
	return pushCmd(screens.NewForm("SSH Key Filename", "Choose a filename for the new key (stored in ~/.ssh)", fmt.Sprintf("ssh-keyname:%s|%s|%s", name, email, mode), []screens.FormInput{
		{Label: "Key Filename:", Value: suggestion, Placeholder: "e.g. git_work", Validate: validateNewSSHKeyFilename},
	}, a.theme))
}

// validateNewSSHKeyFilename combines the filename syntax check with a live
// on-disk collision check, so typing over the pre-filled suggestion with the
// name of a key that already exists is rejected immediately instead of
// silently reusing that file once generation actually runs.
func validateNewSSHKeyFilename(filename string) error {
	if err := validate.SSHKeyFilename(filename); err != nil {
		return err
	}
	path, err := config.SSHKeyPathForFilename(filename)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("a key named %q already exists — pick a different name, or cancel and use \"Use existing key\" instead", filename)
	}
	if _, err := os.Stat(path + ".pub"); err == nil {
		return fmt.Errorf("a public key named %q already exists — pick a different name", filename+".pub")
	}
	return nil
}

// sshKeyPathFormCmd is the manual-entry fallback for typing an exact path to
// an existing key, for keys that live outside the managed SSH directory (and
// so wouldn't show up in sshExistingKeyPickerCmd's list).
func (a *App) sshKeyPathFormCmd(name, email, mode string) tea.Cmd {
	return pushCmd(screens.NewForm("Existing SSH Key", "Path to your SSH private key", fmt.Sprintf("ssh-keypath:%s|%s|%s", name, email, mode), []screens.FormInput{
		{Label: "Key Path:", Placeholder: "e.g. ~/.ssh/id_ed25519", Validate: func(p string) error { return validate.SSHKeyPath(p, true) }},
	}, a.theme))
}

// sshExistingKeyPickerCmd lists the private key files found in the managed
// SSH directory as an arrow-key navigable menu, so picking one doesn't
// require remembering or typing an exact path. A manual-entry fallback is
// always offered too, for keys stored outside ~/.ssh.
func (a *App) sshExistingKeyPickerCmd(name, email, mode string) tea.Cmd {
	keys, err := config.ListSSHKeyFiles()
	if err != nil {
		return tea.Batch(core.ShowToastCmd(fmt.Sprintf("Could not list SSH keys: %v", err), theme.ToastStyleError, 3*time.Second), a.sshKeyPathFormCmd(name, email, mode))
	}
	if len(keys) == 0 {
		return a.sshKeyPathFormCmd(name, email, mode)
	}
	opts := make([]screens.Option, 0, len(keys)+2)
	for _, k := range keys {
		label := k.Name
		if k.Comment != "" {
			label = fmt.Sprintf("%s  (%s)", k.Name, k.Comment)
		}
		opts = append(opts, screens.Option{Label: label, Key: k.Path})
	}
	opts = append(opts, screens.Option{Label: "Enter a path manually…", Key: sshKeyPickManual})
	opts = append(opts, screens.Option{Label: "Cancel", Key: ""})
	return pushCmd(screens.NewOptions(
		"Choose an Existing SSH Key",
		core.OptionsHelp(),
		fmt.Sprintf("ssh-keypick:%s|%s|%s", name, email, mode),
		opts,
		a.theme,
	))
}

// rekeyKeyNameFormCmd prompts for the filename the rotated key should be
// written to, pre-filled with the identity's current key's basename so
// leaving it untouched rotates in place exactly like before. Reused on an
// invalid submission to reopen with what was typed rather than resetting to
// the suggestion.
func (a *App) rekeyKeyNameFormCmd(name, prefill string) tea.Cmd {
	u := a.store.FindUser(name)
	currentPath := ""
	if u != nil {
		currentPath = u.SSHKey
	}
	suggestion := prefill
	if suggestion == "" {
		if currentPath != "" {
			suggestion = filepath.Base(currentPath)
		} else if p, err := config.DefaultSSHKeyPath(name); err == nil {
			suggestion = filepath.Base(p)
		}
	}
	return pushCmd(screens.NewForm("New Key Filename", "Filename for the rotated key (stored in ~/.ssh) — keep the current name to rotate in place", "rekey-keyname:"+name, []screens.FormInput{
		{Label: "Key Filename:", Value: suggestion, Placeholder: "e.g. git_work", Validate: validateRekeyFilename(currentPath)},
	}, a.theme))
}

// validateRekeyFilename allows the special case where the chosen filename
// resolves to the identity's own current key (rotating in place — opRekey
// backs that file up before regenerating it), but rejects any other filename
// that already exists on disk, exactly like validateNewSSHKeyFilename.
func validateRekeyFilename(currentKeyPath string) func(string) error {
	return func(filename string) error {
		if err := validate.SSHKeyFilename(filename); err != nil {
			return err
		}
		path, err := config.SSHKeyPathForFilename(filename)
		if err != nil {
			return err
		}
		if path == currentKeyPath {
			return nil
		}
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("a key named %q already exists — pick a different name", filename)
		}
		return nil
	}
}

// passphraseSetProtectedFormCmd and passphraseSetFormCmd build the
// change-passphrase forms, shared with handleFormResult so a mismatched or
// empty submission reopens the same form instead of dropping back to the
// dashboard with the attempt lost.
func (a *App) passphraseSetProtectedFormCmd(name string) tea.Cmd {
	return pushCmd(screens.NewForm("Change Passphrase", "Enter current and new passphrase for "+name, "passphrase-set-protected:"+name, []screens.FormInput{
		{Label: "Current Passphrase:", IsPassword: true},
		{Label: "New Passphrase:", IsPassword: true, Hint: validate.PassphraseHintLine},
		{Label: "Confirm New Passphrase:", IsPassword: true},
	}, a.theme))
}

func (a *App) passphraseSetFormCmd(name string) tea.Cmd {
	return pushCmd(screens.NewForm("Set Passphrase", "Enter a new passphrase for "+name, "passphrase-set:"+name, []screens.FormInput{
		{Label: "New Passphrase:", IsPassword: true, Hint: validate.PassphraseHintLine},
		{Label: "Confirm New Passphrase:", IsPassword: true},
	}, a.theme))
}

// importPathFormCmd and importPassFormCmd build the two-step bundle-import
// forms. Reused by handleFormResult on failure (empty/missing path, empty
// passphrase) to reopen the same step — importPathFormCmd's prefill lets the
// user just fix a typo'd path instead of retyping it from scratch.
func (a *App) importPathFormCmd(prefill string) tea.Cmd {
	return pushCmd(screens.NewForm("Import Bundle", "Path to the encrypted bundle file (.bundle)", "import:path", []screens.FormInput{
		{Label: "Bundle Path:", Value: prefill, Placeholder: "e.g. ~/git-user-export-2026-01-01.bundle"},
	}, a.theme))
}

func (a *App) importPassFormCmd(bundlePath string) tea.Cmd {
	return pushCmd(screens.NewForm("Import Bundle", "Enter the passphrase for the bundle", "import-pass:"+bundlePath, []screens.FormInput{
		{Label: "Passphrase:", IsPassword: true},
	}, a.theme))
}

// syncSetupFormCmd builds the sync-configuration form. Reused on a
// validation failure so a mismatched confirm-passphrase doesn't also cost the
// user the repository URL they already typed.
func (a *App) syncSetupFormCmd(repoURL string) tea.Cmd {
	return pushCmd(screens.NewForm("Configure Sync", "Keep identities synchronized across devices using a private git repository", "sync-setup", []screens.FormInput{
		{Label: "Repository URL:", Value: repoURL, Placeholder: "git@github.com:user/sync.git", Validate: validate.RepoURL},
		{Label: "Passphrase:", IsPassword: true, Validate: func(p string) error { return validate.Passphrase(p, 8) }},
		{Label: "Confirm Passphrase:", IsPassword: true, Validate: func(p string) error { return validate.Passphrase(p, 8) }},
	}, a.theme))
}

// rekeyPassFormCmd prompts for the rotated key's passphrase. Reused on a
// mismatched confirmation so the user doesn't have to re-confirm the rekey
// itself, just retype the passphrase.
func (a *App) rekeyPassFormCmd(name, keyPath string) tea.Cmd {
	return pushCmd(screens.NewForm("New Key Passphrase", "Optional: protect the new key (leave empty or press Esc to skip)", fmt.Sprintf("rekey-pass:%s|%s", name, keyPath), []screens.FormInput{
		{Label: "New Passphrase:", IsPassword: true, Hint: validate.PassphraseHintLine},
		{Label: "Confirm Passphrase:", IsPassword: true},
	}, a.theme).Skippable())
}

// switchPassphraseFormCmd prompts for the SSH key passphrase during a switch.
// The passphrase is asked entirely inside the TUI.
func (a *App) switchPassphraseFormCmd(name string) tea.Cmd {
	return pushCmd(screens.NewForm("Enter Passphrase", "", "switch-pass:"+name, []screens.FormInput{
		{Label: "Enter Passphrase:", IsPassword: true},
	}, a.theme))
}

// shellPassphraseFormCmd prompts for the SSH key passphrase before opening an
// in-TUI isolated shell (the "shell-session" action) so the key is unlocked
// and loaded into ssh-agent before the subshell takes over the terminal.
func (a *App) shellPassphraseFormCmd(name string) tea.Cmd {
	return pushCmd(screens.NewForm("Enter Passphrase", "", "shell-pass:"+name, []screens.FormInput{
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
