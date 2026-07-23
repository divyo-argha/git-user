package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

type CommandItem struct {
	Title       string
	Description string
	ActionKind  string
}

type CommandPalette struct {
	query    string
	items    []CommandItem
	filtered []CommandItem
	cursor   int
	theme    theme.Theme
}

func NewCommandPalette(th theme.Theme) *CommandPalette {
	items := []CommandItem{
		{Title: "➕ Register New Identity", Description: "Add a new Git profile name & email", ActionKind: "register"},
		{Title: "⚡ Switch Active Identity", Description: "Switch current git user config", ActionKind: "switch"},
		{Title: "🔑 Manage SSH Keys", Description: "Load or configure SSH identity keys", ActionKind: "ssh"},
		{Title: "📁 Bind Repository Path", Description: "Set folder auto-switch rule", ActionKind: "bind"},
		{Title: "📖 Open Help & Documentation", Description: "View keybindings and navigation guide", ActionKind: "help"},
		{Title: "🚪 Quit TUI Application", Description: "Exit back to terminal shell", ActionKind: "quit"},
	}
	cp := &CommandPalette{
		items: items,
		theme: th,
	}
	cp.filterItems()
	return cp
}

func (cp *CommandPalette) Init() tea.Cmd { return nil }

func (cp *CommandPalette) Title() string { return "Command Palette" }

func (cp *CommandPalette) ShortHelp() string {
	return "type • filter commands  ↑/↓ • navigate  enter • execute  esc • cancel"
}

func (cp *CommandPalette) filterItems() {
	if cp.query == "" {
		cp.filtered = make([]CommandItem, len(cp.items))
		copy(cp.filtered, cp.items)
		return
	}

	q := strings.ToLower(cp.query)
	cp.filtered = cp.filtered[:0]
	for _, item := range cp.items {
		if strings.Contains(strings.ToLower(item.Title), q) || strings.Contains(strings.ToLower(item.Description), q) {
			cp.filtered = append(cp.filtered, item)
		}
	}
	if cp.cursor >= len(cp.filtered) {
		cp.cursor = max(0, len(cp.filtered)-1)
	}
}

func (cp *CommandPalette) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case core.KeyEsc, core.KeyCtrlC:
			return cp, func() tea.Msg { return core.ScreenPopMsg{} }
		case core.KeyUp, core.KeyK:
			if cp.cursor > 0 {
				cp.cursor--
			}
		case core.KeyDown, core.KeyJ:
			if cp.cursor < len(cp.filtered)-1 {
				cp.cursor++
			}
		case "backspace":
			if len(cp.query) > 0 {
				cp.query = cp.query[:len(cp.query)-1]
				cp.filterItems()
			}
		case core.KeyEnter:
			if len(cp.filtered) > 0 && cp.cursor < len(cp.filtered) {
				selected := cp.filtered[cp.cursor]
				if selected.ActionKind == "help" {
					return cp, func() tea.Msg {
						return core.ScreenPushMsg{Screen: NewHelpModal(cp.theme)}
					}
				}
				if selected.ActionKind == "quit" {
					return cp, tea.Quit
				}
				return cp, func() tea.Msg {
					return core.ActionResultMsg{Kind: selected.ActionKind}
				}
			}
		default:
			if len(msg.String()) == 1 {
				cp.query += msg.String()
				cp.filterItems()
			}
		}
	}
	return cp, nil
}

func (cp *CommandPalette) View(width, height int) string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(cp.theme.Primary).
		Bold(true).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("🔍 Command Palette"))
	b.WriteString("\n\n")

	prompt := cp.theme.Active().Render("❯ ") + cp.query + cp.theme.Dim().Render("│")
	b.WriteString(prompt)
	b.WriteByte('\n')
	b.WriteString(cp.theme.SeparatorLine(width - 6))
	b.WriteString("\n\n")

	if len(cp.filtered) == 0 {
		b.WriteString(cp.theme.Dim().Render("  No matching commands found."))
	} else {
		for i, item := range cp.filtered {
			cursorStr := "  "
			if i == cp.cursor {
				cursorStr = cp.theme.Selected().Render("❯ ")
			}
			title := item.Title
			if i == cp.cursor {
				title = cp.theme.Selected().Render(item.Title)
			} else {
				title = cp.theme.Bold().Render(item.Title)
			}
			desc := cp.theme.Dim().Render(" (" + item.Description + ")")
			b.WriteString(cursorStr)
			b.WriteString(title)
			b.WriteString(desc)
			b.WriteByte('\n')
		}
	}

	return cp.theme.ActivePane(width, height).Render(b.String())
}
