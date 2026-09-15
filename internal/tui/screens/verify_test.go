package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/stats"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestVerifyScreenLoaded(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	v := NewVerifyScreen(store, th)

	updated, _ := v.Update(verifyLoadedMsg{
		rangeUsed: "HEAD~50..HEAD",
		items: []stats.AuthorStat{
			{DisplayName: "Alice", Email: "alice@example.com", Commits: 3, SignedCommits: 3},
			{DisplayName: "Bob", Email: "bob@example.com", Commits: 2, UnsignedCommits: 2},
		},
	})
	nv := updated.(*VerifyScreen)
	if nv.loading {
		t.Fatal("expected loading false after verifyLoadedMsg")
	}
	if nv.rangeUsed != "HEAD~50..HEAD" {
		t.Errorf("expected rangeUsed to be set, got %q", nv.rangeUsed)
	}
	if len(nv.items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(nv.items))
	}
}

func TestVerifyScreenEditRangeEmitsActionResult(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	v := NewVerifyScreen(store, th)
	v.loading = false

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if cmd == nil {
		t.Fatal("expected a cmd for 'e'")
	}
	msg, ok := cmd().(core.ActionResultMsg)
	if !ok || msg.Kind != "verify-set-range" {
		t.Errorf("expected ActionResultMsg{Kind: verify-set-range}, got %#v", cmd())
	}
}

func TestVerifyScreenSetRangeStartsReload(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	v := NewVerifyScreen(store, th)
	v.loading = false

	cmd := v.SetRange("origin/main..HEAD")
	if cmd == nil {
		t.Fatal("expected SetRange to return a load command")
	}
	if v.rangeStr != "origin/main..HEAD" {
		t.Errorf("expected rangeStr set, got %q", v.rangeStr)
	}
	if !v.loading {
		t.Error("expected loading true after SetRange")
	}
}

func TestVerifyScreenBackPops(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	v := NewVerifyScreen(store, th)
	v.loading = false

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected a cmd for Esc")
	}
	if _, ok := cmd().(core.ScreenPopMsg); !ok {
		t.Errorf("expected ScreenPopMsg, got %#v", cmd())
	}
}
