package tui

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestAppStack(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	startScreen := screens.NewDashboard(store, th)
	app := NewApp(store, startScreen)

	if len(app.screenStack) != 1 {
		t.Errorf("Expected stack length 1, got %d", len(app.screenStack))
	}

	// Push core.Screen
	testScreen := screens.NewConfirm("test", "ctx", th)
	updated, _ := app.Update(core.ScreenPushMsg{Screen: testScreen})
	app = updated.(*App)

	if len(app.screenStack) != 2 {
		t.Errorf("Expected stack length 2 after push, got %d", len(app.screenStack))
	}

	// Pop core.Screen
	updated, _ = app.Update(core.ScreenPopMsg{})
	app = updated.(*App)

	if len(app.screenStack) != 1 {
		t.Errorf("Expected stack length 1 after pop, got %d", len(app.screenStack))
	}

	// Action Result
	updated, cmd := app.Update(core.ActionResultMsg{Kind: "switch", Name: "work"})
	app = updated.(*App)
	if app.action == nil {
		t.Errorf("Expected action to be set")
	}
	if app.action.kind != "switch" || app.action.name != "work" {
		t.Errorf("Expected action kind 'switch' and name 'work'")
	}
	if cmd == nil {
		t.Errorf("Expected tea.Quit cmd")
	}
}
