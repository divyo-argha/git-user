package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/diagnostics"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"github.com/divyo-argha/git-user/internal/validate"
)

// IdentityItem represents a single item in the identity list.
type IdentityItem struct {
	Name        string
	Email       string
	IsActive    bool
	IsTemporary bool
	HasSSHKey   bool
	HasSigning  bool
	BindCount   int
	IsAction    bool
	ActionKey   string

	// HasToken/TokenExpiring/TokenExpired are derived from HTTPSTokenExpiresAt
	// metadata alone (same as doctor's per-profile check) — not a real
	// keyring lookup, since checking every row's OS keychain entry on every
	// refresh tick would be needless I/O for a purely cosmetic badge.
	HasToken      bool
	TokenExpiring bool
	TokenExpired  bool

	// PolicyChecked/PolicyOK are only ever set on the active identity's item,
	// and only when the current repo has a .git-user-policy with a
	// requirement — see computeActivePolicyStatus.
	PolicyChecked bool
	PolicyOK      bool
}

// IdentityList is a scrollable list of identities.
type IdentityList struct {
	items     []IdentityItem
	filtered  []int
	cursor    int
	theme     theme.Theme
	introTick int    // how many items have fully appeared (staggered reveal)
	introDone bool   // true once all items are revealed
	query     string // active filter query, preserved across Refresh
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
		hasToken, expiring, expired := TokenBadgeState(u.HTTPSTokenExpiresAt)
		items = append(items, IdentityItem{
			Name:          u.Name,
			Email:         u.Email,
			IsActive:      u.Name == store.Current,
			IsTemporary:   u.IsTemporary,
			HasSSHKey:     u.SSHKey != "",
			HasSigning:    !u.SignDisabled && u.SignKey != "",
			BindCount:     len(u.BindPaths),
			HasToken:      hasToken,
			TokenExpiring: expiring,
			TokenExpired:  expired,
		})
	}
	items = append(items, IdentityItem{IsAction: true, ActionKey: "register"})
	computeActivePolicyStatus(items, store)
	return items
}

// TokenBadgeState derives a token status badge purely from the
// HTTPSTokenExpiresAt metadata already in config — the same proxy doctor's
// per-profile audit uses, not a real keyring lookup.
func TokenBadgeState(expiresAt string) (hasToken, expiring, expired bool) {
	if expiresAt == "" {
		return false, false, false
	}
	hasToken = true
	t, err := time.Parse(validate.DateLayout, expiresAt)
	if err != nil {
		return true, false, false
	}
	days := int(time.Until(t).Hours() / 24)
	if days < 0 {
		expired = true
	} else if days <= diagnostics.TokenExpiryWarnDays {
		expiring = true
	}
	return
}

// computeActivePolicyStatus checks the active identity against the current
// repo's .git-user-policy (if any) exactly once per rebuild — not per row —
// since it needs a repo-root lookup and a file read, unlike the free
// metadata-only checks above.
func computeActivePolicyStatus(items []IdentityItem, store *config.Store) {
	repoRoot, err := git.RepoRoot()
	if err != nil {
		return
	}
	policy, err := config.LoadRepoPolicy(repoRoot)
	if err != nil || (!policy.RequireSigning && len(policy.AllowedEmailDomains) == 0) {
		return
	}
	for i := range items {
		if !items[i].IsActive {
			continue
		}
		user := store.FindUser(items[i].Name)
		if user == nil {
			continue
		}
		ok := true
		if policy.RequireSigning && (user.SignDisabled || user.SignKey == "") {
			ok = false
		}
		if len(policy.AllowedEmailDomains) > 0 {
			_, domain, _ := strings.Cut(strings.ToLower(user.Email), "@")
			allowed := false
			for _, d := range policy.AllowedEmailDomains {
				if domain == d {
					allowed = true
					break
				}
			}
			if !allowed {
				ok = false
			}
		}
		items[i].PolicyChecked = true
		items[i].PolicyOK = ok
	}
}

