package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestPolicyScreenBackPops(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	p := NewPolicyScreen(store, th)

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected a cmd for Esc")
	}
	if _, ok := cmd().(core.ScreenPopMsg); !ok {
		t.Errorf("expected ScreenPopMsg, got %#v", cmd())
	}
}

func TestPolicyScreenEditEmitsActionResult(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	p := NewPolicyScreen(store, th)

	if p.repoErr != nil {
		t.Skip("not in a git repository — skipping (this test suite runs inside one, so this should not happen)")
	}

	p.actions.FindAndSetCursorByKey("policy-edit")
	if sel := p.actions.Selected(); sel == nil || sel.Key != "policy-edit" {
		t.Fatalf("expected cursor on 'policy-edit', got %#v", sel)
	}
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a cmd for Enter on policy-edit")
	}
	msg, ok := cmd().(core.ActionResultMsg)
	if !ok || msg.Kind != "policy-edit" {
		t.Errorf("expected ActionResultMsg{Kind: policy-edit}, got %#v", cmd())
	}
}
