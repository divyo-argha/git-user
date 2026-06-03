package cmd

import (
	"fmt"
	"os"
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

	tuiCardActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tuiGreen).
			Padding(0, 2).Width(58)

	tuiCardNormal = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(tuiGray).
			Padding(0, 2).Width(58)

	tuiSelected   = lipgloss.NewStyle().Foreground(tuiCyan).Bold(true)
	tuiDim        = lipgloss.NewStyle().Foreground(tuiGray)
	tuiActive     = lipgloss.NewStyle().Foreground(tuiGreen).Bold(true)
	tuiOriginal   = lipgloss.NewStyle().Foreground(tuiOrange)
	tuiDanger     = lipgloss.NewStyle().Foreground(tuiRed)
	tuiHelp       = lipgloss.NewStyle().Foreground(tuiGray).Italic(true)
	tuiBold       = lipgloss.NewStyle().Foreground(tuiWhite).Bold(true)
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

type tuiModel struct {
	screen     tuiScreen
	store      *config.Store
	// main screen
	mainItems  []mainItem
	mainCursor int
	// detail screen
	detailName    string
	detailItems   []detailItem
	detailCursor  int
	// result
	quit    bool
	action  *pendingAction
}

type detailItem struct {
	label    string
	isSep    bool
	key      string
	isDanger bool
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func buildMainItems(store *config.Store) []mainItem {
	items := []mainItem{}
	for _, u := range store.Users {
		label := u.Name
		if u.Name == store.Current {
			label = tuiActive.Render("● "+u.Name) + "  " + tuiDim.Render(u.Email) + "  " + tuiActive.Render("[active]")
		} else {
			label = "○ " + u.Name + "  " + tuiDim.Render(u.Email)
		}
		if u.Source == "original" {
			label += "  " + tuiOriginal.Render("[original]")
		}
		items = append(items, mainItem{label: label, isUser: true, userName: u.Name})
	}
	items = append(items, mainItem{label: lipgloss.NewStyle().Foreground(tuiCyan).Render("+ Create new identity"), isAction: true, actionKey: "register"})
	items = append(items, mainItem{isSep: true})
	items = append(items, mainItem{label: "Session status", isAction: true, actionKey: "session-status"})
	items = append(items, mainItem{label: "Fix remotes (HTTPS → SSH)", isAction: true, actionKey: "fix-remote"})
	items = append(items, mainItem{label: "Security audit", isAction: true, actionKey: "security"})
	items = append(items, mainItem{label: "Doctor (health check)", isAction: true, actionKey: "doctor"})
	items = append(items, mainItem{isSep: true})
	items = append(items, mainItem{label: "Export all identities", isAction: true, actionKey: "export-all"})
	items = append(items, mainItem{label: "Import identities", isAction: true, actionKey: "import"})
	items = append(items, mainItem{label: "Import original gitconfig identity", isAction: true, actionKey: "import-original"})
	items = append(items, mainItem{isSep: true})
	items = append(items, mainItem{label: "Update git-user", isAction: true, actionKey: "update"})
	items = append(items, mainItem{label: tuiDim.Render("Quit"), isAction: true, actionKey: "quit"})
	return items
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
	} else {
		items = append(items, detailItem{label: tuiDim.Render("Show public key (switch first)"), key: "pubkey-locked"})
	}
	items = append(items, detailItem{label: "Add / replace SSH key", key: "bind"})
	items = append(items, detailItem{label: "Rotate SSH key", key: "rekey"})
	if user.SSHKey != "" {
		items = append(items, detailItem{label: "Remove SSH key", key: "unbind"})
	}
	items = append(items, detailItem{isSep: true})

	if user.SSHKey != "" {
		items = append(items, detailItem{label: "Start session", key: "session-start"})
		items = append(items, detailItem{label: "Start session with TTL", key: "session-start-ttl"})
		items = append(items, detailItem{label: "Stop session", key: "session-stop"})
	} else {
		items = append(items, detailItem{label: tuiDim.Render("Start session (no SSH key)"), key: "session-na"})
	}
	items = append(items, detailItem{isSep: true})

	items = append(items, detailItem{label: "Change passphrase", key: "passphrase"})
	items = append(items, detailItem{label: "Export this identity", key: "export"})
	items = append(items, detailItem{isSep: true})

	items = append(items, detailItem{label: tuiDanger.Render("Remove identity"), key: "remove", isDanger: true})
	items = append(items, detailItem{isSep: true})
	items = append(items, detailItem{label: tuiDim.Render("← Back"), key: "back"})

	return items
}

