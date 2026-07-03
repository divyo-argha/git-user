package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runSwitch(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user switch [-c] <name> [email]")
		return fmt.Errorf("missing arguments")
	}

	// Handle restore to original pre-git-user state
	if args[0] == "--original" {
		return runSwitchOriginal()
	}

	createMode := false
	name := ""
	email := ""
	passphrase := ""
	isTemp := false

	if args[0] == "-c" {
		createMode = true
		var otherArgs []string
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--passphrase", "-p":
				if i+1 < len(args) {
					passphrase = args[i+1]
					i++
				}
			case "--temp", "-t":
				isTemp = true
			default:
				otherArgs = append(otherArgs, args[i])
			}
		}
		if len(otherArgs) < 1 {
			ui.Error("usage: git-user switch -c <name> [email] [--passphrase <passphrase>]")
			return fmt.Errorf("missing name")
		}
		name = otherArgs[0]
		if len(otherArgs) > 1 {
			email = otherArgs[1]
		}
	} else {
		name = args[0]
	}

	if !git.IsInstalled() {
		ui.Error("git is not installed or not on PATH")
		return fmt.Errorf("git not found")
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	// Snapshot original gitconfig before first ever switch
	store.SnapshotOriginal(git.CurrentName(), git.CurrentEmail(), git.CurrentSSHCommand(), git.CurrentSigningKey(), git.CurrentSignFormat(), git.CurrentCommitGPGSign())

	// Auto-import original as an identity on first switch if no identities exist yet
	autoImportOriginalIfNeeded(store)
	_ = config.Save(store) 

	if createMode {
		if store.FindUser(name) != nil {
			ui.Errorf("identity %q already exists", name)
			return fmt.Errorf("user exists")
		}

		if err := quickRegister(name, email, passphrase, isTemp, store); err != nil {
			return err
		}

		store, _ = config.Load()
	}

	user := store.FindUser(name)
	if user == nil {
		ui.Errorf("identity %q not found", name)
		ui.Info("Create it with: git-user register")
		ui.Info("Or create and switch: git-user switch -c " + name)
		return fmt.Errorf("user not found")
	}

	// Auto-logout: unload the previous identity's key from ssh-agent
	if store.Current != "" && store.Current != name {
		if prev := store.CurrentUser(); prev != nil {
			if prev.SSHKey != "" && isSSHKeyLoaded(prev.SSHKey) {
				_ = removeSSHKey(prev.SSHKey)
				ui.Info(fmt.Sprintf("Unloaded SSH key for previous identity %q", prev.Name))
			}
			if prev.IsTemporary {
				store.RemoveUser(prev.Name, true)
				ui.Info(fmt.Sprintf("Temporary identity %q deleted.", prev.Name))
			}
		}
	}

	// Passphrase gate
	if user.SSHKey != "" {
		protected, err := isSSHKeyPassphraseProtected(user.SSHKey)
		if err == nil && protected && !isSSHKeyLoaded(user.SSHKey) {
			ui.Info(fmt.Sprintf("Identity %q is protected.", user.Name))
			var passphrase string
			var hasStored bool
			if secret, err := getKeychainPassphrase(user.Name); err == nil && secret != "" {
				if verifyPassphrase(user.SSHKey, secret) {
					passphrase = secret
					hasStored = true
					ui.Info("Retrieved passphrase securely from system keychain.")
				} else {
					ui.Warn("Stored keychain passphrase was incorrect. Stale entry removed.")
					_ = deleteKeychainPassphrase(user.Name)
				}
			}
			if !hasStored {
				var err error
				passphrase, err = readPassphrase("Passphrase: ")
				if err != nil {
					return err
				}
				if !verifyPassphrase(user.SSHKey, passphrase) {
					ui.Error("Incorrect passphrase. Access denied.")
					return fmt.Errorf("incorrect passphrase")
				}
			}

			// Load it into agent
			if ensureSSHAgent() == nil {
				if err := addSSHKeyWithPassphrase(user.SSHKey, passphrase); err != nil {
					ui.Warn(fmt.Sprintf("Could not load key into agent: %v", err))
				} else {
					ui.Success("Key unlocked and loaded into ssh-agent.")
				}
			}
		}
	}

	if err := git.Apply(user.Name, user.Email); err != nil {
		ui.Errorf("applying git config: %v", err)
		return err
	}

	if user.SSHKey != "" {
		if err := git.ConfigureSSH(user.SSHKey); err != nil {
			ui.Warn(fmt.Sprintf("applying SSH config: %v", err))
		}
	} else {
		if err := git.RemoveSSHConfig(); err != nil {
			ui.Warn(fmt.Sprintf("removing SSH config: %v", err))
		}
	}

	if !user.SignDisabled && user.SignKey != "" {
		if err := git.ConfigureSigning(user.SignKey, user.SignFormat); err != nil {
			ui.Warn(fmt.Sprintf("applying signing config: %v", err))
		}
	} else {
		git.RemoveSigningConfig()
	}

	if err := store.SetCurrent(name); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Switched to %q (%s)", user.Name, user.Email))
	if !user.SignDisabled && user.SignKey != "" {
		ui.Success(fmt.Sprintf("Commit Signing: Enabled (%s)", user.SignFormat))
	}

	if user.SSHKey != "" && isSSHKeyLoaded(user.SSHKey) {
		if err := verifySSHConnectionWithKey(user.SSHKey); err != nil {
			ui.Warn("SSH verification failed. The key may not be added to your platform yet.")
			ui.Info(fmt.Sprintf("Test manually with: ssh -i %s -o IdentitiesOnly=yes -T git@github.com", user.SSHKey))
		} else {
			ui.Success("SSH verified: Connection successful!")
		}
	} else if user.SSHKey != "" {
		ui.Info("Skipping SSH verification until the key is loaded")
	}

	if git.IsInRepo() {
		remotes, _ := git.ListRemotes()
		hasHTTPS := false
		for _, remote := range remotes {
			url, err := git.GetRemoteURL(remote)
			if err == nil && strings.HasPrefix(url, "https://") {
				hasHTTPS = true
				break
			}
		}

		if hasHTTPS {
			fmt.Println()
			ui.Warn("This repo uses HTTPS remotes")

			if ui.Confirm("Convert to SSH for passwordless push?", true) {
				_ = runFixRemote(nil)
			}
		}
	}

	return nil
}

