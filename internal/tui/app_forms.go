package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"github.com/divyo-argha/git-user/internal/validate"
)

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
		name, email := msg.Values[0], msg.Values[1]
		if err := validate.IdentityName(name); err != nil {
			return a, tea.Batch(core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second), a.registerFormCmd(action, name, email))
		}
		if err := validate.Email(email); err != nil {
			return a, tea.Batch(core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second), a.registerFormCmd(action, name, email))
		}
		if a.store.IsNameTaken(name) {
			return a, tea.Batch(core.ShowToastCmd(fmt.Sprintf("identity %q already exists", name), theme.ToastStyleError, 3*time.Second), a.registerFormCmd(action, name, email))
		}
		if a.store.IsEmailTaken(email) {
			return a, tea.Batch(core.ShowToastCmd("Email already in use — each identity must have a unique email", theme.ToastStyleError, 3*time.Second), a.registerFormCmd(action, name, email))
		}
		return a, pushCmd(screens.NewOptions(
			"SSH Key Setup",
			core.OptionsHelp(),
			fmt.Sprintf("ssh-setup:%s|%s|%s", name, email, action),
			[]screens.Option{
				{Label: "Generate new key automatically (recommended)", Key: "generate"},
				{Label: "Use existing key (provide path)", Key: "existing"},
				{Label: "Skip for now (set up later)", Key: "skip"},
			},
			a.theme,
		))

	case "rename":
		if err := validate.IdentityName(msg.Values[0]); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("rename", rest, func() (opResult, error) {
			err := opRename(a.store, rest, msg.Values[0])
			return opResult{detail: fmt.Sprintf("Renamed %q → %q", rest, msg.Values[0])}, err
		})

	case "email":
		if err := validate.Email(msg.Values[0]); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("email", rest, func() (opResult, error) {
			err := opChangeEmail(a.store, rest, msg.Values[0])
			return opResult{detail: fmt.Sprintf("Updated %q → email is now %s", rest, msg.Values[0])}, err
		})

	case "ssh-keyname":
		fields := strings.Split(rest, "|")
		name, email, mode := field(fields, 0), field(fields, 1), field(fields, 2)
		filename := msg.Values[0]
		if err := validateNewSSHKeyFilename(filename); err != nil {
			return a, tea.Batch(core.ShowToastCmd(err.Error(), theme.ToastStyleError, 4*time.Second), a.sshKeyNameFormCmd(name, email, mode, filename))
		}
		keyPath, err := config.SSHKeyPathForFilename(filename)
		if err != nil {
			return a, tea.Batch(core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second), a.sshKeyNameFormCmd(name, email, mode, filename))
		}
		return a, a.sshPassphraseFormCmd(name, email, mode, "generate", keyPath)

	case "ssh-passphrase":
		fields := strings.Split(rest, "|")
		name, email, mode, choice, keyPath := field(fields, 0), field(fields, 1), field(fields, 2), field(fields, 3), field(fields, 4)
		newPass := msg.Values[0]
		if newPass != msg.Values[1] {
			return a, tea.Batch(core.ShowToastCmd("Passphrases do not match", theme.ToastStyleError, 3*time.Second), a.sshPassphraseFormCmd(name, email, mode, choice, keyPath))
		}
		return a, pushCmd(screens.NewConfirm(
			"Would you like to sign your Git commits automatically using this identity's SSH key?",
			fmt.Sprintf("ssh-sign:%s|%s|%s|%s|%s|%s", name, email, mode, choice, newPass, keyPath),
			a.theme,
		))

	case "ssh-keypath":
		fields := strings.Split(rest, "|")
		name, email, mode := field(fields, 0), field(fields, 1), field(fields, 2)
		if err := validate.SSHKeyPath(msg.Values[0], true); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		keyPath := expandPath(msg.Values[0])
		return a, pushCmd(screens.NewConfirm(
			"Would you like to sign your Git commits automatically using this identity's SSH key?",
			fmt.Sprintf("ssh-sign:%s|%s|%s|existing||%s", name, email, mode, keyPath),
			a.theme,
		))

	case "bind-path":
		if err := validate.BindPath(msg.Values[0], true); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("bind-path", rest, func() (opResult, error) {
			err := opBindPath(a.store, rest, msg.Values[0])
			return opResult{detail: fmt.Sprintf("Bound directory %q to %q", msg.Values[0], rest)}, err
		})

	case "export", "export-all":
		var names []string
		all := action == "export-all"
		if !all && rest != "" {
			names = strings.Split(rest, ",")
		}
		if err := validate.PassphraseMatch(msg.Values[0], msg.Values[1], 8); err != nil {
			return a, tea.Batch(core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second), a.pushExportForm(names))
		}
		return a, a.runTaskCmd("export", "", func() (opResult, error) {
			return opExport(a.store, names, all, msg.Values[0])
		})

	case "import:path":
		if msg.Values[0] == "" {
			return a, tea.Batch(core.ShowToastCmd("Bundle path cannot be empty", theme.ToastStyleError, 3*time.Second), a.importPathFormCmd(""))
		}
		bundlePath := expandPath(msg.Values[0])
		if _, err := os.Stat(bundlePath); err != nil {
			return a, tea.Batch(core.ShowToastCmd(fmt.Sprintf("Bundle file not found: %s", msg.Values[0]), theme.ToastStyleError, 3*time.Second), a.importPathFormCmd(msg.Values[0]))
		}
		return a, a.importPassFormCmd(bundlePath)

	case "import-pass":
		if msg.Values[0] == "" {
			return a, tea.Batch(core.ShowToastCmd("Passphrase must not be empty", theme.ToastStyleError, 3*time.Second), a.importPassFormCmd(rest))
		}
		return a, a.runTaskCmd("import", "", func() (opResult, error) {
			return opImport(a.store, rest, msg.Values[0], false)
		})

	case "import-original":
		name := msg.Values[0]
		email := msg.Values[1]
		if err := validate.IdentityName(name); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		if err := validate.Email(email); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		return a.startOriginalImport(name, email)

	case "import-original-name":
		// User chose to import under a different (unique) name; keep email.
		name := msg.Values[0]
		if err := validate.IdentityName(name); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		return a.startOriginalImport(name, rest)

	case "import-original-email":
		// User chose a new email too; name is carried in the context.
		email := msg.Values[0]
		if err := validate.Email(email); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		return a.startOriginalImport(rest, email)

	case "import-rename-conflict":
		// Rename the conflicting profile to a free name, then import the
		// original identity under the freed-up desired name.
		fields := strings.SplitN(rest, "|", 2)
		conflictName := ""
		email := ""
		if len(fields) > 0 {
			conflictName = fields[0]
		}
		if len(fields) > 1 {
			email = fields[1]
		}
		newName := msg.Values[0]
		if err := validate.IdentityName(newName); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		if newName == conflictName {
			return a, core.ShowToastCmd("Please enter a different name for the conflicting profile", theme.ToastStyleError, 3*time.Second)
		}
		if a.store.IsNameTaken(newName) {
			return a, core.ShowToastCmd(fmt.Sprintf("Name %q is also taken — choose another", newName), theme.ToastStyleError, 3*time.Second)
		}
		if err := opRename(a.store, conflictName, newName); err != nil {
			return a, core.ShowToastCmd(fmt.Sprintf("Renaming failed: %v", err), theme.ToastStyleError, 3*time.Second)
		}
		return a.startOriginalImport(conflictName, email)

	case "push-cred:github", "push-cred:gitlab":
		platform := strings.TrimPrefix(action, "push-cred:")
		if strings.TrimSpace(msg.Values[0]) == "" {
			return a, tea.Batch(core.ShowToastCmd("Token required to interact with the API", theme.ToastStyleError, 3*time.Second), a.credentialFormCmd(platform))
		}
		return a, a.runTaskCmd("pubkey-push", platform, func() (opResult, error) {
			return opPushKeyWithCredential(a.store, platform, "", msg.Values[0])
		})

	case "push-cred:bitbucket":
		if strings.TrimSpace(msg.Values[0]) == "" || strings.TrimSpace(msg.Values[1]) == "" {
			return a, tea.Batch(core.ShowToastCmd("Username and App Password are required", theme.ToastStyleError, 3*time.Second), a.credentialFormCmd("bitbucket"))
		}
		return a, a.runTaskCmd("pubkey-push", "bitbucket", func() (opResult, error) {
			return opPushKeyWithCredential(a.store, "bitbucket", msg.Values[0], msg.Values[1])
		})

	case "passphrase-set-protected":
		if msg.Values[1] != msg.Values[2] {
			return a, tea.Batch(core.ShowToastCmd("New passphrases do not match", theme.ToastStyleError, 3*time.Second), a.passphraseSetProtectedFormCmd(rest))
		}
		if msg.Values[1] == "" {
			return a, tea.Batch(core.ShowToastCmd("Passphrase must not be empty", theme.ToastStyleError, 3*time.Second), a.passphraseSetProtectedFormCmd(rest))
		}
		return a, a.runTaskCmd("passphrase-set", rest, func() (opResult, error) {
			err := opPassphraseSet(a.store, rest, msg.Values[0], msg.Values[1])
			return opResult{detail: fmt.Sprintf("Passphrase changed for %q", rest)}, err
		})

	case "passphrase-set":
		if msg.Values[0] != msg.Values[1] {
			return a, tea.Batch(core.ShowToastCmd("Passphrases do not match", theme.ToastStyleError, 3*time.Second), a.passphraseSetFormCmd(rest))
		}
		if msg.Values[0] == "" {
			return a, tea.Batch(core.ShowToastCmd("Passphrase must not be empty", theme.ToastStyleError, 3*time.Second), a.passphraseSetFormCmd(rest))
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

	case "rekey-keyname":
		name := rest
		filename := msg.Values[0]
		currentPath := ""
		if u := a.store.FindUser(name); u != nil {
			currentPath = u.SSHKey
		}
		if err := validateRekeyFilename(currentPath)(filename); err != nil {
			return a, tea.Batch(core.ShowToastCmd(err.Error(), theme.ToastStyleError, 4*time.Second), a.rekeyKeyNameFormCmd(name, filename))
		}
		keyPath, err := config.SSHKeyPathForFilename(filename)
		if err != nil {
			return a, tea.Batch(core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second), a.rekeyKeyNameFormCmd(name, filename))
		}
		return a, a.rekeyPassFormCmd(name, keyPath)

	case "rekey-pass":
		fields := strings.SplitN(rest, "|", 2)
		name, keyPath := field(fields, 0), field(fields, 1)
		if msg.Values[0] != msg.Values[1] {
			return a, tea.Batch(core.ShowToastCmd("Passphrases do not match", theme.ToastStyleError, 3*time.Second), a.rekeyPassFormCmd(name, keyPath))
		}
		return a, a.runTaskCmd("rekey", name, func() (opResult, error) {
			return opRekey(a.store, name, keyPath, msg.Values[0])
		})

	case "switch-pass":
		return a, a.runTaskCmd("switch", rest, func() (opResult, error) {
			return opSwitch(a.store, rest, msg.Values[0])
		})

	case "check-ssh-pass":
		return a, a.runTaskCmd("check-ssh", rest, func() (opResult, error) {
			return opCheckSSH(a.store, rest, msg.Values[0])
		})

	case "clone":
		if err := validate.RepoURL(msg.Values[0]); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		repoURL := msg.Values[0]
		destDir := msg.Values[1]
		return a.handleCloneIdentity(fmt.Sprintf("%s|%s", repoURL, destDir))

	case "sync-setup":
		if err := validate.RepoURL(msg.Values[0]); err != nil {
			return a, tea.Batch(core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second), a.syncSetupFormCmd(msg.Values[0]))
		}
		if err := validate.PassphraseMatch(msg.Values[1], msg.Values[2], 8); err != nil {
			return a, tea.Batch(core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second), a.syncSetupFormCmd(msg.Values[0]))
		}
		return a, a.runTaskCmd("sync", "", func() (opResult, error) {
			return opSync(a.store, msg.Values[0], msg.Values[1])
		})

	case "sync-pass":
		return a, a.runTaskCmd("sync", "", func() (opResult, error) {
			return opSync(a.store, "", msg.Values[0])
		})

	case "config-set":
		if err := validate.GitConfigKey(msg.Values[0]); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		if err := validate.GitConfigValue(msg.Values[1]); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("config", rest, func() (opResult, error) {
			return opConfigSet(a.store, rest, msg.Values[0], msg.Values[1])
		})

	case "config-unset":
		if err := validate.GitConfigKey(msg.Values[0]); err != nil {
			return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("config", rest, func() (opResult, error) {
			return opConfigUnset(a.store, rest, msg.Values[0])
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

// startOriginalImport begins importing the original identity under the given
// name/email. It detects conflicts and routes the user to resolve them (pick a
// unique name, pick a new email, or rename the conflicting profile) entirely
// within the TUI before performing the actual import.
func (a *App) startOriginalImport(name, email string) (tea.Model, tea.Cmd) {
	if err := validate.IdentityName(name); err != nil {
		return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
	}
	if err := validate.Email(email); err != nil {
		return a, core.ShowToastCmd(err.Error(), theme.ToastStyleError, 3*time.Second)
	}

	if a.store.IsNameTaken(name) {
		conflicting := name
		return a, pushCmd(screens.NewOptions(
			fmt.Sprintf("Identity %q already exists", name),
			core.OptionsHelp(),
			fmt.Sprintf("import-resolve-name:%s|%s", name, email),
			[]screens.Option{
				{Label: "Import using a different name", Key: "diff-name"},
				{Label: fmt.Sprintf("Rename conflicting profile %q, then import", conflicting), Key: "rename-conflict"},
				{Label: "Cancel", Key: ""},
			},
			a.theme,
		))
	}

	if a.store.IsEmailTaken(email) {
		// The original identity's email is already used by a registered
		// profile — but the original identity is being imported precisely to
		// keep it. Allow choosing a different email, or pick another name.
		return a, pushCmd(screens.NewOptions(
			"Email already in use",
			core.OptionsHelp(),
			fmt.Sprintf("import-resolve-email:%s|%s", name, email),
			[]screens.Option{
				{Label: "Use a different email", Key: "diff-email"},
				{Label: "Import using a different name", Key: "diff-name"},
				{Label: "Cancel", Key: ""},
			},
			a.theme,
		))
	}

	return a, a.runTaskCmd("import-original", "", func() (opResult, error) {
		return opImportOriginal(a.store, name, email)
	})
}
