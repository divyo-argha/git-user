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
	sessionName := os.Getenv("GIT_USER_SESSION")
	authorName := os.Getenv("GIT_AUTHOR_NAME")
	authorEmail := os.Getenv("GIT_AUTHOR_EMAIL")
	isSessionOverride := sessionName != "" || authorName != "" || authorEmail != ""
	isLocalOverride := !isSessionOverride && git.IsInRepo() && git.HasLocalOverride()

	if isSessionOverride {
		if sessionName != "" {
			u = store.FindUser(sessionName)
		}
		if u == nil && authorName != "" {
			for i := range store.Users {
				if store.Users[i].Name == authorName || (store.Users[i].Email == authorEmail && authorEmail != "") {
					u = &store.Users[i]
					break
				}
			}
		}
		if u == nil {
			displayName := sessionName
			if displayName == "" {
				displayName = authorName
			}
			u = &config.User{
				Name:  displayName,
				Email: authorEmail,
			}
		}
	} else if isLocalOverride {
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
			Name    string `json:"name"`
			Email   string `json:"email"`
			Session bool   `json:"session_override,omitempty"`
			Local   bool   `json:"local_override,omitempty"`
			Temp    bool   `json:"temp,omitempty"`
			HasKey  bool   `json:"has_ssh_key"`
		}{
			Name:    u.Name,
			Email:   u.Email,
			Session: isSessionOverride,
			Local:   isLocalOverride,
			Temp:    u.IsTemporary,
			HasKey:  u.SSHKey != "",
		})
		return nil
	}

	if ui.IsPlainOutput(args) {
		fmt.Printf("%s <%s>\n", u.Name, u.Email)
		return nil
	}

	if isSessionOverride {
		ui.Banner("Active Identity (Terminal Session Override)")
	} else if isLocalOverride {
		ui.Banner("Active Identity (Local Repo Override)")
	} else {
		ui.Banner("Active Identity")
	}

	ui.UserRow(u.Name, u.Email, u.SSHKey, true)

	if !isSessionOverride && !isLocalOverride {
		if !git.IsIdentityInSync(u.Name, u.Email) {
			ui.Divider()
			ui.Warn("Git config is out of sync with active identity")
			ui.Info(fmt.Sprintf("Run 'git-user switch %s' to re-apply", u.Name))
		}
	}

	return nil
}
