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

	// Switch now runs entirely in-TUI: no tea.Quit.
	updated, cmd := app.Update(core.ActionResultMsg{Kind: "switch", Name: "work"})
	app = updated.(*App)
	if app.Quit() {
		t.Errorf("Expected app not to quit for in-TUI switch")
	}
	if cmd == nil {
		t.Errorf("Expected a task cmd for switch")
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

	// FormResultMsg for register now pushes the SSH setup options screen
	// instead of exiting to the terminal.
	updated, cmd := app.Update(core.FormResultMsg{Context: "register", Values: []string{"work", "work@corp.com"}})
	app = updated.(*App)
	if cmd == nil {
		t.Fatalf("Expected a push cmd for register form")
	}
	if msg := cmd(); msg != nil {
		if _, ok := msg.(core.ScreenPushMsg); !ok {
			t.Errorf("Expected ScreenPushMsg for register form, got %#v", msg)
		}
	}

	// ConfirmResultMsg for remove now runs an in-TUI task.
	updated, cmd = app.Update(core.ConfirmResultMsg{Context: "remove:work", Confirmed: true})
	app = updated.(*App)
	if app.Quit() {
		t.Errorf("Expected app not to quit for remove")
	}
	if cmd == nil {
		t.Errorf("Expected a task cmd for remove confirm")
	}

	// Test Init
	_ = app.Init()

	// Test handleAction (quit)
	updated, _ = app.Update(core.ActionResultMsg{Kind: "quit"})
	app = updated.(*App)
	if !app.Quit() {
		t.Errorf("Expected Quit to be true")
	}
	app.quit = false

	// Test handleAction (pubkey) — stays in-TUI.
	updated, cmd = app.Update(core.ActionResultMsg{Kind: "pubkey", Name: "work"})
	app = updated.(*App)
	if app.Quit() {
		t.Errorf("Expected app not to quit for pubkey")
	}
	if cmd == nil {
		t.Errorf("Expected a cmd for pubkey")
	}

	// Test handleAction (register-temp) pushes a form.
	updated, cmd = app.Update(core.ActionResultMsg{Kind: "register-temp"})
	app = updated.(*App)
	if cmd == nil {
		t.Errorf("Expected a push cmd for register-temp")
	} else if msg := cmd(); msg != nil {
		if _, ok := msg.(core.ScreenPushMsg); !ok {
			t.Errorf("Expected ScreenPushMsg for register-temp, got %#v", msg)
		}
	}

	// Test handleAction (rename) pushes a form.
	_, cmd = app.Update(core.ActionResultMsg{Kind: "rename", Name: "work"})
	if cmd == nil {
		t.Errorf("Expected a push cmd for rename")
	}

	// Test handleAction (logout) runs in-TUI.
	updated, cmd = app.Update(core.ActionResultMsg{Kind: "logout"})
	app = updated.(*App)
	if app.Quit() {
		t.Errorf("Expected no quit for in-TUI logout")
	}
	if cmd == nil {
		t.Errorf("Expected a task cmd for logout")
	}

	// Test handleAction (email) pushes a form.
	_, cmd = app.Update(core.ActionResultMsg{Kind: "email", Name: "work"})
	if cmd == nil {
		t.Errorf("Expected a push cmd for email")
	}

	// Test handleAction (bind-path) pushes a form.
	_, cmd = app.Update(core.ActionResultMsg{Kind: "bind-path", Name: "work"})
	if cmd == nil {
		t.Errorf("Expected a push cmd for bind-path")
	}

	// Test handleAction (unbind-path) — no paths bound: toast, still in-TUI.
	updated, _ = app.Update(core.ActionResultMsg{Kind: "unbind-path", Name: "work"})
	app = updated.(*App)
	if app.Quit() {
		t.Errorf("Expected no quit for in-TUI unbind-path")
	}
}

