package screens

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

type Detail struct {
	store               *config.Store
	name                string
	actions             components.ActionMenu
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
	d := &Detail{
		store:             store,
		name:              name,
		theme:             th,
		platformStatuses:  map[string]string{"GitHub": "checking", "GitLab": "checking", "Bitbucket": "checking"},
		platformUsernames: make(map[string]string),
	}
	d.refreshActions()
	return d
}

func (d *Detail) refreshActions() {
	user := d.store.FindUser(d.name)
	if user == nil {
		return
	}
	isActive := user.Name == d.store.Current

	var items []components.ActionItem

	// ── PROFILE INFO (Interactive Items) ──────────────────────────────────────
	items = append(items, components.ActionItem{Label: "Identity Details", IsSection: true})

	nameVal := user.Name
	if isActive {
		nameVal = d.theme.Active().Render("● "+user.Name) + " [active]"
	} else {
		nameVal = "○ " + user.Name
	}
	if user.Source == "original" {
		nameVal += " " + d.theme.SuccessStyle().Render("(original)")
	}
	items = append(items, components.ActionItem{Label: fmt.Sprintf("Profile Name : %s", nameVal), Key: "rename"})
	items = append(items, components.ActionItem{Label: fmt.Sprintf("Email Address: %s", user.Email), Key: "email"})

	// ── SSH & SECURITY STATUS (Interactive Items) ─────────────────────────────
	items = append(items, components.ActionItem{Label: "SSH & Security", IsSection: true})

	sshKeyStr := "None"
	if user.SSHKey != "" {
		sshKeyStr = filepath.Base(user.SSHKey)
	}
	items = append(items, components.ActionItem{Label: fmt.Sprintf("SSH Key File : %s", sshKeyStr), Key: "bind"})

	passphraseStr := "Unknown"
	if user.SSHKey != "" {
		if d.passphraseChecked {
			if d.passphraseProtected {
				passphraseStr = "Passphrase Protected ✓"
			} else {
				passphraseStr = "No Passphrase ⚠"
			}
		} else {
			passphraseStr = "checking..."
		}
	}
	items = append(items, components.ActionItem{Label: fmt.Sprintf("Passphrase   : %s", passphraseStr), Key: "passphrase", Disabled: user.SSHKey == ""})

	sessionStr := "not loaded"
	if user.SSHKey != "" {
		if d.keyLoadedChecked {
			if d.keyLoaded {
				sessionStr = "Loaded in agent ✓"
			}
		} else {
			sessionStr = "checking..."
		}
	}
	items = append(items, components.ActionItem{Label: fmt.Sprintf("Agent Status : %s", sessionStr), Key: "check-ssh", Disabled: user.SSHKey == ""})

	// ── PLATFORMS (Interactive check trigger) ──────────────────────────────────
	items = append(items, components.ActionItem{Label: "Verified Platforms", IsSection: true})
	platformsList := []string{"GitHub", "GitLab", "Bitbucket"}
	for _, p := range platformsList {
		status := d.platformStatuses[p]
		var statusStr string
		switch status {
		case "checking":
			statusStr = "checking..."
		case "connected":
			username := d.platformUsernames[p]
			statusStr = fmt.Sprintf("Connected ✓ (%s)", username)
		case "not_added":
			statusStr = "Not added"
		case "network_error":
			statusStr = "Network error ⚠ (stale state)"
		default:
			statusStr = "Not configured"
		}
		items = append(items, components.ActionItem{Label: fmt.Sprintf("%-13s: %s", p, statusStr), Key: "check-ssh", Disabled: user.SSHKey == ""})
	}

	// ── COMMIT SIGNING ────────────────────────────────────────────────────────
	items = append(items, components.ActionItem{Label: "Git Config Options", IsSection: true})

	signingLabel := "Disabled"
	if !user.SignDisabled && user.SignKey != "" {
		signingLabel = fmt.Sprintf("Enabled (%s)", user.SignFormat)
	}
	items = append(items, components.ActionItem{Label: fmt.Sprintf("Commit Signing: %s", signingLabel), Key: "toggle-sign"})

	// ── DOCKER / CLI UTILITIES / EXPORTS ──────────────────────────────────────
	items = append(items, components.ActionItem{Label: "Utilities", IsSection: true})
	if !isActive {
		items = append(items, components.ActionItem{Label: "⚡ Switch to this identity", Key: "switch"})
	}
	if isActive {
		items = append(items, components.ActionItem{Label: "🔑 Show public key", Key: "pubkey"})
		if user.SSHKey != "" {
			items = append(items, components.ActionItem{Label: "🚀 Publish SSH key to platform", Key: "pubkey-push"})
		}
	}
	items = append(items, components.ActionItem{Label: "🔄 Rotate SSH key", Key: "rekey"})
	if user.SSHKey != "" {
		items = append(items, components.ActionItem{Label: "🗑  Remove SSH key", Key: "unbind"})
	}
	items = append(items, components.ActionItem{Label: "📤 Export this identity", Key: "export"})

	// ── DANGER ZONE ───────────────────────────────────────────────────────────
	items = append(items, components.ActionItem{Label: "Danger Zone", IsSection: true})
	items = append(items, components.ActionItem{Label: "🗑  Remove identity", Key: "remove", IsDanger: true})

	// ── FOOTER ────────────────────────────────────────────────────────────────
	items = append(items, components.ActionItem{Label: "", IsSection: true})
	items = append(items, components.ActionItem{Label: "← Back", Key: "back"})

	d.actions = components.NewActionMenu(items, d.theme)
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
			d.refreshActions()
		}
	case core.KeyLoadedMsg:
		d.keyLoadedChecked = true
		d.keyLoaded = msg.Loaded
		d.refreshActions()
	case core.KeyPassphraseMsg:
		d.passphraseChecked = true
		d.passphraseProtected = msg.Protected
		d.refreshActions()
	case core.PlatformConnectionMsg:
		d.platformStatuses[msg.Platform] = msg.Status
		if msg.Username != "" {
			d.platformUsernames[msg.Platform] = msg.Username
		}
		d.refreshActions()
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
	case core.KeyUp, core.KeyK:
		d.actions.CursorUp()
	case core.KeyDown, core.KeyJ:
		d.actions.CursorDown()
	case core.KeyEnter:
		return d.handleEnter()
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
	contentH := height - 4
	paneWidth := width - 6
	if paneWidth > 80 {
		paneWidth = 80
	}

	viewContent := d.actions.View(paneWidth, contentH, true)
	return d.theme.ActionPane(paneWidth, contentH).Render(viewContent)
}
