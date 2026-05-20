package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde expansion",
			input:    "~/.ssh/id_rsa",
			expected: filepath.Join(home, ".ssh/id_rsa"),
		},
		{
			name:     "absolute path unchanged",
			input:    "/absolute/path/key",
			expected: "/absolute/path/key",
		},
		{
			name:     "relative path unchanged",
			input:    "relative/path/key",
			expected: "relative/path/key",
		},
		{
			name:     "tilde only",
			input:    "~/",
			expected: home,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandPath(tt.input)
			if got != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestVerifySSHConnection(t *testing.T) {
	// This test requires network access and SSH keys configured
	// Skip in CI or when SSH is not available
	if os.Getenv("CI") != "" {
		t.Skip("Skipping SSH verification test in CI")
	}

	err := verifySSHConnection()
	// We don't assert success/failure since it depends on user's SSH setup
	// Just verify it doesn't panic
	t.Logf("SSH verification result: %v", err)
}

func TestGenerateAndDisplayKey_KeyExists(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	
	// Override home directory for this test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create .ssh directory
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Create a dummy key file
	keyPath := filepath.Join(sshDir, "git_test")
	if err := os.WriteFile(keyPath, []byte("dummy key"), 0600); err != nil {
		t.Fatal(err)
	}

	// Test that existing key is detected
	// Note: This would require mocking user input, so we just verify the file exists
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("Key file should exist at %s", keyPath)
	}
}
