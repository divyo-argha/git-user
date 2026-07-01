package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	tuiCyan    = lipgloss.Color("#00FFFF")
	tuiGreen   = lipgloss.Color("#00FF00")
	tuiYellow  = lipgloss.Color("#FFFF00")
	tuiGray    = lipgloss.Color("#555555")
	tuiWhite   = lipgloss.Color("#FFFFFF")
	tuiOrange  = lipgloss.Color("#FFAA00")
	tuiRed     = lipgloss.Color("#FF5555")

	tuiSelected   = lipgloss.NewStyle().Foreground(tuiCyan).Bold(true)
	tuiDim        = lipgloss.NewStyle().Foreground(tuiGray)
	tuiActive     = lipgloss.NewStyle().Foreground(tuiGreen).Bold(true)
	tuiOriginal   = lipgloss.NewStyle().Foreground(tuiGreen)
	tuiDanger     = lipgloss.NewStyle().Foreground(tuiRed)
	tuiHelp       = lipgloss.NewStyle().Foreground(tuiGray).Italic(true)
	tuiBold       = lipgloss.NewStyle().Foreground(tuiWhite).Bold(true)

	paneTitleStyle = lipgloss.NewStyle().Foreground(tuiCyan).Bold(true)

	styleActivePane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tuiCyan).
			Padding(0, 2).
			Width(44)

	styleInactivePane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tuiGray).
			Padding(0, 2).
			Width(44)

	styleDetailCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tuiGray).
			Padding(0, 2).
			Width(44)

	styleDetailActiveCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tuiGreen).
			Padding(0, 2).
			Width(44)

	styleDetailActions = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tuiCyan).
			Padding(0, 2).
			Width(44)
)

// ── Screen state ──────────────────────────────────────────────────────────────

type tuiScreen int

const (
	screenMain   tuiScreen = iota
	screenDetail
)

// ── Action result (what to do after TUI exits) ────────────────────────────────

type pendingAction struct {
	kind string // "switch","register","rename","email","bind","rekey","passphrase","session-start","session-stop","export","remove","pubkey","fix-remote","security","doctor","import","update"
	name string // identity name if applicable
	arg  string // extra arg
}

// ── Main item types ───────────────────────────────────────────────────────────

type mainItem struct {
	label     string
	isSep     bool
	isUser    bool
	userName  string
	isAction  bool
	actionKey string
}

// ── Model ─────────────────────────────────────────────────────────────────────

type tuiPane int

const (
	paneIdentities tuiPane = iota
	paneActions
)

type tuiModel struct {
	screen           tuiScreen
	store            *config.Store
	width            int
	height           int
	// main screen
	activePane       tuiPane
	identitiesCursor int
	actionsCursor    int
	identitiesList   []mainItem
	actionsList      []mainItem
	// detail screen
	detailName       string
	detailItems      []detailItem
	detailCursor     int
	// result
	quit   bool
	action *pendingAction
}

