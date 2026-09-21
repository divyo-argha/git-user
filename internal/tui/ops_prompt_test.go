package tui

import (
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/promptops"
	"github.com/divyo-argha/git-user/internal/testutil"
)

func TestOpPromptStatus(t *testing.T) {
	store := &config.Store{
		Current: "alice",
		Users: []config.User{
			{Name: "alice", Email: "alice@example.com"},
		},
	}

	res, err := opPromptStatus(store)
	if err != nil {
		t.Fatalf("opPromptStatus failed: %v", err)
	}
	if !res.showReport {
		t.Errorf("expected showReport to be true")
	}
	if !strings.Contains(res.detail, "TERMINAL PROMPT INDICATOR STATUS & PREVIEW") {
		t.Errorf("expected report header in detail, got: %s", res.detail)
	}
	if !strings.Contains(res.detail, "SHELL INTEGRATIONS:") {
		t.Errorf("expected shell integrations section in detail")
	}
}

func TestOpInstallAndUninstallPrompt(t *testing.T) {
	tmpHome := t.TempDir()
	testutil.SetHomeDir(t, tmpHome)

	// Install
	res, err := opInstallPrompt(promptops.TargetFish)
	if err != nil {
		t.Fatalf("opInstallPrompt failed: %v", err)
	}
	if !res.showReport {
		t.Errorf("expected showReport to be true for install report")
	}

	// Uninstall
	unres, err := opUninstallPrompt(promptops.TargetFish)
	if err != nil {
		t.Fatalf("opUninstallPrompt failed: %v", err)
	}
	if !unres.showReport {
		t.Errorf("expected showReport to be true for uninstall report")
	}
}

func TestOpPromptPreferences(t *testing.T) {
	tmpHome := t.TempDir()
	testutil.SetHomeDir(t, tmpHome)

	store := &config.Store{
		Current: "alice",
		Users: []config.User{
			{Name: "alice", Email: "alice@example.com"},
		},
	}

	// Set custom icon
	res, err := opSetPromptIcon(store, "🚀 ")
	if err != nil {
		t.Fatalf("opSetPromptIcon failed: %v", err)
	}
	if !strings.Contains(res.detail, "🚀") {
		t.Errorf("expected confirmation of icon in detail")
	}
	if store.Prompt == nil || store.Prompt.Icon != "🚀 " {
		t.Errorf("expected store.Prompt.Icon to be '🚀 ', got %+v", store.Prompt)
	}

	// Toggle always
	resAlways, err := opTogglePromptAlways(store)
	if err != nil {
		t.Fatalf("opTogglePromptAlways failed: %v", err)
	}
	if !strings.Contains(resAlways.detail, "ON") {
		t.Errorf("expected ON in toggle result: %s", resAlways.detail)
	}
	if !store.Prompt.Always {
		t.Errorf("expected store.Prompt.Always to be true")
	}

	// Toggle back
	resAlways2, err := opTogglePromptAlways(store)
	if err != nil {
		t.Fatalf("opTogglePromptAlways second call failed: %v", err)
	}
	if !strings.Contains(resAlways2.detail, "OFF") {
		t.Errorf("expected OFF in toggle result: %s", resAlways2.detail)
	}
	if store.Prompt.Always {
		t.Errorf("expected store.Prompt.Always to be false")
	}

	// Toggle plain
	resPlain, err := opTogglePromptPlain(store)
	if err != nil {
		t.Fatalf("opTogglePromptPlain failed: %v", err)
	}
	if !strings.Contains(resPlain.detail, "ON") {
		t.Errorf("expected ON in plain toggle result: %s", resPlain.detail)
	}
	if !store.Prompt.Plain {
		t.Errorf("expected store.Prompt.Plain to be true")
	}

	resPlain2, err := opTogglePromptPlain(store)
	if err != nil {
		t.Fatalf("opTogglePromptPlain second call failed: %v", err)
	}
	if !strings.Contains(resPlain2.detail, "OFF") {
		t.Errorf("expected OFF in plain toggle result: %s", resPlain2.detail)
	}
	if store.Prompt.Plain {
		t.Errorf("expected store.Prompt.Plain to be false")
	}

	// Set icon none and verify opPromptStatus output
	_, _ = opSetPromptIcon(store, "none")
	statusRes, err := opPromptStatus(store)
	if err != nil {
		t.Fatalf("opPromptStatus failed: %v", err)
	}
	if !strings.Contains(statusRes.detail, "(none / disabled)") {
		t.Errorf("expected '(none / disabled)' in status report, got:\n%s", statusRes.detail)
	}
}
