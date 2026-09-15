package cli

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runFixRemote(args []string) error {
	results, err := git.ConvertRemotesToSSH()
	if err != nil {
		ui.Error(err.Error())
		return err
	}

	converted := 0
	for _, r := range results {
		switch {
		case r.ConvertFailed:
			ui.Warn(fmt.Sprintf("%s: could not convert %s", r.Remote, r.OldURL))
		case r.UpdateFailed:
			ui.Warn(fmt.Sprintf("%s: failed to update", r.Remote))
		case r.Converted:
			ui.Success(fmt.Sprintf("%s: %s → %s", r.Remote, r.OldURL, r.NewURL))
			converted++
		}
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