type detailItem struct {
	label    string
	isSep    bool
	key      string
	isDanger bool
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func buildIdentitiesList(store *config.Store) []mainItem {
	items := []mainItem{}
	for _, u := range store.Users {
		label := u.Name
		if u.Name == store.Current {
			label = tuiActive.Render("● "+u.Name) + "  " + tuiDim.Render(u.Email) + "  " + tuiActive.Render("[active]")
		} else {
			label = "○ " + u.Name + "  " + tuiDim.Render(u.Email)
		}
		if u.Source == "original" {
			label += "  " + tuiOriginal.Render("(original)")
		}
		items = append(items, mainItem{label: label, isUser: true, userName: u.Name})
	}
	items = append(items, mainItem{
		label:     lipgloss.NewStyle().Foreground(tuiCyan).Render("+ Register new identity"),
		isAction:  true,
		actionKey: "register",
	})
	return items
}

func buildActionsList() []mainItem {
	return []mainItem{
		{label: "Sign out (logout)", isAction: true, actionKey: "logout"},
		{label: "Fix remotes (HTTPS → SSH)", isAction: true, actionKey: "fix-remote"},
		{label: "Security audit", isAction: true, actionKey: "security"},
		{label: "Doctor (health check)", isAction: true, actionKey: "doctor"},
		{label: "Export all identities", isAction: true, actionKey: "export-all"},
		{label: "Import identities", isAction: true, actionKey: "import"},
		{label: "Import original gitconfig", isAction: true, actionKey: "import-original"},
		{label: "Update git-user", isAction: true, actionKey: "update"},
		{label: tuiDim.Render("Quit"), isAction: true, actionKey: "quit"},
	}
}

func buildDetailItems(user *config.User, store *config.Store) []detailItem {
	items := []detailItem{}

	isActive := user.Name == store.Current
	if !isActive {
		items = append(items, detailItem{label: tuiActive.Render("⚡ Switch to this identity"), key: "switch"})
		items = append(items, detailItem{isSep: true})
	}

	items = append(items, detailItem{label: "Rename", key: "rename"})
	items = append(items, detailItem{label: "Change email", key: "email"})
	items = append(items, detailItem{isSep: true})

	if isActive {
		items = append(items, detailItem{label: "Show public key", key: "pubkey"})
		if user.SSHKey != "" {
			items = append(items, detailItem{label: "Publish SSH key to Git platform", key: "pubkey-push"})
		}
	} else {
		items = append(items, detailItem{label: tuiDim.Render("Show public key (switch first)"), key: "pubkey-locked"})
		if user.SSHKey != "" {
			items = append(items, detailItem{label: tuiDim.Render("Publish SSH key (switch first)"), key: "pubkey-push-locked"})
		}
	}
	items = append(items, detailItem{label: "Add / replace SSH key", key: "bind"})
	items = append(items, detailItem{label: "Rotate SSH key", key: "rekey"})
	if user.SSHKey != "" {
		items = append(items, detailItem{label: "Remove SSH key", key: "unbind"})
	}
	items = append(items, detailItem{isSep: true})

	if user.SSHKey != "" {
		protected, err := isSSHKeyPassphraseProtected(user.SSHKey)
		if err == nil && protected {
			items = append(items, detailItem{label: "Change passphrase", key: "passphrase"})
			items = append(items, detailItem{label: "Remove passphrase", key: "passphrase-remove"})
		} else {
			items = append(items, detailItem{label: "Add passphrase", key: "passphrase"})
		}
	} else {
		items = append(items, detailItem{label: tuiDim.Render("Add passphrase (bind SSH key first)"), key: "passphrase-locked"})
	}
	items = append(items, detailItem{label: "Export this identity", key: "export"})
	items = append(items, detailItem{isSep: true})
	items = append(items, detailItem{label: "Bind directory path", key: "bind-path"})
	if len(user.BindPaths) > 0 {
		items = append(items, detailItem{label: "Unbind directory path", key: "unbind-path"})
	}
	items = append(items, detailItem{isSep: true})

	items = append(items, detailItem{label: tuiDanger.Render("Remove identity"), key: "remove", isDanger: true})
	items = append(items, detailItem{isSep: true})
	items = append(items, detailItem{label: tuiDim.Render("← Back"), key: "back"})

	return items
}

func firstDetailSelectable(items []detailItem) int {
	for i, it := range items {
		if !it.isSep {
			return i
		}
	}
	return 0
}

func initialModel(store *config.Store, startDetail string) tuiModel {
	idList := buildIdentitiesList(store)
	actList := buildActionsList()
	m := tuiModel{
		screen:           screenMain,
		store:            store,
		activePane:       paneIdentities,
		identitiesList:   idList,
		actionsList:      actList,
		identitiesCursor: 0,
		actionsCursor:    0,
	}
	if startDetail != "" {
		m.screen = screenDetail
		m.detailName = startDetail
		user := store.FindUser(startDetail)
		if user != nil {
			m.detailItems = buildDetailItems(user, store)
			m.detailCursor = firstDetailSelectable(m.detailItems)
		}
	}
	return m
}

// ── Bubble Tea interface ───────────────────────────────────────────────────────

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quit = true
			return m, tea.Quit

		case "esc":
			if m.screen == screenDetail {
				m.screen = screenMain
				m.identitiesList = buildIdentitiesList(m.store) // reload in case of changes
				return m, nil
			}
			m.quit = true
			return m, tea.Quit

		case "tab":
			if m.screen == screenMain {
				if m.activePane == paneIdentities {
					m.activePane = paneActions
				} else {
					m.activePane = paneIdentities
				}
			}

		case "left", "h":
			if m.screen == screenMain {
				m.activePane = paneIdentities
			}

		case "right", "l":
			if m.screen == screenMain {
				m.activePane = paneActions
			}

		case "up", "k":
			if m.screen == screenMain {
				if m.activePane == paneIdentities {
					if m.identitiesCursor > 0 {
						m.identitiesCursor--
					}
				} else {
					if m.actionsCursor > 0 {
						m.actionsCursor--
					}
				}
			} else {
				m.detailCursor = prevDetailSelectable(m.detailItems, m.detailCursor)
			}

		case "down", "j":
			if m.screen == screenMain {
				if m.activePane == paneIdentities {
					if m.identitiesCursor < len(m.identitiesList)-1 {
						m.identitiesCursor++
					}
				} else {
					if m.actionsCursor < len(m.actionsList)-1 {
						m.actionsCursor++
					}
				}
			} else {
				m.detailCursor = nextDetailSelectable(m.detailItems, m.detailCursor)
			}

		case "enter":
			if m.screen == screenMain {
				return m.handleMainEnter()
			}
			return m.handleDetailEnter()
		}
	}
	return m, nil
}

