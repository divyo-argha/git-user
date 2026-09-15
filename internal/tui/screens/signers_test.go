package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestSignersScreenBackPops(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	s := NewSignersScreen(store, th)

	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected a cmd for Esc")
	}
	if _, ok := cmd().(core.ScreenPopMsg); !ok {
		t.Errorf("expected ScreenPopMsg, got %#v", cmd())
	}
}

func TestSignersScreenAddIdentityEmitsActionResult(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	s := NewSignersScreen(store, th)

	if s.repoErr != nil {
		t.Skip("not in a git repository")
	}

	s.actions.FindAndSetCursorByKey("signers-add-identity")
	if sel := s.actions.Selected(); sel == nil || sel.Key != "signers-add-identity" {
		t.Fatalf("expected cursor on 'signers-add-identity', got %#v", sel)
	}
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a cmd for Enter")
	}
	msg, ok := cmd().(core.ActionResultMsg)
	if !ok || msg.Kind != "signers-add-identity" {
		t.Errorf("expected ActionResultMsg{Kind: signers-add-identity}, got %#v", cmd())
	}
}
