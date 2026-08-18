package screens

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

type Detail struct {
	store                 *config.Store
	name                  string
	actions               components.ActionMenu
	animFrame             uint64
	theme                 theme.Theme
	keyLoaded             bool
	keyLoadedChecked      bool
	passphraseProtected   bool
	passphraseChecked     bool
	platformStatuses      map[string]string // "checking", "connected", "not_added", "network_error", "locked"
	platformUsernames     map[string]string
	spin                  components.Spinner
	platformChecksStarted bool
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
// resolve.
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
		items = append(items, components.ActionItem{Label: "Primary Action", IsSection: true})
		items = append(items, components.ActionItem{Label: "→ Switch to this identity", Key: "switch"})
	}

	items = append(items, components.ActionItem{Label: "Profile & Git Config", IsSection: true})
	items = append(items, components.ActionItem{Label: "✎ Rename profile", Key: "rename"})
	items = append(items, components.ActionItem{Label: "✉ Change email address", Key: "email"})
	items = append(items, components.ActionItem{Label: "✍ Toggle commit signing", Key: "toggle-sign"})
	items = append(items, components.ActionItem{Label: "⚙ Custom git config", Key: "config"})

	items = append(items, components.ActionItem{Label: "SSH & Security", IsSection: true})
	if user.SSHKey == "" {
		items = append(items, components.ActionItem{Label: "🔑 Bind SSH key file", Key: "bind"})
	} else {
		items = append(items, components.ActionItem{Label: "🔑 Change SSH key file", Key: "bind"})
		items = append(items, components.ActionItem{Label: "⚡ Test SSH connection", Key: "check-ssh"})
		items = append(items, components.ActionItem{Label: "🔒 Manage passphrase", Key: "passphrase"})
		if isActive {
			items = append(items, components.ActionItem{Label: "📋 Show public key", Key: "pubkey"})
			items = append(items, components.ActionItem{Label: "🚀 Publish SSH key to platform", Key: "pubkey-push"})
		} else {
			items = append(items, components.ActionItem{
				Label:    d.theme.Dim().Render("📋 Show public key (switch first)"),
				Key:      "pubkey-locked",
				Disabled: true,
			})
			items = append(items, components.ActionItem{
				Label:    d.theme.Dim().Render("🚀 Publish SSH key (switch first)"),
				Key:      "pubkey-push-locked",
				Disabled: true,
			})
		}
		items = append(items, components.ActionItem{Label: "↻ Rotate SSH key", Key: "rekey"})
		items = append(items, components.ActionItem{Label: "✕ Remove SSH key", Key: "unbind"})
	}

	items = append(items, components.ActionItem{Label: "Directory Bindings", IsSection: true})
	items = append(items, components.ActionItem{Label: "+ Bind a directory", Key: "bind-path"})
	if len(user.BindPaths) > 0 {
		for _, p := range user.BindPaths {
			items = append(items, components.ActionItem{
				Label: fmt.Sprintf("  - Unbind %s", filepath.Base(p)),
				Key:   "unbind-path:" + p,
			})
		}
	}

	items = append(items, components.ActionItem{Label: "Management Options", IsSection: true})
	items = append(items, components.ActionItem{Label: "› Export this identity", Key: "export"})
	items = append(items, components.ActionItem{Label: "› Remove identity", Key: "remove", IsDanger: true})

	items = append(items, components.ActionItem{Label: "", IsSection: true})
	items = append(items, components.ActionItem{Label: "← Back", Key: "back"})

	title := "Actions & Utilities"
	d.actions = components.NewActionMenu(title, items, d.theme)
	if prevKey != "" {
		d.actions.FindAndSetCursorByKey(prevKey)
	} else if !isActive {
		d.actions.FindAndSetCursorByKey("switch")
	} else {
		d.actions.FindAndSetCursorByKey("rename")
	}
}

