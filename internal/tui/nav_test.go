package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/screens"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// pumpApp feeds one message into the app, following any commands it returns
// (the same way the bubbletea runtime does), returning the updated app.
func pumpApp(a *App, m tea.Msg) *App {
	u, cmd := a.Update(m)
	a = u.(*App)
	depth := 0
	for cmd != nil && depth < 20 {
		depth++
		msg := cmd()
		if msg == nil {
			break
		}
		// Ignore recurring component ticks (like cursor blink) in test pump
		if fmt.Sprintf("%T", msg) == "cursor.blinkMsg" {
			break
		}
		u2, cmd2 := a.Update(msg)
		a = u2.(*App)
		cmd = cmd2
	}
	return a
}

// buildDetailRootedApp mirrors Run's stack seeding: the dashboard is always the
// root, with an optional detail screen pushed on top.
func buildDetailRootedApp(t *testing.T, startDetail string) *App {
	t.Helper()
	store := &config.Store{
		Current: "work",
		Users: []config.User{
			{Name: "work", Email: "work@corp.com", SSHKey: "/tmp/k"},
			{Name: "home", Email: "home@personal.com"},
		},
	}
	th := theme.DefaultTheme()
	app := NewApp(store, screens.NewDashboard(store, th))
	if startDetail != "" {
		if store.FindUser(startDetail) != nil {
			app.pushScreen(screens.NewDetail(store, startDetail, th))
		}
	}
	app = pumpApp(app, tea.WindowSizeMsg{Width: 120, Height: 40})
	return app
}

func TestNavStartDetailEscReturnsToDashboard(t *testing.T) {
	// Regression: launching straight into a detail screen (which happens after
	// switch/rename/bind/etc. in cli/tui.go's relaunch loop) used to leave the
	// screen stack as [Detail] only, so Esc / "← Back" were no-ops.
	app := buildDetailRootedApp(t, "work")
	if len(app.screenStack) != 2 {
		t.Fatalf("expected stack [Dashboard, Detail], got %d", len(app.screenStack))
	}
	if fmt.Sprintf("%T", app.activeScreen()) != "*screens.Detail" {
		t.Fatalf("expected Detail active, got %T", app.activeScreen())
	}

	// Esc pops back to the dashboard.
	app = pumpApp(app, tea.KeyMsg{Type: tea.KeyEsc})
	if len(app.screenStack) != 1 {
		t.Fatalf("expected stack [Dashboard] after Esc, got %d", len(app.screenStack))
	}
	if fmt.Sprintf("%T", app.activeScreen()) != "*screens.Dashboard" {
		t.Fatalf("expected Dashboard active, got %T", app.activeScreen())
	}

	// Esc at the root must not exit the TUI and must not crash.
	app = pumpApp(app, tea.KeyMsg{Type: tea.KeyEsc})
	if len(app.screenStack) != 1 {
		t.Fatalf("expected root Esc to keep stack at 1, got %d", len(app.screenStack))
	}
	if app.Quit() {
		t.Fatal("expected root Esc to NOT quit the app")
	}
	if fmt.Sprintf("%T", app.activeScreen()) != "*screens.Dashboard" {
		t.Fatalf("expected Dashboard still active, got %T", app.activeScreen())
	}
}

func TestNavDashboardEnterDetailEsc(t *testing.T) {
	store := &config.Store{
		Current: "work",
		Users:   []config.User{{Name: "work", Email: "work@corp.com"}},
	}
	th := theme.DefaultTheme()
	app := NewApp(store, screens.NewDashboard(store, th))
	app = pumpApp(app, tea.WindowSizeMsg{Width: 120, Height: 40})

	// Enter identity -> Detail
	app = pumpApp(app, tea.KeyMsg{Type: tea.KeyEnter})
	if fmt.Sprintf("%T", app.activeScreen()) != "*screens.Detail" {
		t.Fatalf("expected Detail, got %T", app.activeScreen())
	}
	if len(app.screenStack) != 2 {
		t.Fatalf("expected stack [Dashboard, Detail], got %d", len(app.screenStack))
	}

	// Esc -> back to Dashboard root.
	app = pumpApp(app, tea.KeyMsg{Type: tea.KeyEsc})
	if fmt.Sprintf("%T", app.activeScreen()) != "*screens.Dashboard" {
		t.Fatalf("expected Dashboard, got %T (stack=%d)", app.activeScreen(), len(app.screenStack))
	}
	if len(app.screenStack) != 1 {
		t.Fatalf("expected stack 1 at root, got %d", len(app.screenStack))
	}

	// Confirm dialog: actions pane -> import-export -> back option + Esc.
	app = pumpApp(app, tea.KeyMsg{Type: tea.KeyTab}) // actions pane
	for i := 0; i < 3; i++ {                         // logout -> doctor -> import-original -> import-export
		app = pumpApp(app, tea.KeyMsg{Type: tea.KeyDown})
	}
	app = pumpApp(app, tea.KeyMsg{Type: tea.KeyEnter})
	if fmt.Sprintf("%T", app.activeScreen()) != "*screens.ImportExport" {
		t.Fatalf("expected ImportExport, got %T (stack=%d)", app.activeScreen(), len(app.screenStack))
	}
	app = pumpApp(app, tea.KeyMsg{Type: tea.KeyEsc})
	if fmt.Sprintf("%T", app.activeScreen()) != "*screens.Dashboard" {
		t.Fatalf("expected Dashboard after esc, got %T (stack=%d)", app.activeScreen(), len(app.screenStack))
	}
}
