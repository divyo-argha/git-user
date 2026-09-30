package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestActionMenu(t *testing.T) {
	th := theme.DefaultTheme()

	items := []ActionItem{
		{Label: "Section 1", IsSection: true},
		{Label: "Item 1", Key: "item1"},
		{Label: "Item 2", Key: "item2"},
		{Label: "Section 2", IsSection: true},
		{Label: "Item 3", Key: "item3", Disabled: true},
		{Label: "Item 4", Key: "item4"},
	}

	m := NewActionMenu("Test Menu", items, th)

	// Initial cursor should skip Section 1 and be on Item 1
	if m.Cursor() != 1 {
		t.Errorf("Expected cursor at 1, got %d", m.Cursor())
	}
	if m.Selected().Key != "item1" {
		t.Errorf("Expected item1, got %v", m.Selected().Key)
	}

	// Move down
	m.CursorDown()
	if m.Cursor() != 2 {
		t.Errorf("Expected cursor at 2, got %d", m.Cursor())
	}

	// Move down again - should skip Section 2 and Item 3 (disabled) to land on Item 4
	m.CursorDown()
	if m.Cursor() != 5 {
		t.Errorf("Expected cursor at 5, got %d", m.Cursor())
	}

	// Move up
	m.CursorUp()
	if m.Cursor() != 2 {
		t.Errorf("Expected cursor at 2, got %d", m.Cursor())
	}
}

func TestSystemActions(t *testing.T) {
	th := theme.DefaultTheme()

	// Without fix-remote, with an unimported original identity
	m := SystemActions(th, false, true, true)

	foundFixRemote := false
	foundImportOriginal := false
	importExportLabel := ""
	foundPromptIntegration := false
	foundPolicy := false
	foundVerify := false
	foundRefresh := false
	foundShellIntegration := false
	var promptIntegrationItem *ActionItem
	for i, item := range m.items {
		if item.Key == "fix-remote" {
			foundFixRemote = true
		}
		if item.Key == "import-original" {
			foundImportOriginal = true
		}
		if item.Key == "import-export" {
			importExportLabel = item.Label
		}
		if item.Key == "prompt-integration" {
			foundPromptIntegration = true
			promptIntegrationItem = &m.items[i]
		}
		if item.Key == "policy" {
			foundPolicy = true
		}
		if item.Key == "verify" {
			foundVerify = true
		}
		if item.Key == "refresh" {
			foundRefresh = true
		}
		if item.Key == "shell-integration" {
			foundShellIntegration = true
		}
	}
	if foundFixRemote {
		t.Errorf("SystemActions should NOT include fix-remote when showFixRemote=false")
	}
	if foundImportOriginal {
		t.Errorf("SystemActions should NOT include a separate import-original row — the Import / Export screen offers it")
	}
	if !strings.Contains(importExportLabel, "existing git identity found") {
		t.Errorf("Import / Export label should flag an importable identity, got %q", importExportLabel)
	}
	if !foundPromptIntegration {
		t.Errorf("SystemActions should include the prompt-integration action")
	}
	// Team/org governance features, not relevant to personal multi-account
	// use — kept out of the TUI menu (available as CLI commands).
	if foundPolicy {
		t.Error("SystemActions should NOT include the policy action — it was removed from the TUI menu")
	}
	if foundVerify {
		t.Error("SystemActions should NOT include the verify action — it was removed from the TUI menu")
	}
	// Refresh is now a direct 'r' keybinding on the dashboard, not a menu item.
	if foundRefresh {
		t.Error("SystemActions should NOT include the refresh action — it's now the 'r' key, not a menu item")
	}
	// Shell integration was merged into the terminal prompt indicator item —
	// there is no longer a separate "shell-integration" menu entry, and the
	// merged item's label reflects both.
	if foundShellIntegration {
		t.Error("SystemActions should NOT include a separate shell-integration action — it was merged into prompt-integration")
	}
	if promptIntegrationItem.Label != "❯ Terminal & shell integration" {
		t.Errorf("expected prompt-integration label %q, got %q", "❯ Terminal & shell integration", promptIntegrationItem.Label)
	}

	// With fix-remote
	m2 := SystemActions(th, true, true, true)
	foundFixRemote2 := false
	for _, item := range m2.items {
		if item.Key == "fix-remote" {
			foundFixRemote2 = true
		}
	}
	if !foundFixRemote2 {
		t.Errorf("SystemActions should include fix-remote when showFixRemote=true")
	}

	// Without an unimported original identity the label stays plain.
	m3 := SystemActions(th, false, false, true)
	for _, item := range m3.items {
		if item.Key == "import-export" && item.Label != "⇪ Import / Export" {
			t.Errorf("import-export label should be plain when nothing to import, got %q", item.Label)
		}
	}
}

