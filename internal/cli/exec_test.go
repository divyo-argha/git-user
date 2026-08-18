package cli

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestRunExec_Valid(t *testing.T) {
	setupTestEnv(t)

	store := &config.Store{
		Current: "dev",
		Users: []config.User{
			{Name: "dev", Email: "dev@example.com"},
		},
	}
	if err := config.Save(store); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	err := runExec([]string{"dev", "--", "echo", "hello"})
	if err != nil {
		t.Fatalf("runExec returned error: %v", err)
	}
}

func TestRunExec_NotFound(t *testing.T) {
	setupTestEnv(t)

	store := &config.Store{}
	_ = config.Save(store)

	err := runExec([]string{"nonexistent", "--", "echo", "hello"})
	if err == nil {
		t.Fatal("expected error for nonexistent identity")
	}
}
