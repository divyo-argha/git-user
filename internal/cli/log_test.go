package cli

import (
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestRunSwitch_RecordsAuditLog(t *testing.T) {
	setupTestEnv(t)

	if err := runSwitch([]string{"-c", "dev", "-e", "dev@example.com", "--skip-ssh"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := config.ReadSwitchLog()
	if err != nil {
		t.Fatalf("reading switch log: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 switch-log entry, got %d: %v", len(entries), entries)
	}
	if !strings.Contains(entries[0], "dev") {
		t.Errorf("expected switch-log entry to mention identity %q, got %q", "dev", entries[0])
	}

	if err := runSwitch([]string{"-c", "ops", "-e", "ops@example.com", "--skip-ssh"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entries, err = config.ReadSwitchLog()
	if err != nil {
		t.Fatalf("reading switch log: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 switch-log entries after second switch, got %d: %v", len(entries), entries)
	}
}

func TestRunLog_NoEntries(t *testing.T) {
	setupTestEnv(t)

	if err := runLog([]string{}); err != nil {
		t.Fatalf("unexpected error with no recorded switches: %v", err)
	}
}

func TestRunLog_PlainAndLimit(t *testing.T) {
	setupTestEnv(t)

	for _, name := range []string{"a", "b", "c"} {
		if err := runSwitch([]string{"-c", name, "-e", name + "@example.com", "--skip-ssh"}); err != nil {
			t.Fatalf("switch to %q: %v", name, err)
		}
	}

	if err := runLog([]string{"--plain", "-n", "2"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := config.ReadSwitchLog()
	if err != nil {
		t.Fatalf("reading switch log: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 recorded entries regardless of display --limit, got %d", len(entries))
	}
}
