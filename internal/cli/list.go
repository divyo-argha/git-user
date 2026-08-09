package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runList(args []string) error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if ui.IsJSONOutput(args) {
		type ident struct {
			Name   string `json:"name"`
			Email  string `json:"email"`
			Active bool   `json:"active"`
		}
		out := make([]ident, 0, len(store.Users))
		for _, u := range store.Users {
			out = append(out, ident{
				Name:   u.Name,
				Email:  u.Email,
				Active: u.Name == store.Current,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		_ = enc.Encode(out)
		return nil
	}

	if len(store.Users) == 0 {
		if ui.IsPlainOutput(args) {
			return nil
		}
		ui.Warn("No identities configured yet.")
		ui.Info("Run 'git-user register' to add one.")
		return nil
	}

	if ui.IsPlainOutput(args) {
		for _, u := range store.Users {
			marker := ""
			if u.Name == store.Current {
				marker = " # active"
			}
			fmt.Printf("%s <%s>%s\n", u.Name, u.Email, marker)
		}
		return nil
	}

	ui.Banner("Git Identities")
	for _, u := range store.Users {
		ui.UserRow(u.Name, u.Email, u.SSHKey, u.Name == store.Current)
	}

	if store.Current == "" {
		ui.Warn("No active identity — run 'git-user switch <n>'")
	}
	return nil
}
