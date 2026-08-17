package cli

import "testing"

func TestNormalizeSubcommand(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Current / Whoami
		{"current", "current"},
		{"--current", "current"},
		{"-c", "current"},
		{"whoami", "current"},
		{"--whoami", "current"},
		{"active", "current"},
		{"--active", "current"},

		// List
		{"list", "list"},
		{"--list", "list"},
		{"-l", "list"},
		{"ls", "list"},
		{"--ls", "list"},

		// Switch
		{"switch", "switch"},
		{"--switch", "switch"},
		{"-s", "switch"},
		{"sw", "switch"},
		{"--sw", "switch"},
		{"use", "switch"},
		{"--use", "switch"},

		// Register
		{"register", "register"},
		{"--register", "register"},
		{"-r", "register"},
		{"reg", "register"},
		{"--reg", "register"},
		{"add", "register"},
		{"--add", "register"},
		{"-a", "register"},

		// Remove
		{"remove", "remove"},
		{"--remove", "remove"},
		{"rm", "remove"},
		{"--rm", "remove"},
		{"-rm", "remove"},
		{"delete", "remove"},
		{"--delete", "remove"},
		{"-d", "remove"},
		{"del", "remove"},
		{"--del", "remove"},

		// Diagnostics
		{"doctor", "doctor"},
		{"--doctor", "doctor"},
		{"audit", "audit"},
		{"--audit", "audit"},
		{"security", "audit"},
		{"--security", "audit"},

		// System
		{"logout", "logout"},
		{"--logout", "logout"},
		{"signout", "logout"},
		{"--signout", "logout"},
		{"lo", "logout"},
		{"--lo", "logout"},

		// Help & Version
		{"--help", "help"},
		{"-h", "help"},
		{"-?", "help"},
		{"help", "help"},
		{"--version", "version"},
		{"-v", "version"},
		{"-V", "version"},
		{"version", "version"},
		{"--update", "update"},
		{"-u", "update"},
		{"update", "update"},
		{"--upgrade", "update"},

		// Keys
		{"pubkey", "pubkey"},
		{"--pubkey", "pubkey"},
		{"-k", "pubkey"},
		{"key", "pubkey"},
		{"--key", "pubkey"},
	}

	for _, tt := range tests {
		got := normalizeSubcommand(tt.input)
		if got != tt.want {
			t.Errorf("normalizeSubcommand(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
