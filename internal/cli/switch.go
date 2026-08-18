package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/identity"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
	"github.com/divyo-argha/git-user/internal/ui"
	"github.com/divyo-argha/git-user/internal/validate"
)

func runSwitch(args []string) error {
	localMode := false
	var filteredArgs []string
	for _, a := range args {
		if a == "--local" || a == "-l" {
			localMode = true
		} else {
			filteredArgs = append(filteredArgs, a)
		}
	}
	args = filteredArgs

	if len(args) < 1 {
		ui.Error("usage: git-user switch [-c] <name> [-e <email>] [--local]")
		return fmt.Errorf("missing arguments")
	}

	if localMode && !git.IsInRepo() {
		ui.Error("Not inside a Git repository. --local can only be used within a Git repository.")
		return fmt.Errorf("not in repository")
	}

	// Handle restore to original pre-git-user state
	if args[0] == "--original" {
		return runSwitchOriginal(nil)
	}

	createMode := false
	name := ""
	email := ""
	isTemp := false
	skipSSH := false

	if args[0] == "-c" {
		createMode = true
		var otherArgs []string
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--passphrase", "-p":
				ui.Warn("--passphrase is no longer accepted as a CLI argument (it could leak via `ps` or shell history) — you'll be prompted for it interactively instead.")
				if i+1 < len(args) {
					i++ // skip the (now ignored) value
				}
			case "--temp", "-t":
				isTemp = true
			case "--email", "-e":
				if i+1 < len(args) {
					email = args[i+1]
					i++
				}
			case "--skip-ssh":
				skipSSH = true
			default:
				otherArgs = append(otherArgs, args[i])
			}
		}
		if len(otherArgs) < 1 {
			ui.Error("usage: git-user switch -c <name> [-e <email>] [--skip-ssh]")
			return fmt.Errorf("missing name")
		}
		name = otherArgs[0]
		if len(otherArgs) > 1 && email == "" {
			// Hidden backwards-compatible alias: a trailing positional email.
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

	if createMode {
		if store.FindUser(name) != nil {
			ui.Errorf("identity %q already exists", name)
			return fmt.Errorf("user exists")
		}

		if err := quickRegister(name, email, isTemp, skipSSH, store); err != nil {
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

	if !localMode && store.Current == name && git.IsIdentityInSync(user.Name, user.Email) {
		ui.Info(fmt.Sprintf("Already using identity %q (%s) — nothing to do.", user.Name, user.Email))
		return nil
	}

	// Auto-logout: unload the previous identity's key from ssh-agent. This only
	// applies to a global switch — a `--local` switch never changes the global
	// "current" identity (store.Current, store.Save are untouched below for
	// localMode), so the previous global identity is still active everywhere
	// else. Running this for a local switch would incorrectly unload its key
	// from the agent and, worse, permanently delete a temporary identity's key
	// files while it's still the active identity outside this repo.
	if !localMode && store.Current != "" && store.Current != name {
		if prev := store.CurrentUser(); prev != nil {
			if prev.SSHKey != "" && ssh.IsSSHKeyLoaded(prev.SSHKey) {
				_ = ssh.RemoveSSHKey(prev.SSHKey)
				ui.Info(fmt.Sprintf("Unloaded SSH key for previous identity %q", prev.Name))
			}
			if prev.GetPassphraseMode() == "everytime" && prev.SSHKey != "" {
				_ = ssh.RemoveSSHKey(prev.SSHKey)
			}
			if prev.IsTemporary {
				store.RemoveUser(prev.Name, true)
				ui.Info(fmt.Sprintf("Temporary identity %q deleted.", prev.Name))
				if prev.SSHKey != "" {
					_ = identity.SecureDeleteKeyPair(prev.SSHKey)
					ui.Info(fmt.Sprintf("Temporary SSH key files deleted: %s", prev.SSHKey))
				}
				_ = keyring.DeleteKeychainPassphrase(prev.Name)
			}
		}
	}

	// Warn clearly if the bound SSH key file is missing so a switch never
	// silently produces broken push behavior with the wrong/absent key.
	if user.SSHKey != "" {
		if _, statErr := os.Stat(user.SSHKey); statErr != nil {
			ui.Warn(fmt.Sprintf("Bound SSH key not found: %s", user.SSHKey))
			ui.Info(fmt.Sprintf("Fix it with: git-user bind-key %s --ssh-key <path>", user.Name))
		}
	}

	// Passphrase gate
	if user.SSHKey != "" {
		mode := user.GetPassphraseMode()
		protected, err := isSSHKeyPassphraseProtected(user.SSHKey)
		if err == nil && protected && !ssh.IsSSHKeyLoaded(user.SSHKey) {
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
				var err error
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

			// Load it into agent. EnsureSSHAgent already prints its own
			// "ssh-agent is not running" guidance when it fails, but that's
			// easy to miss among the rest of this command's output and gives
			// no indication afterward that the key never actually got loaded
			// — call it out explicitly so "Switched to X" never reads as
			// "and the key is ready to use" when it isn't.
			if agentErr := ssh.EnsureSSHAgent(); agentErr != nil {
				ui.Warn(fmt.Sprintf("Key for %q was NOT loaded into any ssh-agent — the next push/pull may hang or fail asking for a passphrase.", user.Name))
			} else {
				if err := ssh.AddSSHKeyWithPassphrase(user.SSHKey, passphrase); err != nil {
					ui.Warn(fmt.Sprintf("Could not load key into agent: %v", err))
				} else {
					ui.Success("Key unlocked and loaded into ssh-agent.")
				}
			}
		}
	}

	if localMode {
		if err := git.ApplyScope(user.Name, user.Email, true); err != nil {
			ui.Errorf("applying local git config: %v", err)
			return err
		}

		if err := applyUserSSHConfig(user, true); err != nil {
			ui.Warn(fmt.Sprintf("applying local SSH config: %v", err))
		}

		if !user.SignDisabled && user.SignKey != "" {
			if err := git.ConfigureSigningScope(user.SignKey, user.SignFormat, true); err != nil {
				ui.Warn(fmt.Sprintf("applying local signing config: %v", err))
			}
		} else {
			git.RemoveSigningConfigScope(true)
		}

		if prev := store.CurrentUser(); prev != nil {
			for k := range prev.CustomConfig {
				_ = unsetActiveCustomConfig(k, true)
			}
		}
		for k, v := range user.CustomConfig {
			_ = applyActiveCustomConfig(k, v, true)
		}

		ui.AnimatedSuccess(fmt.Sprintf("Locally switched to %q (%s) for the current repository", user.Name, user.Email))
		if !user.SignDisabled && user.SignKey != "" {
			ui.Success(fmt.Sprintf("Commit Signing: Enabled (%s)", user.SignFormat))
		}
		_ = config.AppendSwitchLog(user.Name, currentDir())
	} else {
		if git.IsInRepo() && git.HasLocalOverride() {
			git.ClearIdentityScope(true)
			ui.Info("Cleared local repository gitconfig overrides to apply global identity.")
		}

		if err := git.Apply(user.Name, user.Email); err != nil {
			ui.Errorf("applying git config: %v", err)
			return err
		}

		if err := applyUserSSHConfig(user, false); err != nil {
			ui.Warn(fmt.Sprintf("applying SSH config: %v", err))
		}

		if !user.SignDisabled && user.SignKey != "" {
			if err := git.ConfigureSigning(user.SignKey, user.SignFormat); err != nil {
				ui.Warn(fmt.Sprintf("applying signing config: %v", err))
			}
		} else {
			git.RemoveSigningConfig()
		}

		if prev := store.CurrentUser(); prev != nil {
			for k := range prev.CustomConfig {
				_ = unsetActiveCustomConfig(k, false)
			}
		}
		for k, v := range user.CustomConfig {
			_ = applyActiveCustomConfig(k, v, false)
		}

		if err := store.SetCurrent(name); err != nil {
			ui.Errorf("%v", err)
			return err
		}

		if err := config.Save(store); err != nil {
			ui.Errorf("saving config: %v", err)
			return err
		}

		ui.AnimatedSuccess(fmt.Sprintf("Switched to %q (%s)", user.Name, user.Email))
		if !user.SignDisabled && user.SignKey != "" {
			ui.Success(fmt.Sprintf("Commit Signing: Enabled (%s)", user.SignFormat))
		}
		_ = config.AppendSwitchLog(user.Name, currentDir())
	}

	if user.SSHKey != "" {
		// Only an agent-loaded key is required when the key is passphrase
		// protected — verifySSHConnectionWithKey passes -i explicitly, so an
		// unprotected key never needs the agent at all. Gating on
		// IsSSHKeyLoaded unconditionally (regardless of protection) skipped
		// verification for every unprotected-key identity, silently hiding
		// real connection problems until the next `git push` failed. Mirrors
		// the same "protected && not loaded" check used by
		// needsPassphraseForSwitch (internal/tui/ops.go).
		protected, _ := isSSHKeyPassphraseProtected(user.SSHKey)
		if !protected || ssh.IsSSHKeyLoaded(user.SSHKey) {
			if err := verifySSHConnectionWithKey(user.SSHKey); err != nil {
				ui.Warn("SSH verification failed. The key may not be added to your platform yet.")
				ui.Info(fmt.Sprintf("Test manually with: ssh -i %s -o IdentitiesOnly=yes -T git@github.com", user.SSHKey))
			} else {
				ui.Success("SSH verified: Connection successful!")
			}
		} else {
			ui.Info("Skipping SSH verification until the key is loaded")
		}
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

func quickRegister(name, email string, isTemp, skipSSH bool, store *config.Store) error {
	ui.Banner("QUICK SETUP: " + name)
	fmt.Println()

	var err error

	if err := validate.IdentityName(name); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if email == "" {
		email, err = ui.Prompt("Email address:")
		if err != nil {
			return err
		}
	}

	for {
		if err := validate.Email(email); err != nil {
			ui.Warn(err.Error())
			email, err = ui.Prompt("Enter a valid email address:")
			if err != nil {
				return err
			}
			continue
		}
		break
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

	if skipSSH {
		if err := config.Save(store); err != nil {
			return err
		}
		fmt.Println()
		ui.Success(fmt.Sprintf("Identity created: %s (%s)", name, email))
		ui.Info("SSH key setup skipped — attach one later with: git-user bind-key " + name)
		fmt.Println()
		return nil
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
		path, err := generateAndDisplayKey(name, email)
		if err != nil {
			ui.Warn("Key generation failed")
			break
		}
		sshKeyPath = path

	case 1: // Use existing key
		keyPath, err := ui.Prompt("Path to SSH key:")
		if err == nil && keyPath != "" {
			if err := validate.SSHKeyPath(keyPath, true); err == nil {
				sshKeyPath = validate.ExpandPath(keyPath)
				ui.Success("Using existing key")
			} else {
				ui.Warn(err.Error())
			}
		}

	case 2: // Skip
		ui.Info("Skipping SSH setup")
	}

	if sshKeyPath != "" {
		if err := store.BindSSHKey(name, sshKeyPath); err != nil {
			ui.Warn("Could not bind SSH key")
		}
		fmt.Println()
		if ui.Confirm("Would you like to sign your Git commits automatically using this identity's SSH key?", true) {
			if err := store.SetSigningKey(name, sshKeyPath, "ssh"); err != nil {
				ui.Warn(fmt.Sprintf("Failed to enable SSH commit signing: %v", err))
			} else {
				ui.Success("Commit signing configured automatically!")
			}
		} else {
			store.ToggleSigning(name, true)
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

// restoreOriginalGitConfig applies a pre-git-user gitconfig snapshot back onto
// the live global git config: identity, SSH command, and signing. Shared by
// `switch --original` and `uninstall`, which both need to put git back the
// way it was before git-user ever touched it.
func restoreOriginalGitConfig(o *config.OriginalConfig) error {
	if err := git.Apply(o.Name, o.Email); err != nil {
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
		format := "gpg"
		if o.SignFormat == "ssh" {
			format = "ssh"
		}
		if err := git.ConfigureSigning(o.SignKey, format); err != nil {
			ui.Warn(fmt.Sprintf("could not restore signing config: %v", err))
		}
	} else {
		git.RemoveSigningConfig()
	}

	return nil
}

func runSwitchOriginal(args []string) error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	o := store.Original
	if o == nil {
		// No snapshot yet: this is a plain import of the current gitconfig
		// identity (interactive name pick when none given).
		ui.Info("No original identity snapshot exists yet — importing the current gitconfig identity instead.")
		return runImportOriginal(args)
	}

	if o.Name == "" && o.Email == "" {
		ui.Warn("Original gitconfig had no user.name or user.email set")
	}

	if err := restoreOriginalGitConfig(o); err != nil {
		ui.Errorf("restoring git config: %v", err)
		return err
	}

	if store.Current != "" {
		if prev := store.CurrentUser(); prev != nil {
			for k := range prev.CustomConfig {
				_ = unsetActiveCustomConfig(k, false)
			}
		}
	}

	// The original identity is now active in the gitconfig. Register it as a
	// managed identity (once) so it also appears in `list` and can be switched
	// to by name. This unifies `switch --original` with `import-original`.
	origName := findOriginalIdentity(store)
	if origName == "" && (o.Name != "" || o.Email != "") {
		origName = o.Name
		if origName == "" {
			origName = "original"
		}
		if store.FindUser(origName) != nil {
			ui.Errorf("identity %q already exists — original was not registered", origName)
		} else {
			store.Users = append(store.Users, config.User{
				Name:       origName,
				Email:      o.Email,
				SSHKey:     extractSSHKeyFromCommand(o.SSHCommand),
				SSHCommand: o.SSHCommand,
				Source:     "original",
			})
		}
	}

	if origName != "" {
		store.Current = origName
	} else {
		store.Current = ""
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	// Verify the restore actually took effect in the gitconfig. If the applied
	// values were overridden (e.g. by an environment variable, an includeIf, or
	// another session racing this one), warn instead of silently reporting
	// success — the active commit identity would not be the restored one.
	if o.Name != "" || o.Email != "" {
		if !git.IsIdentityInSync(o.Name, o.Email) {
			ui.Warn("Restored identity was overridden — the git config no longer matches the restored identity.")
			ui.Info("Check for environment variables (GIT_AUTHOR_NAME, GIT_COMMITTER_NAME) or includeIf rules overriding user.name/user.email.")
		}
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

// findOriginalIdentity returns the name of the managed identity tagged with
// Source == "original", or "" if none has been imported yet.
func findOriginalIdentity(store *config.Store) string {
	for _, u := range store.Users {
		if u.Source == "original" {
			return u.Name
		}
	}
	return ""
}

// applyUserSSHConfig writes core.sshCommand for an identity. When an identity
// was imported from the original gitconfig and carries an explicit SSH command,
// that command is preserved exactly so nothing about the user's existing SSH
// setup is lost. Otherwise the bound SSH key is used, or the command is removed
// when the identity has no key at all.
func applyUserSSHConfig(user *config.User, local bool) error {
	if user.SSHCommand != "" {
		return git.SetSSHCommandScope(user.SSHCommand, local)
	}
	if user.SSHKey != "" {
		return git.ConfigureSSHScope(user.SSHKey, local)
	}
	return git.RemoveSSHConfigScope(local)
}

// currentDir returns the working directory for the switch-log entry, or ""
// if it can't be determined.
func currentDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}
