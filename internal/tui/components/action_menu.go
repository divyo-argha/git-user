package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// ActionItem represents a single item in the action menu.
type ActionItem struct {
	Label     string
	Key       string
	IsSection bool
	IsDanger  bool
	Featured  bool // bold, distinctly coloured at rest (not colour-only: placement and label carry the meaning too)
	Disabled  bool
}

// ActionMenu is the right-pane action menu with sections and icons.
type ActionMenu struct {
	items  []ActionItem
	cursor int
	theme  theme.Theme
	Title  string
}

// NewActionMenu creates an action menu from a list of items.
func NewActionMenu(title string, items []ActionItem, th theme.Theme) ActionMenu {
	m := ActionMenu{Title: title, items: items, theme: th}
	m.cursor = m.nextSelectable(-1)
	return m
}

// SystemActions returns the default system utilities action list.
// showFixRemote controls whether the "Fix remotes (HTTPS → SSH)" entry is
// included; pass true only when the current repo has HTTPS remotes that need
// converting. hasOriginalIdentity flags the "Import / Export" entry when the
// machine's global git config holds a name+email that isn't already one of
// git-user's registered profiles, so the import path stays discoverable
// without a separate, near-duplicate menu row (that screen offers the
// "import original gitconfig" option itself).
func SystemActions(th theme.Theme, showFixRemote, hasOriginalIdentity bool) ActionMenu {
	items := []ActionItem{
		{IsSection: true, Label: "Quick Actions"},
		{Label: "◈ Commit identity stats", Key: "stats", Featured: true},
		{Label: "→ Sign out", Key: "logout"},
	}

	if showFixRemote {
		items = append(items, ActionItem{Label: "⇄ Fix remote → SSH", Key: "fix-remote"})
	}

	importLabel := "⇪ Import / Export"
	if hasOriginalIdentity {
		importLabel += " (existing git identity found)"
	}

	items = append(items,
		ActionItem{IsSection: true, Label: "Health & Security"},
		ActionItem{Label: "✦ Doctor", Key: "doctor"},
		ActionItem{Label: "≡ Identity switch log", Key: "log"},
		ActionItem{IsSection: true, Label: "Profiles & System"},
		ActionItem{Label: importLabel, Key: "import-export"},
		ActionItem{Label: "⚓ Git hooks", Key: "hook"},
		ActionItem{Label: "↻ Sync identities", Key: "sync"},
		ActionItem{Label: "❯ Terminal & shell integration", Key: "prompt-integration"},
		ActionItem{IsSection: true, Label: "Danger Zone"},
		ActionItem{Label: "✖ Uninstall", Key: "uninstall", IsDanger: true},
	)
	return NewActionMenu("System Utilities", items, th)
}

// updateSectionLabel heads the update row. The heading is text, so the
// notice does not depend on colour to stand out.
const updateSectionLabel = "Update Available"

// SetUpdateStatus shows an "Update available" entry at the very top of the
// menu when a newer release exists and removes it otherwise. The menu carries
// no permanent "up to date" row: there is nothing to act on then, and the
// status bar already shows the version.
func (m *ActionMenu) SetUpdateStatus(latestVersion string, updateAvailable bool) {
	selectedKey := ""
	if sel := m.Selected(); sel != nil {
		selectedKey = sel.Key
	}

	idx := -1
	for i := range m.items {
		if m.items[i].Key == "update" {
			idx = i
			break
		}
	}

	switch {
	case updateAvailable && latestVersion != "":
		label := "▲ Update available (" + latestVersion + ")"
		if idx >= 0 {
			m.items[idx].Label = label
			return
		}
		top := []ActionItem{
			{IsSection: true, Label: updateSectionLabel},
			{Label: label, Key: "update"},
		}
		m.items = append(top, m.items...)
	case idx >= 0:
		// Drop the row and its heading (always the item just above it).
		start := idx
		if start > 0 && m.items[start-1].IsSection && m.items[start-1].Label == updateSectionLabel {
			start--
		}
		m.items = append(m.items[:start], m.items[idx+1:]...)
	default:
		return
	}

	if selectedKey != "" && selectedKey != "update" {
		m.FindAndSetCursorByKey(selectedKey)
	} else {
		m.cursor = m.nextSelectable(-1)
	}
}

func (m *ActionMenu) CursorUp()           { m.cursor = m.prevSelectable(m.cursor) }
func (m *ActionMenu) CursorDown()         { m.cursor = m.nextSelectable(m.cursor) }
func (m *ActionMenu) Cursor() int         { return m.cursor }
func (m *ActionMenu) ResetCursor()        { m.cursor = m.nextSelectable(-1) }
func (m *ActionMenu) Items() []ActionItem { return m.items }

