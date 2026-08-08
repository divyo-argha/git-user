package cli

import (
	"fmt"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
	"os"
	"os/exec"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runPassphrase(args []string) error {
	var name string
	var remove bool
	var set bool
	var verify bool
	var modeVal string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--remove" || arg == "-r" {
			remove = true
		} else if arg == "--set" || arg == "-s" {
			set = true
		} else if arg == "--verify" || arg == "-v" {
			verify = true
		} else if (arg == "--mode" || arg == "-m") && i+1 < len(args) {
			modeVal = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "--mode=") {
			modeVal = strings.TrimPrefix(arg, "--mode=")
		} else if !strings.HasPrefix(arg, "-") {
			name = arg
		}
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	var user *config.User
	if name != "" {
		user = store.FindUser(name)
		if user == nil {
			ui.Errorf("identity %q not found", name)
			return fmt.Errorf("user not found")
		}
	} else {
		user = store.CurrentUser()
		if user == nil {
			ui.Error("No active identity.")
			ui.Info("Switch first: git-user switch <name>")
			ui.Info("Or specify identity name: git-user passphrase <name>")
			return fmt.Errorf("no active identity")
		}
	}

	if user.SSHKey == "" {
		ui.Warn(fmt.Sprintf("Identity %q has no SSH key bound", user.Name))
		ui.Info(fmt.Sprintf("Run: git-user bind-key %s", user.Name))
		return fmt.Errorf("no ssh key")
	}

	if _, err := os.Stat(user.SSHKey); err != nil {
		ui.Errorf("SSH key file is not accessible: %s", user.SSHKey)
		ui.Info(fmt.Sprintf("Run 'git-user bind-key %s' to attach an existing key, or 'git-user rekey %s' to create a new one.", user.Name, user.Name))
		return err
	}

	if modeVal != "" {
		modeVal = strings.ToLower(modeVal)
		switch modeVal {
		case "keychain", "persistent":
			modeVal = "persistent"
		case "agent", "login":
			modeVal = "login"
		case "ask", "everytime":
			modeVal = "everytime"
		default:
			ui.Errorf("Invalid passphrase behavior mode %q. Options: persistent, login, everytime", modeVal)
			return fmt.Errorf("invalid mode")
		}
		user.PassphraseMode = modeVal
		_ = config.Save(store)
		if modeVal == "login" || modeVal == "everytime" {
			_ = keyring.DeleteKeychainPassphrase(user.Name)
		}
		if modeVal == "everytime" {
			_ = ssh.RemoveSSHKey(user.SSHKey)
		}
		ui.Success(fmt.Sprintf("Passphrase behavior for %q set to: %s", user.Name, modeVal))
		return nil
	}

	protected, err := isSSHKeyPassphraseProtected(user.SSHKey)
	if err != nil {
		ui.Errorf("Could not inspect SSH key: %v", err)
		return err
	}

	if verify {
		if !protected {
			ui.Info("This SSH key is not passphrase protected.")
			return nil
		}
		testPass, err := readPassphrase("Passphrase to test: ")
		if err != nil {
			return err
		}
		if ssh.VerifyPassphrase(user.SSHKey, testPass) {
			ui.Success("Passphrase verified successfully!")
			return nil
		}
		ui.Error("Incorrect passphrase.")
		return fmt.Errorf("incorrect passphrase")
	}

	ui.Banner("SSH KEY PASSPHRASE")
	fmt.Println()
	ui.Info(fmt.Sprintf("Identity: %s (%s)", user.Name, user.Email))
	ui.Info(fmt.Sprintf("Key: %s", user.SSHKey))
	fmt.Println()

	if !set && !remove {
		if protected {
			ui.Info(fmt.Sprintf("SSH key for %q is currently passphrase protected.", user.Name))
			idx, err := ui.Select("Select passphrase action:", []string{"🔒 Set / Change Passphrase", "🔓 Remove Passphrase"})
			if err != nil {
				ui.Info("Cancelled")
				return nil
			}
			if idx == 1 {
				remove = true
			} else {
				set = true
			}
			fmt.Println()
		} else {
			set = true
		}
	}

	if remove {
		if store.Current != user.Name {
			ui.Warn(fmt.Sprintf("You must be switched to profile %q (active profile) to remove its passphrase.", user.Name))
			ui.Info(fmt.Sprintf("Run 'git-user switch %s' first.", user.Name))
			return fmt.Errorf("must be active identity to remove passphrase")
		}

		if !protected {
			ui.Warn("This key is not passphrase protected. Nothing to remove.")
			return nil
		}
		ui.Info("Enter current passphrase to confirm passphrase removal.")
		oldPassphrase, err := readPassphrase("Current passphrase: ")
		if err != nil {
			return err
		}
		if !ssh.VerifyPassphrase(user.SSHKey, oldPassphrase) {
			ui.Error("Incorrect passphrase. Access denied.")
			return fmt.Errorf("incorrect passphrase")
		}

		if err := changeSSHKeyPassphrase(user.SSHKey, oldPassphrase, ""); err != nil {
			ui.Error("Could not remove passphrase.")
			return err
		}
		ui.Success(fmt.Sprintf("Passphrase security removed for %q.", user.Name))
		_ = keyring.DeleteKeychainPassphrase(user.Name)
		return nil
	}

	oldPassphrase := ""
	if protected {
		ui.Info("This key already has a passphrase. Enter the current passphrase to change it.")
		var err error
		oldPassphrase, err = readPassphrase("Current passphrase: ")
		if err != nil {
			return err
		}
		if !ssh.VerifyPassphrase(user.SSHKey, oldPassphrase) {
			ui.Error("Incorrect passphrase. Access denied.")
			return fmt.Errorf("incorrect passphrase")
		}
	} else {
		ui.Warn("This key is currently not passphrase protected.")
	}

	newPassphrase, err := promptRequiredPassphrase("New passphrase: ", "Confirm new passphrase: ")
	if err != nil {
		return err
	}

	if err := changeSSHKeyPassphrase(user.SSHKey, oldPassphrase, newPassphrase); err != nil {
		if protected {
			ui.Error("Could not change passphrase.")
			ui.Info("The current passphrase may be wrong, or the key may be inaccessible.")
			ui.Info("git-user cannot recover a forgotten SSH key passphrase.")
		} else {
			ui.Error("Could not add passphrase.")
			ui.Info("Check that the key file is readable and writable by your user.")
		}
		return err
	}

	if protected {
		ui.Success(fmt.Sprintf("Passphrase changed for %q", user.Name))
	} else {
		ui.Success(fmt.Sprintf("Passphrase added for %q", user.Name))
	}
	promptAndStoreKeychain(user.Name, user.SSHKey, newPassphrase)
	ui.Info("Run 'ssh-add " + user.SSHKey + "' to load the key into your SSH agent for this session.")
	ui.Info("To change how the key is unlocked, use: git-user passphrase --mode <persistent|login|everytime>")

	return nil
}

