package screens

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// PassphraseMenu allows interactive navigation to set behavior mode, test, lock, change, or remove key passphrases.
type PassphraseMenu struct {
	store               *config.Store
	name                string
	actions             components.ActionMenu
	theme               theme.Theme
	passphraseProtected bool
	passphraseChecked   bool
	keyLoaded           bool
	keyLoadedChecked    bool
}

func NewPassphraseMenu(store *config.Store, name string, th theme.Theme) *PassphraseMenu {
	pm := &PassphraseMenu{
		store: store,
		name:  name,
		theme: th,
	}
	pm.refreshActions()
	return pm
}

func (pm *PassphraseMenu) refreshActions() {
	user := pm.store.FindUser(pm.name)
	if user == nil {
		return
	}

	var prevKey string
	if selected := pm.actions.Selected(); selected != nil {
		prevKey = selected.Key
	}

	var items []components.ActionItem

	mode := user.GetPassphraseMode()
	modeLabel := "Persistent Keychain"
	switch mode {
	case "login":
		modeLabel = "Session Agent (Ask on Login)"
	case "everytime":
		modeLabel = "Ask Every Time"
	}

	items = append(items, components.ActionItem{Label: "Security & Behavior", IsSection: true})
	items = append(items, components.ActionItem{Label: fmt.Sprintf("  Behavior   : %s", modeLabel), Key: "passphrase-mode"})
	items = append(items, components.ActionItem{Label: "  Lock Key   : Unload from SSH Agent", Key: "lock-key", Disabled: pm.keyLoadedChecked && !pm.keyLoaded})
	items = append(items, components.ActionItem{Label: "  Verify     : Test Key Passphrase", Key: "passphrase-verify", Disabled: !pm.passphraseProtected})

	items = append(items, components.ActionItem{Label: "Passphrase Management", IsSection: true})
	items = append(items, components.ActionItem{Label: "› Change     : Set / Change Passphrase", Key: "passphrase-set"})
	items = append(items, components.ActionItem{Label: "› Remove     : Remove Passphrase", Key: "passphrase-remove", Disabled: pm.passphraseChecked && !pm.passphraseProtected})

	items = append(items, components.ActionItem{Label: "", IsSection: true})
	items = append(items, components.ActionItem{Label: "← Back", Key: "back"})

	title := fmt.Sprintf("Passphrase Settings: %s", pm.name)
	pm.actions = components.NewActionMenu(title, items, pm.theme)
	if prevKey != "" {
		pm.actions.FindAndSetCursorByKey(prevKey)
	}
}

func (pm *PassphraseMenu) Init() tea.Cmd {
	user := pm.store.FindUser(pm.name)
	if user != nil && user.SSHKey != "" {
		return tea.Batch(
			core.CheckKeyPassphraseCmd(user.SSHKey),
			core.CheckKeyLoadedCmd(user.SSHKey),
		)
	}
	return nil
}

func (pm *PassphraseMenu) Title() string { return "Passphrase Options: " + pm.name }

func (pm *PassphraseMenu) ShortHelp() string {
	return "↑/↓: navigate • enter: select/toggle • esc: back"
}

func (pm *PassphraseMenu) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	user := pm.store.FindUser(pm.name)
	switch msg := msg.(type) {
	case core.KeyPassphraseMsg:
		if user != nil && msg.Path == user.SSHKey {
			pm.passphraseChecked = true
			pm.passphraseProtected = msg.Protected
			pm.refreshActions()
		}
	case core.KeyLoadedMsg:
		if user != nil && msg.Path == user.SSHKey {
			pm.keyLoadedChecked = true
			pm.keyLoaded = msg.Loaded
			pm.refreshActions()
		}
	case tea.KeyMsg:
		return pm.handleKey(msg)
	}
	return pm, nil
}

func (pm *PassphraseMenu) handleKey(msg tea.KeyMsg) (core.Screen, tea.Cmd) {
	if core.IsEscKey(msg) || msg.String() == "b" || msg.String() == "B" {
		return pm, func() tea.Msg { return core.ScreenPopMsg{} }
	}
	switch msg.String() {
	case core.KeyCtrlC, core.KeyQuit:
		return pm, tea.Quit
	case core.KeyUp, core.KeyK:
		pm.actions.CursorUp()
	case core.KeyDown, core.KeyJ:
		pm.actions.CursorDown()
	case core.KeyEnter:
		return pm.handleEnter()
	}
	return pm, nil
}

