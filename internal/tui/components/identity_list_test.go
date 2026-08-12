package components

import (
	"testing"

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

	// List should have 3 identities + 1 action item (Register)
	if len(list.items) != 4 {
		t.Errorf("Expected 4 items, got %d", len(list.items))
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
	if len(list.items) != 2 {
		t.Errorf("Expected 2 items after refresh, got %d", len(list.items))
	}
}