func (m tuiModel) handleMainEnter() (tea.Model, tea.Cmd) {
	if m.activePane == paneIdentities {
		item := m.identitiesList[m.identitiesCursor]
		if item.isUser {
			user := m.store.FindUser(item.userName)
			if user == nil {
				return m, nil
			}
			m.screen = screenDetail
			m.detailName = item.userName
			m.detailItems = buildDetailItems(user, m.store)
			m.detailCursor = firstDetailSelectable(m.detailItems)
			return m, nil
		}
		if item.isAction {
			m.action = &pendingAction{kind: item.actionKey}
			return m, tea.Quit
		}
	} else {
		item := m.actionsList[m.actionsCursor]
		if item.isAction {
			if item.actionKey == "quit" {
				m.quit = true
				return m, tea.Quit
			}
			m.action = &pendingAction{kind: item.actionKey}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m tuiModel) handleDetailEnter() (tea.Model, tea.Cmd) {
	if m.detailCursor >= len(m.detailItems) {
		return m, nil
	}
	item := m.detailItems[m.detailCursor]
	if item.isSep {
		return m, nil
	}

	switch item.key {
	case "back":
		m.screen = screenMain
		m.identitiesList = buildIdentitiesList(m.store) // reload
		return m, nil
	case "pubkey-locked", "session-na", "passphrase-locked", "pubkey-push-locked":
		return m, nil
	default:
		m.action = &pendingAction{kind: item.key, name: m.detailName}
		return m, tea.Quit
	}
}

// ── Navigation helpers ─────────────────────────────────────────────────────────

func nextDetailSelectable(items []detailItem, cur int) int {
	for i := cur + 1; i < len(items); i++ {
		if !items[i].isSep {
			return i
		}
	}
	return cur
}

func prevDetailSelectable(items []detailItem, cur int) int {
	for i := cur - 1; i >= 0; i-- {
		if !items[i].isSep {
			return i
		}
	}
	return cur
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m tuiModel) View() string {
	if m.screen == screenMain {
		return m.viewMain()
	}
	return m.viewDetail()
}

func (m tuiModel) viewMain() string {
	sb := strings.Builder{}

	// Left Pane (Identities) content
	var leftLines []string
	leftLines = append(leftLines, paneTitleStyle.Render("Git Identities"))
	leftLines = append(leftLines, tuiDim.Render("───────────────────────────────────────────"))
	for i, item := range m.identitiesList {
		prefix := "  "
		if m.activePane == paneIdentities && i == m.identitiesCursor {
			prefix = "▶ "
			lineText := prefix + stripAnsi(item.label)
			leftLines = append(leftLines, tuiSelected.Render(lineText))
		} else if m.activePane != paneIdentities && i == m.identitiesCursor {
			prefix = "▶ "
			lineText := prefix + stripAnsi(item.label)
			leftLines = append(leftLines, tuiDim.Render(lineText))
		} else {
			leftLines = append(leftLines, prefix+item.label)
		}
	}
	// Padding loop removed

	// Right Pane (Utilities) content
	var rightLines []string
	rightLines = append(rightLines, paneTitleStyle.Render("System Utilities"))
	rightLines = append(rightLines, tuiDim.Render("───────────────────────────────────────────"))
	for i, item := range m.actionsList {
		prefix := "  "
		if m.activePane == paneActions && i == m.actionsCursor {
			prefix = "▶ "
			lineText := prefix + stripAnsi(item.label)
			rightLines = append(rightLines, tuiSelected.Render(lineText))
		} else if m.activePane != paneActions && i == m.actionsCursor {
			prefix = "▶ "
			lineText := prefix + stripAnsi(item.label)
			rightLines = append(rightLines, tuiDim.Render(lineText))
		} else {
			rightLines = append(rightLines, prefix+item.label)
		}
	}
	// Padding loop removed

	leftStr := strings.Join(leftLines, "\n")
	rightStr := strings.Join(rightLines, "\n")

	var leftRaw, rightRaw string
	if m.activePane == paneIdentities {
		leftRaw = styleActivePane.Render(leftStr)
		rightRaw = styleInactivePane.Render(rightStr)
	} else {
		leftRaw = styleInactivePane.Render(leftStr)
		rightRaw = styleActivePane.Render(rightStr)
	}

	maxH := lipgloss.Height(leftRaw)
	if lipgloss.Height(rightRaw) > maxH {
		maxH = lipgloss.Height(rightRaw)
	}
	contentH := maxH - 2 // Subtract top and bottom borders
	if contentH < 0 {
		contentH = 0
	}

	var leftBox, rightBox string
	if m.activePane == paneIdentities {
		leftBox = styleActivePane.Height(contentH).Render(leftStr)
		rightBox = styleInactivePane.Height(contentH).Render(rightStr)
	} else {
		leftBox = styleInactivePane.Height(contentH).Render(leftStr)
		rightBox = styleActivePane.Height(contentH).Render(rightStr)
	}

	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "   ", rightBox)

	sb.WriteString("\n" + renderHeader(m.store, m.height) + "\n")
	sb.WriteString(panes + "\n\n")
	sb.WriteString(tuiHelp.Render("  Tab/←/→ switch pane  ↑↓ navigate  Enter select  q quit") + "\n")
	return sb.String()
}

func (m tuiModel) viewDetail() string {
	user := m.store.FindUser(m.detailName)
	if user == nil {
		return "identity not found\n"
	}

	sb := strings.Builder{}

	isActive := user.Name == m.store.Current

	// Left Pane: Info card
	var profileLines []string
	profileLines = append(profileLines, paneTitleStyle.Render("Identity Profile"))
	profileLines = append(profileLines, tuiDim.Render("───────────────────────────────────────────"))
	profileLines = append(profileLines, "")

	// Name
	nameVal := user.Name
	if isActive {
		nameVal = tuiActive.Render("● "+user.Name) + " [active]"
	} else {
		nameVal = "○ " + user.Name
	}
	if user.Source == "original" {
		nameVal += " " + tuiOriginal.Render("(original)")
	}
	profileLines = append(profileLines, fmt.Sprintf("%s\n  %s", tuiDim.Render("Profile Name:"), nameVal))
	profileLines = append(profileLines, "")

	// Email
	profileLines = append(profileLines, fmt.Sprintf("%s\n  %s", tuiDim.Render("Email Address:"), user.Email))
	profileLines = append(profileLines, "")

	// SSH Key
	sshKeyStr := "None"
	if user.SSHKey != "" {
		sshKeyStr = filepath.Base(user.SSHKey)
	}
	profileLines = append(profileLines, fmt.Sprintf("%s\n  %s", tuiDim.Render("SSH Key File:"), sshKeyStr))
	profileLines = append(profileLines, "")

	// Passphrase status
	passphraseStr := tuiDim.Render("Unknown")
	if user.SSHKey != "" {
		if protected, err := isSSHKeyPassphraseProtected(user.SSHKey); err == nil {
			if protected {
				passphraseStr = tuiActive.Render("Passphrase Protected ✓")
			} else {
				passphraseStr = tuiDanger.Render("No Passphrase ⚠")
			}
		}
	}
	profileLines = append(profileLines, fmt.Sprintf("%s\n  %s", tuiDim.Render("Security Status:"), passphraseStr))
	profileLines = append(profileLines, "")

	// Agent status
	sessionStr := tuiDim.Render("not loaded")
	if user.SSHKey != "" && isSSHKeyLoaded(user.SSHKey) {
		sessionStr = tuiActive.Render("Loaded in agent ✓")
	}
	profileLines = append(profileLines, fmt.Sprintf("%s\n  %s", tuiDim.Render("ssh-agent Session:"), sessionStr))
	profileLines = append(profileLines, "")

	profileLines = append(profileLines, tuiDim.Render("Bound Directories:"))
	if len(user.BindPaths) > 0 {
		for _, p := range user.BindPaths {
			displayPath := p
			if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, home) {
				displayPath = "~" + strings.TrimPrefix(p, home)
			}
			if len(displayPath) > 38 {
				displayPath = displayPath[:17] + "..." + displayPath[len(displayPath)-18:]
			}
			profileLines = append(profileLines, "  • "+displayPath)
		}
	} else {
		profileLines = append(profileLines, "  None")
	}

	// Padding loop removed



	// Right Pane: Profile Actions
	var actionLines []string
	actionLines = append(actionLines, paneTitleStyle.Render("Profile Actions"))
	actionLines = append(actionLines, tuiDim.Render("───────────────────────────────────────────"))
	for i, item := range m.detailItems {
		if item.isSep {
			actionLines = append(actionLines, "  "+tuiDim.Render("───────────────────────────────────────────"))
			continue
		}
		prefix := "  "
		if i == m.detailCursor {
			prefix = "▶ "
			raw := stripAnsi(item.label)
			if item.isDanger {
				actionLines = append(actionLines, tuiDanger.Render(prefix+raw))
			} else {
				actionLines = append(actionLines, tuiSelected.Render(prefix+raw))
			}
		} else {
			actionLines = append(actionLines, prefix+item.label)
		}
	}
	// Padding loop removed

	leftStr := strings.Join(profileLines, "\n")
	rightStr := strings.Join(actionLines, "\n")

	var leftRaw string
	if isActive {
		leftRaw = styleDetailActiveCard.Render(leftStr)
	} else {
		leftRaw = styleDetailCard.Render(leftStr)
	}
	rightRaw := styleDetailActions.Render(rightStr)

	maxH := lipgloss.Height(leftRaw)
	if lipgloss.Height(rightRaw) > maxH {
		maxH = lipgloss.Height(rightRaw)
	}
	contentH := maxH - 2 // Subtract top and bottom borders
	if contentH < 0 {
		contentH = 0
	}

	var leftBox string
	if isActive {
		leftBox = styleDetailActiveCard.Height(contentH).Render(leftStr)
	} else {
		leftBox = styleDetailCard.Height(contentH).Render(leftStr)
	}

	rightBox := styleDetailActions.Height(contentH).Render(rightStr)

	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "   ", rightBox)

	sb.WriteString("\n" + renderHeader(m.store, m.height) + "\n")
	sb.WriteString(panes + "\n\n")
	sb.WriteString(tuiHelp.Render("  ↑↓ navigate  Enter select  Esc back  q quit") + "\n")
	return sb.String()
}

