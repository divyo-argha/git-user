package git_test

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/git"
)

func TestConvertHTTPSToSSH(t *testing.T) {
	cases := []struct {
		input   string
		want    string
		changed bool
	}{
		{"https://github.com/user/repo.git", "git@github.com:user/repo.git", true},
		{"https://gitlab.com/org/project.git", "git@gitlab.com:org/project.git", true},
		{"https://bitbucket.org/team/repo.git", "git@bitbucket.org:team/repo.git", true},
		// already SSH — should be unchanged
		{"git@github.com:user/repo.git", "git@github.com:user/repo.git", false},
		// no .git suffix
		{"https://github.com/user/repo", "git@github.com:user/repo.git", true},
	}

	for _, c := range cases {
		got, changed := git.ConvertHTTPSToSSH(c.input)
		if changed != c.changed {
			t.Errorf("ConvertHTTPSToSSH(%q): changed=%v, want %v", c.input, changed, c.changed)
		}
		if got != c.want {
			t.Errorf("ConvertHTTPSToSSH(%q): got %q, want %q", c.input, got, c.want)
		}
	}
}