// Refresh rebuilds the list from a new store, preserving any active filter.
// The periodic background store refresh calls this every few seconds; without
// reapplying the filter here, a filter the user is actively typing would be
// silently cleared out from under them while the query text stayed on screen.
func (l *IdentityList) Refresh(store *config.Store) {
	l.items = buildIdentityItems(store)
	if l.query != "" {
		l.filterByQuery(l.query)
	} else {
		l.applyFilter()
	}
	if l.cursor >= len(l.filtered) {
		l.cursor = max(0, len(l.filtered)-1)
	}
}

// TickIntro advances the staggered-reveal animation by one step.
// Call once per animation frame. Returns true while animation is still
// running so the caller knows when to stop requesting redraws.
func (l *IdentityList) TickIntro() bool {
	if l.introDone {
		return false
	}
	l.introTick++
	if l.introTick >= len(l.filtered) {
		l.introDone = true
	}
	return !l.introDone
}

// IntroComplete returns true once all items have been revealed.
func (l *IdentityList) IntroComplete() bool { return l.introDone }

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

func (l *IdentityList) Cursor() int { return l.cursor }

// FilterByQuery filters the list to items whose name or email contains q
// (case-insensitive). An empty query shows all items.
func (l *IdentityList) FilterByQuery(q string) {
	l.query = q
	l.filterByQuery(q)
}

// filterByQuery applies q to l.items without touching l.query, so Refresh can
// reapply the stored query without it looking like a new filter was typed.
func (l *IdentityList) filterByQuery(q string) {
	l.filtered = l.filtered[:0]
	q = strings.ToLower(q)
	for i, item := range l.items {
		if item.IsAction {
			// Always show the Register action.
			l.filtered = append(l.filtered, i)
			continue
		}
		if strings.Contains(strings.ToLower(item.Name), q) ||
			strings.Contains(strings.ToLower(item.Email), q) {
			l.filtered = append(l.filtered, i)
		}
	}
	// Clamp cursor.
	if l.cursor >= len(l.filtered) {
		l.cursor = max(0, len(l.filtered)-1)
	}
}

// ClearFilter resets the filter and shows all items.
func (l *IdentityList) ClearFilter() {
	l.query = ""
	l.applyFilter()
	l.cursor = 0
}

func (l *IdentityList) applyFilter() {
	l.filtered = make([]int, len(l.items))
	for i := range l.items {
		l.filtered[i] = i
	}
}

// View renders the identity list, clipped to height so it never overflows the pane.
// ViewWithFilter renders the list with an optional filter bar at the top.
// When filterMode is true, a filter prompt is shown and filterQuery is displayed.
func (l IdentityList) ViewWithFilter(width, height int, isActive, filterMode bool, filterQuery string) string {
	if !filterMode {
		return l.View(width, height, isActive)
	}
	// Render a filter bar on top, shrink the list by 1 row.
	matchCount := 0
	for _, idx := range l.filtered {
		if !l.items[idx].IsAction {
			matchCount++
		}
	}
	filterBar := l.theme.InfoStyle().Render(fmt.Sprintf("  / Filter: %s▌  (%d match)", filterQuery, matchCount))
	listView := l.View(width, height-1, isActive)
	return filterBar + "\n" + listView
}

