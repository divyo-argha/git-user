package screens

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"github.com/divyo-argha/git-user/internal/validate"
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

func TestFormValidation(t *testing.T) {
	th := theme.DefaultTheme()

	validateFirst := func(s string) error {
		if s == "invalid" {
			return errors.New("invalid input")
		}
		return nil
	}

	form := NewForm("Title", "Desc", "ctx", []FormInput{
		{Label: "First", Value: "invalid", Validate: validateFirst},
		{Label: "Second"},
	}, th)

	// Enter on first input when invalid should block move and set error message
	updated, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	form = updated.(*Form)
	if form.cursor != 0 {
		t.Errorf("Expected focus to remain at 0 on validation error, got %d", form.cursor)
	}
	if cmd != nil {
		t.Errorf("Expected nil cmd when validation fails on enter, got non-nil")
	}
	if form.errMessage == "" {
		t.Errorf("Expected error message to be set when validation fails")
	}

	// Fix input value
	form.inputs[0].SetValue("valid")
	updated, _ = form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	form = updated.(*Form)
	if form.cursor != 1 {
		t.Errorf("Expected focus to move to 1 after fixing input, got %d", form.cursor)
	}
	if form.errMessage != "" {
		t.Errorf("Expected error message to be cleared, got %q", form.errMessage)
	}
}

// TestFormHintRendersLiveOnEveryKeystroke covers FormInput.Hint end to end:
// unlike Validate, a hint must render as non-blocking informational text
// (e.g. a passphrase strength indicator) that updates on every render, not
// just on submit, and must never appear for an empty field.
func TestFormHintRendersLiveOnEveryKeystroke(t *testing.T) {
	f := NewForm("T", "help", "ctx", []FormInput{
		{Label: "New Passphrase:", IsPassword: true, Hint: validate.PassphraseHintLine},
	}, theme.DefaultTheme())

	if out := f.View(80, 24); strings.Contains(out, "weak") || strings.Contains(out, "Weak") || strings.Contains(out, "Strong") || strings.Contains(out, "Fair") {
		t.Errorf("expected no strength hint for an empty field, got:\n%s", out)
	}

	f.inputs[0].SetValue("aaaa")
	if out := f.View(80, 24); !strings.Contains(out, "Very weak") {
		t.Errorf("expected a 'Very weak' hint after typing a weak value, got:\n%s", out)
	}

	f.inputs[0].SetValue("K9$mQ2!vT8&nP5xyz")
	if out := f.View(80, 24); !strings.Contains(out, "Strong") && !strings.Contains(out, "Good") {
		t.Errorf("expected a stronger hint after typing a strong value, got:\n%s", out)
	}
}
