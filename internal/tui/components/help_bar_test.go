package components

import (
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestHelpBar(t *testing.T) {
	th := theme.DefaultTheme()
	hb := NewHelpBar(th)

	// 1. Empty Text
	if hb.View(80) != "" {
		t.Error("Expected empty view for empty help bar")
	}

	// 2. Simple Help Text
	hb.SetText("Press Q to quit")
	viewSimple := hb.View(80)
	if viewSimple == "" {
		t.Error("Expected non-empty view")
	}

	// 3. Help Text with Bullet points (key caps)
	hb.SetText("q•quit  ctrl+c•force quit")
	viewBullets := hb.View(80)
	if !strings.Contains(viewBullets, "quit") || !strings.Contains(viewBullets, "force quit") {
		t.Errorf("Expected view to contain labels, got: %q", viewBullets)
	}
}
