package screens

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	viewWide := detailActive.View(100, 24)
	if viewWide == "" {
		t.Errorf("Active profile View rendered empty string on wide screen")
	}

	for _, w := range []int{80, 100, 120, 150} {
		for _, h := range []int{24, 30, 40} {
			view := detailActive.View(w, h)
			if view == "" {
				t.Fatalf("View rendered empty string for %dx%d", w, h)
			}
			t.Logf("Terminal %dx%d => View rendered: %dx%d", w, h, lipgloss.Width(view), lipgloss.Height(view))
		}
	}

	viewNarrow := detailActive.View(60, 24)
	if viewNarrow == "" {
		t.Errorf("Active profile View rendered empty string on narrow screen")
	}

	detailMissing := NewDetail(storeActive, "nonexistent", th)
	missingView := detailMissing.View(80, 24)
	if missingView == "" {
		t.Errorf("Expected error view for nonexistent user, got empty string")
	}
}

// "Clone a repo as this identity" lives in the profile menu and dispatches
// with the profile's name, so no identity picker is needed afterwards.
func TestDetail_CloneAsIdentityAction(t *testing.T) {
	store := &config.Store{Users: []config.User{{Name: "eng", Email: "eng@company.com"}}}
	detail := NewDetail(store, "eng", theme.DefaultTheme())

	found := false
	for _, it := range detail.actions.Items() {
		if it.Key == "clone-as" {
			found = true
			if !strings.Contains(it.Label, "Clone a repo as this identity") {
				t.Errorf("unexpected label %q", it.Label)
			}
		}
	}
	if !found {
		t.Fatal("profile menu should offer clone-as")
	}

	detail.actions.FindAndSetCursorByKey("clone-as")
	_, cmd := detail.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on clone-as should produce a command")
	}
	msg, ok := cmd().(core.ActionResultMsg)
	if !ok || msg.Kind != "clone-as" || msg.Name != "eng" {
		t.Errorf("expected ActionResultMsg{clone-as, eng}, got %#v", msg)
	}
}
