package tui

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
)

func TestFirstRunOriginalIdentity(t *testing.T) {
	// No users, not prompted, but an original snapshot exists.
	store := &config.Store{
		Original: &config.OriginalConfig{Name: "alice", Email: "alice@example.com"},
	}
	name, email, ok := firstRunOriginalIdentity(store)
	if !ok || name != "alice" || email != "alice@example.com" {
		t.Errorf("expected alice identity offered, got %q %q ok=%v", name, email, ok)
	}

	// Prompted already → no prompt.
	store.ImportPrompted = true
	if _, _, ok := firstRunOriginalIdentity(store); ok {
		t.Error("expected no prompt when already prompted")
	}

	// Users exist → no prompt.
	store = &config.Store{Users: []config.User{{Name: "bob", Email: "b@x.com"}}}
	if _, _, ok := firstRunOriginalIdentity(store); ok {
		t.Error("expected no prompt when identities already exist")
	}

	// Nothing to import → no prompt.
	store = &config.Store{Original: &config.OriginalConfig{}}
	if _, _, ok := firstRunOriginalIdentity(store); ok {
		t.Error("expected no prompt when there is nothing to import")
	}
}

func TestFirstRunSkipSetsPrompted(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{}
	th := theme.DefaultTheme()
	app := NewApp(store, screens.NewDashboard(store, th))
	app = pumpApp(app, tea.WindowSizeMsg{Width: 80, Height: 40})

	updated, cmd := app.Update(core.ActionResultMsg{Kind: "firstrun-skip"})
	app = updated.(*App)
	if cmd == nil {
		t.Fatal("expected a command from firstrun-skip")
	}
	if !app.store.ImportPrompted {
		t.Error("expected ImportPrompted to be set after skip")
	}
}

func TestFirstRunImportPushesForm(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{Current: ""}
	th := theme.DefaultTheme()
	app := NewApp(store, screens.NewDashboard(store, th))
	app = pumpApp(app, tea.WindowSizeMsg{Width: 80, Height: 40})

	_, cmd := app.Update(core.ActionResultMsg{Kind: "firstrun-import"})
	msg := msgFromCmd(t, cmd)
	if _, ok := msg.(core.ScreenPushMsg); !ok {
		t.Fatalf("expected ScreenPushMsg for firstrun-import, got %T", msg)
	}
}

func TestStartOriginalImportNameConflictOffersResolution(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{
		Users: []config.User{{Name: "work", Email: "work@corp.com"}},
	}
	th := theme.DefaultTheme()
	app := NewApp(store, screens.NewDashboard(store, th))
	app = pumpApp(app, tea.WindowSizeMsg{Width: 80, Height: 40})

	// Trying to import the original under the taken name "work".
	_, cmd := app.startOriginalImport("work", "orig@example.com")
	msg := msgFromCmd(t, cmd)
	push, ok := msg.(core.ScreenPushMsg)
	if !ok {
		t.Fatalf("expected ScreenPushMsg, got %T", msg)
	}
	if _, isOptions := push.Screen.(*screens.Options); !isOptions {
		t.Fatalf("expected Options resolution screen, got %T", push.Screen)
	}
}

func TestStartOriginalImportValidProceeds(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{Users: []config.User{{Name: "work", Email: "work@corp.com"}}}
	th := theme.DefaultTheme()
	app := NewApp(store, screens.NewDashboard(store, th))
	app = pumpApp(app, tea.WindowSizeMsg{Width: 80, Height: 40})

	// Unique name → task runs (no resolution screen).
	_, cmd := app.startOriginalImport("original", "orig@example.com")
	msg := msgFromCmd(t, cmd)
	switch msg.(type) {
	case core.TaskResultMsg:
		// expected: task ran and completed (ImportPrompted may be set)
	default:
		t.Fatalf("expected direct task cmd for valid import, got %T", msg)
	}
}

func TestImportRenameConflictThenImport(t *testing.T) {
	withTempConfig(t)
	store := &config.Store{
		Users: []config.User{{Name: "work", Email: "work@corp.com"}},
	}
	th := theme.DefaultTheme()
	app := NewApp(store, screens.NewDashboard(store, th))
	app = pumpApp(app, tea.WindowSizeMsg{Width: 80, Height: 40})

	// Rename the conflicting "work" profile to "work-legacy", then import
	// original under "work".
	updated, cmd := app.Update(core.FormResultMsg{Context: "import-rename-conflict:work|orig@example.com", Values: []string{"work-legacy"}})
	app = updated.(*App)
	if app.store.FindUser("work-legacy") == nil {
		t.Error("expected conflicting profile to be renamed to work-legacy")
	}

	// The resulting cmd should be a task that imports the original as "work".
	msg := msgFromCmd(t, cmd)
	if tr, ok := msg.(core.TaskResultMsg); ok {
		if tr.Err != nil {
			t.Fatalf("import task failed: %v", tr.Err)
		}
		if app.store.FindUser("work") == nil {
			t.Error("expected original identity imported as work")
		}
	}
}

func msgFromCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("cmd is nil")
	}
	return cmd()
}
