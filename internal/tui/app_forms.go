package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"os"
	"strings"
	"time"
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

	case "clone":
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("Repository URL is required", theme.ToastStyleError, 3*time.Second)
		}
		repoURL := msg.Values[0]
		destDir := msg.Values[1]
		return a.handleCloneIdentity(fmt.Sprintf("%s|%s", repoURL, destDir))

	case "sync-setup":
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("Repository URL is required", theme.ToastStyleError, 3*time.Second)
		}
		if msg.Values[1] == "" {
			return a, core.ShowToastCmd("Passphrase is required", theme.ToastStyleError, 3*time.Second)
		}
		if msg.Values[1] != msg.Values[2] {
			return a, core.ShowToastCmd("Passphrases do not match", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("sync", "", func() (opResult, error) {
			return opSync(a.store, msg.Values[0], msg.Values[1])
		})

	case "sync-pass":
		return a, a.runTaskCmd("sync", "", func() (opResult, error) {
			return opSync(a.store, "", msg.Values[0])
		})

	case "config-set":
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("Config key is required", theme.ToastStyleError, 3*time.Second)
		}
		return a, a.runTaskCmd("config", rest, func() (opResult, error) {
			return opConfigSet(a.store, rest, msg.Values[0], msg.Values[1])
		})

	case "config-unset":
		if msg.Values[0] == "" {
			return a, core.ShowToastCmd("Config key is required", theme.ToastStyleError, 3*time.Second)
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
