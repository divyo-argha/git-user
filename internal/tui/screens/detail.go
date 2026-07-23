package screens

import (
	"fmt"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

type DetailPane int

const (
	DetailPaneProfile DetailPane = iota
	DetailPaneActions
)

type Detail struct {
	store               *config.Store
	name                string
	actions             components.ActionMenu
	activePane          DetailPane
	animFrame           uint64
	theme               theme.Theme
	keyLoaded           bool
	keyLoadedChecked    bool
	passphraseProtected bool
	passphraseChecked   bool
	platformStatuses    map[string]string // "checking", "connected", "not_added", "network_error"
	platformUsernames   map[string]string
}

func NewDetail(store *config.Store, name string, th theme.Theme) *Detail {
	user := store.FindUser(name)
	var actions components.ActionMenu
	if user != nil {
		actions = buildDetailActions(user, store, th)
	}
	return &Detail{
		store:             store,
		name:              name,
		actions:           actions,
		activePane:        DetailPaneActions,
		theme:             th,
		platformStatuses:  map[string]string{"GitHub": "checking", "GitLab": "checking", "Bitbucket": "checking"},
		platformUsernames: make(map[string]string),
	}
}

func buildDetailActions(user *config.User, store *config.Store, th theme.Theme) components.ActionMenu {
	var items []components.ActionItem
	isActive := user.Name == store.Current

	if !isActive {
		items = append(items, components.ActionItem{Label: "Identity", IsSection: true})
		items = append(items, components.ActionItem{Label: "⚡ Switch to this identity", Key: "switch"})
	} else {
		items = append(items, components.ActionItem{Label: "Identity", IsSection: true})
	}

	items = append(items, components.ActionItem{Label: "✏  Rename", Key: "rename"})
	items = append(items, components.ActionItem{Label: "✏  Change email", Key: "email"})
	items = append(items, components.ActionItem{Label: "SSH & Security", IsSection: true})

	if isActive {
		items = append(items, components.ActionItem{Label: "🔑 Show public key", Key: "pubkey"})
		if user.SSHKey != "" {
			items = append(items, components.ActionItem{Label: "🚀 Publish SSH key to platform", Key: "pubkey-push"})
		}
	} else {
		items = append(items, components.ActionItem{Label: "🔑 Show public key (switch first)", Key: "pubkey-locked", Disabled: true})
		if user.SSHKey != "" {
			items = append(items, components.ActionItem{Label: "🚀 Publish SSH key (switch first)", Key: "pubkey-push-locked", Disabled: true})
		}
	}
	items = append(items, components.ActionItem{Label: "⚡ Check SSH connection", Key: "check-ssh"})
	items = append(items, components.ActionItem{Label: "🔗 Add / replace SSH key", Key: "bind"})
	items = append(items, components.ActionItem{Label: "🔄 Rotate SSH key", Key: "rekey"})
	if user.SSHKey != "" {
		items = append(items, components.ActionItem{Label: "🗑  Remove SSH key", Key: "unbind"})
		items = append(items, components.ActionItem{Label: "🔒 Manage passphrase", Key: "passphrase"})
	} else {
		items = append(items, components.ActionItem{Label: "🔒 Add passphrase (bind SSH key first)", Key: "passphrase-locked", Disabled: true})
	}

	items = append(items, components.ActionItem{Label: "Paths & Export", IsSection: true})
	items = append(items, components.ActionItem{Label: "📁 Bind directory path", Key: "bind-path"})
	if len(user.BindPaths) > 0 {
		items = append(items, components.ActionItem{Label: "📁 Unbind directory path", Key: "unbind-path"})
	}
	items = append(items, components.ActionItem{Label: "📤 Export this identity", Key: "export"})
	items = append(items, components.ActionItem{Label: "Danger Zone", IsSection: true})
	items = append(items, components.ActionItem{Label: "🗑  Remove identity", Key: "remove", IsDanger: true})
	items = append(items, components.ActionItem{Label: "", IsSection: true})
	items = append(items, components.ActionItem{Label: "← Back", Key: "back"})

	return components.NewActionMenu(items, th)
}

func (d *Detail) Init() tea.Cmd {
	user := d.store.FindUser(d.name)
	if user != nil && user.SSHKey != "" {
		return tea.Batch(
			core.CheckKeyLoadedCmd(user.SSHKey),
			core.CheckKeyPassphraseCmd(user.SSHKey),
			core.CheckPlatformConnectionCmd(user.SSHKey, "GitHub", "git@github.com", []string{"Hi ", "successfully authenticated"}),
			core.CheckPlatformConnectionCmd(user.SSHKey, "GitLab", "git@gitlab.com", []string{"Welcome to GitLab", "successfully authenticated"}),
			core.CheckPlatformConnectionCmd(user.SSHKey, "Bitbucket", "git@bitbucket.org", []string{"logged in as", "successfully authenticated"}),
		)
	}
	// If no SSH key exists, mark them as not added immediately
	d.platformStatuses["GitHub"] = "not_added"
	d.platformStatuses["GitLab"] = "not_added"
	d.platformStatuses["Bitbucket"] = "not_added"
	return nil
}

func (d *Detail) Title() string { return "Identity: " + d.name }

func (d *Detail) ShortHelp() string { return core.DetailHelp() }

func (d *Detail) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case core.AnimTickMsg:
		d.animFrame++
		return d, nil
	case core.StoreRefreshedMsg:
		if msg.Err == nil && msg.Store != nil {
			d.store = msg.Store
			user := msg.Store.FindUser(d.name)
			if user != nil {
				d.actions = buildDetailActions(user, msg.Store, d.theme)
			}
		}
	case core.KeyLoadedMsg:
		d.keyLoadedChecked = true
		d.keyLoaded = msg.Loaded
	case core.KeyPassphraseMsg:
		d.passphraseChecked = true
		d.passphraseProtected = msg.Protected
	case core.PlatformConnectionMsg:
		d.platformStatuses[msg.Platform] = msg.Status
		if msg.Username != "" {
			d.platformUsernames[msg.Platform] = msg.Username
		}
	case tea.KeyMsg:
		return d.handleKey(msg)
	}
	return d, nil
}

