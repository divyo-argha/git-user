package components

import (
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

	// Without fix-remote
	m := SystemActions(th, false)

	foundFixRemote := false
	foundImportOriginal := false
	for _, item := range m.items {
		if item.Key == "fix-remote" {
			foundFixRemote = true
		}
		if item.Key == "import-original" {
			foundImportOriginal = true
		}
	}
	if foundFixRemote {
		t.Errorf("SystemActions should NOT include fix-remote when showFixRemote=false")
	}
	if !foundImportOriginal {
		t.Errorf("SystemActions should include the import-original action")
	}

	// With fix-remote
	m2 := SystemActions(th, true)
	foundFixRemote2 := false
	for _, item := range m2.items {
		if item.Key == "fix-remote" {
			foundFixRemote2 = true
		}
	}
	if !foundFixRemote2 {
		t.Errorf("SystemActions should include fix-remote when showFixRemote=true")
	}
}

func TestSystemActions_UpdateStatus(t *testing.T) {
	th := theme.DefaultTheme()
	m := SystemActions(th, false)

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
	if updateItem.Label != "▲ Update git-user (v5.0.0 available)" {
		t.Errorf("Unexpected label: %s", updateItem.Label)
	}

	// Disable again when up to date
	m.SetUpdateStatus("v4.8.0", false)
	if !updateItem.Disabled {
		t.Errorf("Expected update item to be disabled when up to date")
	}
}

