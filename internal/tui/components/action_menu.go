package components

import (
	"strings"

	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// ActionItem represents a single item in the action menu.
type ActionItem struct {
	Label     string
	Key       string
	IsSection bool
	IsDanger  bool
	Disabled  bool
}

// ActionMenu is the right-pane action menu with sections and icons.
type ActionMenu struct {
	items  []ActionItem
	cursor int
	theme  theme.Theme
}

// NewActionMenu creates an action menu from a list of items.
func NewActionMenu(items []ActionItem, th theme.Theme) ActionMenu {
	m := ActionMenu{items: items, theme: th}
	m.cursor = m.nextSelectable(-1)
	return m
}

// SystemActions returns the default system utilities action list.
// showFixRemote controls whether the "Fix remotes (HTTPS → SSH)" entry is
// included; pass true only when the current repo has HTTPS remotes that need
// converting.
func SystemActions(th theme.Theme, showFixRemote bool) ActionMenu {
	items := []ActionItem{
		{Label: "Sign out (logout)", Key: "logout"},
	}

	if showFixRemote {
		items = append(items, ActionItem{Label: "Fix remotes (HTTPS → SSH)", Key: "fix-remote"})
	}

	items = append(items,
		ActionItem{IsSection: true, Label: "Diagnostics"},
		ActionItem{Label: "Security audit", Key: "security"},
		ActionItem{Label: "Doctor (health check)", Key: "doctor"},
		ActionItem{Label: "Import / Export…", Key: "import-export"},
		ActionItem{Label: "Update git-user", Key: "update"},
		ActionItem{Label: "Quit", Key: "quit"},
	)
	return NewActionMenu(items, th)
}

func (m *ActionMenu) CursorUp()    { m.cursor = m.prevSelectable(m.cursor) }
func (m *ActionMenu) CursorDown()  { m.cursor = m.nextSelectable(m.cursor) }
func (m *ActionMenu) Cursor() int  { return m.cursor }
func (m *ActionMenu) ResetCursor() { m.cursor = m.nextSelectable(-1) }

func (m *ActionMenu) Selected() *ActionItem {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	return &m.items[m.cursor]
}

func (m *ActionMenu) nextSelectable(from int) int {
	for i := from + 1; i < len(m.items); i++ {
		if !m.items[i].IsSection && !m.items[i].Disabled {
			return i
		}
	}
	if from >= 0 {
		return from
	}
	return 0
}

func (m *ActionMenu) prevSelectable(from int) int {
	for i := from - 1; i >= 0; i-- {
		if !m.items[i].IsSection && !m.items[i].Disabled {
			return i
		}
	}
	return from
}

// PreferredWidth returns the natural rendered width of the widest line in this
// menu, including the title "System Utilities" header. The caller can use this
// to size the right pane exactly to fit the content instead of half the terminal.
// minWidth / maxWidth clamp the result.
func (m *ActionMenu) PreferredWidth(minWidth, maxWidth int) int {
	// Account for border (1 each side) + padding (2 each side from Padding(0,2)) = 6 extra
	const boxOverhead = 6

	widest := len("System Utilities") // title is always present
	for _, item := range m.items {
		var lineLen int
		if item.IsSection {
			// "  ── Label ──" — 2 spaces + 3 + label + 3
			lineLen = 2 + 3 + len(item.Label) + 3
		} else {
			// "▶ Label" (cursor) or "  Label" (normal) — longest form is with cursor prefix
			lineLen = 2 + len(item.Label)
		}
		if lineLen > widest {
			widest = lineLen
		}
	}

	w := widest + boxOverhead
	if w < minWidth {
		w = minWidth
	}
	if w > maxWidth {
		w = maxWidth
	}
	return w
}

// View renders the action menu.
func (m ActionMenu) View(width, height int, isActive bool) string {
	var lines []string

	lines = append(lines, m.theme.PaneTitle().Render("System Utilities"))
	lines = append(lines, m.theme.SeparatorLine(width-6))

	for i, item := range m.items {
		if item.IsSection {
			if i > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, "  "+m.theme.SectionHeader().Render("── "+item.Label+" ──"))
			continue
		}

		isCursor := i == m.cursor
		label := item.Label

		if item.Disabled || item.Key == "quit" {
			label = m.theme.Dim().Render(label)
		}

		if isCursor && isActive {
			raw := stripAnsi(label)
			if item.IsDanger {
				lines = append(lines, m.theme.DangerText().Render("▶ "+raw))
			} else {
				lines = append(lines, m.theme.Selected().Render("▶ "+raw))
			}
		} else if isCursor && !isActive {
			lines = append(lines, m.theme.Dim().Render("▶ "+stripAnsi(label)))
		} else {
			lines = append(lines, "  "+label)
		}
	}

	return strings.Join(lines, "\n")
}