// renderOverview generates the content for the left-hand status & overview pane.
func (d *Detail) renderOverview(width, height int, user *config.User) string {
	var lines []string

	lines = append(lines, d.theme.PaneTitle().Render(fmt.Sprintf("Overview: %s", user.Name)))
	lines = append(lines, "")

	isActive := user.Name == d.store.Current
	var statusBadge string
	if isActive {
		statusBadge = d.theme.Active().Render("● "+user.Name) + " " + d.theme.SuccessStyle().Render("[ACTIVE]")
	} else {
		statusBadge = d.theme.Dim().Render("○ "+user.Name) + " " + d.theme.Dim().Render("[INACTIVE]")
	}

	lines = append(lines, "  "+d.theme.Bold().Render("Profile : ")+statusBadge)
	lines = append(lines, "  "+d.theme.Bold().Render("Email   : ")+user.Email)
	lines = append(lines, "")

	// SSH & Security
	lines = append(lines, d.theme.SectionHeader().Render("SSH & SECURITY"))
	sshKeyStr := d.theme.Dim().Render("None")
	if user.SSHKey != "" {
		sshKeyStr = filepath.Base(user.SSHKey)
	}
	lines = append(lines, "  "+d.theme.Dim().Render("• ")+d.theme.Bold().Render("Key File   : ")+sshKeyStr)

	passphraseStr := d.theme.Dim().Render("Unknown")
	if user.SSHKey != "" {
		if d.passphraseChecked {
			if d.passphraseProtected {
				passphraseStr = d.theme.SuccessStyle().Render("Protected ✓")
			} else {
				passphraseStr = d.theme.ErrorStyle().Render("No Passphrase ⚠")
			}
		} else {
			passphraseStr = d.theme.Dim().Render("checking...")
		}
	} else {
		passphraseStr = d.theme.Dim().Render("None")
	}
	lines = append(lines, "  "+d.theme.Dim().Render("• ")+d.theme.Bold().Render("Passphrase : ")+passphraseStr)

	sessionStr := d.theme.Dim().Render("None")
	if user.SSHKey != "" {
		if d.keyLoadedChecked {
			if d.keyLoaded {
				sessionStr = d.theme.SuccessStyle().Render("Loaded in agent ✓")
			} else {
				sessionStr = d.theme.Dim().Render("Not loaded")
			}
		} else {
			sessionStr = d.theme.Dim().Render("checking...")
		}
	}
	lines = append(lines, "  "+d.theme.Dim().Render("• ")+d.theme.Bold().Render("SSH Agent  : ")+sessionStr)

	signingLabel := d.theme.Dim().Render("Disabled")
	if !user.SignDisabled && user.SignKey != "" {
		signingLabel = d.theme.SuccessStyle().Render(fmt.Sprintf("Enabled (%s)", user.SignFormat))
	}
	lines = append(lines, "  "+d.theme.Dim().Render("• ")+d.theme.Bold().Render("Signing    : ")+signingLabel)
	lines = append(lines, "")

	// Platform Reachability
	lines = append(lines, d.theme.SectionHeader().Render("VERIFIED PLATFORMS"))
	platformsList := []string{"GitHub", "GitLab", "Bitbucket"}
	for _, p := range platformsList {
		status := d.platformStatuses[p]
		var statusStr string
		switch status {
		case "checking":
			statusStr = d.spin.View() + " " + d.theme.Dim().Render("checking...")
		case "connected":
			username := d.platformUsernames[p]
			if username != "" {
				statusStr = d.theme.SuccessStyle().Render(fmt.Sprintf("Connected ✓ (%s)", username))
			} else {
				statusStr = d.theme.SuccessStyle().Render("Connected ✓")
			}
		case "not_added":
			statusStr = d.theme.Dim().Render("Not added")
		case "network_error":
			statusStr = d.theme.WarningStyle().Render("Network error ⚠")
		case "locked":
			statusStr = d.theme.Dim().Render("Locked (needs passphrase)")
		case "":
			statusStr = d.theme.Dim().Render("Not configured")
		default:
			statusStr = d.theme.ErrorStyle().Render(fmt.Sprintf("Unknown (%s)", status))
		}
		lines = append(lines, fmt.Sprintf("  "+d.theme.Dim().Render("• ")+"%-9s: %s", p, statusStr))
	}
	lines = append(lines, "")

	// Directory Bindings
	lines = append(lines, d.theme.SectionHeader().Render("DIRECTORY BINDINGS"))
	if len(user.BindPaths) > 0 {
		for _, p := range user.BindPaths {
			lines = append(lines, "  "+d.theme.Dim().Render("• ")+p)
		}
	} else {
		lines = append(lines, "  "+d.theme.Dim().Render("No directories bound"))
	}

	return strings.Join(lines, "\n")
}

