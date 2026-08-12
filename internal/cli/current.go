package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runCurrent(args []string) error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	var u *config.User
	isLocalOverride := git.IsInRepo() && git.HasLocalOverride()

	if isLocalOverride {
		gitName := git.CurrentName()
		gitEmail := git.CurrentEmail()
		for i := range store.Users {
			if store.Users[i].Name == gitName || (store.Users[i].Email == gitEmail && gitEmail != "") {
				u = &store.Users[i]
				break
			}
		}
		if u == nil && gitName != "" {
			u = &config.User{
				Name:  gitName,
				Email: gitEmail,
			}
		}
	} else {
		u = store.CurrentUser()
	}

	if u == nil {
		if ui.IsPlainOutput(args) || ui.IsJSONOutput(args) {
			return nil
		}
		ui.Warn("No active identity set.")
		ui.Info("Run 'git-user switch <name>' to activate one.")
		return nil
	}

	if ui.IsJSONOutput(args) {
		enc := json.NewEncoder(os.Stdout)
		_ = enc.Encode(struct {
			Name   string `json:"name"`
			Email  string `json:"email"`
			Local  bool   `json:"local_override,omitempty"`
			Temp   bool   `json:"temp,omitempty"`
			HasKey bool   `json:"has_ssh_key"`
		}{
			Name:   u.Name,
			Email:  u.Email,
			Local:  isLocalOverride,
			Temp:   u.IsTemporary,
			HasKey: u.SSHKey != "",
		})
		return nil
	}

	if ui.IsPlainOutput(args) {
		fmt.Printf("%s <%s>\n", u.Name, u.Email)
		return nil
	}

	if isLocalOverride {
		ui.Banner("Active Identity (Local Repo Override)")
	} else {
		ui.Banner("Active Identity")
	}

	ui.UserRow(u.Name, u.Email, u.SSHKey, true)

	if !isLocalOverride {
		if !git.IsIdentityInSync(u.Name, u.Email) {
			ui.Divider()
			ui.Warn("Git config is out of sync with active identity")
			ui.Info(fmt.Sprintf("Run 'git-user switch %s' to re-apply", u.Name))
		}
	}

	return nil
}