// FindAndSetCursorByKey restores the cursor onto the item with the given key,
// preserving selection across a rebuild (e.g. after an async status check
// changes labels). If that item is no longer selectable — disabled by the
// rebuild, or gone entirely — the cursor falls back to the first selectable
// item instead of resting on (or staying on) an item that silently swallows
// Enter, which would otherwise look like an unresponsive button.
func (m *ActionMenu) FindAndSetCursorByKey(key string) {
	for i, item := range m.items {
		if !item.IsSection && item.Key == key {
			if !item.Disabled {
				m.cursor = i
				return
			}
			break
		}
	}
	m.cursor = m.nextSelectable(-1)
}

func (m *ActionMenu) Selected() *ActionItem {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	return &m.items[m.cursor]
}

func (m *ActionMenu) nextSelectable(from int) int {
	for i := from + 1; i < len(m.items); i++ {
		if !m.items[i].IsSection && !m.items[i].Disabled && m.items[i].Key != "" {
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
		if !m.items[i].IsSection && !m.items[i].Disabled && m.items[i].Key != "" {
			return i
		}
	}
	return from
}

// PreferredWidth returns the natural rendered width of the widest line in this
// menu, including the title header. The caller can use this
// to size the right pane exactly to fit the content instead of half the terminal.
// minWidth / maxWidth clamp the result.
func (m *ActionMenu) PreferredWidth(minWidth, maxWidth int) int {
	// Account for border (1 each side = 2) + padding (2 each side = 4) = 6 extra
	const boxOverhead = 6

	title := m.Title
	if title == "" {
		title = "System Utilities"
	}
	widest := lipgloss.Width(title)
	for _, item := range m.items {
		var lineLen int
		if item.IsSection {
			// "  LABEL" — 2 spaces + uppercased label
			lineLen = 2 + lipgloss.Width(strings.ToUpper(item.Label))
		} else {
			// "▶ Label" (cursor) or "  Label" (normal) — longest form is with cursor prefix
			lineLen = 2 + lipgloss.Width(item.Label)
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

// View renders the action menu, windowed to height so it never overflows
// its pane border regardless of terminal size.
func (m ActionMenu) View(width, height int, isActive bool) string {
	var lines []string

	title := m.Title
	if title == "" {
		title = "System Utilities"
	}
	lines = append(lines, m.theme.PaneTitle().Render(title))
	headerRows := 1

	total := len(m.items)

	// How many item rows fit in the remaining pane height?
	// Reserve 2 rows for potential top/bottom scroll indicators.
	availRows := height - headerRows - 2
	if availRows < 1 {
		availRows = 1
	}

	visibleCount := availRows
	if visibleCount > total {
		visibleCount = total
	}

	// Keep cursor inside the window.
	windowStart := m.cursor - visibleCount + 1
	if windowStart < 0 {
		windowStart = 0
	}
	if m.cursor < windowStart {
		windowStart = m.cursor
	}
	windowEnd := windowStart + visibleCount
	if windowEnd > total {
		windowEnd = total
		windowStart = windowEnd - visibleCount
		if windowStart < 0 {
			windowStart = 0
		}
	}

	hiddenAbove := windowStart
	hiddenBelow := total - windowEnd

	if hiddenAbove > 0 {
		lines = append(lines, m.theme.Dim().Render(fmt.Sprintf("  ▲ %d more above", hiddenAbove)))
	} else {
		lines = append(lines, "") // blank spacer keeps layout stable
	}

	for i := windowStart; i < windowEnd; i++ {
		item := m.items[i]

		if item.IsSection {
			lines = append(lines, "  "+m.theme.SectionHeader().Render(strings.ToUpper(item.Label)))
			continue
		}

		isCursor := i == m.cursor
		label := item.Label

		if item.Disabled {
			label = m.theme.Dim().Render(label)
		}

		// resting is how the row looks when it is not under the cursor.
		resting := label
		switch {
		case item.Key == "update":
			resting = m.theme.WarningStyle().Bold(true).Render(label)
		case item.Featured && !item.Disabled:
			resting = m.theme.Featured().Render(label)
		}

		switch {
		case isCursor && isActive:
			raw := stripAnsi(label)
			switch {
			case item.IsDanger:
				lines = append(lines, m.theme.DangerText().Render("▶ "+raw))
			case item.Key == "update":
				lines = append(lines, m.theme.WarningStyle().Bold(true).Render("▶ "+raw))
			default:
				lines = append(lines, m.theme.Selected().Render("▶ "+raw))
			}
		case isCursor:
			raw := stripAnsi(label)
			if item.Key == "update" {
				lines = append(lines, m.theme.WarningStyle().Render("▶ "+raw))
			} else {
				lines = append(lines, m.theme.Dim().Render("▶ "+raw))
			}
		default:
			lines = append(lines, "  "+resting)
		}
	}

	if hiddenBelow > 0 {
		lines = append(lines, m.theme.Dim().Render(fmt.Sprintf("  ▼ %d more below", hiddenBelow)))
	}

	return strings.Join(lines, "\n")
}
