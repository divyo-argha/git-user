package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"github.com/divyo-argha/git-user/internal/version"
)

// TestFixSyncAction_ReappliesActive isolates HOME and GIT_USER_CONFIG so the
// re-apply runs against a temp git config, then verifies the identity was
// written back and the store switched to it.
func TestFixSyncAction_ReappliesActive(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("GIT_USER_CONFIG", filepath.Join(dir, "config.json"))

	store := &config.Store{}
	if err := store.AddUser("eng", "eng@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetCurrent("eng"); err != nil {
		t.Fatal(err)
	}

	app := NewApp(store, screens.NewDashboard(store, theme.DefaultTheme()))

	updated, cmd := app.Update(core.ActionResultMsg{Kind: "fix-sync"})
	app = updated.(*App)
	if app.Quit() {
		t.Fatal("app should not quit for in-TUI fix-sync")
	}
	if cmd == nil {
		t.Fatal("expected a task cmd for fix-sync")
	}

	res := cmd()
	tr, ok := res.(core.TaskResultMsg)
	if !ok {
		t.Fatalf("got %T, want TaskResultMsg", res)
	}
	if !tr.Success {
		t.Fatalf("fix-sync failed: %v", tr.Err)
	}
	if tr.Name != "eng" {
		t.Errorf("TaskResultMsg.Name = %q, want %q", tr.Name, "eng")
	}

	// The identity must be written to the isolated git config.
	cfgData, err := os.ReadFile(filepath.Join(dir, ".gitconfig"))
	if err != nil {
		t.Fatalf("reading isolated .gitconfig: %v", err)
	}
	cfg := string(cfgData)
	if !strings.Contains(cfg, "eng@example.com") {
		t.Errorf("git config does not contain the re-applied email:\n%s", cfg)
	}
}

// TestFixSyncAction_NoActiveIdentity shows an error toast without running a task.
func TestFixSyncAction_NoActiveIdentity(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("GIT_USER_CONFIG", filepath.Join(dir, "config.json"))

	store := &config.Store{}
	app := NewApp(store, screens.NewDashboard(store, theme.DefaultTheme()))

	updated, cmd := app.Update(core.ActionResultMsg{Kind: "fix-sync"})
	app = updated.(*App)
	if app.Quit() {
		t.Fatal("app should not quit")
	}
	if cmd == nil {
		t.Fatal("expected a toast cmd")
	}
	res := cmd()
	if _, ok := res.(core.ToastMsg); !ok {
		t.Fatalf("got %T, want ToastMsg", res)
	}
}

func TestAppStack(t *testing.T) {
	withTempConfig(t)
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
	updated, cmd := app.Update(core.ActionResultMsg{Kind: "switch", Name: "eng"})
	app = updated.(*App)
	if app.Quit() {
		t.Errorf("Expected app not to quit for in-TUI switch")
	}
	if cmd == nil {
		t.Errorf("Expected a task cmd for switch")
	}
}

func TestAppMessagesAndLifecycle(t *testing.T) {
	withTempConfig(t)
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
	updated, cmd := app.Update(core.FormResultMsg{Context: "register", Values: []string{"eng", "eng@corp.com"}})
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
	updated, cmd = app.Update(core.ConfirmResultMsg{Context: "remove:eng", Confirmed: true})
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
	updated, cmd = app.Update(core.ActionResultMsg{Kind: "pubkey", Name: "eng"})
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
	_, cmd = app.Update(core.ActionResultMsg{Kind: "rename", Name: "eng"})
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
	_, cmd = app.Update(core.ActionResultMsg{Kind: "email", Name: "eng"})
	if cmd == nil {
		t.Errorf("Expected a push cmd for email")
	}

	// Test handleAction (bind-path) pushes a form.
	_, cmd = app.Update(core.ActionResultMsg{Kind: "bind-path", Name: "eng"})
	if cmd == nil {
		t.Errorf("Expected a push cmd for bind-path")
	}

	// Test handleAction (unbind-path) — no paths bound: toast, still in-TUI.
	updated, _ = app.Update(core.ActionResultMsg{Kind: "unbind-path", Name: "eng"})
	app = updated.(*App)
	if app.Quit() {
		t.Errorf("Expected no quit for in-TUI unbind-path")
	}
}

