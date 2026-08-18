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
	spin                components.Spinner
}

func NewDetail(store *config.Store, name string, th theme.Theme) *Detail {
	d := &Detail{
		store:             store,
		name:              name,
		theme:             th,
		platformStatuses:  map[string]string{"GitHub": "checking", "GitLab": "checking", "Bitbucket": "checking"},
		platformUsernames: make(map[string]string),
		spin:              components.NewSpinner(th),
	}
	d.refreshActions()
	return d
}

// anyPlatformChecking reports whether at least one platform-connection check
// is still in flight, so the aggregate spinner keeps ticking until all three
// resolve (or the identity has no key / is inactive, in which case none of
// them ever start "checking").
func (d *Detail) anyPlatformChecking() bool {
	for _, status := range d.platformStatuses {
		if status == "checking" {
			return true
		}
	}
	return false
}

func (d *Detail) refreshActions() {
	user := d.store.FindUser(d.name)
	if user == nil {
		return
	}
	isActive := user.Name == d.store.Current

	var prevKey string
	if selected := d.actions.Selected(); selected != nil {
		prevKey = selected.Key
	}

	var items []components.ActionItem

	if !isActive {
		items = append(items, components.ActionItem{Label: "Identity Overview", IsSection: true})
		nameVal := "○ " + user.Name
		items = append(items, components.ActionItem{Label: fmt.Sprintf("Profile Name : %s", nameVal), Key: "rename"})
		items = append(items, components.ActionItem{Label: fmt.Sprintf("Email Address: %s", user.Email), Key: "email"})
		items = append(items, components.ActionItem{Label: fmt.Sprintf("Status       : %s", d.theme.Dim().Render("○ Inactive Profile")), Key: ""})

		// Show directory bindings even for inactive profiles.
		if len(user.BindPaths) > 0 {
			items = append(items, components.ActionItem{Label: "Directory Bindings", IsSection: true})
			for _, p := range user.BindPaths {
				items = append(items, components.ActionItem{
					Label: fmt.Sprintf("  • %s", p),
					Key:   "unbind-path:" + p,
				})
			}
			items = append(items, components.ActionItem{Label: "+ Bind a directory to this profile", Key: "bind-path"})
		} else {
			items = append(items, components.ActionItem{Label: "Directory Bindings", IsSection: true})
			items = append(items, components.ActionItem{Label: d.theme.Dim().Render("  No directories bound"), Key: ""})
			items = append(items, components.ActionItem{Label: "+ Bind a directory to this profile", Key: "bind-path"})
		}

		items = append(items, components.ActionItem{Label: "Primary Action", IsSection: true})
		items = append(items, components.ActionItem{Label: "→ Switch to this identity", Key: "switch"})
		items = append(items, components.ActionItem{Label: "⚡ Switch (this terminal only) — copies activation command", Key: "switch-session"})
		items = append(items, components.ActionItem{Label: "▶ Open isolated shell as this identity", Key: "shell-session"})

		items = append(items, components.ActionItem{Label: "Management Options", IsSection: true})
		items = append(items, components.ActionItem{Label: "› Export this identity", Key: "export"})
		items = append(items, components.ActionItem{Label: "› Remove identity", Key: "remove", IsDanger: true})

		items = append(items, components.ActionItem{Label: "", IsSection: true})
		items = append(items, components.ActionItem{Label: "← Back", Key: "back"})

		title := fmt.Sprintf("Identity Overview: %s (Inactive)", d.name)
		d.actions = components.NewActionMenu(title, items, d.theme)
		if prevKey != "" {
			d.actions.FindAndSetCursorByKey(prevKey)
		} else {
			d.actions.FindAndSetCursorByKey("switch")
		}
		return
	}

	// ── PROFILE INFO (Active Profile Only) ──────────────────────────────────────
	items = append(items, components.ActionItem{Label: "Identity Details", IsSection: true})

	nameVal := d.theme.Active().Render("● "+user.Name) + " [active]"
	items = append(items, components.ActionItem{Label: fmt.Sprintf("Profile Name : %s", nameVal), Key: "rename"})
	items = append(items, components.ActionItem{Label: fmt.Sprintf("Email Address: %s", user.Email), Key: "email"})

	// ── SSH & SECURITY STATUS ─────────────────────────────────────────────────
	items = append(items, components.ActionItem{Label: "SSH & Security", IsSection: true})

	sshKeyStr := d.theme.Dim().Render("None")
	if user.SSHKey != "" {
		sshKeyStr = filepath.Base(user.SSHKey)
	}
	items = append(items, components.ActionItem{Label: fmt.Sprintf("SSH Key File : %s", sshKeyStr), Key: "bind"})

	passphraseStr := d.theme.Dim().Render("Unknown")
	if user.SSHKey != "" {
		if d.passphraseChecked {
			if d.passphraseProtected {
				passphraseStr = d.theme.SuccessStyle().Render("Passphrase Protected ✓")
			} else {
				passphraseStr = d.theme.ErrorStyle().Render("No Passphrase ⚠")
			}
		} else {
			passphraseStr = d.theme.Dim().Render("checking...")
		}
	}
	items = append(items, components.ActionItem{Label: fmt.Sprintf("Passphrase   : %s", passphraseStr), Key: "passphrase", Disabled: user.SSHKey == ""})

	sessionStr := d.theme.Dim().Render("Test Connection")
	if user.SSHKey != "" {
		if d.keyLoadedChecked {
			if d.keyLoaded {
				sessionStr = d.theme.SuccessStyle().Render("Loaded in agent ✓")
			} else {
				sessionStr = d.theme.Dim().Render("Test Connection")
			}
		} else {
			sessionStr = d.theme.Dim().Render("checking...")
		}
	}
	items = append(items, components.ActionItem{Label: fmt.Sprintf("SSH Connection: %s", sessionStr), Key: "check-ssh", Disabled: user.SSHKey == ""})

	// ── PLATFORMS ─────────────────────────────────────────────────────────────
	items = append(items, components.ActionItem{Label: "Verified Platforms", IsSection: true})
	platformsList := []string{"GitHub", "GitLab", "Bitbucket"}
	for _, p := range platformsList {
		status := d.platformStatuses[p]
		var statusStr string
		switch status {
		case "checking":
			statusStr = d.spin.View() + " " + d.theme.Dim().Render("checking...")
		case "connected":
			username := d.platformUsernames[p]
			statusStr = d.theme.SuccessStyle().Render(fmt.Sprintf("Connected ✓ (%s)", username))
		case "not_added":
			statusStr = d.theme.Dim().Render("Not added")
		case "network_error":
			statusStr = d.theme.WarningStyle().Render("Network error ⚠ (stale state)")
		case "":
			statusStr = d.theme.Dim().Render("Not configured")
		default:
			// Unrecognized status string (e.g. a future typo or new status
			// type not yet handled here) — flag it distinctly rather than
			// silently rendering as the genuine not-configured state.
			statusStr = d.theme.ErrorStyle().Render(fmt.Sprintf("Unknown status ⚠ (%q)", status))
		}
		items = append(items, components.ActionItem{Label: fmt.Sprintf("%-13s: %s", p, statusStr), Key: ""})
	}

	// ── COMMIT SIGNING ────────────────────────────────────────────────────────
	items = append(items, components.ActionItem{Label: "Git Config Options", IsSection: true})

	signingLabel := d.theme.Dim().Render("Disabled")
	if !user.SignDisabled && user.SignKey != "" {
		signingLabel = d.theme.SuccessStyle().Render(fmt.Sprintf("Enabled (%s)", user.SignFormat))
	}
	items = append(items, components.ActionItem{Label: fmt.Sprintf("Commit Signing: %s", signingLabel), Key: "toggle-sign"})

	// ── UTILITIES ─────────────────────────────────────────────────────────────
	items = append(items, components.ActionItem{Label: "Utilities", IsSection: true})
	if isActive {
		items = append(items, components.ActionItem{Label: "› Show public key", Key: "pubkey"})
	} else {
		items = append(items, components.ActionItem{
			Label:    d.theme.Dim().Render("› Show public key (switch to this identity first)"),
			Key:      "pubkey-locked",
			Disabled: true,
		})
	}
	if user.SSHKey != "" {
		if isActive {
			items = append(items, components.ActionItem{Label: "› Publish SSH key to platform", Key: "pubkey-push"})
		} else {
			items = append(items, components.ActionItem{
				Label:    d.theme.Dim().Render("› Publish SSH key to platform (switch to this identity first)"),
				Key:      "pubkey-push-locked",
				Disabled: true,
			})
		}
	}
	items = append(items, components.ActionItem{Label: "› Rotate SSH key", Key: "rekey"})
	if user.SSHKey != "" {
		items = append(items, components.ActionItem{Label: "› Remove SSH key", Key: "unbind"})
	}
	items = append(items, components.ActionItem{Label: "› Custom git config", Key: "config"})
	items = append(items, components.ActionItem{Label: "› Export this identity", Key: "export"})

	// ── DANGER ZONE ───────────────────────────────────────────────────────────
	items = append(items, components.ActionItem{Label: "Danger Zone", IsSection: true})
	items = append(items, components.ActionItem{Label: "› Remove identity", Key: "remove", IsDanger: true})

	// ── FOOTER ────────────────────────────────────────────────────────────────
	items = append(items, components.ActionItem{Label: "", IsSection: true})
	items = append(items, components.ActionItem{Label: "← Back", Key: "back"})

	title := fmt.Sprintf("Identity Details: %s", d.name)
	d.actions = components.NewActionMenu(title, items, d.theme)
	if prevKey != "" {
		d.actions.FindAndSetCursorByKey(prevKey)
	}
}

