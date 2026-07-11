package components

import (
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestStatusBar(t *testing.T) {
	th := theme.DefaultTheme()

	store := &config.Store{
		Current: "dev",
		Users: []config.User{
			{Name: "dev", Email: "dev@example.com"},
		},
	}

	bar := NewStatusBar(store, th)

	view := bar.View(80, 40)
	if !strings.Contains(view, "dev") {
		t.Errorf("StatusBar view does not contain active identity name 'dev'")
	}

	// Test agent status update
	bar.SetAgentStatus(true, 1)
	view = bar.View(80, 40)
	if !strings.Contains(view, "Connected") {
		t.Errorf("StatusBar view does not reflect agent connected status")
	}

	// Test store refresh
	store.Current = "prod"
	bar.SetStore(store)
	view = bar.View(80, 40)
	if !strings.Contains(view, "prod") {
		t.Errorf("StatusBar view does not reflect updated identity name 'prod'")
	}
}