func quickRegister(name, email, passphrase string, isTemp bool, store *config.Store) error {
	ui.Banner("QUICK SETUP: " + name)
	fmt.Println()

	var err error

	if email == "" {
		email, err = ui.Prompt("Email address:")
		if err != nil {
			return err
		}
		if email == "" {
			ui.Error("Email is required.")
			return fmt.Errorf("missing email")
		}
	}

	if err := store.AddUser(name, email); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if isTemp {
		u := store.FindUser(name)
		if u != nil {
			u.IsTemporary = true
		}
	}

	fmt.Println()
	ui.Info("SSH Key Setup:")

	idx, err := ui.Select("Choose SSH key setup:", []string{
		"Auto-generate (recommended)",
		"Use existing key",
		"Skip",
	})
	if err != nil {
		idx = 0 // Default to auto-generate
	}

	var sshKeyPath string

	switch idx {
	case 0: // Auto-generate
		path, err := generateAndDisplayKey(name, email, passphrase)
		if err != nil {
			ui.Warn("Key generation failed")
			break
		}
		sshKeyPath = path

	case 1: // Use existing key
		keyPath, err := ui.Prompt("Path to SSH key:")
		if err == nil && keyPath != "" {
			expanded := expandPath(keyPath)
			if _, err := os.Stat(expanded); err == nil {
				sshKeyPath = expanded
				ui.Success("Using existing key")
			} else {
				ui.Warn("Key not found")
			}
		}

	case 2: // Skip
		ui.Info("Skipping SSH setup")
	}

	if sshKeyPath != "" {
		if err := store.BindSSHKey(name, sshKeyPath); err != nil {
			ui.Warn("Could not bind SSH key")
		}
	}

	if err := config.Save(store); err != nil {
		return err
	}

	fmt.Println()
	ui.Success(fmt.Sprintf("Identity created: %s (%s)", name, email))
	if sshKeyPath != "" {
		ui.Success(fmt.Sprintf("SSH key: %s", sshKeyPath))
	}
	fmt.Println()

	return nil
}

func runSwitchOriginal() error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	if store.Original == nil {
		ui.Error("no original identity snapshot found")
		ui.Info("git-user hasn't made a switch yet — your gitconfig is still in its original state")
		return fmt.Errorf("no original snapshot")
	}

	o := store.Original
	if o.Name == "" && o.Email == "" {
		ui.Warn("Original gitconfig had no user.name or user.email set")
	}

	if err := git.Apply(o.Name, o.Email); err != nil {
		ui.Errorf("restoring git config: %v", err)
		return err
	}

	if o.SSHCommand != "" {
		if err := git.SetSSHCommand(o.SSHCommand); err != nil {
			ui.Warn(fmt.Sprintf("could not restore core.sshCommand: %v", err))
		}
	} else {
		git.RemoveSSHConfig()
	}

	if o.SignKey != "" || o.CommitGPGSign != "" {
		if o.SignFormat == "ssh" {
			git.ConfigureSigning(o.SignKey, "ssh")
		} else {
			git.ConfigureSigning(o.SignKey, "gpg")
		}
	} else {
		git.RemoveSigningConfig()
	}

	store.Current = ""
	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success("Restored original identity")
	if o.Name != "" || o.Email != "" {
		ui.Info(fmt.Sprintf("  name:  %s", o.Name))
		ui.Info(fmt.Sprintf("  email: %s", o.Email))
	}
	if o.SSHCommand != "" {
		ui.Info(fmt.Sprintf("  sshCommand: %s", o.SSHCommand))
	}
	fmt.Println()
	ui.Info("To switch back: git-user switch <name>")

	return nil
}

// autoImportOriginalIfNeeded imports the original gitconfig as an identity
// on the very first switch, if it hasn't been imported yet and has valid data.
func autoImportOriginalIfNeeded(store *config.Store) {
	// Skip if already imported
	for _, u := range store.Users {
		if u.Source == "original" {
			return
		}
	}

	o := store.Original
	if o == nil || (o.Name == "" && o.Email == "") {
		return
	}

	importName := o.Name
	if importName == "" {
		importName = "original"
	}

	if store.FindUser(importName) != nil {
		return
	}

	store.Users = append(store.Users, config.User{
		Name:   importName,
		Email:  o.Email,
		SSHKey: extractSSHKeyFromCommand(o.SSHCommand),
		Source: "original",
	})
}