func (d *Detail) Init() tea.Cmd {
	user := d.store.FindUser(d.name)
	if user != nil && user.Name == d.store.Current && user.SSHKey != "" {
		return tea.Batch(
			d.spin.Init(),
			core.CheckKeyLoadedCmd(user.SSHKey),
			core.CheckKeyPassphraseCmd(user.SSHKey),
			core.CheckPlatformConnectionCmd(d.name, user.SSHKey, "GitHub", "git@github.com", []string{"Hi ", "successfully authenticated"}),
			core.CheckPlatformConnectionCmd(d.name, user.SSHKey, "GitLab", "git@gitlab.com", []string{"Welcome to GitLab", "successfully authenticated"}),
			core.CheckPlatformConnectionCmd(d.name, user.SSHKey, "Bitbucket", "git@bitbucket.org", []string{"logged in as", "successfully authenticated"}),
		)
	}
	// If inactive or no SSH key exists, mark as not added
	d.platformStatuses["GitHub"] = "not_added"
	d.platformStatuses["GitLab"] = "not_added"
	d.platformStatuses["Bitbucket"] = "not_added"
	return nil
}

func (d *Detail) Title() string { return "Identity: " + d.name }

func (d *Detail) ShortHelp() string { return core.DetailHelp() }

func (d *Detail) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	user := d.store.FindUser(d.name)
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
		if user != nil && msg.Path == user.SSHKey {
			d.keyLoadedChecked = true
			d.keyLoaded = msg.Loaded
			d.refreshActions()
		}
	case core.KeyPassphraseMsg:
		if user != nil && msg.Path == user.SSHKey {
			d.passphraseChecked = true
			d.passphraseProtected = msg.Protected
			d.refreshActions()
		}
	case core.PlatformConnectionMsg:
		if msg.ProfileName == d.name {
			d.platformStatuses[msg.Platform] = msg.Status
			if msg.Username != "" {
				d.platformUsernames[msg.Platform] = msg.Username
			}
			d.refreshActions()
		}
	case tea.KeyMsg:
		return d.handleKey(msg)
	default:
		// Keep the spinner ticking (its own tea.Tick messages land here) only
		// while a platform-connection check is still in flight.
		if d.anyPlatformChecking() {
			var cmd tea.Cmd
			d.spin, cmd = d.spin.Update(msg)
			return d, cmd
		}
	}
	return d, nil
}

