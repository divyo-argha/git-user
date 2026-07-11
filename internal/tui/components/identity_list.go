package components

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// IdentityItem represents a single item in the identity list.
type IdentityItem struct {
	Name       string
	Email      string
	IsActive   bool
	IsOriginal bool
	HasSSHKey  bool
	HasSigning bool
	BindCount  int
	IsAction   bool
	ActionKey  string
}

// IdentityList is a scrollable, filterable list of identities.
type IdentityList struct {
	items      []IdentityItem
	filtered   []int
	cursor     int
	filter     string
	filtering  bool
	theme      theme.Theme
}

// NewIdentityList creates an identity list from a config store.
func NewIdentityList(store *config.Store, th theme.Theme) IdentityList {
	items := buildIdentityItems(store)
	filtered := make([]int, len(items))
	for i := range items {
		filtered[i] = i
	}
	return IdentityList{items: items, filtered: filtered, theme: th}
}

func buildIdentityItems(store *config.Store) []IdentityItem {
	var items []IdentityItem
	for _, u := range store.Users {
		items = append(items, IdentityItem{
			Name:       u.Name,
			Email:      u.Email,
			IsActive:   u.Name == store.Current,
			IsOriginal: u.Source == "original",
			HasSSHKey:  u.SSHKey != "",
			HasSigning: !u.SignDisabled && u.SignKey != "",
			BindCount:  len(u.BindPaths),
		})
	}
	items = append(items, IdentityItem{IsAction: true, ActionKey: "register"})
	return items
}

// Refresh rebuilds the list from a new store.
func (l *IdentityList) Refresh(store *config.Store) {
	l.items = buildIdentityItems(store)
	l.applyFilter()
	if l.cursor >= len(l.filtered) {
		l.cursor = max(0, len(l.filtered)-1)
	}
}

func (l *IdentityList) CursorUp() {
	if l.cursor > 0 {
		l.cursor--
	}
}

func (l *IdentityList) CursorDown() {
	if l.cursor < len(l.filtered)-1 {
		l.cursor++
	}
}

func (l *IdentityList) Selected() *IdentityItem {
	if len(l.filtered) == 0 || l.cursor >= len(l.filtered) {
		return nil
	}
	idx := l.filtered[l.cursor]
	return &l.items[idx]
}

func (l *IdentityList) Cursor() int     { return l.cursor }
func (l *IdentityList) IsFiltering() bool { return l.filtering }

func (l *IdentityList) SetFilter(query string) {
	l.filter = query
	l.filtering = query != ""
	l.applyFilter()
}

func (l *IdentityList) ClearFilter() {
	l.filter = ""
	l.filtering = false
	l.applyFilter()
}

func (l *IdentityList) applyFilter() {
	if l.filter == "" {
		l.filtered = make([]int, len(l.items))
		for i := range l.items {
			l.filtered[i] = i
		}
		return
	}

	query := strings.ToLower(l.filter)
	l.filtered = l.filtered[:0]
	for i, item := range l.items {
		if item.IsAction {
			continue
		}
		if strings.Contains(strings.ToLower(item.Name), query) ||
			strings.Contains(strings.ToLower(item.Email), query) {
			l.filtered = append(l.filtered, i)
		}
	}
	if l.cursor >= len(l.filtered) {
		l.cursor = max(0, len(l.filtered)-1)
	}
}

// View renders the identity list.
func (l IdentityList) View(width, height int, isActive bool) string {
	var lines []string

	lines = append(lines, l.theme.PaneTitle().Render("Git Identities"))
	lines = append(lines, l.theme.SeparatorLine(width-6))

	if l.filtering {
		filterLine := l.theme.InfoStyle().Render("🔍 ") + l.filter + l.theme.Dim().Render("│")
		lines = append(lines, filterLine)
	}

	for vi, idx := range l.filtered {
		item := l.items[idx]
		isCursor := vi == l.cursor

		if item.IsAction {
			label := l.theme.InfoStyle().Render("+ Register new identity")
			if isCursor && isActive {
				lines = append(lines, l.theme.Selected().Render("▶ "+stripAnsi(label)))
			} else if isCursor && !isActive {
				lines = append(lines, l.theme.Dim().Render("▶ Register new identity"))
			} else {
				lines = append(lines, "  "+label)
			}
			continue
		}

		line := l.renderIdentityLine(item, isCursor, isActive)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (l IdentityList) renderIdentityLine(item IdentityItem, isCursor, isActive bool) string {
	var prefix string
	if item.IsActive {
		prefix = l.theme.Active().Render("● ")
	} else {
		prefix = "○ "
	}

	var badges []string
	if item.HasSSHKey {
		badges = append(badges, l.theme.SuccessStyle().Render("✓SSH"))
	} else {
		badges = append(badges, l.theme.Dim().Render("○SSH"))
	}
	if item.HasSigning {
		badges = append(badges, l.theme.SuccessStyle().Render("✓Sign"))
	}
	if item.BindCount > 0 {
		badges = append(badges, l.theme.Dim().Render(fmt.Sprintf("%d paths", item.BindCount)))
	}

	badgeStr := ""
	if len(badges) > 0 {
		badgeStr = "  " + strings.Join(badges, " ")
	}

	nameStr := item.Name
	if item.IsActive {
		nameStr = l.theme.Active().Render(item.Name) + "  " + l.theme.Dim().Render(item.Email) + "  " + l.theme.Active().Render("[active]")
	} else {
		nameStr = item.Name + "  " + l.theme.Dim().Render(item.Email)
	}

	if item.IsOriginal {
		nameStr += "  " + l.theme.SuccessStyle().Render("(original)")
	}

	fullLine := prefix + nameStr + badgeStr

	if isCursor && isActive {
		return l.theme.Selected().Render("▶ " + stripAnsi(fullLine))
	} else if isCursor && !isActive {
		return l.theme.Dim().Render("▶ " + stripAnsi(fullLine))
	}
	return "  " + fullLine
}

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
