package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runShell(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		runSubcommandHelp("shell")
		return nil
	}

	targetName := args[0]
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := store.FindUser(targetName)
	if user == nil {
		ui.Errorf("identity %q not found", targetName)
		return fmt.Errorf("identity not found: %s", targetName)
	}

	if err := ensureSSHKeyUnlocked(user); err != nil {
		return err
	}

	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		if runtime.GOOS == "windows" {
			shellPath = "powershell.exe"
		} else {
			shellPath = "/bin/sh"
		}
	}

	vars := EnvVars(user)
	// Overrides must come before the inherited environment — see the same
	// note in exec.go: appending after os.Environ() lets a duplicate key
	// from the parent shell win under first-match getenv() semantics.
	env := make([]string, 0, len(vars)+len(os.Environ()))
	for k, v := range vars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	env = append(env, os.Environ()...)

	ui.Banner(fmt.Sprintf("ISOLATED SHELL: %s", user.Name))
	fmt.Printf("  Identity : %s <%s>\n", user.Name, user.Email)
	if user.SSHKey != "" {
		fmt.Printf("  SSH Key  : %s\n", user.SSHKey)
	}
	fmt.Println()
	ui.Info("All Git commands in this subshell will use this identity.")
	ui.Info("Type 'exit' to leave this session and return to your default environment.")
	fmt.Println()

	cmd := exec.Command(shellPath)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	ui.Success(fmt.Sprintf("Exited isolated session for %q. Returned to default profile.", user.Name))
	return nil
}

// ensureSSHKeyUnlocked prompts for and loads a protected SSH key into
// ssh-agent before an isolated shell/terminal starts using it, mirroring the
// passphrase gate in `git-user switch` (switch.go) — otherwise the isolated
// shell looks ready immediately but the first push/pull inside it would stall
// on ssh's own passphrase prompt (or fail outright in a non-interactive
// terminal-emulator launch, which has no tty for ssh to prompt on).
func ensureSSHKeyUnlocked(user *config.User) error {
	if user.SSHKey == "" {
		return nil
	}

	protected, err := isSSHKeyPassphraseProtected(user.SSHKey)
	if err != nil || !protected || ssh.IsSSHKeyLoaded(user.SSHKey) {
		return nil
	}

	mode := user.GetPassphraseMode()
	ui.Info(fmt.Sprintf("Identity %q is protected (mode: %s).", user.Name, mode))

	var passphrase string
	var hasStored bool
	if mode == "persistent" {
		if secret, err := keyring.GetKeychainPassphrase(user.Name); err == nil && secret != "" {
			if ssh.VerifyPassphrase(user.SSHKey, secret) {
				passphrase = secret
				hasStored = true
				ui.Info("Retrieved passphrase securely from system keychain.")
			} else {
				ui.Warn("Stored keychain passphrase was incorrect. Stale entry removed.")
				_ = keyring.DeleteKeychainPassphrase(user.Name)
			}
		}
	}

	if !hasStored {
		passphrase, err = readPassphrase(PassphrasePrompt)
		if err != nil {
			return err
		}
		if !ssh.VerifyPassphrase(user.SSHKey, passphrase) {
			ui.Error("Incorrect passphrase. Access denied.")
			return fmt.Errorf("incorrect passphrase")
		}
		if mode == "persistent" {
			promptAndStoreKeychain(user.Name, user.SSHKey, passphrase)
		}
	}

	if agentErr := ssh.EnsureSSHAgent(); agentErr != nil {
		ui.Warn(fmt.Sprintf("Key for %q was NOT loaded into any ssh-agent — the next push/pull may hang or fail asking for a passphrase.", user.Name))
		return nil
	}
	if err := ssh.AddSSHKeyWithPassphrase(user.SSHKey, passphrase); err != nil {
		ui.Warn(fmt.Sprintf("Could not load key into agent: %v", err))
		return nil
	}
	ui.Success("Key unlocked and loaded into ssh-agent.")
	return nil
}
