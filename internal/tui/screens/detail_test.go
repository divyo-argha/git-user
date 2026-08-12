package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestDetail(t *testing.T) {
	th := theme.DefaultTheme()

	store := &config.Store{
		Current: "personal",
		Users:   []config.User{{Name: "eng", Email: "eng@company.com"}},
	}
	detail := NewDetail(store, "eng", th)

	// Test Initial cursor focus on switch for inactive profile
	selected := detail.actions.Selected()
	if selected == nil || selected.Key != "switch" {
		t.Errorf("Expected default cursor focus on switch action for inactive profile")
	}

	// Test Esc returns pop
	_, cmd := detail.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("Expected cmd on Esc")
	}
	msg := cmd()
	if _, ok := msg.(core.ScreenPopMsg); !ok {
		t.Errorf("Expected core.ScreenPopMsg on Esc")
	}

	// Test Enter on focused switch action
	_, cmd = detail.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("Expected cmd on Enter")
	}
	msg = cmd()
	if actionMsg, ok := msg.(core.ActionResultMsg); ok {
		if actionMsg.Kind != "switch" {
			t.Errorf("Expected switch action, got %s", actionMsg.Kind)
		}
	} else {
		t.Errorf("Expected core.ActionResultMsg on Enter, got %T", msg)
	}

	// Test 's' hotkey triggers switch action
	_, cmd = detail.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatalf("Expected cmd on 's' hotkey")
	}
	msg = cmd()
	if actionMsg, ok := msg.(core.ActionResultMsg); ok {
		if actionMsg.Kind != "switch" {
			t.Errorf("Expected switch action on 's' hotkey, got %s", actionMsg.Kind)
		}
	} else {
		t.Errorf("Expected core.ActionResultMsg on 's' hotkey, got %T", msg)
	}

	// Test active profile detail view
	storeActive := &config.Store{
		Current: "eng",
		Users:   []config.User{{Name: "eng", Email: "eng@company.com", SSHKey: "/path/to/key"}},
	}
	detailActive := NewDetail(storeActive, "eng", th)
	viewStr := detailActive.View(80, 24)
	if viewStr == "" {
		t.Errorf("Active profile View rendered empty string")
	}
}