func promptRequiredPassphrase(prompt, confirmPrompt string) (string, error) {
	passphrase, err := readPassphrase(prompt)
	if err != nil {
		return "", err
	}
	if passphrase == "" {
		ui.Error("Passphrase must not be empty.")
		return "", fmt.Errorf("empty passphrase")
	}

	confirm, err := readPassphrase(confirmPrompt)
	if err != nil {
		return "", err
	}
	if passphrase != confirm {
		ui.Error("Passphrases do not match.")
		return "", fmt.Errorf("passphrase mismatch")
	}

	return passphrase, nil
}

func changeSSHKeyPassphrase(keyPath, oldPassphrase, newPassphrase string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	cmd := exec.Command("ssh-keygen", "-p", "-f", keyPath)
	env := os.Environ()

	env = append(env, "GIT_USER_ASKPASS_MODE=true")
	env = append(env, "GIT_USER_OLD_PASSPHRASE="+oldPassphrase)
	env = append(env, "GIT_USER_NEW_PASSPHRASE="+newPassphrase)
	env = append(env, "SSH_ASKPASS="+exe)
	env = append(env, "SSH_ASKPASS_REQUIRE=force")

	hasDisplay := false
	for _, e := range env {
		if strings.HasPrefix(e, "DISPLAY=") {
			hasDisplay = true
			break
		}
	}
	if !hasDisplay {
		env = append(env, "DISPLAY=dummy:0")
	}

	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
