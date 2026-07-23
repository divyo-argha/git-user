package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/divyo-argha/git-user/internal/bundle"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runExport(args []string) error {
	if len(args) == 0 {
		ui.Error("usage: git-user export --all | git-user export <name> [name...]")
		return fmt.Errorf("missing arguments")
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}
	if len(store.Users) == 0 {
		ui.Warn("No identities to export.")
		return nil
	}

	// Resolve which identities to export
	var selected []config.User
	if args[0] == "--all" {
		for _, u := range store.Users {
			if !u.IsTemporary {
				selected = append(selected, u)
			}
		}
	} else {
		for _, name := range args {
			u := store.FindUser(name)
			if u == nil {
				ui.Errorf("identity %q not found", name)
				return fmt.Errorf("user not found")
			}
			if u.IsTemporary {
				ui.Warn(fmt.Sprintf("skipping temporary identity %q", name))
				continue
			}
			selected = append(selected, *u)
		}
	}

	if len(selected) == 0 {
		ui.Warn("No eligible identities to export.")
		return nil
	}

	fmt.Println()
	ui.Warn("This file will contain your PRIVATE SSH keys.")
	ui.Warn("Keep it secure and delete it after importing on the new machine.")
	fmt.Println()

	passphrase, err := readPassphrase("Enter passphrase to encrypt bundle: ")
	if err != nil {
		return err
	}
	confirm, err := readPassphrase("Confirm passphrase: ")
	if err != nil {
		return err
	}
	if passphrase != confirm {
		ui.Error("Passphrases do not match.")
		return fmt.Errorf("passphrase mismatch")
	}
	if passphrase == "" {
		ui.Error("Passphrase must not be empty.")
		return fmt.Errorf("empty passphrase")
	}

	var identities []bundle.Identity
	passphraseSkipped := 0
	for _, u := range selected {
		id := bundle.Identity{Name: u.Name, Email: u.Email}
		if u.SSHKey != "" {
			privKey, err := os.ReadFile(u.SSHKey)
			if err != nil {
				ui.Warn(fmt.Sprintf("Could not read private key for %q: %v. Exporting without SSH key.", u.Name, err))
			} else {
				// Skip passphrase-protected keys — they cannot be safely bundled
				// without knowing the passphrase, which we don't prompt for here.
				protected, err := isSSHKeyPassphraseProtected(u.SSHKey)
				if err == nil && protected {
					ui.Warn(fmt.Sprintf("Skipping SSH key for %q: key is passphrase-protected and cannot be bundled safely.", u.Name))
					passphraseSkipped++
				} else {
					id.PrivateKey = privKey
					// Public key is optional but try to read it
					id.PublicKey, _ = os.ReadFile(u.SSHKey + ".pub")
				}
			}
		}
		identities = append(identities, id)
	}

	if passphraseSkipped > 0 {
		fmt.Println()
		ui.Warn(fmt.Sprintf("%d identit%s will be exported without SSH keys (passphrase-protected).", passphraseSkipped, plural(passphraseSkipped)))
		ui.Info("To include these keys, remove their passphrase first: git-user passphrase <name> --remove")
		fmt.Println()
	}

	ui.Info("Encrypting… (this takes a few seconds)")
	encrypted, err := bundle.Encrypt(identities, passphrase)
	if err != nil {
		ui.Errorf("encrypting bundle: %v", err)
		return err
	}

	home, _ := os.UserHomeDir()
	baseName := fmt.Sprintf("git-user-export-%s", time.Now().Format("2006-01-02"))
	outPath := filepath.Join(home, baseName+".bundle")

	counter := 1
	for {
		if _, err := os.Stat(outPath); os.IsNotExist(err) {
			break
		}
		outPath = filepath.Join(home, fmt.Sprintf("%s-%d.bundle", baseName, counter))
		counter++
	}

	if err := os.WriteFile(outPath, encrypted, 0600); err != nil {
		ui.Errorf("writing bundle: %v", err)
		return err
	}

	fmt.Println()
	ui.Success(fmt.Sprintf("Exported %d identit%s to %s", len(identities), plural(len(identities)), outPath))
	fmt.Println()
	for _, id := range identities {
		ui.Info(fmt.Sprintf("  • %s (%s)", id.Name, id.Email))
	}
	fmt.Println()
	ui.Info("Transfer this file to your new machine, then run:")
	fmt.Printf("  git-user import %s\n", outPath)
	return nil
}
