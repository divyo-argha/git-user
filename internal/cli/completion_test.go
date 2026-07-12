package cli

import (
	"testing"
)

func TestRunCompletion(t *testing.T) {
	setupTestEnv(t)

	// Test Case 1: Missing shell argument
	err := runCompletion([]string{})
	if err == nil {
		t.Fatal("expected error with no arguments, got nil")
	}

	// Test Case 2: Unsupported shell
	err = runCompletion([]string{"invalid"})
	if err == nil {
		t.Fatal("expected error with unsupported shell, got nil")
	}

	// Test Case 3: Bash shell
	err = runCompletion([]string{"bash"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test Case 4: Zsh shell
	err = runCompletion([]string{"zsh"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test Case 5: Fish shell
	err = runCompletion([]string{"fish"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
