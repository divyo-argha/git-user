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

// PassphraseMenu allows interactive navigation to set or remove key passphrases.
type PassphraseMenu struct {
	store               *config.Store
	name                string
	actions             components.ActionMenu
	theme               theme.Theme
	passphraseProtected bool
	passphraseChecked   bool
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

	items = append(items, components.ActionItem{Label: fmt.Sprintf("Passphrase Options: %s", pm.name), IsSection: true})

	keyStr := filepath.Base(user.SSHKey)
	if keyStr == "" {
		keyStr = "None"
	}
	items = append(items, components.ActionItem{Label: fmt.Sprintf("SSH Key Path : %s", keyStr), Key: ""})

	statusStr := pm.theme.Dim().Render("checking...")
	if pm.passphraseChecked {
		if pm.passphraseProtected {
			statusStr = pm.theme.SuccessStyle().Render("Passphrase Protected ✓")
		} else {
			statusStr = pm.theme.ErrorStyle().Render("No Passphrase ⚠")
		}
	}
	items = append(items, components.ActionItem{Label: fmt.Sprintf("Current Status : %s", statusStr), Key: ""})

	items = append(items, components.ActionItem{Label: "Actions", IsSection: true})
	items = append(items, components.ActionItem{Label: "🔒 Set / Change Passphrase", Key: "passphrase-set"})
	items = append(items, components.ActionItem{Label: "🔓 Remove Passphrase", Key: "passphrase-remove", Disabled: pm.passphraseChecked && !pm.passphraseProtected})

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
		return core.CheckKeyPassphraseCmd(user.SSHKey)
	}
	return nil
}

func (pm *PassphraseMenu) Title() string { return "Passphrase Options: " + pm.name }

func (pm *PassphraseMenu) ShortHelp() string { return "↑/↓: navigate • enter: select • esc: back" }

func (pm *PassphraseMenu) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	user := pm.store.FindUser(pm.name)
	switch msg := msg.(type) {
	case core.KeyPassphraseMsg:
		if user != nil && msg.Path == user.SSHKey {
			pm.passphraseChecked = true
			pm.passphraseProtected = msg.Protected
			pm.refreshActions()
		}
	case tea.KeyMsg:
		return pm.handleKey(msg)
	}
	return pm, nil
}

func (pm *PassphraseMenu) handleKey(msg tea.KeyMsg) (core.Screen, tea.Cmd) {
	switch msg.String() {
	case core.KeyCtrlC, core.KeyQuit:
		return pm, tea.Quit
	case core.KeyEsc:
		return pm, func() tea.Msg { return core.ScreenPopMsg{} }
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
	case "passphrase-set", "passphrase-remove":
		return pm, func() tea.Msg {
			return core.ActionResultMsg{Kind: item.Key, Name: pm.name}
		}
	}
	return pm, nil
}

func (pm *PassphraseMenu) View(width, height int) string {
	contentH := height - 4
	paneWidth := width - 6
	if paneWidth > 80 {
		paneWidth = 80
	}

	viewContent := pm.actions.View(paneWidth, contentH, true)
	return pm.theme.ActionPane(paneWidth, contentH).Render(viewContent)
}
