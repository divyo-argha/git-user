package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runPubkey(args []string) error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	name := store.Current
	if len(args) > 0 {
		name = args[0]
	}

	if name == "" {
		ui.Error("no active identity — specify one: git-user pubkey <name>")
		return fmt.Errorf("no identity")
	}

	// Security check: only allow viewing active identity's key
	if name != store.Current {
		ui.Error("access denied: you can only view the public key of the active identity")
		ui.Info(fmt.Sprintf("Switch first: git-user switch %s", name))
		return fmt.Errorf("access denied")
	}

	user := store.FindUser(name)
	if user == nil {
		ui.Errorf("identity %q not found", name)
		return fmt.Errorf("user not found")
	}

	if user.SSHKey == "" {
		ui.Errorf("identity %q has no SSH key bound", name)
		ui.Info(fmt.Sprintf("Bind one with: git-user bind %s", name))
		return fmt.Errorf("no ssh key")
	}

	pubKeyPath := user.SSHKey + ".pub"
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		ui.Errorf("public key not found at %s", pubKeyPath)
		return err
	}

	// Show fingerprint (reads only the public key — no passphrase needed)
	fingerprintOut, err := exec.Command("ssh-keygen", "-lf", pubKeyPath).Output()
	if err != nil {
		ui.Errorf("could not read key: %v", err)
		return err
	}

	fmt.Println()
	ui.Divider()
	ui.Banner(fmt.Sprintf("PUBLIC KEY — %s (%s)", user.Name, user.Email))
	fmt.Println()
	fmt.Println(strings.TrimSpace(string(pubKeyBytes)))
	fmt.Println()
	ui.Info(fmt.Sprintf("Fingerprint: %s", strings.TrimSpace(string(fingerprintOut))))
	ui.Divider()
	fmt.Println()
	ui.Info("Add this key to your Git platform(s):")
	fmt.Println("  GitHub:    Settings → SSH and GPG keys → New SSH key")
	fmt.Println("  GitLab:    Preferences → SSH Keys → Add new key")
	fmt.Println("  Bitbucket: Personal settings → SSH keys → Add key")
	fmt.Println()
	ui.Info("The same public key can be added to multiple platforms.")
	ui.Info("The private key stays on your machine and is never shared.")

	return nil
}
