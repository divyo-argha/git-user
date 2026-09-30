package components

import (
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
	m := SystemActions(th, false, true)

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
	m2 := SystemActions(th, true, true)
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
	m3 := SystemActions(th, false, false)
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
	m := SystemActions(th, false, false)

	// No permanent "up to date" row: nothing to act on.
	if hasKey(m, "update") {
		t.Fatal("update row should be absent by default")
	}

	// Appears when an update is available, above the Danger Zone.
	m.SetUpdateStatus("v5.0.0", true)
	if !hasKey(m, "update") {
		t.Fatal("expected update row when update is available")
	}
	for i, it := range m.items {
		if it.Key == "update" {
			if it.Label != "▲ Update available (v5.0.0)" || it.Disabled {
				t.Errorf("unexpected update item: %+v", it)
			}
			if next := m.items[i+1]; !(next.IsSection && next.Label == "Danger Zone") {
				t.Errorf("update row should sit just above Danger Zone, next item is %+v", next)
			}
		}
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

	// Disappears again when up to date.
	m.SetUpdateStatus("v4.8.0", false)
	if hasKey(m, "update") {
		t.Error("update row should be removed when up to date")
	}
}

func TestSystemActions_UpdateStatusKeepsCursor(t *testing.T) {
	th := theme.DefaultTheme()
	m := SystemActions(th, false, false)
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
	m := SystemActions(th, true, true)

	const height = 8
	out := m.View(40, height, true)

	lineCount := strings.Count(out, "\n") + 1
	if lineCount > height {
		t.Errorf("ActionMenu.View rendered %d lines, want <= %d (height)", lineCount, height)
	}
}
