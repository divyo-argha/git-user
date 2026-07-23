package screens

import (
	"github.com/divyo-argha/git-user/internal/tui/core"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestDetail(t *testing.T) {
	th := theme.DefaultTheme()

	store := &config.Store{
		Users: []config.User{{Name: "work", Email: "work@company.com"}},
	}
	detail := NewDetail(store, "work", th)

	// Test Initial
	if detail.actions.Cursor() != 1 {
		t.Errorf("Expected menu cursor at 1 (Profile Name)")
	}

	// Test down
	updatedModel, _ := detail.Update(tea.KeyMsg{Type: tea.KeyDown})
	detail = updatedModel.(*Detail)
	if detail.actions.Cursor() != 2 {
		t.Errorf("Expected menu cursor at 2 (Email Address)")
	}

	// Test Esc returns pop
	_, cmd := detail.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("Expected cmd on Esc")
	}
	msg := cmd()
	if _, ok := msg.(core.ScreenPopMsg); !ok {
		t.Errorf("Expected core.ScreenPopMsg on Esc")
	}

	// Navigate to the "Switch to this identity" option
	// (Index 1 is name, 2 email, 4 key, 7 passphrase, 11 checking, 13 switch)
	// Let's reset cursor, and press down until we find key == "switch"
	detail.actions.ResetCursor()
	for i := 0; i < 30; i++ {
		selected := detail.actions.Selected()
		if selected != nil && selected.Key == "switch" {
			break
		}
		detail.actions.CursorDown()
	}

	_, cmd = detail.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("Expected cmd on Enter")
	}
	msg = cmd()
	if actionMsg, ok := msg.(core.ActionResultMsg); ok {
		if actionMsg.Kind != "switch" {
			t.Errorf("Expected switch action, got %s", actionMsg.Kind)
		}
	} else {
		t.Errorf("Expected core.ActionResultMsg on Enter, got %T", msg)
	}
}
