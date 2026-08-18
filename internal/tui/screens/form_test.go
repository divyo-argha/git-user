package screens

import (
	"github.com/divyo-argha/git-user/internal/tui/core"
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

	// Enter on 1 returns core.FormResultMsg
	_, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("Expected cmd on Enter")
	}
	res := cmd()
	if formRes, ok := res.(core.FormResultMsg); ok {
		if formRes.Context != "ctx" {
			t.Errorf("Expected context ctx, got %s", formRes.Context)
		}
		if len(formRes.Values) != 2 {
			t.Errorf("Expected 2 values, got %d", len(formRes.Values))
		}
	} else {
		t.Errorf("Expected core.FormResultMsg")
	}

	// Esc returns core.ScreenPopMsg
	_, cmd = form.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("Expected cmd on Esc")
	}
	res = cmd()
	if _, ok := res.(core.ScreenPopMsg); !ok {
		t.Errorf("Expected core.ScreenPopMsg on Esc, got %T", res)
	}
}

func TestFormSkippable(t *testing.T) {
	th := theme.DefaultTheme()

	form := NewForm("Title", "Desc", "ctx", []FormInput{
		{Label: "First"},
		{Label: "Second"},
	}, th).Skippable()

	// Type something into the focused first field so a skip can be verified
	// to discard it rather than submit it.
	form.inputs[0].SetValue("typed")

	_, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("Expected cmd on Esc")
	}
	res := cmd()
	formRes, ok := res.(core.FormResultMsg)
	if !ok {
		t.Fatalf("Expected core.FormResultMsg on Esc for a skippable form, got %T", res)
	}
	if formRes.Context != "ctx" {
		t.Errorf("Expected context ctx, got %s", formRes.Context)
	}
	if len(formRes.Values) != 2 {
		t.Fatalf("Expected 2 values, got %d", len(formRes.Values))
	}
	for i, v := range formRes.Values {
		if v != "" {
			t.Errorf("Expected value %d to be empty on skip, got %q", i, v)
		}
	}
}
