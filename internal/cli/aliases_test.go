package cli

import (
	"os"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

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
		{"delete", "remove"},
		{"--delete", "remove"},

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

		// Terminal Sessions
		{"env", "env"},
		{"--env", "env"},
		{"shell", "shell"},
		{"--shell", "shell"},
		{"exec", "exec"},
		{"--exec", "exec"},
		{"init", "init"},
		{"--init", "init"},

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

// Shorthands dropped for safety/ambiguity must no longer normalize to a
// command, and each must have a hint naming its replacement.
func TestRemovedAliases(t *testing.T) {
	for alias := range removedAliasHints {
		if got := normalizeSubcommand(alias); got != alias {
			t.Errorf("normalizeSubcommand(%q) = %q; a removed alias must pass through unchanged", alias, got)
		}
		if removedAliasHints[alias] == "" {
			t.Errorf("removed alias %q needs a hint", alias)
		}
	}
	for _, alias := range []string{"-d", "del", "--del", "-rm", "run", "--run", "fix", "--fix"} {
		if _, ok := removedAliasHints[alias]; !ok {
			t.Errorf("%q should have a removal hint", alias)
		}
	}
	// The kept spellings still work.
	for in, want := range map[string]string{"rm": "remove", "delete": "remove", "--rm": "remove", "exec": "exec", "repair": "refresh", "--repair": "refresh", "refresh": "refresh", "doctor": "doctor"} {
		if got := normalizeSubcommand(in); got != want {
			t.Errorf("normalizeSubcommand(%q) = %q, want %q", in, got, want)
		}
	}
}

// A removed shorthand must fail as an unknown command and must never act on
// the identity it was given (the point of dropping -d/del was that a typo
// could delete something).
func TestExecute_RemovedAliasesFailSafely(t *testing.T) {
	setupTestEnv(t)
	store, _ := config.Load()
	_ = store.AddUser("keep", "keep@example.com")
	_ = config.Save(store)

	orig := os.Args
	t.Cleanup(func() { os.Args = orig })

	for _, alias := range []string{"-d", "del", "--del", "-rm", "run", "--run", "fix", "--fix"} {
		os.Args = []string{"git-user", alias, "keep"}
		if err := Execute(); err == nil {
			t.Errorf("%q should be an unknown command", alias)
		}
		if s, _ := config.Load(); s.FindUser("keep") == nil {
			t.Fatalf("%q deleted the identity", alias)
		}
	}

	// A kept spelling still works end to end.
	os.Args = []string{"git-user", "rm", "keep", "--yes"}
	if err := Execute(); err != nil {
		t.Fatalf("rm keep --yes: %v", err)
	}
	if s, _ := config.Load(); s.FindUser("keep") != nil {
		t.Error("'rm' should still remove the identity")
	}
}
