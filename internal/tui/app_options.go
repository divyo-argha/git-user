package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"strings"
	"time"
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