func firstSelectable(items []mainItem) int {
	for i, it := range items {
		if !it.isSep {
			return i
		}
	}
	return 0
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
	items := buildMainItems(store)
	m := tuiModel{
		screen:     screenMain,
		store:      store,
		mainItems:  items,
		mainCursor: firstSelectable(items),
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
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quit = true
			return m, tea.Quit

		case "esc":
			if m.screen == screenDetail {
				m.screen = screenMain
				return m, nil
			}
			m.quit = true
			return m, tea.Quit

		case "up", "k":
			if m.screen == screenMain {
				m.mainCursor = prevSelectable(m.mainItems, m.mainCursor)
			} else {
				m.detailCursor = prevDetailSelectable(m.detailItems, m.detailCursor)
			}

		case "down", "j":
			if m.screen == screenMain {
				m.mainCursor = nextSelectable(m.mainItems, m.mainCursor)
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
	item := m.mainItems[m.mainCursor]

	if item.isUser {
		// Open detail screen for this identity
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
		switch item.actionKey {
		case "quit":
			m.quit = true
			return m, tea.Quit
		default:
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
		return m, nil
	case "pubkey-locked", "session-na":
		// non-actionable, just stay
		return m, nil
	default:
		m.action = &pendingAction{kind: item.key, name: m.detailName}
		return m, tea.Quit
	}
}

// ── Navigation helpers ─────────────────────────────────────────────────────────

func nextSelectable(items []mainItem, cur int) int {
	for i := cur + 1; i < len(items); i++ {
		if !items[i].isSep {
			return i
		}
	}
	return cur
}

func prevSelectable(items []mainItem, cur int) int {
	for i := cur - 1; i >= 0; i-- {
		if !items[i].isSep {
			return i
		}
	}
	return cur
}

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

	sb.WriteString("\n" + renderHeader(m.store) + "\n\n")

	for i, item := range m.mainItems {
		if item.isSep {
			sb.WriteString("  " + tuiDim.Render("──────────────────────────────────────────────") + "\n")
			continue
		}
		if i == m.mainCursor {
			sb.WriteString("  " + tuiSelected.Render("▶ "+stripAnsi(item.label)) + "\n")
		} else {
			sb.WriteString("    " + item.label + "\n")
		}
	}

	sb.WriteString("\n" + tuiHelp.Render("  ↑↓ navigate  Enter select  q quit") + "\n")
	return sb.String()
}

func (m tuiModel) viewDetail() string {
	user := m.store.FindUser(m.detailName)
	if user == nil {
		return "identity not found\n"
	}

	sb := strings.Builder{}

	// Info card
	isActive := user.Name == m.store.Current
	cardStyle := tuiCardNormal
	if isActive {
		cardStyle = tuiCardActive
	}

	nameStr := tuiBold.Render(user.Name)
	if isActive {
		nameStr = tuiActive.Render("● " + user.Name)
	}
	if user.Source == "original" {
		nameStr += "  " + tuiOriginal.Render("[original]")
	}

	sshLine := tuiDim.Render("(no SSH key)")
	if user.SSHKey != "" {
		sshLine = tuiDim.Render(user.SSHKey)
	}

	sessionLine := tuiDim.Render("not loaded")
	if user.SSHKey != "" && isSSHKeyLoaded(user.SSHKey) {
		sessionLine = tuiActive.Render("key loaded ✓")
	}

	syncLine := ""
	if isActive {
		gitEmail := git.CurrentEmail()
		if gitEmail == user.Email {
			syncLine = "\n  " + tuiDim.Render("Sync     ") + tuiActive.Render("in sync ✓")
		} else {
			syncLine = "\n  " + tuiDim.Render("Sync     ") + tuiDanger.Render("out of sync ⚠")
		}
	}

	cardContent := fmt.Sprintf("  %s\n\n  %s  %s\n  %s  %s\n  %s  %s%s",
		nameStr,
		tuiDim.Render("Email   "), user.Email,
		tuiDim.Render("SSH Key "), sshLine,
		tuiDim.Render("Session "), sessionLine,
		syncLine,
	)
	sb.WriteString("\n" + cardStyle.Render(cardContent) + "\n\n")

	// Actions
	for i, item := range m.detailItems {
		if item.isSep {
			sb.WriteString("  " + tuiDim.Render("──────────────────────────────────────────────") + "\n")
			continue
		}
		if i == m.detailCursor {
			raw := stripAnsi(item.label)
			if item.isDanger {
				sb.WriteString("  " + tuiDanger.Render("▶ "+raw) + "\n")
			} else {
				sb.WriteString("  " + tuiSelected.Render("▶ "+raw) + "\n")
			}
		} else {
			sb.WriteString("    " + item.label + "\n")
		}
	}

	sb.WriteString("\n" + tuiHelp.Render("  ↑↓ navigate  Enter select  Esc back  q quit") + "\n")
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
		runPassphrase(nil)

	case "session-start":
		runSession([]string{"start", act.name})

	case "session-start-ttl":
		ttl, err := ui.Prompt("TTL duration (e.g. 4h, 30m):")
		if err != nil || ttl == "" {
			ui.Info("Cancelled")
			return
		}
		runSession([]string{"start", act.name, "--ttl", ttl})

	case "session-stop":
		runSession([]string{"stop", act.name})

	case "session-status":
		runSession([]string{"status"})

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

// ensure os is used
var _ = os.Stderr