// stripAnsi removes ANSI escape sequences for plain text rendering in selected state.
func stripAnsi(s string) string {
	var result strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// ── Entry points ──────────────────────────────────────────────────────────────

func runTui() error {
	return launchTUI("")
}

func runTuiForIdentity(name string) error {
	return launchTUI(name)
}

func launchTUI(startDetail string) error {
	for {
		store, err := config.Load()
		if err != nil {
			return err
		}

		m := initialModel(store, startDetail)
		p := tea.NewProgram(m, tea.WithAltScreen())
		finalRaw, err := p.Run()
		if err != nil {
			return err
		}

		final := finalRaw.(tuiModel)

		if final.quit || final.action == nil {
			return nil
		}

		act := final.action

		// Execute action outside TUI (needs terminal I/O)
		fmt.Println()
		executeAction(act, store)
		fmt.Println()

		if act.kind == "quit" {
			return nil
		}

		// After remove, go back to main
		if act.kind == "remove" {
			startDetail = ""
		} else if act.name != "" {
			startDetail = act.name
		} else {
			startDetail = ""
		}

		// Prompt to return
		fmt.Print(tuiDim.Render("  Press Enter to return to menu..."))
		fmt.Scanln()
	}
}

func executeAction(act *pendingAction, store *config.Store) {
	switch act.kind {
	case "register":
		runRegister(nil)

	case "switch":
		runSwitch([]string{act.name})

	case "rename":
		newName, err := ui.Prompt(fmt.Sprintf("New name for %q:", act.name))
		if err != nil || newName == "" {
			ui.Info("Cancelled")
			return
		}
		if store.FindUser(newName) != nil {
			ui.Errorf("Identity %q already exists", newName)
			return
		}
		u := store.FindUser(act.name)
		if u == nil {
			return
		}
		u.Name = newName
		if store.Current == act.name {
			store.Current = newName
		}
		config.Save(store)
		ui.Success(fmt.Sprintf("Renamed %q → %q", act.name, newName))

	case "email":
		newEmail, err := ui.Prompt(fmt.Sprintf("New email for %q:", act.name))
		if err != nil || newEmail == "" {
			ui.Info("Cancelled")
			return
		}
		runEdit([]string{act.name, newEmail})

	case "pubkey":
		runPubkey(nil)

	case "pubkey-push":
		runPubkeyPush(nil)

	case "bind":
		runBind([]string{act.name})

	case "unbind":
		u := store.FindUser(act.name)
		if u == nil {
			return
		}
		if !ui.Confirm(fmt.Sprintf("Remove SSH key binding from %q? (file not deleted)", act.name), false) {
			ui.Info("Cancelled")
			return
		}
		u.SSHKey = ""
		config.Save(store)
		if store.Current == act.name {
			git.RemoveSSHConfig()
		}
		ui.Success("SSH key removed from identity")

	case "rekey":
		runRekey([]string{act.name})

	case "passphrase":
		runPassphrase([]string{act.name})

	case "passphrase-remove":
		runPassphrase([]string{act.name, "--remove"})

	case "bind-path":
		path, err := ui.Prompt("Directory path to bind:")
		if err != nil || path == "" {
			ui.Info("Cancelled")
			return
		}
		runBindPath([]string{act.name, path})

	case "unbind-path":
		u := store.FindUser(act.name)
		if u == nil {
			return
		}
		if len(u.BindPaths) == 0 {
			ui.Info("No paths bound to this identity")
			return
		}
		var path string
		if len(u.BindPaths) == 1 {
			path = u.BindPaths[0]
			if !ui.Confirm(fmt.Sprintf("Unbind directory %q?", path), false) {
				ui.Info("Cancelled")
				return
			}
		} else {
			idx, err := ui.Select("Select directory to unbind:", u.BindPaths)
			if err != nil {
				ui.Info("Cancelled")
				return
			}
			path = u.BindPaths[idx]
		}
		runUnbindPath([]string{act.name, path})

	case "logout":
		runLogout(nil)

	case "export":
		runExport([]string{act.name})

	case "export-all":
		runExport([]string{"--all"})

	case "import":
		path, err := ui.Prompt("Path to bundle file:")
		if err != nil || path == "" {
			ui.Info("Cancelled")
			return
		}
		runImport([]string{path})

	case "import-original":
		runImportOriginal(nil)

	case "remove":
		if !ui.Confirm(fmt.Sprintf("Remove identity %q? This cannot be undone.", act.name), false) {
			ui.Info("Cancelled")
			return
		}
		runRemove([]string{act.name})

	case "fix-remote":
		runFixRemote(nil)

	case "security":
		runSecurityCheck(nil)

	case "doctor":
		runDoctor(nil)

	case "update":
		if err := RunUpdate(); err != nil {
			ui.Errorf("Update failed: %v", err)
		}
	}
}

// handleUnknownArg checks if arg is an identity name and opens detail view,
// otherwise returns false so root.go can show "unknown command".
func handleUnknownArg(name string) bool {
	store, err := config.Load()
	if err != nil {
		return false
	}
	// Check exact match first
	if store.FindUser(name) != nil {
		launchTUI(name)
		return true
	}
	// Suggest similar names
	var similar []string
	lower := strings.ToLower(name)
	for _, u := range store.Users {
		if strings.Contains(strings.ToLower(u.Name), lower) {
			similar = append(similar, u.Name)
		}
	}
	if len(similar) > 0 {
		ui.Errorf("identity %q not found — did you mean: %s", name, strings.Join(similar, ", "))
		return true
	}
	return false
}
