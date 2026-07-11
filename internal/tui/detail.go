package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

type Detail struct {
	store               *config.Store
	name                string
	actions             components.ActionMenu
	theme               theme.Theme
	keyLoaded           bool
	keyLoadedChecked    bool
	passphraseProtected bool
	passphraseChecked   bool
}

func NewDetail(store *config.Store, name string, th theme.Theme) *Detail {
	user := store.FindUser(name)
	var actions components.ActionMenu
	if user != nil {
		actions = buildDetailActions(user, store, th)
	}
	return &Detail{
		store:   store,
		name:    name,
		actions: actions,
		theme:   th,
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
	items = append(items, components.ActionItem{Label: "🔗 Add / replace SSH key", Key: "bind"})
	items = append(items, components.ActionItem{Label: "🔄 Rotate SSH key", Key: "rekey"})
	if user.SSHKey != "" {
		items = append(items, components.ActionItem{Label: "🗑  Remove SSH key", Key: "unbind"})
		items = append(items, components.ActionItem{Label: "🔒 Manage passphrase", Key: "passphrase"})
	} else {
		items = append(items, components.ActionItem{Label: "🔒 Add passphrase (bind SSH key first)", Key: "passphrase-locked", Disabled: true})
	}

	items = append(items, components.ActionItem{Label: "Paths", IsSection: true})
	items = append(items, components.ActionItem{Label: "📁 Bind directory path", Key: "bind-path"})
	if len(user.BindPaths) > 0 {
		items = append(items, components.ActionItem{Label: "📁 Unbind directory path", Key: "unbind-path"})
	}

	items = append(items, components.ActionItem{Label: "Export", IsSection: true})
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
			CheckKeyLoadedCmd(user.SSHKey),
			CheckKeyPassphraseCmd(user.SSHKey),
		)
	}
	return nil
}

func (d *Detail) Title() string { return "Identity: " + d.name }

func (d *Detail) ShortHelp() string { return DetailHelp() }

func (d *Detail) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case StoreRefreshedMsg:
		if msg.Err == nil && msg.Store != nil {
			d.store = msg.Store
			user := msg.Store.FindUser(d.name)
			if user != nil {
				d.actions = buildDetailActions(user, msg.Store, d.theme)
			}
		}
	case KeyLoadedMsg:
		d.keyLoadedChecked = true
		d.keyLoaded = msg.Loaded
	case KeyPassphraseMsg:
		d.passphraseChecked = true
		d.passphraseProtected = msg.Protected
	case tea.KeyMsg:
		return d.handleKey(msg)
	}
	return d, nil
}

func (d *Detail) handleKey(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case KeyCtrlC, KeyQuit:
		return d, tea.Quit
	case KeyEsc:
		return d, func() tea.Msg { return ScreenPopMsg{} }
	case KeyUp, KeyK:
		d.actions.CursorUp()
	case KeyDown, KeyJ:
		d.actions.CursorDown()
	case KeyEnter:
		return d.handleEnter()
	}
	return d, nil
}

func (d *Detail) handleEnter() (Screen, tea.Cmd) {
	item := d.actions.Selected()
	if item == nil {
		return d, nil
	}

	switch item.Key {
	case "back":
		return d, func() tea.Msg { return ScreenPopMsg{} }
	case "pubkey-locked", "pubkey-push-locked", "passphrase-locked":
		return d, nil
	default:
		return d, func() tea.Msg {
			return ActionResultMsg{Kind: item.Key, Name: d.name}
		}
	}
}

func (d *Detail) View(width, height int) string {
	user := d.store.FindUser(d.name)
	if user == nil {
		return "identity not found\n"
	}

	paneWidth := theme.PaneWidth(width)
	contentH := height - 4

	leftContent := d.renderProfileCard(user, paneWidth)
	rightContent := d.actions.View(paneWidth, contentH, true)

	isActive := user.Name == d.store.Current

	var leftBox, rightBox string
	if isActive {
		leftBox = d.theme.DetailCardActive(paneWidth, contentH).Render(leftContent)
	} else {
		leftBox = d.theme.DetailCardInactive(paneWidth, contentH).Render(leftContent)
	}
	rightBox = d.theme.ActionPane(paneWidth, contentH).Render(rightContent)

	if theme.IsSingleColumn(width) {
		return rightBox
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
