package tui

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"strings"
	"time"
)

// ── Task Results ──────────────────────────────────────────────────────────────

func (a *App) handleTaskResult(msg core.TaskResultMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if msg.Err != nil {
		a.removeKeyPath = ""
		switch {
		case errors.Is(msg.Err, ErrNeedsPassphrase):
			if msg.Kind == "check-ssh" {
				cmds = append(cmds, a.checkSSHPassphraseFormCmd(msg.Name))
			} else {
				cmds = append(cmds, a.switchPassphraseFormCmd(msg.Name))
			}
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
		if msg.ShowReport {
			cmds = append(cmds, pushCmd(screens.NewReport(titleForKind(msg.Kind), msg.Detail, a.theme)))
		}
		if git.HasHTTPSRemotes() {
			cmds = append(cmds, pushCmd(screens.NewConfirm(
				"This repo uses HTTPS remotes. Convert to SSH for passwordless push?",
				"switch-https",
				a.theme,
			)))
		}
	case "logout":
		cmds = append(cmds, core.ShowToastCmd(firstLine(msg.Detail), theme.ToastStyleSuccess, 3*time.Second))
	case "update":
		if msg.Success && !strings.Contains(strings.ToLower(msg.Detail), "requires administrator") {
			a.statusBar.SetVersionStatus("", false)
			for _, s := range a.screenStack {
				if d, ok := s.(*screens.Dashboard); ok {
					d.SetVersionStatus("", false)
				}
			}
		}
		if msg.ShowReport {
			cmds = append(cmds, pushCmd(screens.NewReport(titleForKind(msg.Kind), msg.Detail, a.theme)))
		} else {
			cmds = append(cmds, core.ShowToastCmd(firstLine(msg.Detail), theme.ToastStyleSuccess, 3*time.Second))
		}
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
	case "rename":
		return "Identity Renamed"
	case "email":
		return "Email Updated"
	case "remove":
		return "Identity Removed"
	case "delete-key":
		return "SSH Key Files Deleted"
	case "bind":
		return "SSH Key Configured"
	case "unbind":
		return "SSH Key Unbound"
	case "rekey":
		return "SSH Key Rotated"
	case "bind-path":
		return "Directory Bound"
	case "unbind-path":
		return "Directory Unbound"
	case "passphrase-set":
		return "Passphrase Set"
	case "passphrase-remove":
		return "Passphrase Removed"
	case "passphrase-verify":
		return "Passphrase Verified"
	case "export", "export-current", "export-all":
		return "Export"
	case "import":
		return "Import"
	case "import-original":
		return "Original Identity Imported"
	case "check-ssh":
		return "SSH Connection Check"
	case "pubkey":
		return "Public Key"
	case "pubkey-push":
		return "Publish SSH Key"
	case "security":
		return "Security Audit"
	case "doctor":
		return "Diagnostics"
	case "refresh":
		return "Refresh"
	case "log":
		return "Identity Switch Log"
	case "uninstall":
		return "Uninstall"
	case "fix-remote":
		return "Remote Conversion"
	case "clone":
		return "Clone Repository"
	case "stats":
		return "Commit Identity Stats"
	case "sync":
		return "Sync"
	case "hook":
		return "Git Hooks"
	case "config":
		return "Custom Git Config"
	case "switch":
		return "Switched Identity"
	case "logout":
		return "Signed Out"
	case "update":
		return "Self-Update"
	}
	return "Result"
}
