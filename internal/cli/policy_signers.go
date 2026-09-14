package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runSigners(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user policy signers <add|list|remove>")
		return fmt.Errorf("missing subcommand")
	}

	switch args[0] {
	case "add":
		return runSignersAdd(args[1:])
	case "list":
		return runSignersList()
	case "remove":
		return runSignersRemove(args[1:])
	default:
		ui.Errorf("unknown signers subcommand: %s", args[0])
		return fmt.Errorf("unknown subcommand")
	}
}

func runSignersAdd(args []string) error {
	repoRoot, err := git.RepoRoot()
	if err != nil {
		ui.Error("Not in a git repository")
		return fmt.Errorf("not in repo")
	}

	var identityName, emailFlag, pubkeyFlag string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--email":
			if i+1 < len(args) {
				emailFlag = args[i+1]
				i++
			}
		case "--pubkey-file":
			if i+1 < len(args) {
				pubkeyFlag = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") && identityName == "" {
				identityName = args[i]
			}
		}
	}

	var principals []string
	var keyBlob string

	if emailFlag != "" {
		if pubkeyFlag == "" {
			ui.Error("--pubkey-file is required when using --email")
			return fmt.Errorf("missing --pubkey-file")
		}
		pubKey, err := os.ReadFile(pubkeyFlag)
		if err != nil {
			ui.Errorf("reading %s: %v", pubkeyFlag, err)
			return err
		}
		principals = []string{emailFlag}
		keyBlob = strings.TrimSpace(string(pubKey))
	} else {
		store, err := config.Load()
		if err != nil {
			ui.Errorf("loading config: %v", err)
			return err
		}
		name := identityName
		if name == "" {
			name = store.Current
		}
		user := store.FindUser(name)
		if user == nil {
			ui.Errorf("identity %q not found", name)
			return fmt.Errorf("identity not found")
		}
		if user.SSHKey == "" {
			ui.Errorf("identity %q has no SSH key bound — use --email/--pubkey-file for a non-local contributor", user.Name)
			return fmt.Errorf("no SSH key")
		}
		pubKey, err := os.ReadFile(user.SSHKey + ".pub")
		if err != nil {
			ui.Errorf("reading %s.pub: %v", user.SSHKey, err)
			return err
		}
		principals = append([]string{user.Email}, user.Aliases...)
		keyBlob = strings.TrimSpace(string(pubKey))
	}

	entries, err := config.LoadAllowedSigners(repoRoot)
	if err != nil {
		ui.Errorf("reading %s: %v", config.AllowedSignersFileName, err)
		return err
	}
	entries = config.UpsertSignerEntry(entries, principals, keyBlob)
	if err := config.SaveAllowedSigners(repoRoot, entries); err != nil {
		ui.Errorf("writing %s: %v", config.AllowedSignersFileName, err)
		return err
	}

	ui.Success(fmt.Sprintf("Added %s to %s", strings.Join(principals, ", "), config.AllowedSignersFileName))
	suggestAllowedSignersConfig(repoRoot)
	return nil
}

func runSignersList() error {
	repoRoot, err := git.RepoRoot()
	if err != nil {
		ui.Error("Not in a git repository")
		return fmt.Errorf("not in repo")
	}

	entries, err := config.LoadAllowedSigners(repoRoot)
	if err != nil {
		ui.Errorf("reading %s: %v", config.AllowedSignersFileName, err)
		return err
	}
	if len(entries) == 0 {
		ui.Info(fmt.Sprintf("No entries in %s.", config.AllowedSignersFileName))
		return nil
	}
	for _, e := range entries {
		fmt.Printf("  %-40s  %s\n", strings.Join(e.Principals, ","), e.KeyBlob)
	}
	return nil
}

func runSignersRemove(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user policy signers remove <principal>")
		return fmt.Errorf("missing principal")
	}
	repoRoot, err := git.RepoRoot()
	if err != nil {
		ui.Error("Not in a git repository")
		return fmt.Errorf("not in repo")
	}

	entries, err := config.LoadAllowedSigners(repoRoot)
	if err != nil {
		ui.Errorf("reading %s: %v", config.AllowedSignersFileName, err)
		return err
	}
	before := len(entries)
	entries = config.RemoveSignerEntries(entries, args[0])
	if len(entries) == before {
		ui.Info(fmt.Sprintf("No entry found for %q", args[0]))
		return nil
	}
	if err := config.SaveAllowedSigners(repoRoot, entries); err != nil {
		ui.Errorf("writing %s: %v", config.AllowedSignersFileName, err)
		return err
	}
	ui.Success(fmt.Sprintf("Removed %q from %s", args[0], config.AllowedSignersFileName))
	return nil
}

func suggestAllowedSignersConfig(repoRoot string) {
	current, _ := exec.Command("git", "config", "--local", "gpg.ssh.allowedSignersFile").Output()
	if strings.TrimSpace(string(current)) == config.AllowedSignersFileName {
		return
	}
	cmd := exec.Command("git", "config", "--local", "gpg.ssh.allowedSignersFile", config.AllowedSignersFileName)
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		ui.Warn(fmt.Sprintf("Could not set gpg.ssh.allowedSignersFile: %v", err))
		return
	}
	ui.Info(fmt.Sprintf("Set local git config to use %s for signature verification", config.AllowedSignersFileName))
}