// collectMsgs walks a tea.Cmd (including nested tea.Batch) and returns every
// message it produces, for asserting on toasts/screens hidden inside a batch.
func collectMsgs(cmd tea.Cmd) []tea.Msg {
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

// TestHandleToggleSignGivesFeedback guards against a regression where
// toggling commit signing gave the user zero feedback of any kind (no toast,
// no report) — it silently mutated the store and called config.Save with its
// error return value dropped entirely, so a failed save was indistinguishable
// from a successful one.
func TestHandleToggleSignGivesFeedback(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{Users: []config.User{{Name: "eng", Email: "eng@corp.com", SSHKey: "/tmp/nonexistent-key"}}}
	th := theme.DefaultTheme()
	app := NewApp(store, screens.NewDashboard(store, th))

	// Happy path: a success toast must appear (previously there was none).
	_, cmd := app.Update(core.ActionResultMsg{Kind: "toggle-sign", Name: "eng"})
	msgs := collectMsgs(cmd)
	foundToast := false
	for _, m := range msgs {
		if toast, ok := m.(core.ToastMsg); ok {
			foundToast = true
			if toast.Style != theme.ToastStyleSuccess {
				t.Errorf("expected a success toast, got style %v: %q", toast.Style, toast.Text)
			}
		}
	}
	if !foundToast {
		t.Error("expected handleToggleSign to produce a toast (previously it gave no feedback at all)")
	}

	// Failure path: force config.Save to fail via the store's stale-config
	// guard (config.ErrConfigChanged — the file changed on disk after this
	// store was loaded), and confirm the failure surfaces as an error toast
	// instead of being silently dropped.
	primer, _ := config.Load()
	_ = primer.AddUser("eng", "eng@example.com")
	if err := config.Save(primer); err != nil {
		t.Fatalf("priming config: %v", err)
	}
	store2, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	// Modify the file on disk after store2 loaded it, so store2's own Save
	// call below finds its loadedHash stale and returns ErrConfigChanged.
	stale, _ := config.Load()
	_ = stale.AddUser("someone-else", "someone-else@example.com")
	if err := config.Save(stale); err != nil {
		t.Fatalf("staling config: %v", err)
	}

	app2 := NewApp(store2, screens.NewDashboard(store2, th))
	_, cmd2 := app2.Update(core.ActionResultMsg{Kind: "toggle-sign", Name: "eng"})
	msgs2 := collectMsgs(cmd2)
	foundErrorToast := false
	for _, m := range msgs2 {
		if toast, ok := m.(core.ToastMsg); ok && toast.Style == theme.ToastStyleError {
			foundErrorToast = true
		}
	}
	if !foundErrorToast {
		t.Error("expected a failed config.Save to surface as an error toast, not silently succeed")
	}
}

func TestHandleTaskResultSwitchShowsReportWhenWarnings(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{Users: []config.User{{Name: "eng", Email: "eng@corp.com"}}}
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
		Name:       "eng",
		Success:    true,
		Detail:     "Switched to \"eng\" (eng@corp.com)\n⚠ Bound SSH key not found: /nope",
		ShowReport: true,
	})
	if !reportPushed(collect(cmd)) {
		t.Error("expected a Report screen to be pushed when a switch produces warnings")
	}

	// Without warnings only the toast is shown.
	_, cmd = app.handleTaskResult(core.TaskResultMsg{
		Kind:       "switch",
		Name:       "eng",
		Success:    true,
		Detail:     "Switched to \"eng\" (eng@corp.com)",
		ShowReport: false,
	})
	if reportPushed(collect(cmd)) {
		t.Error("expected no Report screen when a switch has no warnings")
	}
}

