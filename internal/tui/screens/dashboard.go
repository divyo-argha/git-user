package screens

import (
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
	filterInput string
	filtering   bool
	animFrame   uint64
	theme       theme.Theme
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
	if d.filtering {
		return core.FilterHelp()
	}
	return core.DashboardHelp()
}

func (d *Dashboard) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case core.StoreRefreshedMsg:
		if msg.Err == nil && msg.Store != nil {
			d.store = msg.Store
			d.identities.Refresh(msg.Store)
		}
	case tea.MouseMsg:
		return d.handleMouse(msg)
	case tea.KeyMsg:
		if d.filtering {
			return d.handleFilterKey(msg)
		}
		return d.handleKey(msg)
	}
	return d, nil
}

func (d *Dashboard) handleMouse(msg tea.MouseMsg) (core.Screen, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionPress {
			if msg.X < 40 {
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
	switch msg.String() {
	case core.KeyCtrlC, core.KeyQuit:
		return d, tea.Quit
	case core.KeyEsc:
		return d, tea.Quit
	case core.KeyTab:
		if d.activePane == PaneIdentities {
			d.activePane = PaneActions
		} else {
			d.activePane = PaneIdentities
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
	case core.KeyFilter:
		d.filtering = true
		d.filterInput = ""
		d.identities.SetFilter("")
		d.activePane = PaneIdentities
	case core.KeyEnter:
		return d.handleEnter()
	}
	return d, nil
}

func (d *Dashboard) handleFilterKey(msg tea.KeyMsg) (core.Screen, tea.Cmd) {
	switch msg.String() {
	case core.KeyEsc:
		d.filtering = false
		d.filterInput = ""
		d.identities.ClearFilter()
		return d, nil
	case core.KeyEnter:
		d.filtering = false
		return d.handleEnter()
	case core.KeyUp, core.KeyK:
		d.identities.CursorUp()
	case core.KeyDown, core.KeyJ:
		d.identities.CursorDown()
	case "backspace":
		if len(d.filterInput) > 0 {
			d.filterInput = d.filterInput[:len(d.filterInput)-1]
			d.identities.SetFilter(d.filterInput)
		} else {
			d.filtering = false
			d.identities.ClearFilter()
		}
	case core.KeyCtrlC:
		return d, tea.Quit
	default:
		key := msg.String()
		if len(key) == 1 {
			d.filterInput += key
			d.identities.SetFilter(d.filterInput)
		}
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
		if item.Key == "quit" {
			return d, tea.Quit
		}
		return d, func() tea.Msg {
			return core.ActionResultMsg{Kind: item.Key}
		}
	}
	return d, nil
}

func (d *Dashboard) View(width, height int) string {
	contentH := height - 4

	if theme.IsSingleColumn(width) {
		paneWidth := theme.PaneWidth(width)
		return d.viewSingleColumn(paneWidth, contentH)
	}

	// Right pane: sized to fit its own content, not half the terminal.
	rightWidth := d.actions.PreferredWidth(28, 48)
	// Left pane: all remaining space minus the gap.
	leftWidth := width - rightWidth - theme.PaneGap
	if leftWidth < 20 {
		leftWidth = 20
	}

	leftContent := d.identities.View(leftWidth, contentH, d.activePane == PaneIdentities)
	rightContent := d.actions.View(rightWidth, contentH, d.activePane == PaneActions)

	var leftBox, rightBox string
	if d.activePane == PaneIdentities {
		leftBox = d.theme.PulsingActivePane(leftWidth, contentH, d.animFrame).Render(leftContent)
		rightBox = d.theme.InactivePane(rightWidth, contentH).Render(rightContent)
	} else {
		leftBox = d.theme.InactivePane(leftWidth, contentH).Render(leftContent)
		rightBox = d.theme.PulsingActivePane(rightWidth, contentH, d.animFrame).Render(rightContent)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "   ", rightBox)
}

func (d *Dashboard) viewSingleColumn(width, height int) string {
	if d.activePane == PaneIdentities {
		content := d.identities.View(width, height, true)
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