func (d *Detail) Init() tea.Cmd {
	user := d.store.FindUser(d.name)
	if user != nil && user.SSHKey != "" {
		return tea.Batch(
			d.spin.Init(),
			core.CheckKeyLoadedCmd(user.SSHKey),
			core.CheckKeyPassphraseCmd(user.SSHKey),
		)
	}
	d.platformStatuses["GitHub"] = "not_added"
	d.platformStatuses["GitLab"] = "not_added"
	d.platformStatuses["Bitbucket"] = "not_added"
	return nil
}

func (d *Detail) maybeStartPlatformChecks(user *config.User) tea.Cmd {
	if d.platformChecksStarted || user == nil || user.SSHKey == "" {
		return nil
	}
	if !d.passphraseChecked {
		return nil
	}
	if d.passphraseProtected && !(d.keyLoadedChecked && d.keyLoaded) {
		if !d.keyLoadedChecked {
			return nil
		}
		d.platformChecksStarted = true
		d.platformStatuses["GitHub"] = "locked"
		d.platformStatuses["GitLab"] = "locked"
		d.platformStatuses["Bitbucket"] = "locked"
		return nil
	}

	d.platformChecksStarted = true
	return tea.Batch(
		core.CheckPlatformConnectionCmd(d.name, user.SSHKey, "GitHub", "git@github.com", []string{"Hi ", "successfully authenticated"}),
		core.CheckPlatformConnectionCmd(d.name, user.SSHKey, "GitLab", "git@gitlab.com", []string{"Welcome to GitLab", "successfully authenticated"}),
		core.CheckPlatformConnectionCmd(d.name, user.SSHKey, "Bitbucket", "git@bitbucket.org", []string{"logged in as", "successfully authenticated"}),
	)
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
			cmd := d.maybeStartPlatformChecks(user)
			d.refreshActions()
			if cmd != nil {
				return d, cmd
			}
		}
	case core.KeyPassphraseMsg:
		if user != nil && msg.Path == user.SSHKey {
			d.passphraseChecked = true
			d.passphraseProtected = msg.Protected
			cmd := d.maybeStartPlatformChecks(user)
			d.refreshActions()
			if cmd != nil {
				return d, cmd
			}
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
	user := d.store.FindUser(d.name)
	if user == nil {
		return d.theme.ErrorStyle().Render("Identity not found: " + d.name)
	}

	contentH := height - 4
	if contentH < 10 {
		contentH = 10
	}

	if theme.IsSingleColumn(width) {
		paneWidth := theme.PaneWidth(width)
		viewContent := d.actions.View(paneWidth, contentH, true)
		return d.theme.ActionPane(paneWidth, contentH).Render(viewContent)
	}

	// 2-Column Responsive Layout
	rightWidth := d.actions.PreferredWidth(30, 46)
	leftWidth := width - rightWidth - theme.PaneGap - 2*theme.PaneBorder
	if leftWidth < 28 {
		leftWidth = 28
	}

	leftContent := d.renderOverview(leftWidth, contentH, user)
	rightContent := d.actions.View(rightWidth, contentH, true)

	leftBox := d.theme.InactivePane(leftWidth, contentH).Render(leftContent)
	rightBox := d.theme.PulsingActivePane(rightWidth, contentH, d.animFrame).Render(rightContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "   ", rightBox)
}
