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

	m := NewActionMenu(items, th)

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
	m := SystemActions(th)

	// Ensure system actions contains quit
	foundQuit := false
	for _, item := range m.items {
		if item.Key == "quit" {
			foundQuit = true
			break
		}
	}
	if !foundQuit {
		t.Errorf("SystemActions is missing quit action")
	}
}
