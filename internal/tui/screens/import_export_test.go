package screens

import (
	"os/exec"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/testutil"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestImportExport(t *testing.T) {
	testutil.Sandbox(t) // hasUnimportedOriginalIdentity reads global git config
	store := &config.Store{}
	th := theme.DefaultTheme()
	s := NewImportExport(store, th)

	// 1. Initial values
	if s.Init() != nil {
		t.Error("Expected Init() to be nil")
	}
	if s.Title() != "Import / Export" {
		t.Errorf("Expected title 'Import / Export', got %q", s.Title())
	}
	if s.ShortHelp() == "" {
		t.Error("Expected non-empty short help")
	}

	// 2. Navigation Up/Down
	// Start cursor is 0
	if s.cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", s.cursor)
	}

	// Key Down
	_, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if s.cursor != 1 {
		t.Errorf("Expected cursor at 1 after 'j', got %d", s.cursor)
	}

	// Key Up
	_, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if s.cursor != 0 {
		t.Errorf("Expected cursor at 0 after 'k', got %d", s.cursor)
	}

	// Key Down limit check
	for i := 0; i < 10; i++ {
		_, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	maxIdx := len(s.options) - 1
	if s.cursor != maxIdx {
		t.Errorf("Expected cursor to stop at %d, got %d", maxIdx, s.cursor)
	}

	// Key Up limit check
	for i := 0; i < 10; i++ {
		_, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	}
	if s.cursor != 0 {
		t.Errorf("Expected cursor to stop at 0, got %d", s.cursor)
	}

	// 3. Update Exit keys
	_, cmdEsc := s.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmdEsc == nil {
		t.Error("Expected tea.Cmd on Escape key")
	}
	msgEsc := cmdEsc()
	if _, ok := msgEsc.(core.ScreenPopMsg); !ok {
		t.Errorf("Expected ScreenPopMsg, got %#v", msgEsc)
	}

	_, cmdB := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if cmdB == nil {
		t.Error("Expected tea.Cmd on 'b'")
	}

	_, cmdCtrlC := s.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmdCtrlC == nil {
		t.Error("Expected tea.Cmd on Ctrl+C")
	}

	// 4. Update Enter Action
	s.cursor = 0 // "export-current"
	_, cmdEnter := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmdEnter == nil {
		t.Error("Expected tea.Cmd on Enter")
	}
	msgEnter := cmdEnter()
	actionMsg, ok := msgEnter.(core.ActionResultMsg)
	if !ok || actionMsg.Kind != "export-current" {
		t.Errorf("Expected ActionResultMsg with kind 'export-current', got %#v", msgEnter)
	}

	// 5. Update Enter Back Option
	s.cursor = len(s.options) - 1 // "back"
	_, cmdBackEnter := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmdBackEnter == nil {
		t.Error("Expected tea.Cmd on Enter on 'back'")
	}
	msgBackEnter := cmdBackEnter()
	if _, ok := msgBackEnter.(core.ScreenPopMsg); !ok {
		t.Errorf("Expected ScreenPopMsg, got %#v", msgBackEnter)
	}

	// 6. View rendering
	viewStr := s.View(80, 20)
	if !strings.Contains(viewStr, "Import / Export") {
		t.Error("Expected view to render title")
	}
	if !strings.Contains(viewStr, "Export current identity") {
		t.Error("Expected view to render options")
	}
}

func setGlobalGitIdentity(t *testing.T, name, email string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if err := exec.Command("git", "config", "--global", "user.name", name).Run(); err != nil {
		t.Fatalf("git config --global user.name: %v", err)
	}
	if err := exec.Command("git", "config", "--global", "user.email", email).Run(); err != nil {
		t.Fatalf("git config --global user.email: %v", err)
	}
}

// TestHasUnimportedOriginalIdentity_NoGlobalConfig guards against showing
// "import existing git identity" when there's nothing set globally to import
// in the first place (a fresh machine, or one where the user only ever set
// identity via git-user).
func TestHasUnimportedOriginalIdentity_NoGlobalConfig(t *testing.T) {
	testutil.Sandbox(t)
	store := &config.Store{}
	if hasUnimportedOriginalIdentity(store) {
		t.Error("expected no unimported identity when global git config has no user.name/user.email set")
	}
}

// TestHasUnimportedOriginalIdentity_NotYetRegistered is the core case this
// gating exists for: a real global identity that isn't one of git-user's
// profiles yet.
func TestHasUnimportedOriginalIdentity_NotYetRegistered(t *testing.T) {
	testutil.Sandbox(t)
	setGlobalGitIdentity(t, "Dev", "dev@example.com")

	store := &config.Store{}
	if !hasUnimportedOriginalIdentity(store) {
		t.Error("expected an unimported identity when global git config has a name+email not in the store")
	}
}

// TestHasUnimportedOriginalIdentity_AlreadyRegistered guards against the
// literal bug reported: the option must disappear once a profile already
// exists with the exact same name AND email as the global git config —
// there is nothing new left to import.
func TestHasUnimportedOriginalIdentity_AlreadyRegistered(t *testing.T) {
	testutil.Sandbox(t)
	setGlobalGitIdentity(t, "Dev", "dev@example.com")

	store := &config.Store{
		Users: []config.User{{Name: "Dev", Email: "dev@example.com", Source: "original"}},
	}
	if hasUnimportedOriginalIdentity(store) {
		t.Error("expected no unimported identity once a profile already has the same name+email")
	}
}

// TestHasUnimportedOriginalIdentity_DifferentEmailStillShows guards against
// over-matching: a profile that merely shares the global config's *name*
// (e.g. a renamed profile, or a coincidence) but has a different email is
// NOT the same identity — the import option must still show, since the
// global config's actual name+email pair is still unregistered.
func TestHasUnimportedOriginalIdentity_DifferentEmailStillShows(t *testing.T) {
	testutil.Sandbox(t)
	setGlobalGitIdentity(t, "Dev", "dev@example.com")

	store := &config.Store{
		Users: []config.User{{Name: "Dev", Email: "someone-else@example.com", Source: "work"}},
	}
	if !hasUnimportedOriginalIdentity(store) {
		t.Error("expected an unimported identity when only the name matches, not the email")
	}
}

// TestBuildImportExportOptions_HidesImportOriginalWhenAlreadyRegistered is
// the end-to-end guard for the ImportExport sub-screen specifically (the
// second entry point into the same feature, beyond the Dashboard menu).
func TestBuildImportExportOptions_HidesImportOriginalWhenAlreadyRegistered(t *testing.T) {
	testutil.Sandbox(t)
	setGlobalGitIdentity(t, "Dev", "dev@example.com")

	store := &config.Store{
		Users: []config.User{{Name: "Dev", Email: "dev@example.com", Source: "original"}},
	}
	for _, opt := range buildImportExportOptions(store) {
		if opt.key == "import-original" {
			t.Error("expected import-original to be hidden once the identity is already registered")
		}
	}
}