func menuKeys(m ActionMenu) []string {
	var keys []string
	for _, it := range m.items {
		if !it.IsSection {
			keys = append(keys, it.Key)
		}
	}
	return keys
}

func hasKey(m ActionMenu, key string) bool {
	for _, k := range menuKeys(m) {
		if k == key {
			return true
		}
	}
	return false
}

func TestSystemActions_UpdateStatus(t *testing.T) {
	th := theme.DefaultTheme()
	m := SystemActions(th, false, false, true)

	// No permanent "up to date" row: nothing to act on.
	if hasKey(m, "update") {
		t.Fatal("update row should be absent by default")
	}

	// Appears when an update is available, at the very top under its own heading.
	m.SetUpdateStatus("v5.0.0", true)
	if !hasKey(m, "update") {
		t.Fatal("expected update row when update is available")
	}
	if !(m.items[0].IsSection && m.items[0].Label == "Update Available") {
		t.Errorf("first item should be the Update Available heading, got %+v", m.items[0])
	}
	if it := m.items[1]; it.Key != "update" || it.Label != "▲ Update available (v5.0.0)" || it.Disabled {
		t.Errorf("second item should be the enabled update row, got %+v", it)
	}
	// Focus is never stolen: the cursor stays on whatever the user had selected.
	if sel := m.Selected(); sel == nil || sel.Key != "stats" {
		t.Errorf("cursor should stay on its previous item, got %+v", sel)
	}

	// Label refreshes in place without duplicating the row.
	m.SetUpdateStatus("v5.1.0", true)
	n := 0
	for _, k := range menuKeys(m) {
		if k == "update" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("expected exactly one update row, got %d", n)
	}

	// Disappears again when up to date, heading included.
	m.SetUpdateStatus("v4.8.0", false)
	if hasKey(m, "update") {
		t.Error("update row should be removed when up to date")
	}
	for _, it := range m.items {
		if it.IsSection && it.Label == "Update Available" {
			t.Error("the Update Available heading should be removed with the row")
		}
	}
	if m.items[0].Label != "Quick Actions" {
		t.Errorf("menu should start with Quick Actions again, got %+v", m.items[0])
	}
}

func TestSystemActions_UpdateStatusKeepsCursor(t *testing.T) {
	th := theme.DefaultTheme()
	m := SystemActions(th, false, false, true)
	m.FindAndSetCursorByKey("stats")

	m.SetUpdateStatus("v5.0.0", true)
	if sel := m.Selected(); sel == nil || sel.Key != "stats" {
		t.Errorf("cursor should stay on stats after the update row is inserted, got %+v", sel)
	}

	// Cursor on the update row, then it is removed: fall back to a selectable item.
	m.FindAndSetCursorByKey("update")
	m.SetUpdateStatus("", false)
	sel := m.Selected()
	if sel == nil || sel.IsSection || sel.Key == "" || sel.Disabled {
		t.Errorf("cursor should land on a selectable item after removal, got %+v", sel)
	}
}

