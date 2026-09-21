package policyops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
)

func BuildPolicyFile(requireSigning bool, domains []string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# git-user repository policy\n# See: git-user hook --help\n\n")
	fmt.Fprintf(&b, "require_signing=%t\n", requireSigning)
	if len(domains) > 0 {
		fmt.Fprintf(&b, "allowed_email_domains=%s\n", strings.Join(domains, ","))
	}
	return []byte(b.String())
}

func WritePolicy(repoRoot string, requireSigning bool, domains []string) error {
	return os.WriteFile(filepath.Join(repoRoot, config.RepoPolicyFileName), BuildPolicyFile(requireSigning, domains), 0644)
}

func ResolveSignerFromIdentity(store *config.Store, name string) (principals []string, keyBlob string, err error) {
	if name == "" {
		name = store.Current
	}
	user := store.FindUser(name)
	if user == nil {
		return nil, "", fmt.Errorf("identity %q not found", name)
	}
	if user.SSHKey == "" {
		return nil, "", fmt.Errorf("identity %q has no SSH key bound — use --email/--pubkey-file for a non-local contributor", user.Name)
	}
	pubKey, err := os.ReadFile(user.SSHKey + ".pub")
	if err != nil {
		return nil, "", fmt.Errorf("reading %s.pub: %w", user.SSHKey, err)
	}
	principals = append([]string{user.Email}, user.Aliases...)
	return principals, strings.TrimSpace(string(pubKey)), nil
}

func ResolveSignerFromEmail(email, pubkeyFile string) (principals []string, keyBlob string, err error) {
	if pubkeyFile == "" {
		return nil, "", fmt.Errorf("--pubkey-file is required when using --email")
	}
	pubKey, err := os.ReadFile(pubkeyFile)
	if err != nil {
		return nil, "", fmt.Errorf("reading %s: %w", pubkeyFile, err)
	}
	return []string{email}, strings.TrimSpace(string(pubKey)), nil
}

func UpsertSigner(repoRoot string, principals []string, keyBlob string) error {
	entries, err := config.LoadAllowedSigners(repoRoot)
	if err != nil {
		return err
	}
	entries = config.UpsertSignerEntry(entries, principals, keyBlob)
	return config.SaveAllowedSigners(repoRoot, entries)
}

func RemoveSigner(repoRoot, principal string) (removed bool, err error) {
	entries, err := config.LoadAllowedSigners(repoRoot)
	if err != nil {
		return false, err
	}
	before := len(entries)
	entries = config.RemoveSignerEntries(entries, principal)
	if len(entries) == before {
		return false, nil
	}
	return true, config.SaveAllowedSigners(repoRoot, entries)
}

func WireAllowedSignersConfig(repoRoot string) (changed bool, err error) {
	checkCmd := exec.Command("git", "config", "--local", "gpg.ssh.allowedSignersFile")
	checkCmd.Dir = repoRoot
	current, _ := checkCmd.Output()
	if strings.TrimSpace(string(current)) == config.AllowedSignersFileName {
		return false, nil
	}
	cmd := exec.Command("git", "config", "--local", "gpg.ssh.allowedSignersFile", config.AllowedSignersFileName)
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		return false, err
	}
	return true, nil
}
