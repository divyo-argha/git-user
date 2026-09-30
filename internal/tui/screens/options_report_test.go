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

	// Enter also pops.
	_, enterCmd := r.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if enterCmd == nil {
		t.Fatal("expected cmd on enter")
	}
	enterMsg := enterCmd()
	if _, ok := enterMsg.(core.ScreenPopMsg); !ok {
		t.Fatalf("expected ScreenPopMsg on enter, got %#v", enterMsg)
	}
}

func TestReportCopy_WholeReportByDefault(t *testing.T) {
	th := theme.DefaultTheme()
	r := NewReport("Report", "line one\nline two\nline three", th)

	var copied string
	orig := clipboardWriteFn
	clipboardWriteFn = func(text string) error { copied = text; return nil }
	defer func() { clipboardWriteFn = orig }()

	_, cmd := r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if cmd == nil {
		t.Fatal("expected cmd on 'c'")
	}
	cmd()

	want := "line one\nline two\nline three"
	if copied != want {
		t.Errorf("expected whole report copied without WithCopyText, got %q, want %q", copied, want)
	}
}

func TestReportCopy_ScopedToCopyTextWhenSet(t *testing.T) {
	th := theme.DefaultTheme()
	r := NewReport("Public Key", "PUBLIC KEY — dev (dev@example.com)\n\nssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIexample dev@example.com\n\nFingerprint: SHA256:abc", th).
		WithCopyText("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIexample dev@example.com")

	var copied string
	orig := clipboardWriteFn
	clipboardWriteFn = func(text string) error { copied = text; return nil }
	defer func() { clipboardWriteFn = orig }()

	_, cmd := r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if cmd == nil {
		t.Fatal("expected cmd on 'c'")
	}
	cmd()

	want := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIexample dev@example.com"
	if copied != want {
		t.Errorf("expected only the public key copied, got %q, want %q", copied, want)
	}

	if !strings.Contains(r.ShortHelp(), "copy key") {
		t.Errorf("expected ShortHelp to mention copying the key when WithCopyText is set, got %q", r.ShortHelp())
	}
}

func TestReport_WithoutCopyHidesHintAndIgnoresKey(t *testing.T) {
	th := theme.DefaultTheme()

	normal := NewReport("T", "body", th)
	if !strings.Contains(normal.ShortHelp(), "copy") {
		t.Errorf("normal report should advertise copy, got %q", normal.ShortHelp())
	}

	ref := NewReport("Keys", "body", th).WithoutCopy()
	if strings.Contains(ref.ShortHelp(), "copy") {
		t.Errorf("reference screen must not advertise copy, got %q", ref.ShortHelp())
	}
	if !strings.Contains(ref.ShortHelp(), "?") {
		t.Errorf("reference screen should say '?' closes it, got %q", ref.ShortHelp())
	}
	if _, cmd := ref.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}); cmd != nil {
		t.Error("'c' must do nothing on a reference screen")
	}
}
