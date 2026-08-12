package cli

import "testing"

func TestWantsHelp(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"--help"}, true},
		{[]string{"-h"}, true},
		{[]string{"help"}, true},
		{[]string{"eng"}, false},
		{[]string{""}, false},
		{[]string{"-c", "eng", "--help"}, true},
		{[]string{}, false},
	}
	for _, c := range cases {
		if got := wantsHelp(c.args); got != c.want {
			t.Errorf("wantsHelp(%v) = %v, want %v", c.args, got, c.want)
		}
	}
}

func TestCommandUsageHasEntries(t *testing.T) {
	commands := []string{"register", "switch", "list", "remove", "passphrase", "export", "import", "clone", "stats", "config", "sync", "hook", "rename", "bind-key", "audit", "pubkey"}
	for _, c := range commands {
		if commandUsage(c) == "" {
			t.Errorf("commandUsage(%q) is empty", c)
		}
	}
	if commandUsage("nope") != "" {
		t.Error("expected empty usage for unknown command")
	}
}
