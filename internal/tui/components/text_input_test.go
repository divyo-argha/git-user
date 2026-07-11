package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestTextInput(t *testing.T) {
	th := theme.DefaultTheme()

	input := NewTextInput(th, "Placeholder", false)

	// Ensure value is initially empty
	if input.Value() != "" {
		t.Errorf("Expected empty string, got %v", input.Value())
	}

	// Test set value
	input.SetValue("hello")
	if input.Value() != "hello" {
		t.Errorf("Expected 'hello', got %v", input.Value())
	}

	// Test update via KeyMsg
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}}
	updated, _ := input.Update(msg)
	
	// 'w' is appended because we aren't fully managing the internal cursor state in this simple test,
	// but the textinput package handles append when focused or just running.
	// Oh wait, it won't append if it's not focused!
	input.Focus()
	updated, _ = input.Update(msg)
	if updated.Value() != "hellow" {
		t.Errorf("Expected 'hellow', got %v", updated.Value())
	}
}
