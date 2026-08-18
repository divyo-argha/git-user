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

// ── Confirm Results ───────────────────────────────────────────────────────────

func (a *App) handleConfirmResult(msg core.ConfirmResultMsg) (tea.Model, tea.Cmd) {
	parts := strings.SplitN(msg.Context, ":", 2)
	action := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	}

	// ssh-sign is an in-flow yes/no choice ("enable commit signing?"), not a
	// cancel gate — declining it must still complete registration/binding and
	// show the public key, just with signing left off.
	if action == "ssh-sign" {
		fields := strings.Split(rest, "|") // name|email|mode|choice|pass|keyPath
		name := field(fields, 0)
		email := field(fields, 1)
		mode := field(fields, 2)
		choice := field(fields, 3)
		pass := field(fields, 4)
		keyPath := field(fields, 5)
		return a, a.runTaskCmd("attach-key", name, func() (opResult, error) {
			return opAttachKey(a.store, name, email, mode, choice, pass, keyPath, msg.Confirmed)
		})
	}

	if !msg.Confirmed {
		return a, core.ShowToastCmd("Cancelled", theme.ToastStyleInfo, 2*time.Second)
	}

	switch action {
	case "quit-confirmed":
		// User confirmed quitting from the dashboard.
		a.quit = true
		return a, tea.Quit

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
		return a, pushCmd(screens.NewForm("New Key Passphrase", "Optional: protect the new key (leave empty or press Esc to skip)", "rekey-pass:"+rest, []screens.FormInput{
			{Label: "New Passphrase:", IsPassword: true},
			{Label: "Confirm Passphrase:", IsPassword: true},
		}, a.theme).Skippable())

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

	case "switch-https":
		return a, a.runTaskCmd("fix-remote", "", func() (opResult, error) {
			return opFixRemote()
		})

	case "uninstall-confirmed":
		return a, a.runTaskCmd("uninstall", "", func() (opResult, error) {
			return opUninstall(a.store)
		})
	}

	return a, nil
}
