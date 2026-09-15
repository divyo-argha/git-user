package cli

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/policyops"
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
		principals, keyBlob, err = policyops.ResolveSignerFromEmail(emailFlag, pubkeyFlag)
		if err != nil {
			ui.Errorf("%v", err)
			return err
		}
	} else {
		store, err := config.Load()
		if err != nil {
			ui.Errorf("loading config: %v", err)
			return err
		}
		principals, keyBlob, err = policyops.ResolveSignerFromIdentity(store, identityName)
		if err != nil {
			ui.Errorf("%v", err)
			return err
		}
	}

	if err := policyops.UpsertSigner(repoRoot, principals, keyBlob); err != nil {
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

	removed, err := policyops.RemoveSigner(repoRoot, args[0])
	if err != nil {
		ui.Errorf("%v", err)
		return err
	}
	if !removed {
		ui.Info(fmt.Sprintf("No entry found for %q", args[0]))
		return nil
	}
	ui.Success(fmt.Sprintf("Removed %q from %s", args[0], config.AllowedSignersFileName))
	return nil
}