func TestActionMenu_ViewFitsHeight(t *testing.T) {
	th := theme.DefaultTheme()
	m := SystemActions(th, true, true, true)

	const height = 8
	out := m.View(40, height, true)

	lineCount := strings.Count(out, "\n") + 1
	if lineCount > height {
		t.Errorf("ActionMenu.View rendered %d lines, want <= %d (height)", lineCount, height)
	}
}

func TestSystemActions_StatsIsFeaturedNearTop(t *testing.T) {
	m := SystemActions(theme.DefaultTheme(), false, false, true)

	var statsIdx, logoutIdx = -1, -1
	for i, it := range m.items {
		switch it.Key {
		case "stats":
			statsIdx = i
			if !it.Featured {
				t.Error("stats should be a featured item")
			}
		case "logout":
			logoutIdx = i
		default:
			if it.Featured {
				t.Errorf("only stats should be featured, got %q", it.Key)
			}
		}
	}
	if statsIdx != 1 || logoutIdx < statsIdx {
		t.Errorf("stats should be the first item under Quick Actions (idx 1), got stats=%d logout=%d", statsIdx, logoutIdx)
	}
	n := 0
	for _, k := range menuKeys(m) {
		if k == "stats" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("stats should appear exactly once, got %d", n)
	}
}

func TestActionMenu_FeaturedAndUpdateRenderBoldAndDistinct(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	th := theme.DefaultTheme()
	m := SystemActions(th, false, false, true)
	m.SetUpdateStatus("v5.0.0", true)
	m.FindAndSetCursorByKey("doctor") // cursor elsewhere so stats/update render at rest

	out := m.View(60, 40, true)
	lineWith := func(text string) string {
		for _, l := range strings.Split(out, "\n") {
			if strings.Contains(l, text) {
				return l
			}
		}
		t.Fatalf("line containing %q not found in:\n%s", text, out)
		return ""
	}
	stats, update, plain := lineWith("Commit identity stats"), lineWith("Update available"), lineWith("Identity switch log")

	for name, l := range map[string]string{"stats": stats, "update": update} {
		if !strings.Contains(l, "\x1b[1") && !strings.Contains(l, ";1m") {
			t.Errorf("%s row should be bold at rest, got %q", name, l)
		}
	}
	if !strings.Contains(stats, "\x1b[") || strings.Contains(plain, "\x1b[") {
		t.Errorf("stats should be styled and ordinary rows plain.\nstats=%q\nplain=%q", stats, plain)
	}
	if stats == update || strings.Split(stats, "m")[0] == strings.Split(update, "m")[0] {
		t.Errorf("stats and update must use different colours.\nstats=%q\nupdate=%q", stats, update)
	}
	// Without colour the notice must still be discoverable as text.
	lipgloss.SetColorProfile(termenv.Ascii)
	plainOut := m.View(60, 40, true)
	if !strings.Contains(plainOut, "UPDATE AVAILABLE") || !strings.Contains(plainOut, "Update available (v5.0.0)") {
		t.Errorf("update notice must be readable without colour:\n%s", plainOut)
	}
}

func TestSystemActions_CloneMovedToProfileMenu(t *testing.T) {
	m := SystemActions(theme.DefaultTheme(), true, true, true)
	if hasKey(m, "clone") {
		t.Error("clone is identity-specific and now lives in the profile menu, not System Utilities")
	}
}

func TestSystemActions_SignOutOnlyWhenSignedIn(t *testing.T) {
	th := theme.DefaultTheme()
	if !hasKey(SystemActions(th, false, false, true), "logout") {
		t.Error("Sign out should be offered while an identity is active")
	}
	out := SystemActions(th, false, false, false)
	if hasKey(out, "logout") {
		t.Error("Sign out must be hidden when nobody is signed in")
	}
	// The rest of Quick Actions is unaffected and the cursor lands on a real item.
	if !hasKey(out, "stats") {
		t.Error("stats should still be offered")
	}
	if sel := out.Selected(); sel == nil || sel.IsSection || sel.Key == "" {
		t.Errorf("cursor should start on a selectable item, got %+v", sel)
	}
}
