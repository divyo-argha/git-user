package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/diagnostics"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestHealthLoadedPopulatesRowsAndCursor(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	h := NewHealth(store, th)

	report := diagnostics.Report{
		Checks: []diagnostics.Check{
			{IsProgress: true, Category: "System", Message: "Checking config file permissions..."},
			{ID: "config-perms", Category: "System", Name: "Config file permissions", Status: diagnostics.StatusPass, Message: "Config file permissions OK (0600)", Scored: true},
			{ID: "signing", Category: "Active Identity", Name: "Commit signing", Status: diagnostics.StatusWarn, Message: "No commit signing key configured for this identity", FixHint: "Run 'git-user sign work --on'", Scored: true},
			{ID: "profile-passphrase", Category: "Profiles & Security Audit", Name: "Passphrase protection", Subject: "work", Status: diagnostics.StatusWarn, Message: `Profile "work" SSH key has no passphrase`},
		},
		Issues:      2,
		ScoreTotal:  2,
		ScorePassed: 1,
	}

	updated, cmd := h.Update(healthLoadedMsg{report: report})
	if cmd != nil {
		t.Fatalf("expected no follow-up cmd from healthLoadedMsg, got one")
	}
	nh := updated.(*Health)

	if nh.loading {
		t.Fatal("expected loading to be false after healthLoadedMsg")
	}
	if len(nh.selectable) != 3 {
		t.Fatalf("expected 3 selectable check rows (progress line excluded), got %d", len(nh.selectable))
	}
	if nh.cursor != 0 {
		t.Fatalf("expected cursor to reset to 0, got %d", nh.cursor)
	}

	// Section and subsection headers should be present alongside the checks.
	var sawSection, sawSubsection bool
	for _, r := range nh.rows {
		if r.kind == healthRowSection {
			sawSection = true
		}
		if r.kind == healthRowSubsection && r.label == "work" {
			sawSubsection = true
		}
	}
	if !sawSection {
		t.Error("expected at least one section header row")
	}
	if !sawSubsection {
		t.Error("expected a subsection row for profile 'work'")
	}
}

func TestHealthNavigationAndExpand(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	h := NewHealth(store, th)

	report := diagnostics.Report{
		Checks: []diagnostics.Check{
			{ID: "a", Category: "System", Status: diagnostics.StatusPass, Message: "first"},
			{ID: "b", Category: "System", Status: diagnostics.StatusWarn, Message: "second", FixHint: "do the thing"},
		},
	}
	updated, _ := h.Update(healthLoadedMsg{report: report})
	h = updated.(*Health)

	if h.cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", h.cursor)
	}

	updated, _ = h.Update(tea.KeyMsg{Type: tea.KeyDown})
	h = updated.(*Health)
	if h.cursor != 1 {
		t.Fatalf("expected cursor 1 after down, got %d", h.cursor)
	}

	rowIdx := h.cursorRowIndex()
	if h.expanded[rowIdx] {
		t.Fatal("expected row not expanded initially")
	}
	updated, _ = h.Update(tea.KeyMsg{Type: tea.KeyEnter})
	h = updated.(*Health)
	if !h.expanded[rowIdx] {
		t.Fatal("expected row expanded after enter")
	}
}

func TestHealthBackPopsScreen(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	h := NewHealth(store, th)
	h.loading = false

	_, cmd := h.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected a cmd for Esc")
	}
	msg := cmd()
	if _, ok := msg.(core.ScreenPopMsg); !ok {
		t.Errorf("expected ScreenPopMsg, got %#v", msg)
	}
}
