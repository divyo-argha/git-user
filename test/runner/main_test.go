package main

import (
	"testing"
)

func TestSimplifyPackageName_Empty(t *testing.T) {
	got := simplifyPackageName("")
	if got != "" {
		t.Errorf("simplifyPackageName('') = %q, want ''", got)
	}
}

func TestSimplifyPackageName_Root(t *testing.T) {
	got := simplifyPackageName("github.com/divyo-argha/git-user")
	if got != "." {
		t.Errorf("simplifyPackageName(root) = %q, want '.'", got)
	}
}

func TestSimplifyPackageName_Subpackage(t *testing.T) {
	tests := []struct {
		full string
		want string
	}{
		{"github.com/divyo-argha/git-user/internal/cli", "internal/cli"},
		{"github.com/divyo-argha/git-user/internal/config", "internal/config"},
		{"github.com/divyo-argha/git-user/internal/tui/core", "internal/tui/core"},
		{"github.com/divyo-argha/git-user/cmd/git-user", "cmd/git-user"},
		{"github.com/divyo-argha/git-user/logo", "logo"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := simplifyPackageName(tt.full)
			if got != tt.want {
				t.Errorf("simplifyPackageName(%q) = %q, want %q", tt.full, got, tt.want)
			}
		})
	}
}

func TestSimplifyPackageName_Unrelated(t *testing.T) {
	got := simplifyPackageName("some/other/package")
	if got != "some/other/package" {
		t.Errorf("simplifyPackageName(foreign) = %q, want unchanged", got)
	}
}

func TestPackageStat_Defaults(t *testing.T) {
	ps := PackageStat{}
	if ps.Name != "" {
		t.Errorf("PackageStat.Name should be empty, got %q", ps.Name)
	}
	if ps.Status != "" {
		t.Errorf("PackageStat.Status should be empty, got %q", ps.Status)
	}
}