func (d *Detail) handleKey(msg tea.KeyMsg) (core.Screen, tea.Cmd) {
	if core.IsEscKey(msg) || msg.String() == "b" || msg.String() == "B" {
		return d, func() tea.Msg { return core.ScreenPopMsg{} }
	}
	switch msg.String() {
	case core.KeyCtrlC:
		return d, tea.Quit
	case core.KeyQuit:
		return d, func() tea.Msg { return core.ActionResultMsg{Kind: "quit-confirm"} }
	case "s", "S":
		user := d.store.FindUser(d.name)
		if user != nil && user.Name != d.store.Current {
			return d, func() tea.Msg {
				return core.ActionResultMsg{Kind: "switch", Name: d.name}
			}
		}
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

	switch {
	case item.Key == "back":
		return d, func() tea.Msg { return core.ScreenPopMsg{} }
	case item.Key == "pubkey-locked" || item.Key == "pubkey-push-locked" || item.Key == "passphrase-locked" || item.Key == "":
		return d, nil
	case len(item.Key) > 12 && item.Key[:12] == "unbind-path:":
		// "unbind-path:<path>" — confirm then unbind.
		path := item.Key[12:]
		return d, func() tea.Msg {
			return core.ActionResultMsg{Kind: "unbind-path-confirm", Name: d.name + "|" + path}
		}
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