func (d *Detail) handleKey(msg tea.KeyMsg) (core.Screen, tea.Cmd) {
	switch msg.String() {
	case core.KeyCtrlC, core.KeyQuit:
		return d, tea.Quit
	case core.KeyEsc:
		return d, func() tea.Msg { return core.ScreenPopMsg{} }
	case core.KeyTab:
		if d.activePane == DetailPaneProfile {
			d.activePane = DetailPaneActions
		} else {
			d.activePane = DetailPaneProfile
		}
	case core.KeyLeft, core.KeyH:
		d.activePane = DetailPaneProfile
	case core.KeyRight, core.KeyL:
		d.activePane = DetailPaneActions
	case core.KeyUp, core.KeyK:
		if d.activePane == DetailPaneActions {
			d.actions.CursorUp()
		}
	case core.KeyDown, core.KeyJ:
		if d.activePane == DetailPaneActions {
			d.actions.CursorDown()
		}
	case core.KeyEnter:
		if d.activePane == DetailPaneActions {
			return d.handleEnter()
		}
	}
	return d, nil
}

func (d *Detail) handleEnter() (core.Screen, tea.Cmd) {
	item := d.actions.Selected()
	if item == nil {
		return d, nil
	}

	switch item.Key {
	case "back":
		return d, func() tea.Msg { return core.ScreenPopMsg{} }
	case "pubkey-locked", "pubkey-push-locked", "passphrase-locked":
		return d, nil
	default:
		return d, func() tea.Msg {
			return core.ActionResultMsg{Kind: item.Key, Name: d.name}
		}
	}
}

func (d *Detail) View(width, height int) string {
	user := d.store.FindUser(d.name)
	if user == nil {
		return "identity not found\n"
	}

	contentH := height - 4
	if theme.IsSingleColumn(width) {
		paneWidth := theme.PaneWidth(width)
		rightContent := d.actions.View(paneWidth, contentH, true)
		return d.theme.ActionPane(paneWidth, contentH).Render(rightContent)
	}

	// Right pane: sized to fit its natural content (clamped 28..48)
	rightWidth := d.actions.PreferredWidth(28, 48)
	// Left pane: consumes all remaining width
	leftWidth := width - rightWidth - theme.PaneGap
	if leftWidth < 20 {
		leftWidth = 20
	}

	leftContent := d.renderProfileCard(user, leftWidth)
	rightContent := d.actions.View(rightWidth, contentH, d.activePane == DetailPaneActions)

	var leftBox, rightBox string
	if d.activePane == DetailPaneProfile {
		leftBox = d.theme.PulsingActivePane(leftWidth, contentH, d.animFrame).Render(leftContent)
		rightBox = d.theme.InactivePane(rightWidth, contentH).Render(rightContent)
	} else {
		leftBox = d.theme.InactivePane(leftWidth, contentH).Render(leftContent)
		rightBox = d.theme.PulsingActivePane(rightWidth, contentH, d.animFrame).Render(rightContent)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "   ", rightBox)
}


