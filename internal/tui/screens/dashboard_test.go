package screens

import (
	"github.com/divyo-argha/git-user/internal/tui/core"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestDashboard(t *testing.T) {
	th := theme.DefaultTheme()

	store := &config.Store{
		Current: "eng",
		Users: []config.User{
			{Name: "personal", Email: "personal@example.com"},
			{Name: "eng", Email: "eng@company.com"},
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

	// Test enter on identity list triggers core.ScreenPushMsg
	// Cursor should be at 0 ("personal")
	_, cmd := dash.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("Expected cmd on Enter")
	}
	msg := cmd()
	pushMsg, ok := msg.(core.ScreenPushMsg)
	if !ok {
		t.Errorf("Expected core.ScreenPushMsg on identity list enter")
	}
	_, okDetail := pushMsg.Screen.(*Detail)
	if !okDetail {
		t.Errorf("Expected pushed screen to be Detail")
	}

	// Quitting
	_, cmd = dash.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Errorf("Expected tea.Quit command on 'q'")
	}
}

func TestDashboardOutOfSyncWarningAndFix(t *testing.T) {
	th := theme.DefaultTheme()
	store := &config.Store{
		Current: "eng",
		Users: []config.User{
			{Name: "eng", Email: "eng@company.com"},
		},
	}
	dash := NewDashboard(store, th)

	// Initially no warning.
	if dash.syncOut {
		t.Error("expected syncOut=false before any sync status arrives")
	}

	// Report an out-of-sync git config.
	dash.Update(core.SyncStatusMsg{InSync: false})
	if !dash.syncOut {
		t.Fatal("expected syncOut=true after out-of-sync report")
	}

	// Help text advertises the fix key.
	if !strings.Contains(dash.ShortHelp(), "f•re-apply") && !strings.Contains(dash.ShortHelp(), "f re-apply") {
		t.Errorf("ShortHelp should advertise the fix key, got: %q", dash.ShortHelp())
	}

	// View shows the warning banner.
	if v := dash.View(80, 20); !strings.Contains(v, "out of sync") {
		t.Error("dashboard view should contain the out-of-sync warning")
	}

	// 'f' dispatches the fix-sync action.
	_, cmd := dash.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if cmd == nil {
		t.Fatal("expected cmd for 'f' when out of sync")
	}
	msg := cmd()
	action, ok := msg.(core.ActionResultMsg)
	if !ok || action.Kind != "fix-sync" {
		t.Errorf("expected ActionResultMsg{Kind: fix-sync}, got %#v", msg)
	}

	// Re-synced: warning disappears, 'f' no longer dispatches.
	dash.Update(core.SyncStatusMsg{InSync: true})
	if dash.syncOut {
		t.Error("expected syncOut=false after in-sync report")
	}
	if v := dash.View(80, 20); strings.Contains(v, "out of sync") {
		t.Error("dashboard view should not contain the warning when in sync")
	}
	_, cmd = dash.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if cmd != nil {
		t.Error("expected no cmd for 'f' when in sync")
	}
}
