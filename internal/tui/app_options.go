package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"github.com/divyo-argha/git-user/internal/validate"
)

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
			return a, a.sshKeyNameFormCmd(name, email, mode, "")
		case "existing":
			return a, a.sshExistingKeyPickerCmd(name, email, mode)
		default:
			// skip
			return a, a.runTaskCmd("attach-key", name, func() (opResult, error) {
				return opAttachKey(a.store, name, email, mode, "skip", "", "", false)
			})
		}

	case "ssh-keypick":
		fields := strings.Split(data, "|") // name|email|mode
		name := field(fields, 0)
		email := field(fields, 1)
		mode := field(fields, 2)
		if msg.Choice == sshKeyPickManual {
			return a, a.sshKeyPathFormCmd(name, email, mode)
		}
		keyPath := msg.Choice
		if _, err := os.Stat(keyPath); err != nil {
			return a, core.ShowToastCmd(fmt.Sprintf("key file not found: %s", keyPath), theme.ToastStyleError, 3*time.Second)
		}
		return a, pushCmd(screens.NewConfirm(
			"Would you like to sign your Git commits automatically using this identity's SSH key?",
			fmt.Sprintf("ssh-sign:%s|%s|%s|existing||%s", name, email, mode, keyPath),
			a.theme,
		))

	case "import-resolve-name":
		fields := strings.SplitN(data, "|", 2)
		name := ""
		email := ""
		if len(fields) > 0 {
			name = fields[0]
		}
		if len(fields) > 1 {
			email = fields[1]
		}
		switch msg.Choice {
		case "diff-name":
			// Prompt for a fresh, unique profile name (keep the email).
			return a, pushCmd(screens.NewForm("Import Under a Different Name", fmt.Sprintf("The name %q is taken — choose a unique one", name), "import-original-name:"+email, []screens.FormInput{
				{Label: "Profile Name:", Placeholder: "e.g. original-2", Validate: validate.IdentityName},
			}, a.theme))
		case "rename-conflict":
			// Rename the conflicting profile (name) to a free name, then import
			// the original under the desired name.
			return a, pushCmd(screens.NewForm("Rename Conflicting Profile", fmt.Sprintf("New name for the existing profile %q to free up %q", name, name), fmt.Sprintf("import-rename-conflict:%s|%s", name, email), []screens.FormInput{
				{Label: "New Name:", Placeholder: "e.g. " + name + "-work", Validate: validate.IdentityName},
			}, a.theme))
		default:
			return a, core.ShowToastCmd("Cancelled", theme.ToastStyleInfo, 2*time.Second)
		}

	case "import-resolve-email":
		fields := strings.SplitN(data, "|", 2)
		name := ""
		if len(fields) > 0 {
			name = fields[0]
		}
		switch msg.Choice {
		case "diff-email":
			return a, pushCmd(screens.NewForm("Import With a Different Email", "The email is already in use — choose a different one", "import-original-email:"+name, []screens.FormInput{
				{Label: "Email Address:", Placeholder: "e.g. you+work@example.com", Validate: validate.Email},
			}, a.theme))
		case "diff-name":
			return a, pushCmd(screens.NewForm("Import Under a Different Name", "Choose a unique profile name", "import-original-name:", []screens.FormInput{
				{Label: "Profile Name:", Placeholder: "e.g. original", Validate: validate.IdentityName},
			}, a.theme))
		default:
			return a, core.ShowToastCmd("Cancelled", theme.ToastStyleInfo, 2*time.Second)
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

	case "clone-identity":
		fields := strings.Split(data, "|")
		repoURL := fields[0]
		destDir := ""
		if len(fields) > 1 {
			destDir = fields[1]
		}
		identity := msg.Choice
		return a, a.runTaskCmd("clone", "", func() (opResult, error) {
			return opClone(a.store, repoURL, destDir, identity, false)
		})

	case "hook":
		return a, a.runTaskCmd("hook", msg.Choice, func() (opResult, error) {
			return opHook(msg.Choice)
		})

	case "config-action":
		name := data
		switch msg.Choice {
		case "list":
			return a, a.runTaskCmd("config", name, func() (opResult, error) {
				return opConfigList(a.store, name)
			})
		case "set":
			return a, pushCmd(screens.NewForm("Set Config Key", "Custom git config key to set for "+name, "config-set:"+name, []screens.FormInput{
				{Label: "Key:", Placeholder: "e.g. init.defaultBranch", Validate: validate.GitConfigKey},
				{Label: "Value:", Placeholder: "e.g. main", Validate: validate.GitConfigValue},
			}, a.theme))
		case "unset":
			return a, pushCmd(screens.NewForm("Unset Config Key", "Custom git config key to remove for "+name, "config-unset:"+name, []screens.FormInput{
				{Label: "Key:", Placeholder: "e.g. init.defaultBranch", Validate: validate.GitConfigKey},
			}, a.theme))
		}
	}

	return a, nil
}
