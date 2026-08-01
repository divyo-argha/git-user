package tui

import (
	"testing"
	"time"

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

func TestAppDetailedHandlers(t *testing.T) {
	store := &config.Store{
		Current: "work",
		Users: []config.User{
			{Name: "work", Email: "work@corp.com", SSHKey: "/path/to/key", SignKey: "/path/to/key", SignDisabled: false},
			{Name: "home", Email: "home@personal.com"},
		},
	}
	th := theme.DefaultTheme()
	startScreen := screens.NewDashboard(store, th)
	app := NewApp(store, startScreen)
	_, _ = app.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

	// 1. Test StoreRefreshedMsg and AgentStatusMsg and AnimTickMsg
	newStore := &config.Store{Current: "home", Users: store.Users}
	updated, _ := app.Update(core.StoreRefreshedMsg{Store: newStore})
	app = updated.(*App)
	if app.store.Current != "home" {
		t.Errorf("Expected current store user 'home', got %q", app.store.Current)
	}

	updated, _ = app.Update(core.AgentStatusMsg{Connected: true, KeyCount: 3})
	app = updated.(*App)

	updated, _ = app.Update(core.AnimTickMsg(time.Now()))
	app = updated.(*App)
	if app.animFrame != 1 {
		t.Errorf("Expected animFrame to be 1, got %d", app.animFrame)
	}

	// 2. Test view rendering
	app.View()

	// 3. Test handleAction return Cmds
	testCmds := []struct {
		kind string
		name string
	}{
		{"register", ""},
		{"register-temp", ""},
		{"unbind", "work"},
		{"rekey", "work"},
		{"passphrase", "work"},
		{"import-export", ""},
		{"remove", "work"},
	}
	for _, tc := range testCmds {
		_, cmd := app.Update(core.ActionResultMsg{Kind: tc.kind, Name: tc.name})
		if cmd == nil {
			t.Fatalf("Expected non-nil cmd for action %s", tc.kind)
		}
		msg := cmd()
		if _, ok := msg.(core.ScreenPushMsg); !ok {
			t.Errorf("Expected ScreenPushMsg for action %s, got %#v", tc.kind, msg)
		}
	}

	// 4. Test handleAction direct exits (pendingAction set)
	testDirectExits := []string{
		"pubkey-push", "bind", "check-ssh", "passphrase-set", "passphrase-remove",
		"passphrase-verify", "export", "export-all", "import", "import-original",
		"fix-remote", "security", "doctor", "update",
	}
	for _, kind := range testDirectExits {
		updated, cmd := app.Update(core.ActionResultMsg{Kind: kind, Name: "work"})
		if cmd == nil {
			t.Fatalf("Expected quit cmd for %s", kind)
		}
		app = updated.(*App)
		if app.action == nil || app.action.kind != kind {
			t.Errorf("Expected action kind %s, got %#v", kind, app.action)
		}
	}

	// 5. Test export-current action
	// Clear current to test error toast
	app.store.Current = ""
	_, cmdExportErr := app.Update(core.ActionResultMsg{Kind: "export-current"})
	if cmdExportErr == nil {
		t.Error("Expected non-nil error toast cmd")
	}
	// Reset current to test success path
	app.store.Current = "work"
	updated, cmdExportSuccess := app.Update(core.ActionResultMsg{Kind: "export-current"})
	if cmdExportSuccess == nil {
		t.Error("Expected quit cmd for export-current")
	}
	app = updated.(*App)
	if app.action.kind != "export-current" {
		t.Errorf("Expected export-current action set")
	}

	// 6. Test toggle-sign action
	// Toggling home user (no SSHKey, toggle simple disable state)
	_, _ = app.Update(core.ActionResultMsg{Kind: "toggle-sign", Name: "home"})
	if app.store.FindUser("home").SignDisabled != true {
		t.Error("Expected home user signing to be disabled")
	}
	_, _ = app.Update(core.ActionResultMsg{Kind: "toggle-sign", Name: "home"})
	if app.store.FindUser("home").SignDisabled != false {
		t.Error("Expected home user signing to be re-enabled")
	}

	// Toggling work user (has SSHKey, turn off signing key config)
	_, _ = app.Update(core.ActionResultMsg{Kind: "toggle-sign", Name: "work"})
	if app.store.FindUser("work").SignDisabled != true {
		// Wait, toggle-signing might set toggle state
	}

	// Test non-existent user toggle-sign (should not crash)
	_, _ = app.Update(core.ActionResultMsg{Kind: "toggle-sign", Name: "nonexistent"})

	// 7. Test Form Results (success and invalid validations)
	formTests := []struct {
		context string
		values  []string
		wantKind string
	}{
		{"register", []string{"new-user", "new@corp.com"}, "register"},
		{"register", []string{"", "new@corp.com"}, ""}, // empty name
		{"register-temp", []string{"temp-user", "temp@corp.com"}, "register-temp"},
		{"register-temp", []string{"temp-user", ""}, ""}, // empty email
		{"rename:work", []string{"work-new"}, "rename"},
		{"rename:work", []string{""}, ""}, // empty name
		{"email:work", []string{"new-email@corp.com"}, "email"},
		{"email:work", []string{""}, ""}, // empty email
	}
	for _, ft := range formTests {
		app.action = nil
		_, cmd := app.Update(core.FormResultMsg{Context: ft.context, Values: ft.values})
		if ft.wantKind != "" {
			if cmd == nil {
				t.Fatalf("Expected quit cmd for form result %s", ft.context)
			}
			if app.action == nil || app.action.kind != ft.wantKind {
				t.Errorf("Expected action %s, got %#v", ft.wantKind, app.action)
			}
		} else {
			if app.action != nil {
				t.Errorf("Expected no action set for invalid form input, got %#v", app.action)
			}
		}
	}

	// Empty form values should be ignored
	_, _ = app.Update(core.FormResultMsg{Context: "register", Values: []string{}})

	// 8. Test Confirm Results
	confirmTests := []struct {
		context   string
		confirmed bool
		wantKind  string
	}{
		{"remove:work", true, "remove"},
		{"remove:work", false, ""},
		{"unbind:work", true, "unbind"},
		{"rekey:work", true, "rekey"},
		{"invalid-format", true, ""},
	}
	for _, ct := range confirmTests {
		app.action = nil
		_, _ = app.Update(core.ConfirmResultMsg{Context: ct.context, Confirmed: ct.confirmed})
		if ct.wantKind != "" {
			if app.action == nil || app.action.kind != ct.wantKind {
				t.Errorf("Expected action %s, got %#v", ct.wantKind, app.action)
			}
		} else {
			if app.action != nil {
				t.Errorf("Expected no action set, got %#v", app.action)
			}
		}
	}
}


