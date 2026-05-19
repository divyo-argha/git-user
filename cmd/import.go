package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/divyo-argha/git-user/internal/bundle"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runImport(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user import <bundle-file>")
		return fmt.Errorf("missing bundle file")
	}
	inPath := expandPath(args[0])

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

	imported := 0
	skipped := 0
	for _, id := range identities {
		if store.FindUser(id.Name) != nil {
			ui.Warn(fmt.Sprintf("Skipping %q — identity already exists", id.Name))
			skipped++
			continue
		}

		if err := store.AddUser(id.Name, id.Email); err != nil {
			ui.Errorf("adding %q: %v", id.Name, err)
			continue
		}

		if len(id.PrivateKey) > 0 {
			keyPath := filepath.Join(sshDir, fmt.Sprintf("git_%s", id.Name))
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
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	fmt.Println()
	if imported > 0 {
		ui.Info(fmt.Sprintf("Imported %d identit%s. Run 'git-user switch <name>' to activate one.", imported, plural(imported)))
	}
	if skipped > 0 {
		ui.Info(fmt.Sprintf("%d identit%s skipped (already exist).", skipped, plural(skipped)))
	}
	return nil
}