func (d *Detail) renderProfileCard(user *config.User, width int) string {
	var lines []string
	isActive := user.Name == d.store.Current

	lines = append(lines, d.theme.PaneTitle().Render("Identity Profile"))
	lines = append(lines, d.theme.SeparatorLine(width-6))
	lines = append(lines, "")

	nameVal := user.Name
	if isActive {
		nameVal = d.theme.Active().Render("● "+user.Name) + " [active]"
	} else {
		nameVal = "○ " + user.Name
	}
	if user.Source == "original" {
		nameVal += " " + d.theme.SuccessStyle().Render("(original)")
	}
	lines = append(lines, fmt.Sprintf("%s\n  %s", d.theme.Dim().Render("Profile Name:"), nameVal))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("%s\n  %s", d.theme.Dim().Render("Email Address:"), user.Email))
	lines = append(lines, "")

	sshKeyStr := "None"
	if user.SSHKey != "" {
		sshKeyStr = filepath.Base(user.SSHKey)
	}
	lines = append(lines, fmt.Sprintf("%s\n  %s", d.theme.Dim().Render("SSH Key File:"), sshKeyStr))
	lines = append(lines, "")

	passphraseStr := d.theme.Dim().Render("Unknown")
	if user.SSHKey != "" {
		if d.passphraseChecked {
			if d.passphraseProtected {
				passphraseStr = d.theme.Active().Render("Passphrase Protected ✓")
			} else {
				passphraseStr = d.theme.DangerText().Render("No Passphrase ⚠")
			}
		} else {
			passphraseStr = d.theme.Dim().Render("checking...")
		}
	}
	lines = append(lines, fmt.Sprintf("%s\n  %s", d.theme.Dim().Render("Security Status:"), passphraseStr))
	lines = append(lines, "")

	sessionStr := d.theme.Dim().Render("not loaded")
	if user.SSHKey != "" {
		if d.keyLoadedChecked {
			if d.keyLoaded {
				sessionStr = d.theme.Active().Render("Loaded in agent ✓")
			}
		} else {
			sessionStr = d.theme.Dim().Render("checking...")
		}
	}
	lines = append(lines, fmt.Sprintf("%s\n  %s", d.theme.Dim().Render("ssh-agent Session:"), sessionStr))
	lines = append(lines, "")

	// ── Platform Connections ──────────────────────────────────────────────
	lines = append(lines, d.theme.Dim().Render("Platform Connections:"))
	platformsList := []string{"GitHub", "GitLab", "Bitbucket"}
	for _, p := range platformsList {
		status := d.platformStatuses[p]
		var statusStr string
		switch status {
		case "checking":
			statusStr = d.theme.Dim().Render("checking...")
		case "connected":
			username := d.platformUsernames[p]
			statusStr = d.theme.Active().Render(fmt.Sprintf("Connected ✓ (%s)", username))
		case "not_added":
			statusStr = d.theme.Dim().Render("Not added")
		case "network_error":
			statusStr = d.theme.WarningStyle().Render("Network error ⚠ (stale state)")
		default:
			statusStr = d.theme.Dim().Render("Not configured")
		}
		lines = append(lines, fmt.Sprintf("  • %-10s %s", p+":", statusStr))
	}
	lines = append(lines, "")

	if !user.SignDisabled && user.SignKey != "" {
		lines = append(lines, fmt.Sprintf("%s\n  %s",
			d.theme.Dim().Render("Commit Signing:"),
			d.theme.Active().Render(fmt.Sprintf("Enabled (%s)", user.SignFormat)),
		))
	} else {
		lines = append(lines, fmt.Sprintf("%s\n  %s",
			d.theme.Dim().Render("Commit Signing:"),
			d.theme.Dim().Render("Disabled"),
		))
	}
	lines = append(lines, "")

	lines = append(lines, d.theme.Dim().Render("Bound Directories:"))
	if len(user.BindPaths) > 0 {
		for _, p := range user.BindPaths {
			displayPath := p
			if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
				displayPath = "~" + strings.TrimPrefix(p, home)
			}
			if len(displayPath) > 38 {
				displayPath = displayPath[:17] + "..." + displayPath[len(displayPath)-18:]
			}
			lines = append(lines, "  • "+displayPath)
		}
	} else {
		lines = append(lines, "  None")
	}

	return strings.Join(lines, "\n")
}