func (l IdentityList) View(width, height int, isActive bool) string {
	var lines []string

	lines = append(lines, l.theme.PaneTitle().Render("Git Identities"))
	lines = append(lines, l.theme.SeparatorLine(width-6))

	// Header rows already consumed (title + separator).
	headerRows := 2

	if len(l.items) <= 1 {
		lines = append(lines, "")
		lines = append(lines, l.theme.InfoStyle().Render("  ✦ Welcome to git-user!"))
		lines = append(lines, l.theme.Dim().Render("  No custom profiles registered yet."))
		lines = append(lines, l.theme.Dim().Render("  Select '+ Register new identity' below."))
		lines = append(lines, "")
	}

	total := len(l.filtered)
	if total == 0 {
		return strings.Join(lines, "\n")
	}

	// How many item rows fit in the remaining pane height?
	// Reserve 2 rows for potential top/bottom indicators.
	availRows := height - headerRows - 2
	if availRows < 1 {

		availRows = 1
	}

	// Compute the scroll window [windowStart, windowStart+visibleCount).
	visibleCount := availRows
	if visibleCount > total {
		visibleCount = total
	}

	// Keep cursor inside the window.
	windowStart := l.cursor - visibleCount + 1
	if windowStart < 0 {
		windowStart = 0
	}
	if l.cursor < windowStart {
		windowStart = l.cursor
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

	// Top scroll indicator
	if hiddenAbove > 0 {
		lines = append(lines, l.theme.Dim().Render(fmt.Sprintf("  ▲ %d more above", hiddenAbove)))
	} else {
		lines = append(lines, "") // blank spacer keeps layout stable
	}

	// Visible items
	for vi := windowStart; vi < windowEnd; vi++ {
		idx := l.filtered[vi]
		item := l.items[idx]
		isCursor := vi == l.cursor

		// ── Staggered reveal ────────────────────────────────────────────────
		// While the intro animation is running, items above introTick are
		// shown in full; items at or beyond introTick are dim placeholders.
		if !l.introDone && vi >= l.introTick {
			dot := l.theme.Dim().Render("  · ")
			name := l.theme.Dim().Render(item.Name)
			if item.IsAction {
				name = l.theme.Dim().Render("Register new identity")
			}
			lines = append(lines, dot+name)
			continue
		}

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

	// Bottom scroll indicator
	if hiddenBelow > 0 {
		lines = append(lines, l.theme.Dim().Render(fmt.Sprintf("  ▼ %d more below", hiddenBelow)))
	}

	return strings.Join(lines, "\n")
}

func (l IdentityList) renderIdentityLine(item IdentityItem, isCursor, isActive bool) string {
	var prefix string
	if item.IsActive {
		prefix = l.theme.Active().Render("● ")
	} else {
		prefix = l.theme.Dim().Render("○ ")
	}

	var badges []string
	if item.HasSSHKey {
		badges = append(badges, l.theme.PillBadge().Render("SSH"))
	}
	if item.HasSigning {
		badges = append(badges, l.theme.PillBadge().Render("SIGN"))
	}
	if item.HasToken {
		switch {
		case item.TokenExpired:
			badges = append(badges, l.theme.PillDanger().Render("TOKEN"))
		case item.TokenExpiring:
			badges = append(badges, l.theme.PillWarning().Render("TOKEN"))
		default:
			badges = append(badges, l.theme.PillBadge().Render("TOKEN"))
		}
	}
	if item.IsActive && item.PolicyChecked {
		if item.PolicyOK {
			badges = append(badges, l.theme.PillBadge().Render("POLICY"))
		} else {
			badges = append(badges, l.theme.PillDanger().Render("POLICY"))
		}
	}
	if item.BindCount > 0 {
		badges = append(badges, l.theme.Dim().Render(fmt.Sprintf("• %d paths", item.BindCount)))
	}

	badgeStr := ""
	if len(badges) > 0 {
		badgeStr = "  " + strings.Join(badges, " ")
	}

	nameStr := l.theme.Bold().Render(item.Name) + "  " + l.theme.Dim().Render("<"+item.Email+">")
	if item.IsActive {
		nameStr = l.theme.Active().Render(item.Name) + "  " + l.theme.Dim().Render("<"+item.Email+">") + "  " + l.theme.PillActive().Render("● ACTIVE")
	}

	if item.IsTemporary {
		nameStr += "  " + l.theme.PillWarning().Render("TEMP")
	}

	fullLine := prefix + nameStr + badgeStr

	if isCursor && isActive {
		return l.theme.Selected().Render("▶ ") + fullLine
	} else if isCursor && !isActive {
		return l.theme.Dim().Render("▶ ") + fullLine
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