func TestHandleTaskResultSwitchShowsReportWhenWarnings(t *testing.T) {
	store := &config.Store{Users: []config.User{{Name: "work", Email: "work@corp.com"}}}
	th := theme.DefaultTheme()
	app := NewApp(store, screens.NewDashboard(store, th))

	collect := func(cmd tea.Cmd) []tea.Msg {
		var msgs []tea.Msg
		var walk func(c tea.Cmd)
		walk = func(c tea.Cmd) {
			if c == nil {
				return
			}
			m := c()
			if m == nil {
				return
			}
			msgs = append(msgs, m)
			if b, ok := m.(tea.BatchMsg); ok {
				for _, inner := range b {
					walk(inner)
				}
			}
		}
		walk(cmd)
		return msgs
	}
	reportPushed := func(msgs []tea.Msg) bool {
		for _, m := range msgs {
			if push, ok := m.(core.ScreenPushMsg); ok {
				if _, ok := push.Screen.(*screens.Report); ok {
					return true
				}
			}
		}
		return false
	}

	// With warnings the switch result surfaces a Report screen.
	_, cmd := app.handleTaskResult(core.TaskResultMsg{
		Kind:       "switch",
		Name:       "work",
		Success:    true,
		Detail:     "Switched to \"work\" (work@corp.com)\n⚠ Bound SSH key not found: /nope",
		ShowReport: true,
	})
	if !reportPushed(collect(cmd)) {
		t.Error("expected a Report screen to be pushed when a switch produces warnings")
	}

	// Without warnings only the toast is shown.
	_, cmd = app.handleTaskResult(core.TaskResultMsg{
		Kind:       "switch",
		Name:       "work",
		Success:    true,
		Detail:     "Switched to \"work\" (work@corp.com)",
		ShowReport: false,
	})
	if reportPushed(collect(cmd)) {
		t.Error("expected no Report screen when a switch has no warnings")
	}
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

	// 3. Test handleAction screens that push a screen
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
		if msg == nil {
			t.Fatalf("Expected a message for action %s", tc.kind)
		}
		if _, ok := msg.(core.ScreenPushMsg); !ok {
			t.Errorf("Expected ScreenPushMsg for action %s, got %#v", tc.kind, msg)
		}
	}

	// 4. Test handleAction in-TUI operations (no pendingAction, no quit).
	testInTUIOps := []string{
		"switch", "logout", "fix-remote", "security", "doctor", "check-ssh",
		"pubkey-push", "bind", "passphrase-set", "passphrase-remove", "passphrase-verify",
		"export", "export-all", "import", "import-original", "update",
	}
	for _, kind := range testInTUIOps {
		updated, cmd := app.Update(core.ActionResultMsg{Kind: kind, Name: "work"})
		if cmd == nil {
			t.Fatalf("Expected non-nil cmd for %s", kind)
		}
		app = updated.(*App)
		if app.Quit() {
			t.Errorf("Expected app not to quit for %s", kind)
		}
	}

	// 5. Test export-current action stays in-TUI.
	app.store.Current = ""
	_, cmdExportErr := app.Update(core.ActionResultMsg{Kind: "export-current"})
	if cmdExportErr == nil {
		t.Error("Expected non-nil error toast cmd")
	}
	app.store.Current = "work"
	updated, cmdExportSuccess := app.Update(core.ActionResultMsg{Kind: "export-current"})
	if cmdExportSuccess == nil {
		t.Error("Expected non-nil cmd for export-current")
	}
	app = updated.(*App)

	// 6. Test toggle-sign action
	_, _ = app.Update(core.ActionResultMsg{Kind: "toggle-sign", Name: "home"})
	if app.store.FindUser("home").SignDisabled != true {
		t.Error("Expected home user signing to be disabled")
	}
	_, _ = app.Update(core.ActionResultMsg{Kind: "toggle-sign", Name: "home"})
	if app.store.FindUser("home").SignDisabled != false {
		t.Error("Expected home user signing to be re-enabled")
	}
	_, _ = app.Update(core.ActionResultMsg{Kind: "toggle-sign", Name: "nonexistent"})

	// 7. Test Form Results (validation still happens in-TUI).
	formTests := []struct {
		context string
		values  []string
	}{
		{"register", []string{"new-user", "new@corp.com"}},
		{"register", []string{"", "new@corp.com"}},    // empty name → error toast, no action
		{"register", []string{"x", "not-an-email"}},   // invalid email → error toast
		{"register-temp", []string{"temp-user", "temp@corp.com"}},
		{"register-temp", []string{"temp-user", ""}}, // empty email → error toast
		{"rename:work", []string{"work-new"}},
		{"rename:work", []string{""}},
		{"email:work", []string{"new-email@corp.com"}},
		{"email:work", []string{""}},
	}
	for _, ft := range formTests {
		updated, cmd := app.Update(core.FormResultMsg{Context: ft.context, Values: ft.values})
		app = updated.(*App)
		if cmd == nil {
			t.Fatalf("Expected non-nil cmd for form result %s %v", ft.context, ft.values)
		}
		if app.Quit() {
			t.Errorf("Expected app not to quit for form result %s", ft.context)
		}
	}

	// Empty form values should be ignored
	_, _ = app.Update(core.FormResultMsg{Context: "register", Values: []string{}})

	// 8. Test Confirm Results stay in-TUI.
	confirmTests := []struct {
		context   string
		confirmed bool
	}{
		{"remove:work", true},
		{"remove:work", false},
		{"unbind:work", true},
		{"rekey:work", true},
		{"invalid-format", true},
	}
	for _, ct := range confirmTests {
		updated, cmd := app.Update(core.ConfirmResultMsg{Context: ct.context, Confirmed: ct.confirmed})
		app = updated.(*App)
		if app.Quit() {
			t.Errorf("Expected app not to quit for confirm %s", ct.context)
		}
		_ = cmd
	}
}
