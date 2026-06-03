package ui

import (
	"testing"
)

func TestStyleFunctions(t *testing.T) {
	// Test that style functions don't panic
	_ = StyleSuccess()
	_ = StyleDim()
}

func TestIsTTY(t *testing.T) {
	// Just verify it doesn't panic
	_ = isTTY()
}

func TestMessageFunctions(t *testing.T) {
	// These functions write to stdout/stderr
	// We just verify they don't panic
	
	Success("test success")
	Successf("test %s", "success")
	Info("test info")
	Warn("test warning")
	Error("test error")
	Errorf("test %s", "error")
}

func TestFormatFunctions(t *testing.T) {
	// Test formatting functions don't panic
	Header("Test Header")
	Banner("Test Banner")
	Divider()
	UserRow("test", "test@example.com", "/path/to/key", true, false)
	UserRow("test2", "test2@example.com", "", false, true)
	UserDetails("test", "test@example.com", "/path/to/key")
}

func TestRawMode(t *testing.T) {
	// Test that RawMode doesn't panic
	// We can't really test the functionality without a real TTY
	err := RawMode(true)
	if err == nil {
		// If it succeeded, turn it back off
		RawMode(false)
	}
}

func TestPrompt(t *testing.T) {
	// Prompt requires stdin, so we skip actual testing
	// Just verify the function exists and has correct signature
	t.Skip("Prompt requires interactive input")
}

func TestSelect(t *testing.T) {
	// Select requires interactive input, so we skip actual testing
	t.Skip("Select requires interactive input")
}

func TestSelectModelInit(t *testing.T) {
	// Test that SelectModel can be created and initialized
	model := SelectModel{
		label:   "Test",
		options: []string{"option1", "option2", "option3"},
		cursor:  0,
		chosen:  -1,
	}

	// Test Init
	if cmd := model.Init(); cmd != nil {
		t.Error("Init() should return nil")
	}

	// Test View doesn't panic
	view := model.View()
	if view == "" {
		t.Error("View() should return non-empty string")
	}
}