func TestAppDetailedHandlers(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{
		Current: "eng",
		Users: []config.User{
			{Name: "eng", Email: "eng@corp.com", SSHKey: "/path/to/key", SignKey: "/path/to/key", SignDisabled: false},
			{Name: "private", Email: "private@example.com"},
		},
	}
	th := theme.DefaultTheme()
	startScreen := screens.NewDashboard(store, th)
	app := NewApp(store, startScreen)
	_, _ = app.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

	// 1. Test StoreRefreshedMsg and AgentStatusMsg and AnimTickMsg
	newStore := &config.Store{Current: "private", Users: store.Users}
	updated, _ := app.Update(core.StoreRefreshedMsg{Store: newStore})
	app = updated.(*App)
	if app.store.Current != "private" {
		t.Errorf("Expected current store user 'private', got %q", app.store.Current)
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
		{"unbind", "eng"},
		{"rekey", "eng"},
		{"passphrase", "eng"},
		{"import-export", ""},
		{"remove", "eng"},
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
		updated, cmd := app.Update(core.ActionResultMsg{Kind: kind, Name: "eng"})
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
	app.store.Current = "eng"
	updated, cmdExportSuccess := app.Update(core.ActionResultMsg{Kind: "export-current"})
	if cmdExportSuccess == nil {
		t.Error("Expected non-nil cmd for export-current")
	}
	app = updated.(*App)

	// 6. Test toggle-sign action
	_, _ = app.Update(core.ActionResultMsg{Kind: "toggle-sign", Name: "private"})
	if app.store.FindUser("private").SignDisabled != true {
		t.Error("Expected private user signing to be disabled")
	}
	_, _ = app.Update(core.ActionResultMsg{Kind: "toggle-sign", Name: "private"})
	if app.store.FindUser("private").SignDisabled != false {
		t.Error("Expected private user signing to be re-enabled")
	}
	_, _ = app.Update(core.ActionResultMsg{Kind: "toggle-sign", Name: "nonexistent"})

	// 7. Test Form Results (validation still happens in-TUI).
	formTests := []struct {
		context string
		values  []string
	}{
		{"register", []string{"new-user", "new@corp.com"}},
		{"register", []string{"", "new@corp.com"}},  // empty name → error toast, no action
		{"register", []string{"x", "not-an-email"}}, // invalid email → error toast
		{"register-temp", []string{"guest-user", "guest@corp.com"}},
		{"register-temp", []string{"guest-user", ""}}, // empty email → error toast
		{"rename:eng", []string{"eng-new"}},
		{"rename:eng", []string{""}},
		{"email:eng", []string{"new-email@corp.com"}},
		{"email:eng", []string{""}},
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
		{"remove:eng", true},
		{"remove:eng", false},
		{"unbind:eng", true},
		{"rekey:eng", true},
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

	// 9. Declining commit signing on "ssh-sign" must still complete
	// registration (and thus still show the public key report) instead of
	// being treated as cancelling the whole flow.
	updated, cmd := app.Update(core.ConfirmResultMsg{
		Context:   "ssh-sign:decline-sign|decline-sign@corp.com|register|skip||",
		Confirmed: false,
	})
	app = updated.(*App)
	if cmd == nil {
		t.Fatalf("Expected a task cmd for declined ssh-sign, not a cancel toast")
	}
	msg := cmd()
	result, ok := msg.(core.TaskResultMsg)
	if !ok {
		t.Fatalf("Expected core.TaskResultMsg for declined ssh-sign, got %#v", msg)
	}
	if !result.Success {
		t.Errorf("Expected declined ssh-sign to still succeed, got err: %v", result.Err)
	}
	if app.store.FindUser("decline-sign") == nil {
		t.Errorf("Expected identity to be registered even though commit signing was declined")
	}
}

func TestUpdateVersionRefreshesImmediately(t *testing.T) {
	withTempConfig(t)
	origVer := version.Version
	origBuild := version.BuildVersion
	t.Cleanup(func() {
		version.Version = origVer
		version.BuildVersion = origBuild
	})
	version.Version = "v4.8.1"
	version.BuildVersion = ""

	store := &config.Store{
		Current: "work",
		Users: []config.User{
			{Name: "work", Email: "work@example.com"},
		},
	}
	th := theme.DefaultTheme()
	startScreen := screens.NewDashboard(store, th)
	app := NewApp(store, startScreen)
	_, _ = app.Update(tea.WindowSizeMsg{Width: 80, Height: 40})

	// Check initial version is v4.8.1
	if version.GetVersion() != "v4.8.1" {
		t.Fatalf("Expected initial version v4.8.1, got %q", version.GetVersion())
	}
	if !strings.Contains(app.View(), "Version v4.8.1") {
		t.Errorf("Expected dashboard view to contain 'Version v4.8.1'")
	}

	// Remote check reports v5.0.0 is available
	updated, _ := app.Update(core.VersionCheckMsg{
		CurrentVersion:  "v4.8.1",
		LatestVersion:   "v5.0.0",
		UpdateAvailable: true,
	})
	app = updated.(*App)

	if !strings.Contains(app.View(), "Update available: v5.0.0") {
		t.Errorf("Expected status bar to show update available pill")
	}

	// Update task completes with new version v5.0.0
	updateDetail := "✨ Successfully updated git-user!\n\n   v4.8.1 ──▶ v5.0.0 (verified)\n\n  Run 'git-user' to launch the interactive dashboard."
	updated, cmd := app.Update(core.TaskResultMsg{
		Kind:       "update",
		Success:    true,
		Detail:     updateDetail,
		ShowReport: true,
	})
	app = updated.(*App)

	// Version must update immediately in memory
	if version.GetVersion() != "v5.0.0" {
		t.Errorf("Expected version to update immediately to v5.0.0, got %q", version.GetVersion())
	}

	// Dispatch commands from task result (pushing Report screen)
	if cmd != nil {
		var walk func(c tea.Cmd)
		walk = func(c tea.Cmd) {
			if c == nil {
				return
			}
			m := c()
			if m == nil {
				return
			}
			if b, ok := m.(tea.BatchMsg); ok {
				for _, inner := range b {
					walk(inner)
				}
			} else if push, ok := m.(core.ScreenPushMsg); ok {
				up, _ := app.Update(push)
				app = up.(*App)
			}
		}
		walk(cmd)
	}

	// Active screen is Report
	if len(app.screenStack) != 2 {
		t.Fatalf("Expected report screen pushed, stack length = %d", len(app.screenStack))
	}
	if _, ok := app.activeScreen().(*screens.Report); !ok {
		t.Fatalf("Expected active screen to be *screens.Report, got %T", app.activeScreen())
	}

	// Status bar view on Report screen must show Version v5.0.0 and no warning pill
	reportView := app.View()
	if !strings.Contains(reportView, "Version v5.0.0") {
		t.Errorf("Expected view on report screen to show 'Version v5.0.0', got:\n%s", reportView)
	}
	if strings.Contains(reportView, "Update available") {
		t.Errorf("Status bar should not show update available after successful update")
	}

	// User presses Enter / Esc to pop the report and return to the home screen (Dashboard)
	updated, _ = app.Update(core.ScreenPopMsg{})
	app = updated.(*App)

	if len(app.screenStack) != 1 {
		t.Fatalf("Expected stack length 1 after pop, got %d", len(app.screenStack))
	}
	if _, ok := app.activeScreen().(*screens.Dashboard); !ok {
		t.Fatalf("Expected active screen to be Dashboard, got %T", app.activeScreen())
	}

	// Home screen view must immediately show Version v5.0.0
	homeView := app.View()
	if !strings.Contains(homeView, "Version v5.0.0") {
		t.Errorf("Expected home screen view to show 'Version v5.0.0', got:\n%s", homeView)
	}
	if strings.Contains(homeView, "Version v4.8.1") {
		t.Errorf("Home screen view must not show old Version v4.8.1 after update, got:\n%s", homeView)
	}
	if strings.Contains(homeView, "Update available") {
		t.Errorf("Home screen view must not show update available pill after update")
	}
}

func TestExtractUpdatedVersion(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "unicode arrow with verified",
			input:  "✨ Successfully updated git-user!\n\n   v4.8.1 ──▶ v5.0.0 (verified)\n",
			expect: "v5.0.0",
		},
		{
			name:   "unicode arrow without v",
			input:  "✨ Successfully updated git-user!\n\n   4.8.1 ──▶ 5.0.0 (verified)\n",
			expect: "5.0.0",
		},
		{
			name:   "arrow in update info",
			input:  "Updating git-user: v4.8.1 → v4.9.0 (linux amd64)\nDownloading...",
			expect: "v4.9.0",
		},
		{
			name:   "ascii arrow",
			input:  "v4.8.1 -> v4.9.0\n",
			expect: "v4.9.0",
		},
		{
			name:   "download verified line",
			input:  "Download verified: git-user v5.1.2\n",
			expect: "v5.1.2",
		},
		{
			name:   "already up to date",
			input:  "✨ git-user is already up to date!\n\n   v4.8.1 (latest release)\n",
			expect: "v4.8.1",
		},
		{
			name:   "empty output",
			input:  "",
			expect: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractUpdatedVersion(tc.input)
			if got != tc.expect {
				t.Errorf("extractUpdatedVersion(%q) = %q, want %q", tc.input, got, tc.expect)
			}
		})
	}
}
