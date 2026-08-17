package screens

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

type Pane int

const (
	PaneIdentities Pane = iota
	PaneActions
)

type Dashboard struct {
	store       *config.Store
	identities  components.IdentityList
	actions     components.ActionMenu
	activePane  Pane
	animFrame   uint64
	theme       theme.Theme
	filterMode  bool   // true when '/' has been pressed and user is typing
	filterQuery string // current filter text
	leftWidth   int    // updated each View(); used for mouse pane detection
	syncOut     bool   // git config does not match the active identity
}

func NewDashboard(store *config.Store, th theme.Theme) *Dashboard {
	return &Dashboard{
		store:      store,
		identities: components.NewIdentityList(store, th),
		actions:    components.SystemActions(th, git.HasHTTPSRemotes()),
		activePane: PaneIdentities,
		theme:      th,
	}
}

func (d *Dashboard) Init() tea.Cmd { return nil }
func (d *Dashboard) Title() string { return "Dashboard" }
func (d *Dashboard) ShortHelp() string {
	if d.filterMode {
		return core.FilterHelp()
	}
	if d.syncOut {
		return core.DashboardHelp() + "  f•re-apply active identity"
	}
	return core.DashboardHelp()
}

func (d *Dashboard) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case core.AnimTickMsg:
		d.animFrame++
		d.identities.TickIntro()
		return d, nil
	case core.StoreRefreshedMsg:
		if msg.Err == nil && msg.Store != nil {
			d.store = msg.Store
			d.identities.Refresh(msg.Store)
		}
	case core.SyncStatusMsg:
		d.syncOut = msg.Err == nil && !msg.InSync
	case core.VersionCheckMsg:
		d.actions.SetUpdateStatus(msg.LatestVersion, msg.UpdateAvailable)
		return d, nil
	case tea.MouseMsg:
		return d.handleMouse(msg)
	case tea.KeyMsg:
		return d.handleKey(msg)
	}
	return d, nil
}

func (d *Dashboard) handleMouse(msg tea.MouseMsg) (core.Screen, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionPress {
			// Use tracked leftWidth from last View() call for accurate detection.
			splitAt := d.leftWidth + theme.PaneBorder
			if splitAt < 20 {
				splitAt = 40 // fallback before first render
			}
			if msg.X < splitAt {
				d.activePane = PaneIdentities
			} else {
				d.activePane = PaneActions
			}
		}
	case tea.MouseButtonWheelUp:
		if d.activePane == PaneIdentities {
			d.identities.CursorUp()
		} else {
			d.actions.CursorUp()
		}
	case tea.MouseButtonWheelDown:
		if d.activePane == PaneIdentities {
			d.identities.CursorDown()
		} else {
			d.actions.CursorDown()
		}
	}
	return d, nil
}

func (d *Dashboard) handleKey(msg tea.KeyMsg) (core.Screen, tea.Cmd) {
	// ── Filter mode: characters type into the query ───────────────────────
	if d.filterMode {
		switch msg.String() {
		case core.KeyCtrlC:
			return d, tea.Quit
		case core.KeyEsc:
			// Esc: clear filter and exit filter mode.
			d.filterMode = false
			d.filterQuery = ""
			d.identities.ClearFilter()
			return d, nil
		case core.KeyEnter:
			// Enter: accept current selection and exit filter mode.
			d.filterMode = false
			return d.handleEnter()
		case "backspace", "ctrl+h":
			if len(d.filterQuery) > 0 {
				d.filterQuery = d.filterQuery[:len([]rune(d.filterQuery))-1]
				d.identities.FilterByQuery(d.filterQuery)
			}
			if d.filterQuery == "" {
				d.identities.ClearFilter()
			}
			return d, nil
		case core.KeyUp, core.KeyK:
			d.identities.CursorUp()
			return d, nil
		case core.KeyDown, core.KeyJ:
			d.identities.CursorDown()
			return d, nil
		default:
			// Printable character: append to query.
			if len(msg.Runes) > 0 {
				d.filterQuery += string(msg.Runes)
				d.identities.FilterByQuery(d.filterQuery)
			}
			return d, nil
		}
	}

	// ── Normal mode ───────────────────────────────────────────────────────
	switch msg.String() {
	case core.KeyCtrlC:
		return d, tea.Quit
	case core.KeyQuit:
		// 'q' shows a quit confirmation instead of quitting immediately.
		return d, func() tea.Msg {
			return core.ActionResultMsg{Kind: "quit-confirm"}
		}
	case core.KeyEsc:
		// Root screen: nothing to go back to.
		return d, core.ShowToastCmd("At the main menu — press q to quit", theme.ToastStyleInfo, 2*time.Second)
	case core.KeyFilter:
		// '/' enters filter mode on the identities pane.
		d.filterMode = true
		d.activePane = PaneIdentities
		return d, nil
	case core.KeyTab:
		if d.activePane == PaneIdentities {
			d.activePane = PaneActions
		} else {
			d.activePane = PaneIdentities
		}
	case "s", "S":
		if d.activePane == PaneIdentities {
			item := d.identities.Selected()
			if item != nil && !item.IsAction && !item.IsActive {
				return d, func() tea.Msg {
					return core.ActionResultMsg{Kind: "switch", Name: item.Name}
				}
			}
		}
	case "f", "F":
		// Re-apply the active identity when the git config drifted out of sync.
		if d.syncOut {
			return d, func() tea.Msg {
				return core.ActionResultMsg{Kind: "fix-sync"}
			}
		}
	case core.KeyLeft, core.KeyH:
		d.activePane = PaneIdentities
	case core.KeyRight, core.KeyL:
		d.activePane = PaneActions
	case core.KeyUp, core.KeyK:
		if d.activePane == PaneIdentities {
			d.identities.CursorUp()
		} else {
			d.actions.CursorUp()
		}
	case core.KeyDown, core.KeyJ:
		if d.activePane == PaneIdentities {
			d.identities.CursorDown()
		} else {
			d.actions.CursorDown()
		}
	case core.KeyEnter:
		return d.handleEnter()
	}
	return d, nil
}

