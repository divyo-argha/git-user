package cli

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/gitenv"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ui"
)

// runToken manages a per-identity HTTPS personal-access-token (or app
// password), stored in the OS keyring the same way SSH key passphrases are
// (internal/keyring). This is for the case `git-user` otherwise steers
// people away from — HTTPS remotes — for hosts/networks where SSH genuinely
// isn't an option (a corporate proxy blocking port 22, some CI runners): the
// token is wired up as core.askpass so those remotes get correct,
// per-identity credentials automatically, without ever writing the token
// itself into git config. See internal/gitenv.AskpassCommand and
// internal/cli/askpass_helper.go for how it's consumed.
func runToken(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		runSubcommandHelp("token")
		return nil
	}

	var name string
	set := false
	remove := false
	var username string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--set", "-s":
			set = true
		case "--remove", "-r":
			remove = true
		case "--username", "-u":
			if i+1 < len(args) {
				username = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") && name == "" {
				name = args[i]
			}
		}
	}

	if name == "" {
		ui.Error("usage: git-user token <name> [--set] [--remove] [--username <user>]")
		return fmt.Errorf("missing identity name")
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := store.FindUser(name)
	if user == nil {
		ui.Errorf("identity %q not found", name)
		return fmt.Errorf("identity not found: %s", name)
	}

	if remove {
		if err := keyring.DeleteHTTPSToken(user.Name); err != nil {
			ui.Errorf("removing token: %v", err)
			return err
		}
		if store.Current == user.Name {
			git.RemoveAskpassConfig()
		}
		ui.Success(fmt.Sprintf("HTTPS token removed for %q.", user.Name))
		return nil
	}

	if set {
		token, err := readPassphrase("Enter Personal Access Token: ")
		if err != nil {
			return err
		}
		if strings.TrimSpace(token) == "" {
			ui.Error("token cannot be empty")
			return fmt.Errorf("empty token")
		}
		if err := keyring.SetHTTPSToken(user.Name, token); err != nil {
			ui.Errorf("storing token: %v", err)
			return err
		}
		if username != "" && username != user.HTTPSUsername {
			if err := store.SetHTTPSUsername(user.Name, username); err != nil {
				ui.Warn(fmt.Sprintf("Could not save username: %v", err))
			} else if err := config.Save(store); err != nil {
				ui.Warn(fmt.Sprintf("Could not save config: %v", err))
			}
		}
		ui.Success(fmt.Sprintf("Token stored securely for %q.", user.Name))
		if store.Current == user.Name {
			if cmd, err := gitenv.AskpassCommand(user.Name); err == nil {
				if err := git.ConfigureAskpass(cmd); err != nil {
					ui.Warn(fmt.Sprintf("Could not apply core.askpass: %v", err))
				} else {
					ui.Info("HTTPS remotes will now authenticate as this identity automatically.")
				}
			}
		} else {
			ui.Info(fmt.Sprintf("It will take effect the next time you switch to %q.", user.Name))
		}
		return nil
	}

	if username != "" {
		if err := store.SetHTTPSUsername(user.Name, username); err != nil {
			ui.Errorf("saving username: %v", err)
			return err
		}
		if err := config.Save(store); err != nil {
			ui.Errorf("saving config: %v", err)
			return err
		}
		ui.Success(fmt.Sprintf("HTTPS username for %q set to %q.", user.Name, username))
		return nil
	}

	// Status.
	if keyring.HasHTTPSToken(user.Name) {
		ui.Success(fmt.Sprintf("%q has an HTTPS token stored (username: %s).", user.Name, user.GetHTTPSUsername()))
	} else {
		ui.Info(fmt.Sprintf("%q has no HTTPS token stored.", user.Name))
		ui.Info(fmt.Sprintf("  Set one with: git-user token %s --set", user.Name))
	}
	return nil
}
