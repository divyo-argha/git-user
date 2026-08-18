package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/identity"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ui"
)

// These are byte-for-byte the blocks prompt.go's installZsh/installBash/
// installStarship/installFish append or write. Kept in sync with them so
// uninstall can find-and-remove exactly what was added, and nothing else.
const (
	zshPromptBlock = `
# --- git-user prompt integration ---
function _git_user_prompt() {
  local user=$(git-user prompt 2>/dev/null)
  if [[ -n "$user" ]]; then
    echo "%F{blue} ${user}%f"
  fi
}
RPROMPT='$(_git_user_prompt)'
`
	bashPromptBlock = `
# --- git-user prompt integration ---
__git_user_prompt() {
  local user=$(git-user prompt 2>/dev/null)
  if [ -n "$user" ]; then
    echo -e "\033[1;34m ${user}\033[0m "
  fi
}
# Prepend to PS1 dynamically
PROMPT_COMMAND='PS1="$(__git_user_prompt)\u@\h:\w\$ "'
`
	starshipPromptBlock = `
[custom.gituser]
command = "git-user prompt"
when = "git rev-parse --is-inside-work-tree 2>/dev/null"
format = "[$output]($style) "
style = "bold blue"
`
	fishPromptFile = `function fish_right_prompt
  set -l git_user (git-user prompt 2>/dev/null)
  if test -n "$git_user"
    set_color blue
    echo -n " $git_user"
    set_color normal
  end
end
`
)

// runUninstall removes git-user's footprint from the machine as completely as
// it can: it restores whatever git identity/config existed before git-user
// was first used, strips the global git config it manages, deletes its
// config directory and (optionally) the SSH keys it generated, removes
// keychain-stored passphrases, and cleans up shell prompt integration.
func runUninstall(args []string) error {
	autoYes := false
	deleteKeys := false
	keepKeys := false
	removeBinary := false
	for _, a := range args {
		switch a {
		case "--yes", "-y":
			autoYes = true
		case "--delete-keys":
			deleteKeys = true
		case "--keep-keys":
			keepKeys = true
		case "--remove-binary":
			removeBinary = true
		}
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	dataDir := filepath.Dir(config.ConfigPath())

	ui.Banner("UNINSTALL GIT-USER")
	fmt.Println()
	ui.Warn("This will permanently:")
	ui.Info("  • Restore your original git identity/config, if git-user saved one")
	ui.Info("  • Remove git-user's global git config (signing key, sshCommand, directory-binding includeIf rules)")
	ui.Info(fmt.Sprintf("  • Delete %s (all identities, audit log, sync state)", dataDir))
	ui.Info("  • Remove any passphrases it stored in the system keychain")
	ui.Info("  • Remove shell prompt integration it installed (zsh/bash/starship/fish), if any")
	fmt.Println()

	if !autoYes {
		if !ui.IsTTY() {
			ui.Error("Refusing to uninstall non-interactively without --yes.")
			return fmt.Errorf("confirmation required")
		}
		if !ui.Confirm("This cannot be undone. Continue?", false) {
			ui.Info("Cancelled — nothing was changed.")
			return nil
		}
	}

	generatedKeys, externalKeys := classifyIdentityKeys(store)

	if !deleteKeys && !keepKeys && len(generatedKeys) > 0 && ui.IsTTY() {
		deleteKeys = ui.Confirm(fmt.Sprintf(
			"Also permanently delete the %d SSH private key(s) git-user generated? "+
				"(They may still be registered on GitHub/GitLab/Bitbucket — this only removes the local copy.)",
			len(generatedKeys)), false)
	}

	fmt.Println()

	// 1. Restore (or clear) the global git identity/config.
	if store.Original != nil {
		if err := restoreOriginalGitConfig(store.Original); err != nil {
			ui.Warn(fmt.Sprintf("Could not fully restore original git config: %v", err))
		} else {
			ui.Success("Restored original git identity/config.")
		}
	} else {
		git.RemoveSSHConfig()
		git.RemoveSigningConfig()
		ui.Info("No pre-git-user snapshot was recorded — left user.name/user.email untouched, removed git-user's sshCommand/signing config.")
	}

	// 2. Remove directory-binding includeIf entries git-user owns.
	removeManagedIncludeIfs()
	ui.Success("Removed directory-binding (includeIf) git config.")

	// 3. Remove keychain passphrases and (optionally) generated SSH keys.
	for _, u := range store.Users {
		_ = keyring.DeleteKeychainPassphrase(u.Name)
	}
	ui.Success("Removed stored keychain passphrases.")

	if deleteKeys {
		for _, path := range generatedKeys {
			if err := identity.SecureDeleteKeyPair(path); err != nil {
				ui.Warn(fmt.Sprintf("Could not delete key %s: %v", path, err))
			} else {
				ui.Success(fmt.Sprintf("Deleted SSH key pair: %s(.pub)", path))
			}
			_ = os.Remove(path + ".backup")
			_ = os.Remove(path + ".backup.pub")
		}
	} else if len(generatedKeys) > 0 {
		ui.Info("Kept the SSH keys git-user generated (pass --delete-keys next time to remove them too):")
		for _, path := range generatedKeys {
			ui.Info("  " + path)
		}
	}
	if len(externalKeys) > 0 {
		ui.Info("Left these untouched — they were bound, not generated, by git-user:")
		for _, path := range externalKeys {
			ui.Info("  " + path)
		}
	}

	// 4. Remove shell prompt integration.
	removePromptIntegration()

	// 5. Remove git-user's own data directory.
	if err := os.RemoveAll(dataDir); err != nil {
		ui.Warn(fmt.Sprintf("Could not remove %s: %v", dataDir, err))
	} else {
		ui.Success(fmt.Sprintf("Removed %s", dataDir))
	}

	fmt.Println()
	ui.Divider()
	ui.Info("Not removed automatically — clean these up yourself if applicable:")
	ui.Info("  • Pre-commit hooks from 'git-user hook install' in individual repos — run 'git-user hook uninstall' inside them, or delete .git/hooks/pre-commit there.")
	if runtime.GOOS == "darwin" {
		ui.Info("  • The 'Host *' AddKeysToAgent/UseKeychain block git-user may have added to ~/.ssh/config, if you no longer want it.")
	}

	if !removeBinary && !autoYes && ui.IsTTY() {
		removeBinary = ui.Confirm("Also delete the git-user binary itself?", false)
	}
	if removeBinary {
		if exe, err := os.Executable(); err == nil {
			if err := os.Remove(exe); err != nil {
				ui.Warn(fmt.Sprintf("Could not remove binary at %s: %v — remove it manually.", exe, err))
			} else {
				ui.Success(fmt.Sprintf("Removed binary: %s", exe))
			}
		}
	}

	fmt.Println()
	ui.Success("git-user has been uninstalled.")
	return nil
}

// classifyIdentityKeys splits the SSH keys referenced by the config into ones
// git-user itself generated (living at the default ~/.ssh/git_<name> path it
// creates on `register`/`rekey`) versus ones the user pointed it at via
// `bind-key --ssh-key <existing path>`. Only the former are ever offered for
// deletion — a key git-user didn't create isn't git-user's to destroy.
func classifyIdentityKeys(store *config.Store) (generated, external []string) {
	seen := make(map[string]bool)
	for _, u := range store.Users {
		if u.SSHKey == "" || seen[u.SSHKey] {
			continue
		}
		seen[u.SSHKey] = true
		if _, err := os.Stat(u.SSHKey); err != nil {
			continue
		}
		if expected, err := config.DefaultSSHKeyPath(u.Name); err == nil && u.SSHKey == expected {
			generated = append(generated, u.SSHKey)
		} else {
			external = append(external, u.SSHKey)
		}
	}
	return generated, external
}

// removeManagedIncludeIfs strips only the global includeIf entries that point
// at git-user's own profile-*.gitconfig snippet files, leaving any unrelated
// includeIf rules the user configured by hand alone.
func removeManagedIncludeIfs() {
	out, err := exec.Command("git", "config", "--global", "--get-regexp", `includeif\..*\.path`).Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		if strings.Contains(value, "profile-") && strings.HasSuffix(value, ".gitconfig") {
			_ = exec.Command("git", "config", "--global", "--unset-all", key).Run()
		}
	}
}

