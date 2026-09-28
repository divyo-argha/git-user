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
	foundPromptIntegration := false
	foundPolicy := false
	foundVerify := false
	foundRefresh := false
	var shellIntegrationItem *ActionItem
	for i, item := range m.items {
		if item.Key == "fix-remote" {
			foundFixRemote = true
		}
		if item.Key == "import-original" {
			foundImportOriginal = true
		}
		if item.Key == "prompt-integration" {
			foundPromptIntegration = true
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
			shellIntegrationItem = &m.items[i]
		}
	}
	if foundFixRemote {
		t.Errorf("SystemActions should NOT include fix-remote when showFixRemote=false")
	}
	if !foundImportOriginal {
		t.Errorf("SystemActions should include import-original when showImportOriginal=true")
	}
	if !foundPromptIntegration {
		t.Errorf("SystemActions should include the prompt-integration action")
	}
	// Team/org governance features, not relevant to personal multi-account
	// use — removed from the TUI menu (still available as CLI commands).
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
	// Shell integration is now its own item, decluttered from the
	// per-identity "open in new terminal" picker (that lives on the
	// identity's own detail screen instead).
	if shellIntegrationItem == nil {
		t.Fatal("SystemActions should include the shell-integration action")
	}
	if shellIntegrationItem.Label != "⌘ Shell integration" {
		t.Errorf("expected shell-integration label %q, got %q", "⌘ Shell integration", shellIntegrationItem.Label)
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

	// Without an unimported original identity — nothing left to import, so
	// the entry must not appear (this is the actual gating this test guards).
	m3 := SystemActions(th, false, false)
	for _, item := range m3.items {
		if item.Key == "import-original" {
			t.Error("SystemActions should NOT include import-original when showImportOriginal=false")
		}
	}
}

func TestSystemActions_UpdateStatus(t *testing.T) {
	th := theme.DefaultTheme()
	m := SystemActions(th, false, false)

	// By default update item should be disabled and show "up to date"
	var updateItem *ActionItem
	for i := range m.items {
		if m.items[i].Key == "update" {
			updateItem = &m.items[i]
			break
		}
	}
	if updateItem == nil {
		t.Fatalf("Update action item not found in SystemActions")
	}
	if !updateItem.Disabled {
		t.Errorf("Expected update item to be disabled by default")
	}

	// Enable when update is available
	m.SetUpdateStatus("v5.0.0", true)
	if updateItem.Disabled {
		t.Errorf("Expected update item to be enabled when update is available")
	}
	if updateItem.Label != "▲ Update available (v5.0.0)" {
		t.Errorf("Unexpected label: %s", updateItem.Label)
	}

	// Disable again when up to date
	m.SetUpdateStatus("v4.8.0", false)
	if !updateItem.Disabled {
		t.Errorf("Expected update item to be disabled when up to date")
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

