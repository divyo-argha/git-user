package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/divyo-argha/git-user/internal/bundle"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runImport(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user import [--force] <bundle-file>")
		return fmt.Errorf("missing arguments")
	}

	var force bool
	var bundleFile string
	for _, a := range args {
		if a == "--force" || a == "-f" {
			force = true
		} else {
			bundleFile = a
		}
	}

	if bundleFile == "" {
		ui.Error("usage: git-user import [--force] <bundle-file>")
		return fmt.Errorf("missing bundle file")
	}
	inPath := expandPath(bundleFile)

	data, err := os.ReadFile(inPath)
	if err != nil {
		ui.Errorf("reading bundle: %v", err)
		return err
	}

	passphrase, err := readPassphrase("Enter passphrase: ")
	if err != nil {
		return err
	}

	ui.Info("Decrypting…")
	identities, err := bundle.Decrypt(data, passphrase)
	if err != nil {
		ui.Error(err.Error())
		return err
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		ui.Errorf("creating .ssh directory: %v", err)
		return err
	}

	// originalCurrent is the identity that was active before this import ran.
	// An --force (or interactive "Overwrite") conflict resolution can remove
	// it out from under the user (config.Store.RemoveUser clears store.Current
	// when the removed identity was active) — removeConflicting tracks that so
	// whatever identity replaces it can be restored as current afterward,
	// instead of leaving the live git config pointing at a now-gone identity.
	originalCurrent := store.Current
	originalCurrentRemoved := false
	removeConflicting := func(uname string) {
		if uname == originalCurrent {
			originalCurrentRemoved = true
		}
		_ = store.RemoveUser(uname, true)
	}

	imported := 0
	skipped := 0
	var firstImportedName string
	restoredCurrent := ""
	for _, id := range identities {
		conflictMsg := ""
		if store.IsNameTaken(id.Name) {
			conflictMsg = fmt.Sprintf("Identity name %q is already taken", id.Name)
		} else if store.IsEmailTaken(id.Email) {
			conflictMsg = fmt.Sprintf("Email %q is already used by another identity", id.Email)
		}

		if conflictMsg != "" {
			if force {
				if store.IsNameTaken(id.Name) {
					removeConflicting(id.Name)
				}
				if store.IsEmailTaken(id.Email) {
					for _, u := range store.Users {
						if u.Email == id.Email {
							removeConflicting(u.Name)
							break
						}
					}
				}
			} else if !ui.IsTTY() {
				ui.Warn(fmt.Sprintf("Skipping %q — conflict (%s) and no --force", id.Name, conflictMsg))
				skipped++
				continue
			} else {
				ui.Warn(fmt.Sprintf("Conflict for %q: %s", id.Name, conflictMsg))
				choice, err := ui.Select("How would you like to proceed?", []string{"Skip", "Overwrite (removes conflicting local identity)", "Rename (import with a different name)"})
				if err != nil || choice == 0 {
					ui.Info(fmt.Sprintf("Skipped %q", id.Name))
					skipped++
					continue
				} else if choice == 1 { // Overwrite
					if store.IsNameTaken(id.Name) {
						removeConflicting(id.Name)
					}
					if store.IsEmailTaken(id.Email) {
						for _, u := range store.Users {
							if u.Email == id.Email {
								removeConflicting(u.Name)
								break
							}
						}
					}
				} else if choice == 2 { // Rename
					newName, err := ui.Prompt(fmt.Sprintf("Enter new name for %q:", id.Name))
					if err != nil || newName == "" {
						ui.Info(fmt.Sprintf("Skipped %q", id.Name))
						skipped++
						continue
					}
					id.Name = newName
					if store.IsNameTaken(id.Name) || store.IsEmailTaken(id.Email) {
						ui.Error(fmt.Sprintf("Still conflicts after rename. Skipping %q.", id.Name))
						skipped++
						continue
					}
				}
			}
		}

		if err := store.AddUser(id.Name, id.Email); err != nil {
			ui.Errorf("adding %q: %v", id.Name, err)
			continue
		}

		if len(id.PrivateKey) > 0 {
			keyPath, err := config.DefaultSSHKeyPath(id.Name)
			if err != nil {
				ui.Errorf("skipping %q: %v", id.Name, err)
				continue
			}
			if err := os.WriteFile(keyPath, id.PrivateKey, 0600); err != nil {
				ui.Errorf("writing private key for %q: %v", id.Name, err)
				continue
			}
			if len(id.PublicKey) > 0 {
				_ = os.WriteFile(keyPath+".pub", id.PublicKey, 0644)
			}
			_ = store.BindSSHKey(id.Name, keyPath)
			ui.Success(fmt.Sprintf("Imported: %s (%s) → %s", id.Name, id.Email, keyPath))
		} else {
			ui.Success(fmt.Sprintf("Imported: %s (%s) — no SSH key", id.Name, id.Email))
		}
		imported++
		if firstImportedName == "" {
			firstImportedName = id.Name
		}
		if originalCurrentRemoved && store.Current == "" {
			restoredCurrent = id.Name
			originalCurrentRemoved = false
		}
	}

	// Reactivate: prefer restoring whichever imported identity replaced the
	// one that was active before this import overwrote it; otherwise, if
	// there was no active identity at all, auto-activate the first freshly
	// imported one instead of leaving it inert. Either way this keeps the
	// live git config (name/email/SSH command/signing) from silently going
	// stale or pointing at an identity that no longer exists.
	activateName := restoredCurrent
	if activateName == "" && originalCurrent == "" && firstImportedName != "" {
		activateName = firstImportedName
	}
	if activateName != "" {
		if u := store.FindUser(activateName); u != nil {
			if err := git.Apply(u.Name, u.Email); err != nil {
				ui.Warn(fmt.Sprintf("Could not activate identity %q: %v", activateName, err))
			} else {
				store.Current = activateName
				ui.Info(fmt.Sprintf("Activated identity %q", activateName))
				if err := applyUserSSHConfig(u, false); err != nil {
					ui.Warn(fmt.Sprintf("Could not apply SSH config for %q: %v", activateName, err))
				}
				if !u.SignDisabled && u.SignKey != "" {
					if err := git.ConfigureSigning(u.SignKey, u.SignFormat); err != nil {
						ui.Warn(fmt.Sprintf("Could not apply signing config for %q: %v", activateName, err))
					}
				} else {
					git.RemoveSigningConfig()
				}
			}
		}
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	fmt.Println()
	if imported > 0 {
		if activateName == "" {
			ui.Info(fmt.Sprintf("Imported %d identit%s. Run 'git-user switch <name>' to activate one.", imported, plural(imported)))
		} else {
			ui.Info(fmt.Sprintf("Imported %d identit%s.", imported, plural(imported)))
		}
	}
	if skipped > 0 {
		ui.Info(fmt.Sprintf("%d identit%s skipped (already exist).", skipped, plural(skipped)))
	}
	return nil
}
