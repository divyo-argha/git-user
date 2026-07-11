package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestForm(t *testing.T) {
	th := theme.DefaultTheme()

	form := NewForm("Title", "Desc", "ctx", []FormInput{
		{Label: "First"},
		{Label: "Second"},
	}, th)

	// Focus is on 0
	if form.cursor != 0 {
		t.Errorf("Expected focus at 0")
	}

	// Tab to 1
	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyTab})
	form = updated.(*Form)
	if form.cursor != 1 {
		t.Errorf("Expected focus at 1")
	}

	// Enter on 1 returns FormResultMsg
	_, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("Expected cmd on Enter")
	}
	res := cmd()
	if formRes, ok := res.(FormResultMsg); ok {
		if formRes.Context != "ctx" {
			t.Errorf("Expected context ctx, got %s", formRes.Context)
		}
		if len(formRes.Values) != 2 {
			t.Errorf("Expected 2 values, got %d", len(formRes.Values))
		}
	} else {
		t.Errorf("Expected FormResultMsg")
	}

	// Esc returns ScreenPopMsg
	_, cmd = form.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("Expected cmd on Esc")
	}
	res = cmd()
	if _, ok := res.(ScreenPopMsg); !ok {
		t.Errorf("Expected ScreenPopMsg on Esc, got %T", res)
	}
}