func (d *Dashboard) handleEnter() (core.Screen, tea.Cmd) {
	if d.activePane == PaneIdentities {
		item := d.identities.Selected()
		if item == nil {
			return d, nil
		}
		if item.IsAction {
			return d, func() tea.Msg {
				return core.ActionResultMsg{Kind: item.ActionKey}
			}
		}
		user := d.store.FindUser(item.Name)
		if user != nil {
			return d, func() tea.Msg {
				return core.ScreenPushMsg{
					Screen: NewDetail(d.store, user.Name, d.theme),
				}
			}
		}
	} else {
		item := d.actions.Selected()
		if item == nil {
			return d, nil
		}
		return d, func() tea.Msg {
			return core.ActionResultMsg{Kind: item.Key}
		}
	}
	return d, nil
}

func (d *Dashboard) View(width, height int) string {
	contentH := height - 4

	var view string
	if theme.IsSingleColumn(width) {
		paneWidth := theme.PaneWidth(width)
		view = d.viewSingleColumn(paneWidth, contentH)
	} else {
		// Right pane: sized to fit its own content, not half the terminal.
		rightWidth := d.actions.PreferredWidth(28, 48)
		// Left pane: all remaining space minus the gap and the border columns each
		// pane style adds on top of its requested Width (PaneBorder per pane).
		leftWidth := width - rightWidth - theme.PaneGap - 2*theme.PaneBorder
		if leftWidth < 20 {
			leftWidth = 20
		}

		// Track leftWidth for mouse click detection.
		d.leftWidth = leftWidth

		leftContent := d.identities.ViewWithFilter(leftWidth, contentH, d.activePane == PaneIdentities, d.filterMode, d.filterQuery)
		rightContent := d.actions.View(rightWidth, contentH, d.activePane == PaneActions)

		var leftBox, rightBox string
		if d.activePane == PaneIdentities {
			leftBox = d.theme.PulsingActivePane(leftWidth, contentH, d.animFrame).Render(leftContent)
			rightBox = d.theme.InactivePane(rightWidth, contentH).Render(rightContent)
		} else {
			leftBox = d.theme.InactivePane(leftWidth, contentH).Render(leftContent)
			rightBox = d.theme.PulsingActivePane(rightWidth, contentH, d.animFrame).Render(rightContent)
		}

		view = lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "   ", rightBox)
	}

	if d.syncOut {
		view = syncWarningBanner(width, d.theme) + "\n" + view
	}
	return view
}

// syncWarningBanner renders the out-of-sync warning at full terminal width,
// truncated to fit so the terminal never wraps it mid-phrase.
func syncWarningBanner(width int, th theme.Theme) string {
	msg := "⚠ Git config is out of sync with the active identity — press f to re-apply"
	max := width - 4
	if max < 8 {
		max = 8
	}
	runes := []rune(msg)
	if len(runes) > max {
		runes = append(runes[:max-1], '…')
	}
	return th.WarningStyle().Bold(true).Render(string(runes))
}

func (d *Dashboard) viewSingleColumn(width, height int) string {
	if d.activePane == PaneIdentities {
		content := d.identities.ViewWithFilter(width, height, true, d.filterMode, d.filterQuery)
		return d.theme.ActivePane(width, height).Render(content)
	}
	content := d.actions.View(width, height, true)
	return d.theme.ActivePane(width, height).Render(content)
}

func (d *Dashboard) Refresh(store *config.Store) {
	d.store = store
	d.identities.Refresh(store)
}

func (d *Dashboard) SetStore(store *config.Store) {
	d.store = store
	d.identities.Refresh(store)
}

func (d *Dashboard) SetVersionStatus(latestVersion string, updateAvailable bool) {
	d.actions.SetUpdateStatus(latestVersion, updateAvailable)
}

