package cmd

import (
	"fmt"
	"os"

	"github.com/divyo-argha/git-user/internal/bundle"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
	"golang.org/x/term"
)

func runExport(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user export <output-file>")
		return fmt.Errorf("missing output file")
	}
	outPath := expandPath(args[0])

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}
	if len(store.Users) == 0 {
		ui.Warn("No identities to export.")
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
	for _, u := range store.Users {
		id := bundle.Identity{Name: u.Name, Email: u.Email}
		if u.SSHKey != "" {
			id.PrivateKey, _ = os.ReadFile(u.SSHKey)
			id.PublicKey, _ = os.ReadFile(u.SSHKey + ".pub")
		}
		identities = append(identities, id)
	}

	ui.Info("Encrypting… (this takes a few seconds)")
	encrypted, err := bundle.Encrypt(identities, passphrase)
	if err != nil {
		ui.Errorf("encrypting bundle: %v", err)
		return err
	}

	if err := os.WriteFile(outPath, encrypted, 0600); err != nil {
		ui.Errorf("writing bundle: %v", err)
		return err
	}

	fmt.Println()
	ui.Success(fmt.Sprintf("Exported %d identit%s to %s", len(identities), plural(len(identities)), outPath))
	ui.Info("Transfer this file to your new machine, then run:")
	ui.Info("  git-user import " + outPath)
	return nil
}

func readPassphrase(prompt string) (string, error) {
	fmt.Print(prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		// fallback for non-TTY (e.g. tests)
		var s string
		_, err2 := fmt.Scanln(&s)
		if err2 != nil {
			return "", fmt.Errorf("reading passphrase: %w", err)
		}
		return s, nil
	}
	return string(b), nil
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