func (pm *PassphraseMenu) handleEnter() (core.Screen, tea.Cmd) {
	item := pm.actions.Selected()
	if item == nil || item.Disabled {
		return pm, nil
	}

	switch item.Key {
	case "back":
		return pm, func() tea.Msg { return core.ScreenPopMsg{} }

	case "passphrase-mode":
		user := pm.store.FindUser(pm.name)
		if user == nil {
			return pm, nil
		}
		curr := user.GetPassphraseMode()
		next := "persistent"
		labelMsg := "Persistent Keychain"
		switch curr {
		case "persistent":
			next = "login"
			labelMsg = "Session Agent (Ask on Login)"
		case "login":
			next = "everytime"
			labelMsg = "Ask Every Time"
		case "everytime":
			next = "persistent"
			labelMsg = "Persistent Keychain"
		}
		user.PassphraseMode = next
		_ = config.Save(pm.store)
		if next == "login" || next == "everytime" {
			_ = keyring.DeleteKeychainPassphrase(user.Name)
		}
		if next == "everytime" && user.SSHKey != "" {
			_ = ssh.RemoveSSHKey(user.SSHKey)
			pm.keyLoaded = false
		}
		pm.refreshActions()
		return pm, core.ShowToastCmd("Passphrase mode: "+labelMsg, theme.ToastStyleSuccess, 2*time.Second)

	case "lock-key":
		user := pm.store.FindUser(pm.name)
		if user != nil && user.SSHKey != "" {
			_ = ssh.RemoveSSHKey(user.SSHKey)
			pm.keyLoaded = false
			pm.keyLoadedChecked = true
			pm.refreshActions()
			return pm, core.ShowToastCmd("SSH key unloaded from agent", theme.ToastStyleInfo, 2*time.Second)
		}

	case "passphrase-set", "passphrase-verify":
		return pm, func() tea.Msg {
			return core.ActionResultMsg{Kind: item.Key, Name: pm.name}
		}

	case "passphrase-remove":
		if pm.store.Current != pm.name {
			return pm, core.ShowToastCmd(fmt.Sprintf("Must switch to profile %q to remove its passphrase", pm.name), theme.ToastStyleError, 3*time.Second)
		}
		return pm, func() tea.Msg {
			return core.ActionResultMsg{Kind: item.Key, Name: pm.name}
		}
	}
	return pm, nil
}

func (pm *PassphraseMenu) View(width, height int) string {
	var sb strings.Builder

	boxWidth := 58
	if width < 68 {
		boxWidth = width - 10
	}
	if boxWidth < 36 {
		boxWidth = 36
	}

	padTop := (height - 15) / 2
	if padTop < 0 {
		padTop = 0
	}
	for i := 0; i < padTop; i++ {
		sb.WriteString("\n")
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, pm.theme.Bold().Render("  PASSPHRASE OPTIONS: "+pm.name))
	lines = append(lines, "")

	user := pm.store.FindUser(pm.name)
	if user != nil {
		nameStr := user.Name
		if user.Name == pm.store.Current {
			nameStr += " " + pm.theme.Active().Render("(active profile)")
		}
		lines = append(lines, fmt.Sprintf("  Profile  : %s", nameStr))
		if user.SSHKey != "" {
			lines = append(lines, fmt.Sprintf("  Key File : %s", filepath.Base(user.SSHKey)))
		}
	}

	statusStr := pm.theme.Dim().Render("checking...")
	if pm.passphraseChecked {
		if pm.passphraseProtected {
			statusStr = pm.theme.SuccessStyle().Render("Passphrase Protected ✓")
		} else {
			statusStr = pm.theme.ErrorStyle().Render("No Passphrase ⚠")
		}
	}
	lines = append(lines, fmt.Sprintf("  Security : %s", statusStr))

	if pm.store.Current != pm.name {
		lines = append(lines, pm.theme.WarningStyle().Render("  ⚠ Must switch to this profile to remove passphrase"))
	}

	lines = append(lines, "")
	lines = append(lines, pm.theme.Dim().Render("  Options & Actions:"))

	items := pm.actions.Items()
	cursor := pm.actions.Cursor()
	for i, item := range items {
		if item.IsSection || item.Label == "" {
			continue
		}
		label := item.Label
		if i == cursor {
			label = pm.theme.Selected().Render("▶ " + label)
		} else if item.Disabled {
			label = pm.theme.Dim().Render("  " + label)
		} else {
			label = "  " + label
		}
		lines = append(lines, "  "+label)
	}

	lines = append(lines, "")
	lines = append(lines, pm.theme.Dim().Render("  ↑/↓/j/k: navigate • Enter: select/toggle • Esc: back"))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := pm.theme.ActionPane(boxWidth, 0).Render(content)

	padLeft := (width - boxWidth - 6) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	padStr := strings.Repeat(" ", padLeft)
	for _, line := range strings.Split(box, "\n") {
		sb.WriteString(padStr)
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	return sb.String()
}
