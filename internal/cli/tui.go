package cli

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/tui"
	"github.com/divyo-argha/git-user/internal/ui"
)

// runTui starts the interactive Bubble Tea UI at the dashboard.
func runTui() error {
	return launchTUI("")
}

// runTuiForIdentity starts the interactive Bubble Tea UI focusing on a specific identity.
func runTuiForIdentity(name string) error {
	return launchTUI(name)
}

func launchTUI(startDetail string) error {
	for {
		store, err := config.Load()
		if err != nil {
			return err
		}

		kind, name, arg, err := tui.Run(store, startDetail)
		if err != nil {
			return err
		}

		if kind == "" || kind == "quit" {
			return nil
		}

		// Execute action outside TUI (needs terminal I/O)
		fmt.Println()
		executeAction(kind, name, arg, store)
		fmt.Println()

		// After remove, go back to main
		if kind == "remove" {
			startDetail = ""
		} else if name != "" {
			startDetail = name
		} else {
			startDetail = ""
		}

		// Prompt to return
		fmt.Print(tuiDim.Render("  Press Enter to return to menu..."))
		fmt.Scanln()
	}
}

func executeAction(kind string, name string, arg string, store *config.Store) {
	switch kind {
	case "register":
		if name != "" && arg != "" {
			runRegister([]string{name, arg})
		} else {
			runRegister(nil)
		}

	case "switch":
		runSwitch([]string{name})

	case "rename":
		if arg != "" {
			newName := arg
			if store.FindUser(newName) != nil {
				ui.Errorf("Identity %q already exists", newName)
				return
			}
			u := store.FindUser(name)
			if u == nil {
				return
			}
			u.Name = newName
			if store.Current == name {
				store.Current = newName
			}
			config.Save(store)
			ui.Success(fmt.Sprintf("Renamed %q → %q", name, newName))
		}

	case "email":
		if arg != "" {
			runEdit([]string{name, arg})
		}

	case "pubkey":
		runPubkey(nil)

	case "pubkey-push":
		runPubkeyPush(nil)

	case "bind":
		runBind([]string{name})

	case "check-ssh":
		store, _ := config.Load()
		if u := store.FindUser(name); u != nil && u.SSHKey != "" {
			ui.Banner(fmt.Sprintf("CHECKING SSH CONNECTION: %s", name))
			if err := verifySSHConnectionWithKey(u.SSHKey); err != nil {
				ui.Warn("SSH verification failed. Make sure your public key is added to GitHub/GitLab.")
			} else {
				ui.Success("SSH connection verified successfully!")
			}
		} else {
			ui.Warn(fmt.Sprintf("No SSH key bound to identity %q", name))
			ui.Info("Bind a key first using: git-user bind " + name)
		}

	case "unbind":
		u := store.FindUser(name)
		if u == nil {
			return
		}
		if !ui.Confirm(fmt.Sprintf("Remove SSH key binding from %q? (file not deleted)", name), false) {
			ui.Info("Cancelled")
			return
		}
		u.SSHKey = ""
		config.Save(store)
		if store.Current == name {
			git.RemoveSSHConfig()
		}
		ui.Success("SSH key removed from identity")

	case "rekey":
		runRekey([]string{name})

	case "passphrase":
		runPassphrase([]string{name})

	case "passphrase-remove":
		runPassphrase([]string{name, "--remove"})

	case "bind-path":
		path, err := ui.Prompt("Directory path to bind:")
		if err != nil || path == "" {
			ui.Info("Cancelled")
			return
		}
		runBindPath([]string{name, path})

	case "unbind-path":
		u := store.FindUser(name)
		if u == nil {
			return
		}
		if len(u.BindPaths) == 0 {
			ui.Info("No paths bound to this identity")
			return
		}
		var path string
		if len(u.BindPaths) == 1 {
			path = u.BindPaths[0]
			if !ui.Confirm(fmt.Sprintf("Unbind directory %q?", path), false) {
				ui.Info("Cancelled")
				return
			}
		} else {
			idx, err := ui.Select("Select directory to unbind:", u.BindPaths)
			if err != nil {
				ui.Info("Cancelled")
				return
			}
			path = u.BindPaths[idx]
		}
		runUnbindPath([]string{name, path})

	case "logout":
		runLogout(nil)

	case "export":
		runExport([]string{name})

	case "export-current":
		if name != "" {
			runExport([]string{name})
		} else {
			ui.Error("No active identity to export")
		}

	case "export-all":
		runExport([]string{"--all"})

	case "import":
		path, err := ui.Prompt("Path to bundle file:")
		if err != nil || path == "" {
			ui.Info("Cancelled")
			return
		}
		runImport([]string{path})

	case "import-original":
		runImportOriginal(nil)

	case "remove":
		if !ui.Confirm(fmt.Sprintf("Remove identity %q? This cannot be undone.", name), false) {
			ui.Info("Cancelled")
			return
		}
		runRemove([]string{name})

	case "fix-remote":
		runFixRemote(nil)

	case "security":
		runSecurityCheck(nil)

	case "doctor":
		runDoctor(nil)

	case "update":
		if err := RunUpdate(); err != nil {
			ui.Errorf("Update failed: %v", err)
		}
	}
}

// handleUnknownArg checks if arg is an identity name and opens detail view,
// otherwise returns false so root.go can show "unknown command".
func handleUnknownArg(name string) bool {
	store, err := config.Load()
	if err != nil {
		return false
	}
	// Check exact match first
	if store.FindUser(name) != nil {
		if err := runTuiForIdentity(name); err != nil {
			ui.Errorf("TUI error: %v", err)
		}
		return true
	}
	// Suggest similar names
	var similar []string
	lower := strings.ToLower(name)
	for _, u := range store.Users {
		if strings.Contains(strings.ToLower(u.Name), lower) {
			similar = append(similar, u.Name)
		}
	}
	if len(similar) > 0 {
		ui.Errorf("identity %q not found — did you mean: %s", name, strings.Join(similar, ", "))
		return true
	}
	return false
}
