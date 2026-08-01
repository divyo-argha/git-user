package screens

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestImportExport(t *testing.T) {
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
	maxIdx := len(importExportOptions) - 1
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
	s.cursor = len(importExportOptions) - 1 // "back"
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
