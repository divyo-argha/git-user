package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestStripNoColorFlag(t *testing.T) {
	tests := []struct {
		in    []string
		out   []string
		found bool
	}{
		{[]string{"list"}, []string{"list"}, false},
		{[]string{"--no-color", "list"}, []string{"list"}, true},
		{[]string{"list", "--no-color", "--json"}, []string{"list", "--json"}, true},
		{[]string{"exec", "--", "tool", "--no-color"}, []string{"exec", "--", "tool", "--no-color"}, false},
		{[]string{"--no-color", "exec", "--", "--no-color"}, []string{"exec", "--", "--no-color"}, true},
		{nil, []string{}, false},
	}
	for _, tt := range tests {
		got, found := StripNoColorFlag(tt.in)
		if !reflect.DeepEqual(got, tt.out) || found != tt.found {
			t.Errorf("StripNoColorFlag(%v) = %v, %v; want %v, %v", tt.in, got, found, tt.out, tt.found)
		}
	}
}

func TestNoColorRequested(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	if NoColorRequested() {
		t.Error("should not request no-color by default")
	}
	t.Setenv("NO_COLOR", "1")
	if !NoColorRequested() {
		t.Error("NO_COLOR=1 should disable colour")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if !NoColorRequested() {
		t.Error("TERM=dumb should disable colour")
	}
}

func TestApplyColorPreference_RendersPlain(t *testing.T) {
	prev := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	lipgloss.SetColorProfile(termenv.TrueColor)
	if out := styleSuccess.Render("ok"); !strings.Contains(out, "\x1b[") {
		t.Fatalf("precondition: expected ANSI codes with TrueColor, got %q", out)
	}

	args := ApplyColorPreference([]string{"--no-color", "list"})
	if !reflect.DeepEqual(args, []string{"list"}) {
		t.Errorf("flag not stripped: %v", args)
	}
	if out := styleSuccess.Render("ok"); strings.Contains(out, "\x1b[") {
		t.Errorf("expected plain text after --no-color, got %q", out)
	}
}