// removePromptIntegration removes exactly the blocks prompt.go's installers
// append, by literal match — so a file the user has since edited around the
// block is left alone (with a warning) rather than partially mangled.
func removePromptIntegration() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	removeBlock := func(path, block, label string) {
		content, err := os.ReadFile(path)
		if err != nil {
			return
		}
		if !strings.Contains(string(content), block) {
			if strings.Contains(string(content), "git-user prompt") {
				ui.Warn(fmt.Sprintf("%s contains a modified git-user prompt block — remove it manually.", path))
			}
			return
		}
		updated := strings.Replace(string(content), block, "", 1)
		if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
			ui.Warn(fmt.Sprintf("Could not clean up %s: %v", path, err))
			return
		}
		ui.Success(fmt.Sprintf("Removed %s prompt integration from %s", label, path))
	}

	removeBlock(filepath.Join(home, ".zshrc"), zshPromptBlock, "zsh")
	removeBlock(filepath.Join(home, ".bashrc"), bashPromptBlock, "bash")
	removeBlock(filepath.Join(home, ".config", "starship.toml"), starshipPromptBlock, "starship")

	fishPath := filepath.Join(home, ".config", "fish", "functions", "fish_right_prompt.fish")
	if content, err := os.ReadFile(fishPath); err == nil {
		if strings.TrimSpace(string(content)) == strings.TrimSpace(fishPromptFile) {
			if err := os.Remove(fishPath); err == nil {
				ui.Success("Removed fish prompt integration: " + fishPath)
			}
		} else if strings.Contains(string(content), "git-user prompt") {
			ui.Warn(fmt.Sprintf("%s references git-user but has custom content — remove it manually.", fishPath))
		}
	}
}
