package screens

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestOptionsNavigationAndResult(t *testing.T) {
	th := theme.DefaultTheme()
	o := NewOptions("Pick one", "help", "ctx", []Option{
		{Label: "First", Key: "a"},
		{Label: "Second", Key: "b"},
		{Label: "Cancel", Key: ""},
	}, th)

	// Navigate down and select second option.
	updated, _ := o.Update(tea.KeyMsg{Type: tea.KeyDown})
	o = updated.(*Options)
	updated, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEnter})
	o = updated.(*Options)
	if cmd == nil {
		t.Fatal("expected a cmd on enter")
	}
	msg := cmd()
	res, ok := msg.(core.OptionResultMsg)
	if !ok {
		t.Fatalf("expected OptionResultMsg, got %#v", msg)
	}
	if res.Context != "ctx" || res.Choice != "b" {
		t.Errorf("expected choice b for ctx, got %q for %q", res.Choice, res.Context)
	}

	// Esc cancels.
	updated, cmd = o.Update(tea.KeyMsg{Type: tea.KeyEsc})
	o = updated.(*Options)
	msg = cmd()
	res, ok = msg.(core.OptionResultMsg)
	if !ok {
		t.Fatalf("expected OptionResultMsg on esc, got %#v", msg)
	}
	if res.Choice != "" {
		t.Errorf("expected empty choice on cancel, got %q", res.Choice)
	}

	// Rendering includes the title and options.
	view := o.View(80, 30)
	if !strings.Contains(view, "Pick one") {
		t.Error("expected title in view")
	}
	if !strings.Contains(view, "First") {
		t.Error("expected option label in view")
	}
}

func TestReportRenderAndScroll(t *testing.T) {
	th := theme.DefaultTheme()
	// Generate enough lines to exceed a height-24 viewport (maxLines = 24-8 = 16).
	// Need > 16 lines so maxScrollOffset > 0.
	var lineSlice []string
	for i := 1; i <= 25; i++ {
		lineSlice = append(lineSlice, fmt.Sprintf("line%d", i))
	}
	lines := strings.Join(lineSlice, "\n")
	r := NewReport("My Report", lines, th)

	// Must call View first so r.maxLines is initialised.
	view := r.View(80, 24)
	if !strings.Contains(view, "My Report") {
		t.Error("expected title in view")
	}
	if !strings.Contains(view, "line1") {
		t.Error("expected first line in view")
	}

	// With 25 lines and maxLines=16, maxScrollOffset = 25-16 = 9. Scroll should eng.
	updated, _ := r.Update(tea.KeyMsg{Type: tea.KeyDown})
	r = updated.(*Report)
	if r.offset != 1 {
		t.Errorf("expected offset 1 after scroll, got %d", r.offset)
	}

	// Scroll back up.
	updated, _ = r.Update(tea.KeyMsg{Type: tea.KeyUp})
	r = updated.(*Report)
	if r.offset != 0 {
		t.Errorf("expected offset 0 after scrolling up, got %d", r.offset)
	}

	// Esc pops.
	updated, cmd := r.Update(tea.KeyMsg{Type: tea.KeyEsc})
	_ = updated
	if cmd == nil {
		t.Fatal("expected cmd on esc")
	}
	msg := cmd()
	if _, ok := msg.(core.ScreenPopMsg); !ok {
		t.Fatalf("expected ScreenPopMsg on esc, got %#v", msg)
	}
}
