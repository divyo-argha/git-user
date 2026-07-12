package screens

import (
	"github.com/divyo-argha/git-user/internal/tui/core"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestConfirm(t *testing.T) {
	th := theme.DefaultTheme()
	confirm := NewConfirm("Are you sure?", "delete", th)

	// Starts at Cancel (1)
	if confirm.cursor != 1 {
		t.Errorf("Expected cursor at 1 (Cancel)")
	}

	// Press left -> Yes (0)
	updated, _ := confirm.Update(tea.KeyMsg{Type: tea.KeyLeft})
	confirm = updated.(*Confirm)
	if confirm.cursor != 0 {
		t.Errorf("Expected cursor at 0 (Yes)")
	}

	// Press right -> Cancel (1)
	updated, _ = confirm.Update(tea.KeyMsg{Type: tea.KeyRight})
	confirm = updated.(*Confirm)
	if confirm.cursor != 1 {
		t.Errorf("Expected cursor at 1 (Cancel)")
	}

	// Enter on Cancel (1) returns false
	_, cmd := confirm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("Expected command on Enter")
	}
	res := cmd()
	if confirmRes, ok := res.(core.ConfirmResultMsg); ok {
		if confirmRes.Confirmed {
			t.Errorf("Expected Confirmed=false on Cancel")
		}
	} else {
		t.Errorf("Expected core.ConfirmResultMsg")
	}

	// Shortcut y returns true
	_, cmd = confirm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatalf("Expected command on 'y'")
	}
	res = cmd()
	if confirmRes, ok := res.(core.ConfirmResultMsg); ok {
		if !confirmRes.Confirmed {
			t.Errorf("Expected Confirmed=true on 'y'")
		}
		if confirmRes.Context != "delete" {
			t.Errorf("Expected context 'delete', got %s", confirmRes.Context)
		}
	}
}
