package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestIdentityList(t *testing.T) {
	th := theme.DefaultTheme()

	store := &config.Store{
		Current: "eng",
		Users: []config.User{
			{Name: "personal", Email: "personal@example.com"},
			{Name: "eng", Email: "eng@company.com"},
			{Name: "shared", Email: "shared@example.com"},
		},
	}

	list := NewIdentityList(store, th)

	// List should have 3 identities + 2 action items (Register, Create temporary)
	if len(list.items) != 5 {
		t.Errorf("Expected 5 items, got %d", len(list.items))
	}

	// Active user should be eng
	// Initial cursor should be 0 (personal)
	if list.Cursor() != 0 {
		t.Errorf("Expected cursor at 0, got %d", list.Cursor())
	}

	list.CursorDown()
	list.CursorDown()
	if list.Cursor() != 2 {
		t.Errorf("Expected cursor at 2, got %d", list.Cursor())
	}

	// Test Refresh
	store.Users = []config.User{
		{Name: "only", Email: "only@example.com"},
	}
	list.Refresh(store)
	if len(list.items) != 3 {
		t.Errorf("Expected 3 items after refresh, got %d", len(list.items))
	}
}

func actionRows(l IdentityList) []IdentityItem {
	var out []IdentityItem
	for _, it := range l.items {
		if it.IsAction {
			out = append(out, it)
		}
	}
	return out
}

// "Create temporary profile" lives in the left panel, directly under
// "Register new identity", after all real identities.
func TestIdentityList_RegisterActionsOrderedAtBottom(t *testing.T) {
	store := &config.Store{Users: []config.User{{Name: "a", Email: "a@x.com"}, {Name: "b", Email: "b@x.com"}}}
	list := NewIdentityList(store, theme.DefaultTheme())

	n := len(list.items)
	reg, tmp := list.items[n-2], list.items[n-1]
	if !reg.IsAction || reg.ActionKey != "register" || reg.ActionLabel != "+ Register new identity" {
		t.Errorf("second-to-last row should be Register, got %+v", reg)
	}
	if !tmp.IsAction || tmp.ActionKey != "register-temp" || tmp.ActionLabel != "~ Create temporary profile" {
		t.Errorf("last row should be Create temporary profile, got %+v", tmp)
	}
	if len(actionRows(list)) != 2 {
		t.Errorf("expected exactly two action rows, got %d", len(actionRows(list)))
	}

	// Reachable with the keyboard and dispatches the right action.
	for i := 0; i < n; i++ {
		list.CursorDown()
	}
	if sel := list.Selected(); sel == nil || sel.ActionKey != "register-temp" {
		t.Errorf("cursor should be able to reach Create temporary profile, got %+v", sel)
	}
}

func TestIdentityList_TemporaryNoLongerInSystemMenu(t *testing.T) {
	m := SystemActions(theme.DefaultTheme(), true, true, true)
	for _, k := range menuKeys(m) {
		if k == "register-temp" {
			t.Error("register-temp moved to the left panel and must not also be in the System menu")
		}
	}
}

func TestIdentityList_ActionRowsSurviveFilterAndDoNotCountAsMatches(t *testing.T) {
	store := &config.Store{Users: []config.User{{Name: "work", Email: "w@x.com"}, {Name: "home", Email: "h@x.com"}}}
	list := NewIdentityList(store, theme.DefaultTheme())

	list.FilterByQuery("zzz-no-match")
	if len(list.filtered) != 2 {
		t.Fatalf("both action rows should stay visible under a filter, got %d rows", len(list.filtered))
	}
	out := list.ViewWithFilter(60, 20, true, true, "zzz-no-match")
	if !strings.Contains(out, "(0 match)") || !strings.Contains(out, "Create temporary profile") {
		t.Errorf("filter bar should report 0 matches and still list the actions:\n%s", out)
	}
}

func TestIdentityList_EmptyStateShownWithTwoActionRows(t *testing.T) {
	list := NewIdentityList(&config.Store{}, theme.DefaultTheme())
	out := list.View(60, 20, true)
	if !strings.Contains(out, "Welcome to git-user") {
		t.Errorf("welcome text should show when there are no profiles:\n%s", out)
	}
	list = NewIdentityList(&config.Store{Users: []config.User{{Name: "a", Email: "a@x.com"}}}, theme.DefaultTheme())
	if strings.Contains(list.View(60, 20, true), "Welcome to git-user") {
		t.Error("welcome text should be hidden once a profile exists")
	}
}

func TestIdentityList_ActionRowsStyledDistinctly(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	store := &config.Store{Users: []config.User{{Name: "alpha", Email: "a@x.com"}, {Name: "beta", Email: "b@x.com"}}}
	list := NewIdentityList(store, theme.DefaultTheme())
	list.introDone = true
	out := list.View(70, 20, true) // cursor on "alpha"

	find := func(text string) string {
		for _, l := range strings.Split(out, "\n") {
			if strings.Contains(l, text) {
				return l
			}
		}
		t.Fatalf("%q not in output:\n%s", text, out)
		return ""
	}
	reg, tmp, name := find("Register new identity"), find("Create temporary profile"), find("beta")
	for label, l := range map[string]string{"register": reg, "temporary": tmp} {
		if !strings.Contains(l, "\x1b[1;") {
			t.Errorf("%s row should be bold at rest: %q", label, l)
		}
	}
	colour := func(l string) string { return strings.SplitN(l[strings.Index(l, "\x1b["):], "m", 2)[0] }
	if reg == "" || colour(reg) != colour(tmp) {
		t.Errorf("both action rows should share the action colour:\n%q\n%q", reg, tmp)
	}
	if strings.Contains(name, colour(reg)) {
		t.Errorf("identity names must not use the action colour: %q vs %q", name, colour(reg))
	}

	// No colour: the glyph and wording still identify the rows.
	lipgloss.SetColorProfile(termenv.Ascii)
	plain := list.View(70, 20, true)
	if !strings.Contains(plain, "+ Register new identity") || !strings.Contains(plain, "~ Create temporary profile") {
		t.Errorf("rows must be readable without colour:\n%s", plain)
	}
}
