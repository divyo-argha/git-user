// Package sshconf manages git-user-owned blocks inside ~/.ssh/config.
//
// Safety model
// ────────────
//   • We only touch blocks we own (delimited by "# git-user:begin <alias>" /
//     "# git-user:end <alias>" markers).
//   • All other content in ~/.ssh/config is preserved byte-for-byte.
//   • Writes are atomic: we write to a temp file then rename over the target.
//   • File permissions are enforced at 0600 (OpenSSH requirement).
//   • No plaintext secrets are ever written; only the key *path* is referenced.
package sshconf

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	beginMarker = "# git-user:begin"
	endMarker   = "# git-user:end"
)

// Block is one Host entry that git-user manages.
type Block struct {
	// Alias is the Host value, e.g. "github-work".
	Alias string
	// Hostname is the real server, e.g. "github.com".
	Hostname string
	// User is the SSH login name — always "git" for GitHub/GitLab/Bitbucket.
	User string
	// IdentityFile is the absolute path to the private key.
	IdentityFile string
	// AddKeysToAgent enables ssh-agent integration.
	AddKeysToAgent bool
}

var sshConfigPath string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("cannot determine home directory: " + err.Error())
	}
	sshConfigPath = filepath.Join(home, ".ssh", "config")
}

// SSHConfigPath returns the resolved path (useful for display).
func SSHConfigPath() string { return sshConfigPath }

// Upsert writes (or replaces) the managed block for b.Alias in ~/.ssh/config.
// All content outside our markers is left untouched.
func Upsert(b Block) error {
	if err := ensureSSHDir(); err != nil {
		return err
	}

	raw, err := readOrEmpty(sshConfigPath)
	if err != nil {
		return err
	}

	block := renderBlock(b)
	updated := replaceOrAppend(raw, b.Alias, block)
	return atomicWrite(sshConfigPath, updated)
}

// Remove deletes the managed block for alias from ~/.ssh/config.
// If the alias is not found it is a no-op (idempotent).
func Remove(alias string) error {
	raw, err := readOrEmpty(sshConfigPath)
	if err != nil {
		return err
	}
	updated := replaceOrAppend(raw, alias, "") // empty replacement = delete
	return atomicWrite(sshConfigPath, updated)
}

// BlockExists reports whether alias already has a managed block.
func BlockExists(alias string) (bool, error) {
	raw, err := readOrEmpty(sshConfigPath)
	if err != nil {
		return false, err
	}
	begin := fmt.Sprintf("%s %s", beginMarker, alias)
	return bytes.Contains(raw, []byte(begin)), nil
}

// ListManagedAliases returns all Host aliases currently managed by git-user.
func ListManagedAliases() ([]string, error) {
	raw, err := readOrEmpty(sshConfigPath)
	if err != nil {
		return nil, err
	}
	var aliases []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, beginMarker+" ") {
			alias := strings.TrimPrefix(line, beginMarker+" ")
			aliases = append(aliases, strings.TrimSpace(alias))
		}
	}
	return aliases, nil
}

// ValidateKeyFile checks that the given path looks like a usable private key.
// It does NOT decrypt the key — it only verifies the file exists and has safe
// permissions (0600 or 0400), which is what OpenSSH enforces at runtime.
func ValidateKeyFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("key file not found: %s", path)
		}
		return fmt.Errorf("cannot stat key file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a key file", path)
	}
	perm := info.Mode().Perm()
	if perm&0o077 != 0 {
		return fmt.Errorf(
			"key file %s has unsafe permissions (%04o) — run: chmod 600 %s",
			path, perm, path,
		)
	}
	return nil
}

// ─── internal helpers ────────────────────────────────────────────────────────

func renderBlock(b Block) string {
	var sb strings.Builder
	begin := fmt.Sprintf("%s %s", beginMarker, b.Alias)
	end := fmt.Sprintf("%s %s", endMarker, b.Alias)

	agent := "no"
	if b.AddKeysToAgent {
		agent = "yes"
	}
	user := b.User
	if user == "" {
		user = "git"
	}

	sb.WriteString(begin + "\n")
	sb.WriteString(fmt.Sprintf("Host %s\n", b.Alias))
	sb.WriteString(fmt.Sprintf("    HostName %s\n", b.Hostname))
	sb.WriteString(fmt.Sprintf("    User %s\n", user))
	sb.WriteString(fmt.Sprintf("    IdentityFile %s\n", b.IdentityFile))
	sb.WriteString(fmt.Sprintf("    IdentitiesOnly yes\n"))
	sb.WriteString(fmt.Sprintf("    AddKeysToAgent %s\n", agent))
	sb.WriteString(end + "\n")
	return sb.String()
}

// replaceOrAppend finds the begin/end markers for alias and replaces the
// content between them with newBlock. If alias is not found, newBlock is
// appended. If newBlock is empty the whole block (including markers) is deleted.
func replaceOrAppend(raw []byte, alias, newBlock string) []byte {
	begin := fmt.Sprintf("%s %s", beginMarker, alias)
	end := fmt.Sprintf("%s %s", endMarker, alias)

	lines := strings.Split(string(raw), "\n")
	var out []string
	inside := false
	found := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == begin {
			inside = true
			found = true
			if newBlock != "" {
				// Write the replacement block (already includes markers).
				out = append(out, strings.TrimRight(newBlock, "\n"))
			}
			continue
		}
		if inside {
			if trimmed == end {
				inside = false
			}
			continue
		}
		out = append(out, line)
	}

	if !found && newBlock != "" {
		// Append — ensure a blank line separator.
		if len(out) > 0 && out[len(out)-1] != "" {
			out = append(out, "")
		}
		out = append(out, strings.TrimRight(newBlock, "\n"))
	}

	result := strings.Join(out, "\n")
	// Normalise: ensure the file ends with exactly one newline.
	result = strings.TrimRight(result, "\n") + "\n"
	return []byte(result)
}

func ensureSSHDir() error {
	dir := filepath.Dir(sshConfigPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating ~/.ssh: %w", err)
	}
	return nil
}

func readOrEmpty(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []byte{}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return data, nil
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".ssh_config_tmp_*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	defer func() {
		tmp.Close()
		os.Remove(tmpName) // no-op if rename succeeded
	}()

	if err := tmp.Chmod(0600); err != nil {
		return fmt.Errorf("setting permissions on temp file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming temp file to %s: %w", path, err)
	}
	return nil
}
