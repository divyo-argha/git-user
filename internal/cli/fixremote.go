package cli

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runFixRemote(args []string) error {
	if !git.IsInstalled() {
		ui.Error("git is not installed")
		return fmt.Errorf("git not found")
	}

	if !git.IsInRepo() {
		ui.Error("not in a git repository")
		return fmt.Errorf("not in repo")
	}

	remotes, err := git.ListRemotes()
	if err != nil || len(remotes) == 0 {
		ui.Error("no remotes found")
		return fmt.Errorf("no remotes")
	}

	converted := 0
	for _, remote := range remotes {
		url, err := git.GetRemoteURL(remote)
		if err != nil {
			continue
		}

		if !strings.HasPrefix(url, "https://") {
			continue
		}

		sshURL, ok := git.ConvertHTTPSToSSH(url)
		if !ok {
			ui.Warn(fmt.Sprintf("%s: could not convert %s", remote, url))
			continue
		}

		if err := git.SetRemoteURL(remote, sshURL); err != nil {
			ui.Warn(fmt.Sprintf("%s: failed to update", remote))
			continue
		}

		ui.Success(fmt.Sprintf("%s: %s → %s", remote, url, sshURL))
		converted++
	}

	if converted == 0 {
		ui.Info("All remotes already use SSH")
	} else {
		fmt.Println()
		ui.Success(fmt.Sprintf("Converted %d remote(s) to SSH", converted))
		ui.Info("Try: git push")
	}

	return nil
}
