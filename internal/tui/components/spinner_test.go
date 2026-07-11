package components

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestSpinner(t *testing.T) {
	th := theme.DefaultTheme()
	s := NewSpinner(th)

	cmd := s.Init()
	if cmd == nil {
		t.Errorf("Expected non-nil cmd from Spinner.Init()")
	}

	view := s.View()
	if view == "" {
		t.Errorf("Expected non-empty string from Spinner.View()")
	}
}
