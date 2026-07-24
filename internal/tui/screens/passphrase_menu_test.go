package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestPassphraseMenu(t *testing.T) {
	t.Setenv("GIT_USER_CONFIG", t.TempDir()+"/config.json")
	th := theme.DefaultTheme()
	store := &config.Store{
		Current: "work",
		Users:   []config.User{{Name: "work", Email: "work@company.com", SSHKey: "/path/to/key"}},
	}

	pm := NewPassphraseMenu(store, "work", th)

	if pm.Title() != "Passphrase Options: work" {
		t.Errorf("Unexpected title: %s", pm.Title())
	}

	// Test Esc returns pop command
	_, cmd := pm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("Expected command on Esc key")
	}
	msg := cmd()
	if _, ok := msg.(core.ScreenPopMsg); !ok {
		t.Errorf("Expected core.ScreenPopMsg on Esc, got %T", msg)
	}

	// Test mode toggling (passphrase-mode)
	pm.actions.FindAndSetCursorByKey("passphrase-mode")
	_, cmd = pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("Expected toast command on mode toggle")
	}
	u := store.FindUser("work")
	if u.GetPassphraseMode() != "login" {
		t.Errorf("Expected mode to cycle to login, got %s", u.GetPassphraseMode())
	}

	// Test navigation and selecting passphrase-set action
	pm.actions.FindAndSetCursorByKey("passphrase-set")
	_, cmd = pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("Expected command on Enter key")
	}
	msg = cmd()
	if actionMsg, ok := msg.(core.ActionResultMsg); ok {
		if actionMsg.Kind != "passphrase-set" {
			t.Errorf("Expected passphrase-set action, got %s", actionMsg.Kind)
		}
		if actionMsg.Name != "work" {
			t.Errorf("Expected user work, got %s", actionMsg.Name)
		}
	} else {
		t.Errorf("Expected core.ActionResultMsg on Enter, got %T", msg)
	}

	// Test selecting passphrase-remove action on active profile
	pm.actions.FindAndSetCursorByKey("passphrase-remove")
	_, cmd = pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("Expected command on Enter key")
	}
	msg = cmd()
	if actionMsg, ok := msg.(core.ActionResultMsg); ok {
		if actionMsg.Kind != "passphrase-remove" {
			t.Errorf("Expected passphrase-remove action, got %s", actionMsg.Kind)
		}
	} else {
		t.Errorf("Expected core.ActionResultMsg on Enter, got %T", msg)
	}

	// Test selecting passphrase-remove on non-active profile shows toast
	storeInactive := &config.Store{
		Current: "personal",
		Users:   []config.User{{Name: "work", Email: "work@company.com", SSHKey: "/path/to/key"}},
	}
	pmInactive := NewPassphraseMenu(storeInactive, "work", th)
	pmInactive.actions.FindAndSetCursorByKey("passphrase-remove")
	_, cmd = pmInactive.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("Expected toast command on non-active profile passphrase-remove")
	}
	msg = cmd()
	if toastMsg, ok := msg.(core.ToastMsg); ok {
		if toastMsg.Text == "" {
			t.Errorf("Expected non-empty toast error message")
		}
	} else {
		t.Errorf("Expected ToastMsg on non-active profile passphrase-remove, got %T", msg)
	}

	// Test View rendering
	viewStr := pm.View(80, 24)
	if viewStr == "" {
		t.Errorf("View rendered empty string")
	}
}
