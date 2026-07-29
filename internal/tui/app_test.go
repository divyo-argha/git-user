package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestAppMessagesAndLifecycle(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	startScreen := screens.NewDashboard(store, th)
	app := NewApp(store, startScreen)

	// Test Initial loading View
	view := app.View()
	if view != "Loading..." {
		t.Errorf("Expected Loading..., got %q", view)
	}

	// Send WindowSizeMsg
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	app = updated.(*App)
	if app.width != 80 || app.height != 40 {
		t.Errorf("Expected size 80x40, got %dx%d", app.width, app.height)
	}

	updated, _ = app.Update(core.ToastMsg{Text: "Test message", Style: theme.ToastStyleSuccess})
	app = updated.(*App)
	if !app.toast.IsVisible() {
		t.Errorf("Expected toast to be visible")
	}

	// Send ToastExpiredMsg
	updated, _ = app.Update(core.ToastExpiredMsg{})
	app = updated.(*App)
	if app.toast.IsVisible() {
		t.Errorf("Expected toast to be hidden")
	}

	// Send FormResultMsg
	updated, _ = app.Update(core.FormResultMsg{Context: "register", Values: []string{"work", "work@corp.com"}})
	app = updated.(*App)
	if app.action == nil || app.action.kind != "register" {
		t.Errorf("Expected register action set")
	}

	// Send ConfirmResultMsg
	updated, _ = app.Update(core.ConfirmResultMsg{Context: "remove:work", Confirmed: true})
	app = updated.(*App)
	if app.action == nil || app.action.kind != "remove" {
		t.Errorf("Expected remove action set")
	}

	// Test Init
	_ = app.Init()

	// Test handleAction (quit)
	updated, _ = app.Update(core.ActionResultMsg{Kind: "quit"})
	app = updated.(*App)
	if !app.Quit() {
		t.Errorf("Expected Quit to be true")
	}

	// Test handleAction (pubkey)
	updated, _ = app.Update(core.ActionResultMsg{Kind: "pubkey", Name: "work"})
	app = updated.(*App)
	kind, name, _ := app.PendingAction()
	if kind != "pubkey" || name != "work" {
		t.Errorf("Expected pending action pubkey for work, got %s for %s", kind, name)
	}

	// Test handleAction (register-temp)
	_, _ = app.Update(core.ActionResultMsg{Kind: "register-temp"})

	// Test handleAction (rename)
	_, _ = app.Update(core.ActionResultMsg{Kind: "rename", Name: "work"})

	// Test handleAction (logout)
	updated, _ = app.Update(core.ActionResultMsg{Kind: "logout"})
	app = updated.(*App)
	kind, _, _ = app.PendingAction()
	if kind != "logout" {
		t.Errorf("Expected logout pending action")
	}

	// Test handleAction (email)
	_, _ = app.Update(core.ActionResultMsg{Kind: "email", Name: "work"})

	// Test handleAction (bind-path)
	_, _ = app.Update(core.ActionResultMsg{Kind: "bind-path", Name: "work"})

	// Test handleAction (unbind-path)
	_, _ = app.Update(core.ActionResultMsg{Kind: "unbind-path", Name: "work"})
}

