package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestDashboard(t *testing.T) {
	th := theme.DefaultTheme()

	store := &config.Store{
		Current: "work",
		Users: []config.User{
			{Name: "personal", Email: "personal@example.com"},
			{Name: "work", Email: "work@company.com"},
		},
	}

	dash := NewDashboard(store, th)

	// Test Initial Pane
	if dash.activePane != PaneIdentities {
		t.Errorf("Expected active pane to be identity list")
	}

	// Test switching pane
	updated, _ := dash.Update(tea.KeyMsg{Type: tea.KeyTab})
	dash = updated.(*Dashboard)
	if dash.activePane != PaneActions {
		t.Errorf("Expected active pane to be action menu after tab")
	}

	updated, _ = dash.Update(tea.KeyMsg{Type: tea.KeyLeft})
	dash = updated.(*Dashboard)
	if dash.activePane != PaneIdentities {
		t.Errorf("Expected active pane to be identity list after left")
	}

	// Test enter on identity list triggers ScreenPushMsg
	// Cursor should be at 0 ("personal")
	_, cmd := dash.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("Expected cmd on Enter")
	}
	msg := cmd()
	pushMsg, ok := msg.(ScreenPushMsg)
	if !ok {
		t.Errorf("Expected ScreenPushMsg on identity list enter")
	}
	_, okDetail := pushMsg.Screen.(*Detail)
	if !okDetail {
		t.Errorf("Expected pushed screen to be Detail")
	}

	// Test filtering
	updated, _ = dash.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	dash = updated.(*Dashboard)
	if !dash.filtering {
		t.Errorf("Expected filtering to be true after '/'")
	}

	// Exit filtering
	updated, _ = dash.Update(tea.KeyMsg{Type: tea.KeyEsc})
	dash = updated.(*Dashboard)
	if dash.filtering {
		t.Errorf("Expected filtering to be false after esc")
	}

	// Quitting
	_, cmd = dash.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Errorf("Expected tea.Quit command on 'q'")
	}
}
