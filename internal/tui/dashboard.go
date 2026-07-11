package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

type Pane int

const (
	PaneIdentities Pane = iota
	PaneActions
)

type Dashboard struct {
	store        *config.Store
	identities   components.IdentityList
	actions      components.ActionMenu
	activePane   Pane
	filterInput  string
	filtering    bool
	theme        theme.Theme
}

func NewDashboard(store *config.Store, th theme.Theme) *Dashboard {
	return &Dashboard{
		store:      store,
		identities: components.NewIdentityList(store, th),
		actions:    components.SystemActions(th),
		activePane: PaneIdentities,
		theme:      th,
	}
}

func (d *Dashboard) Init() tea.Cmd { return nil }
func (d *Dashboard) Title() string { return "Dashboard" }
func (d *Dashboard) ShortHelp() string {
	if d.filtering {
		return FilterHelp()
	}
	return DashboardHelp()
}

func (d *Dashboard) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case StoreRefreshedMsg:
		if msg.Err == nil && msg.Store != nil {
			d.store = msg.Store
			d.identities.Refresh(msg.Store)
		}
	case tea.KeyMsg:
		if d.filtering {
			return d.handleFilterKey(msg)
		}
		return d.handleKey(msg)
	}
	return d, nil
}

func (d *Dashboard) handleKey(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case KeyCtrlC, KeyQuit:
		return d, tea.Quit
	case KeyEsc:
		return d, tea.Quit
	case KeyTab:
		if d.activePane == PaneIdentities {
			d.activePane = PaneActions
		} else {
			d.activePane = PaneIdentities
		}
	case KeyLeft, KeyH:
		d.activePane = PaneIdentities
	case KeyRight, KeyL:
		d.activePane = PaneActions
	case KeyUp, KeyK:
		if d.activePane == PaneIdentities {
			d.identities.CursorUp()
		} else {
			d.actions.CursorUp()
		}
	case KeyDown, KeyJ:
		if d.activePane == PaneIdentities {
			d.identities.CursorDown()
		} else {
			d.actions.CursorDown()
		}
	case KeyFilter:
		d.filtering = true
		d.filterInput = ""
		d.identities.SetFilter("")
		d.activePane = PaneIdentities
	case KeyEnter:
		return d.handleEnter()
	}
	return d, nil
}

func (d *Dashboard) handleFilterKey(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case KeyEsc:
		d.filtering = false
		d.filterInput = ""
		d.identities.ClearFilter()
		return d, nil
	case KeyEnter:
		d.filtering = false
		return d.handleEnter()
	case KeyUp, KeyK:
		d.identities.CursorUp()
	case KeyDown, KeyJ:
		d.identities.CursorDown()
	case "backspace":
		if len(d.filterInput) > 0 {
			d.filterInput = d.filterInput[:len(d.filterInput)-1]
			d.identities.SetFilter(d.filterInput)
		} else {
			d.filtering = false
			d.identities.ClearFilter()
		}
	case KeyCtrlC:
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

func (d *Dashboard) handleEnter() (Screen, tea.Cmd) {
	if d.activePane == PaneIdentities {
		item := d.identities.Selected()
		if item == nil {
			return d, nil
		}
		if item.IsAction {
			return d, func() tea.Msg {
				return ActionResultMsg{Kind: item.ActionKey}
			}
		}
		user := d.store.FindUser(item.Name)
		if user != nil {
			return d, func() tea.Msg {
				return ScreenPushMsg{
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
			return ActionResultMsg{Kind: item.Key}
		}
	}
	return d, nil
}

func (d *Dashboard) View(width, height int) string {
	paneWidth := theme.PaneWidth(width)
	contentH := height - 4

	if theme.IsSingleColumn(width) {
		return d.viewSingleColumn(paneWidth, contentH)
	}

	leftContent := d.identities.View(paneWidth, contentH, d.activePane == PaneIdentities)
	rightContent := d.actions.View(paneWidth, contentH, d.activePane == PaneActions)

	var leftBox, rightBox string
	if d.activePane == PaneIdentities {
		leftBox = d.theme.ActivePane(paneWidth, contentH).Render(leftContent)
		rightBox = d.theme.InactivePane(paneWidth, contentH).Render(rightContent)
	} else {
		leftBox = d.theme.InactivePane(paneWidth, contentH).Render(leftContent)
		rightBox = d.theme.ActivePane(paneWidth, contentH).Render(rightContent)
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
