package tui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/divyo-argha/git-user/internal/bundle"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/keyring"
	"github.com/divyo-argha/git-user/internal/ssh"
	xssh "golang.org/x/crypto/ssh"
)

// opResult carries the outcome of an in-TUI operation. Detail is rendered on a
// Report screen when ShowReport is true; otherwise it is shown as a toast.
type opResult struct {
	detail     string
	showReport bool
}

// Sentinel errors used to signal that the UI must prompt for more input.
var (
	ErrNeedsPassphrase  = errors.New("passphrase required")
	ErrNeedsCredential  = errors.New("platform credential required")
)

// ── Shared helpers ────────────────────────────────────────────────────────────

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

func isSSHKeyPassphraseProtected(keyPath string) (bool, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return false, err
	}
	_, err = xssh.ParseRawPrivateKey(data)
	if err == nil {
		return false, nil
	}
	var passphraseErr *xssh.PassphraseMissingError
	if errors.As(err, &passphraseErr) {
		return true, nil
	}
	return false, err
}

// needsPassphraseForSwitch reports whether switching to the identity requires
// interactive passphrase entry (protected, not loaded, not in keychain).
func needsPassphraseForSwitch(store *config.Store, name string) bool {
	user := store.FindUser(name)
	if user == nil || user.SSHKey == "" {
		return false
	}
	protected, err := isSSHKeyPassphraseProtected(user.SSHKey)
	if err != nil || !protected || ssh.IsSSHKeyLoaded(user.SSHKey) {
		return false
	}
	if user.GetPassphraseMode() == "persistent" {
		if secret, kerr := keyring.GetKeychainPassphrase(user.Name); kerr == nil && secret != "" {
			return false
		}
	}
	return true
}

// ── Switch / Logout / Remove ──────────────────────────────────────────────────

// opSwitch switches the active identity. Passphrase is optional; if the key is
// protected and not loaded it must be provided (or retrieved from the keychain).
func opSwitch(store *config.Store, name, passphrase string) (opResult, error) {
	if !git.IsInstalled() {
		return opResult{}, fmt.Errorf("git is not installed or not on PATH")
	}

	store.SnapshotOriginal(git.CurrentName(), git.CurrentEmail(), git.CurrentSSHCommand(), git.CurrentSigningKey(), git.CurrentSignFormat(), git.CurrentCommitGPGSign())

	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}

	var warnings []string

	// Auto-logout: unload the previous identity's key and clean up temporaries.
	if store.Current != "" && store.Current != name {
		if prev := store.CurrentUser(); prev != nil {
			if prev.SSHKey != "" && ssh.IsSSHKeyLoaded(prev.SSHKey) {
				_ = ssh.RemoveSSHKey(prev.SSHKey)
			}
			if prev.GetPassphraseMode() == "everytime" && prev.SSHKey != "" {
				_ = ssh.RemoveSSHKey(prev.SSHKey)
			}
			if prev.IsTemporary {
				store.RemoveUser(prev.Name, true)
				if prev.SSHKey != "" {
					_ = os.Remove(prev.SSHKey)
					_ = os.Remove(prev.SSHKey + ".pub")
				}
				_ = keyring.DeleteKeychainPassphrase(prev.Name)
			}
		}
	}

	// Warn if the bound SSH key file is missing.
	if user.SSHKey != "" {
		if _, statErr := os.Stat(user.SSHKey); statErr != nil {
			warnings = append(warnings, fmt.Sprintf("Bound SSH key not found: %s — fix it with bind using the new key path.", user.SSHKey))
		}
	}

	// Passphrase gate.
	if user.SSHKey != "" {
		mode := user.GetPassphraseMode()
		protected, perr := isSSHKeyPassphraseProtected(user.SSHKey)
		if perr == nil && protected && !ssh.IsSSHKeyLoaded(user.SSHKey) {
			p := passphrase
			if p == "" && mode == "persistent" {
				if secret, kerr := keyring.GetKeychainPassphrase(user.Name); kerr == nil && secret != "" {
					if ssh.VerifyPassphrase(user.SSHKey, secret) {
						p = secret
					} else {
						_ = keyring.DeleteKeychainPassphrase(user.Name)
					}
				}
			}
			if p == "" {
				return opResult{}, ErrNeedsPassphrase
			}
			if !ssh.VerifyPassphrase(user.SSHKey, p) {
				return opResult{}, fmt.Errorf("incorrect passphrase")
			}
			if passphrase != "" && mode == "persistent" {
				_ = keyring.SetKeychainPassphrase(user.Name, passphrase)
			}
			if ssh.EnsureSSHAgent() == nil {
				if err := ssh.AddSSHKeyWithPassphrase(user.SSHKey, p); err != nil {
					warnings = append(warnings, fmt.Sprintf("Could not load key into agent: %v", err))
				}
			}
		}
	}

	if err := git.Apply(user.Name, user.Email); err != nil {
		return opResult{}, fmt.Errorf("applying git config: %w", err)
	}
	if err := applyUserSSHConfig(user, false); err != nil {
		warnings = append(warnings, fmt.Sprintf("applying SSH config: %v", err))
	}
	if !user.SignDisabled && user.SignKey != "" {
		if err := git.ConfigureSigning(user.SignKey, user.SignFormat); err != nil {
			warnings = append(warnings, fmt.Sprintf("applying signing config: %v", err))
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
		return opResult{}, err
	}
	if err := config.Save(store); err != nil {
		return opResult{}, fmt.Errorf("saving config: %w", err)
	}

	report := fmt.Sprintf("Switched to %q (%s)\n", user.Name, user.Email)
	if !user.SignDisabled && user.SignKey != "" {
		report += fmt.Sprintf("Commit Signing: Enabled (%s)\n", user.SignFormat)
	}
	if user.SSHKey != "" && !ssh.IsSSHKeyLoaded(user.SSHKey) {
		report += "Note: use 'Check SSH connection' from the profile menu to verify the key works.\n"
	}
	for _, w := range warnings {
		report += "⚠ " + w + "\n"
	}

	return opResult{detail: report, showReport: false}, nil
}

// applyUserSSHConfig mirrors the CLI logic for core.sshCommand.
func applyUserSSHConfig(user *config.User, local bool) error {
	if user.SSHCommand != "" {
		return git.SetSSHCommandScope(user.SSHCommand, local)
	}
	if user.SSHKey != "" {
		return git.ConfigureSSHScope(user.SSHKey, local)
	}
	return git.RemoveSSHConfigScope(local)
}

func applyActiveCustomConfig(key, value string, local bool) error {
	scope := "--global"
	if local {
		scope = "--local"
	}
	return exec.Command("git", "config", scope, key, value).Run()
}

func unsetActiveCustomConfig(key string, local bool) error {
	scope := "--global"
	if local {
		scope = "--local"
	}
	return exec.Command("git", "config", scope, "--unset-all", key).Run()
}

// opLogout signs out of the current identity.
func opLogout(store *config.Store) (opResult, error) {
	user := store.CurrentUser()
	if user == nil {
		return opResult{detail: "Already signed out — no active identity."}, nil
	}
	if user.SSHKey != "" && ssh.IsSSHKeyLoaded(user.SSHKey) {
		_ = ssh.RemoveSSHKey(user.SSHKey)
	}
	git.ClearIdentity()
	if user.IsTemporary {
		store.RemoveUser(user.Name, true)
		if user.SSHKey != "" {
			_ = os.Remove(user.SSHKey)
			_ = os.Remove(user.SSHKey + ".pub")
		}
		_ = keyring.DeleteKeychainPassphrase(user.Name)
	}
	store.Current = ""
	if err := config.Save(store); err != nil {
		return opResult{}, fmt.Errorf("saving config: %w", err)
	}
	return opResult{detail: fmt.Sprintf("Signed out from %q. No active git identity.", user.Name)}, nil
}

// opRename renames an identity.
func opRename(store *config.Store, name, newName string) error {
	if store.FindUser(newName) != nil {
		return fmt.Errorf("identity %q already exists", newName)
	}
	u := store.FindUser(name)
	if u == nil {
		return fmt.Errorf("identity %q not found", name)
	}
	u.Name = newName
	if store.Current == name {
		store.Current = newName
	}
	return config.Save(store)
}

// opChangeEmail updates an identity's email, reapplying git config when active.
func opChangeEmail(store *config.Store, name, newEmail string) error {
	for _, u := range store.Users {
		if u.Name != name && u.Email == newEmail {
			return fmt.Errorf("email already in use — each identity must have a unique email")
		}
	}
	if err := store.UpdateUser(name, newEmail); err != nil {
		return err
	}
	if store.Current == name {
		u := store.FindUser(name)
		if err := git.Apply(u.Name, u.Email); err != nil {
			return fmt.Errorf("re-applying git config: %w", err)
		}
	}
	return config.Save(store)
}

// opRemove removes an identity and returns the bound SSH key path so the UI can
// offer to delete the key files. The confirmation dialog already covers the
// destructive nature, so the active identity may also be removed (its git
// config is cleared afterwards).
func opRemove(store *config.Store, name string) (string, error) {
	user := store.FindUser(name)
	if user == nil {
		return "", fmt.Errorf("identity %q not found", name)
	}
	sshKey := user.SSHKey
	if err := store.RemoveUser(name, true); err != nil {
		return "", err
	}
	_ = keyring.DeleteKeychainPassphrase(name)
	if store.Current == "" {
		git.ClearIdentity()
	}
	if err := config.Save(store); err != nil {
		return "", err
	}
	return sshKey, nil
}

// opDeleteKeyFiles removes an SSH key pair.
func opDeleteKeyFiles(keyPath string) {
	if keyPath == "" {
		return
	}
	_ = os.Remove(keyPath)
	_ = os.Remove(keyPath + ".pub")
}

// ── Bind / Rekey / Passphrase ─────────────────────────────────────────────────

// opGenerateKey creates an ed25519 key non-interactively.
func opGenerateKey(name, email, passphrase string) (string, error) {
	home, _ := os.UserHomeDir()
	keyPath := filepath.Join(home, ".ssh", fmt.Sprintf("git_%s", name))
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return "", fmt.Errorf("creating .ssh directory: %w", err)
	}
	if _, err := os.Stat(keyPath); err == nil {
		return keyPath, nil
	}
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath, "-N", passphrase)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if passphrase != "" {
		_ = keyring.SetKeychainPassphrase(name, passphrase)
	}
	return keyPath, nil
}

// opRegisterFinish completes profile creation: adds the user, binds the key if
// generated, and configures commit signing.
func opRegisterFinish(store *config.Store, name, email string, isTemp bool, keyPath, passphrase string, signEnabled bool) (opResult, error) {
	if err := store.AddUser(name, email); err != nil {
		return opResult{}, err
	}
	if isTemp {
		if u := store.FindUser(name); u != nil {
			u.IsTemporary = true
		}
	}
	if keyPath != "" {
		if err := store.BindSSHKey(name, keyPath); err != nil {
			return opResult{}, fmt.Errorf("binding SSH key: %w", err)
		}
		if signEnabled {
			if err := store.SetSigningKey(name, keyPath, "ssh"); err != nil {
				return opResult{}, fmt.Errorf("failed to enable SSH commit signing: %w", err)
			}
		} else {
			store.ToggleSigning(name, true)
		}
	}
	if err := config.Save(store); err != nil {
		return opResult{}, err
	}

	report := fmt.Sprintf("Identity created: %s (%s)\n", name, email)
	if keyPath != "" {
		report += fmt.Sprintf("SSH key: %s\n", keyPath)
		report += "Activate it from the dashboard or the profile detail view.\n"
		if pub, err := os.ReadFile(keyPath + ".pub"); err == nil {
			report += "\nPublic key:\n" + strings.TrimSpace(string(pub)) + "\n"
		}
	} else {
		report += "No SSH key set — bind one later from the profile detail view.\n"
	}
	return opResult{detail: report, showReport: true}, nil
}

// opBind associates an existing SSH key with an identity.
func opBind(store *config.Store, name, keyPath string, signEnabled bool) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	expanded := expandPath(keyPath)
	if _, err := os.Stat(expanded); err != nil {
		return opResult{}, fmt.Errorf("SSH key file %q does not exist", keyPath)
	}
	if err := store.BindSSHKey(name, expanded); err != nil {
		return opResult{}, err
	}
	if signEnabled {
		if err := store.SetSigningKey(name, expanded, "ssh"); err != nil {
			return opResult{}, fmt.Errorf("failed to enable SSH commit signing: %w", err)
		}
	} else {
		store.ToggleSigning(name, true)
	}
	if err := config.Save(store); err != nil {
		return opResult{}, err
	}
	report := fmt.Sprintf("SSH key configured for %q\nKey: %s\n", name, expanded)
	if signEnabled {
		report += "Commit Signing: Enabled\n"
	} else {
		report += "Commit Signing: Disabled\n"
	}
	if store.Current == name {
		if err := git.ConfigureSSH(expanded); err != nil {
			report += fmt.Sprintf("⚠ updating git SSH config: %v\n", err)
		}
		if signEnabled {
			if err := git.ConfigureSigning(expanded, "ssh"); err != nil {
				report += fmt.Sprintf("⚠ updating git signing config: %v\n", err)
			}
		} else {
			git.RemoveSigningConfig()
		}
		report += "Git config updated for the active identity.\n"
	}
	return opResult{detail: report, showReport: true}, nil
}

// opUnbind removes an identity's SSH key binding.
func opUnbind(store *config.Store, name string) error {
	u := store.FindUser(name)
	if u == nil {
		return fmt.Errorf("identity %q not found", name)
	}
	u.SSHKey = ""
	if err := config.Save(store); err != nil {
		return err
	}
	if store.Current == name {
		git.RemoveSSHConfig()
	}
	return nil
}

// opAttachKey resolves the SSH key choice from the in-TUI setup chain
// (generate / existing / skip) and either completes profile creation
// (register, register-temp) or binds the key to an existing identity (bind).
func opAttachKey(store *config.Store, name, email, mode, choice, passphrase, keyPath string, signEnabled bool) (opResult, error) {
	switch choice {
	case "generate":
		kp, err := opGenerateKey(name, email, passphrase)
		if err != nil {
			return opResult{}, err
		}
		keyPath = kp
	case "existing":
		expanded := expandPath(keyPath)
		if _, err := os.Stat(expanded); err != nil {
			return opResult{}, fmt.Errorf("key file not found: %s", keyPath)
		}
		keyPath = expanded
	case "skip", "":
		keyPath = ""
	}

	if mode == "bind" {
		if keyPath == "" {
			return opResult{}, fmt.Errorf("no SSH key configured")
		}
		return opBind(store, name, keyPath, signEnabled)
	}

	isTemp := mode == "register-temp"
	return opRegisterFinish(store, name, email, isTemp, keyPath, passphrase, signEnabled)
}

// opRekey rotates an identity's SSH key (non-interactive, forced).
func opRekey(store *config.Store, name, passphrase string) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	keyPath := filepath.Join(sshDir, fmt.Sprintf("git_%s", name))
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return opResult{}, fmt.Errorf("creating .ssh directory: %w", err)
	}

	backupPath := keyPath + ".backup"
	hasOldKey := false
	if _, err := os.Stat(keyPath); err == nil {
		hasOldKey = true
		if err := os.Rename(keyPath, backupPath); err != nil {
			return opResult{}, fmt.Errorf("backing up key: %w", err)
		}
		if _, err := os.Stat(keyPath + ".pub"); err == nil {
			_ = os.Rename(keyPath+".pub", backupPath+".pub")
		}
	}

	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", user.Email, "-f", keyPath, "-N", passphrase)
	if out, err := cmd.CombinedOutput(); err != nil {
		if hasOldKey {
			_ = os.Rename(backupPath, keyPath)
			_ = os.Rename(backupPath+".pub", keyPath+".pub")
		}
		return opResult{}, fmt.Errorf("generating SSH key: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if passphrase != "" {
		_ = keyring.SetKeychainPassphrase(name, passphrase)
	}

	if err := store.BindSSHKey(name, keyPath); err != nil {
		return opResult{}, err
	}
	if err := config.Save(store); err != nil {
		return opResult{}, err
	}

	report := fmt.Sprintf("SSH key rotated successfully for %s\nOld key backed up with .backup extension\n\n", name)
	if pub, err := os.ReadFile(keyPath + ".pub"); err == nil {
		report += "REPLACE YOUR OLD KEY WITH THIS NEW PUBLIC KEY\n"
		report += strings.TrimSpace(string(pub)) + "\n\n"
	}
	report += "• GitHub: Settings → SSH and GPG keys → Delete old key → Add new key\n"
	report += "• GitLab: Preferences → SSH Keys → Remove old key → Add new key\n"
	report += "• Bitbucket: Personal settings → SSH keys → Delete old → Add new\n"
	return opResult{detail: report, showReport: true}, nil
}

// changeSSHKeyPassphrase changes an SSH key passphrase non-interactively.
func changeSSHKeyPassphrase(keyPath, oldPassphrase, newPassphrase string) error {
	cmd := exec.Command("ssh-keygen", "-p", "-f", keyPath, "-P", oldPassphrase, "-N", newPassphrase)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ssh-keygen: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// opPassphraseSet sets or changes an identity's key passphrase.
func opPassphraseSet(store *config.Store, name, oldPass, newPass string) error {
	user := store.FindUser(name)
	if user == nil {
		return fmt.Errorf("identity %q not found", name)
	}
	if newPass == "" {
		return fmt.Errorf("passphrase must not be empty")
	}
	if err := changeSSHKeyPassphrase(user.SSHKey, oldPass, newPass); err != nil {
		return err
	}
	if user.GetPassphraseMode() == "persistent" {
		_ = keyring.SetKeychainPassphrase(name, newPass)
	}
	return nil
}

// opPassphraseVerify verifies a key passphrase.
func opPassphraseVerify(store *config.Store, name, pass string) error {
	user := store.FindUser(name)
	if user == nil {
		return fmt.Errorf("identity %q not found", name)
	}
	if !ssh.VerifyPassphrase(user.SSHKey, pass) {
		return fmt.Errorf("incorrect passphrase")
	}
	return nil
}

// opPassphraseRemove removes a key passphrase (only valid for the active identity).
func opPassphraseRemove(store *config.Store, name, currentPass string) error {
	user := store.FindUser(name)
	if user == nil {
		return fmt.Errorf("identity %q not found", name)
	}
	if store.Current != name {
		return fmt.Errorf("must switch to profile %q to remove its passphrase", name)
	}
	if err := changeSSHKeyPassphrase(user.SSHKey, currentPass, ""); err != nil {
		return err
	}
	_ = keyring.DeleteKeychainPassphrase(name)
	return nil
}

// ── Path bindings ─────────────────────────────────────────────────────────────

func opBindPath(store *config.Store, name, path string) error {
	expanded := expandPath(path)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory %q does not exist", path)
		}
		return fmt.Errorf("error reading directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %q is a file, not a directory", path)
	}
	if err := store.BindPathToUser(name, abs); err != nil {
		return err
	}
	return config.Save(store)
}

func opUnbindPath(store *config.Store, name, path string) error {
	expanded := expandPath(path)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	if err := store.UnbindPathFromUser(name, abs); err != nil {
		return err
	}
	return config.Save(store)
}

// ── Export / Import ───────────────────────────────────────────────────────────

// opExport bundles identities into an encrypted file.
func opExport(store *config.Store, names []string, all bool, passphrase string) (opResult, error) {
	if passphrase == "" {
		return opResult{}, fmt.Errorf("passphrase must not be empty")
	}

	var selected []config.User
	if all {
		for _, u := range store.Users {
			if !u.IsTemporary {
				selected = append(selected, u)
			}
		}
	} else {
		for _, name := range names {
			u := store.FindUser(name)
			if u == nil {
				return opResult{}, fmt.Errorf("identity %q not found", name)
			}
			if u.IsTemporary {
				continue
			}
			selected = append(selected, *u)
		}
	}
	if len(selected) == 0 {
		return opResult{detail: "No eligible identities to export."}, nil
	}

	var identities []bundle.Identity
	passphraseSkipped := 0
	for _, u := range selected {
		id := bundle.Identity{Name: u.Name, Email: u.Email}
		if u.SSHKey != "" {
			privKey, err := os.ReadFile(u.SSHKey)
			if err != nil {
				continue
			}
			protected, perr := isSSHKeyPassphraseProtected(u.SSHKey)
			if perr == nil && protected {
				passphraseSkipped++
				continue
			}
			id.PrivateKey = privKey
			id.PublicKey, _ = os.ReadFile(u.SSHKey + ".pub")
		}
		identities = append(identities, id)
	}

	encrypted, err := bundle.Encrypt(identities, passphrase)
	if err != nil {
		return opResult{}, fmt.Errorf("encrypting bundle: %w", err)
	}

	home, _ := os.UserHomeDir()
	baseName := fmt.Sprintf("git-user-export-%s", time.Now().Format("2006-01-02"))
	outPath := filepath.Join(home, baseName+".bundle")
	counter := 1
	for {
		if _, err := os.Stat(outPath); os.IsNotExist(err) {
			break
		}
		outPath = filepath.Join(home, fmt.Sprintf("%s-%d.bundle", baseName, counter))
		counter++
	}
	if err := os.WriteFile(outPath, encrypted, 0600); err != nil {
		return opResult{}, fmt.Errorf("writing bundle: %w", err)
	}

	report := fmt.Sprintf("Exported %d identit%s to %s\n", len(identities), plural(len(identities)), outPath)
	for _, id := range identities {
		report += fmt.Sprintf("  • %s (%s)\n", id.Name, id.Email)
	}
	if passphraseSkipped > 0 {
		report += fmt.Sprintf("\n⚠ %d identit%s exported without SSH keys (passphrase-protected).\n", passphraseSkipped, plural(passphraseSkipped))
	}
	report += "\nTransfer this file to your new machine, then import it from the Import/Export menu.\n"
	return opResult{detail: report, showReport: true}, nil
}

// opImport restores identities from an encrypted bundle. Conflicting identities
// are skipped unless force is true (in which case they are overwritten).
func opImport(store *config.Store, bundlePath, passphrase string, force bool) (opResult, error) {
	inPath := expandPath(bundlePath)
	data, err := os.ReadFile(inPath)
	if err != nil {
		return opResult{}, fmt.Errorf("reading bundle: %w", err)
	}
	identities, err := bundle.Decrypt(data, passphrase)
	if err != nil {
		return opResult{}, err
	}

	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return opResult{}, fmt.Errorf("creating .ssh directory: %w", err)
	}

	imported := 0
	skipped := 0
	report := ""
	for _, id := range identities {
		conflictMsg := ""
		if store.IsNameTaken(id.Name) {
			conflictMsg = fmt.Sprintf("Identity name %q is already taken", id.Name)
		} else if store.IsEmailTaken(id.Email) {
			conflictMsg = fmt.Sprintf("Email %q is already used by another identity", id.Email)
		}
		if conflictMsg != "" {
			if force {
				if store.IsNameTaken(id.Name) {
					_ = store.RemoveUser(id.Name, true)
				}
				if store.IsEmailTaken(id.Email) {
					for _, u := range store.Users {
						if u.Email == id.Email {
							_ = store.RemoveUser(u.Name, true)
							break
						}
					}
				}
			} else {
				report += fmt.Sprintf("Skipped %q — conflict (%s). Remove the conflicting identity and try again.\n", id.Name, conflictMsg)
				skipped++
				continue
			}
		}
		if err := store.AddUser(id.Name, id.Email); err != nil {
			report += fmt.Sprintf("Could not add %q: %v\n", id.Name, err)
			continue
		}
		if len(id.PrivateKey) > 0 {
			keyPath := filepath.Join(sshDir, fmt.Sprintf("git_%s", id.Name))
			if err := os.WriteFile(keyPath, id.PrivateKey, 0600); err != nil {
				report += fmt.Sprintf("Could not write private key for %q: %v\n", id.Name, err)
				continue
			}
			if len(id.PublicKey) > 0 {
				_ = os.WriteFile(keyPath+".pub", id.PublicKey, 0644)
			}
			_ = store.BindSSHKey(id.Name, keyPath)
		}
		imported++
	}
	if err := config.Save(store); err != nil {
		return opResult{}, err
	}

	report = fmt.Sprintf("Imported %d identit%s. Switch to one to activate it.\n", imported, plural(imported)) + report
	if skipped > 0 {
		report += fmt.Sprintf("%d identit%s skipped (already exist).\n", skipped, plural(skipped))
	}
	return opResult{detail: report, showReport: true}, nil
}

// opImportOriginal imports the pre-git-user gitconfig identity under a chosen name.
func opImportOriginal(store *config.Store, name, email string) (opResult, error) {
	for _, u := range store.Users {
		if u.Source == "original" {
			return opResult{}, fmt.Errorf("original identity already imported as %q", u.Name)
		}
	}
	if store.FindUser(name) != nil {
		return opResult{}, fmt.Errorf("identity %q already exists — use a different name", name)
	}

	var origName, origEmail, sshCommand string
	if store.Original != nil {
		origName = store.Original.Name
		origEmail = store.Original.Email
		sshCommand = store.Original.SSHCommand
	} else {
		origName = git.CurrentName()
		origEmail = git.CurrentEmail()
		sshCommand = git.CurrentSSHCommand()
	}

	if name == "" {
		name = origName
		if name == "" {
			name = "original"
		}
	}
	if email == "" {
		email = origEmail
	}
	if email == "" {
		return opResult{}, fmt.Errorf("email is required")
	}

	sshKey := extractSSHKeyFromCommand(sshCommand)
	store.Users = append(store.Users, config.User{
		Name:       name,
		Email:      email,
		SSHKey:     sshKey,
		SSHCommand: sshCommand,
		Source:     "original",
	})
	store.Current = name
	store.SnapshotOriginal(origName, origEmail, sshCommand, git.CurrentSigningKey(), git.CurrentSignFormat(), git.CurrentCommitGPGSign())
	if err := config.Save(store); err != nil {
		return opResult{}, err
	}

	report := fmt.Sprintf("Imported original identity as %q and set it active\n", name)
	report += fmt.Sprintf("  Email: %s\n", email)
	if sshKey != "" {
		report += fmt.Sprintf("  SSH Key: %s\n", sshKey)
	}
	if sshCommand != "" {
		report += fmt.Sprintf("  SSH Command: %s\n", sshCommand)
	}
	report += "\nIt stays your active identity. Switch away from the dashboard.\n"
	return opResult{detail: report, showReport: true}, nil
}

func extractSSHKeyFromCommand(cmd string) string {
	parts := strings.Fields(cmd)
	for i, p := range parts {
		if p == "-i" && i+1 < len(parts) {
			return strings.Trim(parts[i+1], `"'`)
		}
	}
	return ""
}

// ── Reports ───────────────────────────────────────────────────────────────────

// opPubkey returns the active identity's public key text.
func opPubkey(store *config.Store, name string) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	if name != store.Current {
		return opResult{}, fmt.Errorf("to view %s's public key, switch to it first", name)
	}
	if user.SSHKey == "" {
		return opResult{}, fmt.Errorf("identity %q has no SSH key bound", name)
	}
	pubKeyPath := user.SSHKey + ".pub"
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return opResult{}, fmt.Errorf("public key not found at %s", pubKeyPath)
	}
	report := fmt.Sprintf("PUBLIC KEY — %s (%s)\n\n", user.Name, user.Email)
	report += strings.TrimSpace(string(pubKeyBytes)) + "\n"
	if fp, err := exec.Command("ssh-keygen", "-lf", pubKeyPath).Output(); err == nil {
		report += "\nFingerprint: " + strings.TrimSpace(string(fp)) + "\n"
	}
	report += "\nAdd this key to your Git platform(s):\n"
	report += "  GitHub:    Settings → SSH and GPG keys → New SSH key\n"
	report += "  GitLab:    Preferences → SSH Keys → Add new key\n"
	report += "  Bitbucket: Personal settings → SSH keys → Add key\n"
	report += "\nThe same public key can be added to multiple platforms.\n"
	report += "The private key stays on your machine and is never shared.\n"
	return opResult{detail: report, showReport: true}, nil
}

// opCheckSSH tests the SSH connection for an identity.
func opCheckSSH(store *config.Store, name string) (opResult, error) {
	user := store.FindUser(name)
	if user == nil {
		return opResult{}, fmt.Errorf("identity %q not found", name)
	}
	if user.SSHKey == "" {
		return opResult{}, fmt.Errorf("no SSH key bound to identity %q", name)
	}
	report := fmt.Sprintf("Checking SSH connection for %s (%s)\n\n", user.Name, user.Email)
	ok, detail := verifySSHConnectionWithKey(user.SSHKey)
	if ok {
		report += "SSH connection verified successfully!\n"
	} else {
		report += "SSH verification failed.\n"
		report += detail
	}
	if !ssh.IsSSHKeyLoaded(user.SSHKey) {
		report += "\nKey is not loaded in the SSH agent. Unlock it by switching to this identity.\n"
	}
	return opResult{detail: report, showReport: true}, nil
}

// verifySSHConnectionWithKey tests SSH auth across the major platforms.
func verifySSHConnectionWithKey(keyPath string) (bool, string) {
	platforms := []struct {
		host    string
		success []string
	}{
		{"git@github.com", []string{"Hi ", "successfully authenticated"}},
		{"git@gitlab.com", []string{"Welcome to GitLab", "successfully authenticated"}},
		{"git@bitbucket.org", []string{"logged in as", "successfully authenticated"}},
	}
	for _, p := range platforms {
		args := []string{"-T", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=5"}
		if keyPath != "" {
			args = append(args, "-i", keyPath, "-o", "IdentitiesOnly=yes")
		}
		args = append(args, p.host)
		cmd := exec.Command("ssh", args...)
		if keyPath != "" {
			cmd.Env = append(os.Environ(), "SSH_AUTH_SOCK=")
		}
		output, _ := cmd.CombinedOutput()
		out := string(output)
		for _, marker := range p.success {
			if strings.Contains(out, marker) {
				return true, ""
			}
		}
	}
	return false, "  • The public key may not be added to GitHub/GitLab yet\n  • The key may not be loaded in ssh-agent\n  • Network connectivity issues\n"
}

// opFixRemote converts HTTPS remotes to SSH.
func opFixRemote() (opResult, error) {
	if !git.IsInstalled() {
		return opResult{}, fmt.Errorf("git is not installed")
	}
	if !git.IsInRepo() {
		return opResult{}, fmt.Errorf("not in a git repository")
	}
	remotes, err := git.ListRemotes()
	if err != nil || len(remotes) == 0 {
		return opResult{}, fmt.Errorf("no remotes found")
	}
	converted := 0
	report := ""
	for _, remote := range remotes {
		url, err := git.GetRemoteURL(remote)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(url, "https://") {
			continue
		}
		sshURL, ok := git.ConvertHTTPSToSSH(url)
		if !ok {
			report += fmt.Sprintf("%s: could not convert %s\n", remote, url)
			continue
		}
		if err := git.SetRemoteURL(remote, sshURL); err != nil {
			report += fmt.Sprintf("%s: failed to update\n", remote)
			continue
		}
		report += fmt.Sprintf("%s: %s → %s\n", remote, url, sshURL)
		converted++
	}
	if converted == 0 {
		report = "All remotes already use SSH.\n"
	} else {
		report = fmt.Sprintf("Converted %d remote(s) to SSH.\n", converted) + report
	}
	return opResult{detail: report, showReport: true}, nil
}

// opSecurity audits identity and config file security.
func opSecurity(store *config.Store) (opResult, error) {
	report := ""
	issues := 0
	configPath := config.ConfigPath()
	if info, err := os.Stat(configPath); err == nil {
		mode := info.Mode().Perm()
		if mode != 0600 {
			report += fmt.Sprintf("⚠ Config file has insecure permissions: %o (fix: chmod 600 %s)\n", mode, configPath)
			issues++
		} else {
			report += "✓ Config file permissions OK (0600)\n"
		}
	}
	for _, user := range store.Users {
		report += fmt.Sprintf("\n%s (%s)\n", user.Name, user.Email)
		if user.SSHKey == "" {
			report += "  ⚠ No SSH key bound (bind one from the profile view)\n"
			issues++
			continue
		}
		info, err := os.Stat(user.SSHKey)
		if err != nil {
			report += fmt.Sprintf("  ⚠ SSH key not found: %s\n", user.SSHKey)
			issues++
			continue
		}
		mode := info.Mode().Perm()
		if mode != 0600 {
			report += fmt.Sprintf("  ⚠ Insecure key permissions: %o (fix: chmod 600 %s)\n", mode, user.SSHKey)
			issues++
		} else {
			report += fmt.Sprintf("  ✓ Permissions OK: %s\n", filepath.Base(user.SSHKey))
		}
		protected, err := isSSHKeyPassphraseProtected(user.SSHKey)
		if err != nil {
			report += "  ⚠ Could not verify passphrase protection\n"
			issues++
		} else if protected {
			report += "  ✓ Passphrase protected\n"
		} else {
			report += "  ⚠ No passphrase detected — use the Passphrase menu to add one\n"
			issues++
		}
	}
	report += "\n"
	if issues == 0 {
		report += "No security issues found\n"
	} else {
		report += fmt.Sprintf("Found %d security issue(s)\n", issues)
	}
	return opResult{detail: report, showReport: true}, nil
}

// opDoctor runs a health check.
func opDoctor(store *config.Store) (opResult, error) {
	report := ""
	issues := 0
	user := store.CurrentUser()
	if store.Current == "" || user == nil {
		report += "⚠ No active identity set — switch to one from the dashboard\n"
		issues++
	} else {
		report += fmt.Sprintf("✓ Active identity: %s (%s)\n", user.Name, user.Email)
	}
	if user != nil && store.Current != "" {
		gitName := git.CurrentGlobalName()
		gitEmail := git.CurrentGlobalEmail()
		if gitName != user.Name || gitEmail != user.Email {
			report += fmt.Sprintf("⚠ Git config mismatch: expected %s <%s>, got %s <%s>. Re-switch to resync.\n", user.Name, user.Email, gitName, gitEmail)
			issues++
		} else {
			report += "✓ Git config in sync\n"
		}
		if user.SSHKey != "" {
			if info, err := os.Stat(user.SSHKey); err != nil {
				report += fmt.Sprintf("⚠ SSH key file not found: %s\n", user.SSHKey)
				issues++
			} else {
				report += fmt.Sprintf("✓ SSH key exists: %s\n", user.SSHKey)
				_ = info
			}
		} else {
			report += "⚠ No SSH key configured for this identity\n"
			issues++
		}
	}
	if git.IsInstalled() {
		report += "✓ Git is installed\n"
	} else {
		report += "⚠ Git is not installed or not on PATH\n"
		issues++
	}
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		report += "⚠ ssh-keygen not found on PATH\n"
		issues++
	} else {
		report += "✓ ssh-keygen is available\n"
	}
	home, _ := os.UserHomeDir()
	if entries, err := os.ReadDir(filepath.Join(home, ".ssh")); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".backup") {
				report += fmt.Sprintf("ℹ Stale backup key found: ~/.ssh/%s (safe to delete once the new key works)\n", e.Name())
			}
		}
	}
	report += "\n"
	if issues == 0 {
		report += "All checks passed! Your git-user setup is healthy.\n"
	} else {
		report += fmt.Sprintf("Found %d issue(s). See suggestions above.\n", issues)
	}
	return opResult{detail: report, showReport: true}, nil
}

// ── Pubkey push ───────────────────────────────────────────────────────────────

// opPushKey attempts to publish the active identity's public key to a platform,
// preferring the platform CLI (gh/glab). It returns ErrNeedsCredential when a
// token / username+password must be entered by the user.
func opPushKey(store *config.Store, platform string) (opResult, error) {
	user := store.CurrentUser()
	if user == nil {
		return opResult{}, fmt.Errorf("no active identity is set")
	}
	if user.SSHKey == "" {
		return opResult{}, fmt.Errorf("no SSH key is bound to the active identity %q", user.Name)
	}
	pubKeyPath := user.SSHKey + ".pub"
	if _, err := os.ReadFile(pubKeyPath); err != nil {
		return opResult{}, fmt.Errorf("could not read public key file %s", pubKeyPath)
	}

	switch platform {
	case "github":
		if _, err := exec.LookPath("gh"); err == nil {
			if exec.Command("gh", "auth", "status").Run() == nil {
				addCmd := exec.Command("gh", "ssh-key", "add", pubKeyPath, "--title", fmt.Sprintf("git-user: %s", user.Name))
				if out, err := addCmd.CombinedOutput(); err == nil {
					return opResult{detail: "SSH key successfully added to GitHub via gh CLI!"}, nil
				} else {
					_ = out
				}
			}
		}
		return opResult{}, ErrNeedsCredential
	case "gitlab":
		if _, err := exec.LookPath("glab"); err == nil {
			if exec.Command("glab", "auth", "status").Run() == nil {
				addCmd := exec.Command("glab", "ssh-key", "add", pubKeyPath, "--title", fmt.Sprintf("git-user: %s", user.Name))
				if out, err := addCmd.CombinedOutput(); err == nil {
					return opResult{detail: "SSH key successfully added to GitLab via glab CLI!"}, nil
				} else {
					_ = out
				}
			}
		}
		return opResult{}, ErrNeedsCredential
	case "bitbucket":
		return opResult{}, ErrNeedsCredential
	}
	return opResult{}, fmt.Errorf("unsupported platform")
}

// opPushKeyWithCredential publishes a key using an API token (or username +
// app password for Bitbucket).
func opPushKeyWithCredential(store *config.Store, platform, username, credential string) (opResult, error) {
	user := store.CurrentUser()
	if user == nil {
		return opResult{}, fmt.Errorf("no active identity is set")
	}
	if user.SSHKey == "" {
		return opResult{}, fmt.Errorf("no SSH key is bound to the active identity %q", user.Name)
	}
	pubKeyPath := user.SSHKey + ".pub"
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return opResult{}, fmt.Errorf("could not read public key file %s", pubKeyPath)
	}
	pubKey := strings.TrimSpace(string(pubKeyBytes))

	switch platform {
	case "github":
		return pushKeyGitHub(user.Name, pubKey, credential)
	case "gitlab":
		return pushKeyGitLab(user.Name, pubKey, credential)
	case "bitbucket":
		return pushKeyBitbucket(user.Name, pubKey, username, credential)
	}
	return opResult{}, fmt.Errorf("unsupported platform")
}

func pushKeyGitHub(profileName, pubKey, token string) (opResult, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"title": fmt.Sprintf("git-user: %s", profileName),
		"key":   pubKey,
	})
	req, err := http.NewRequest("POST", "https://api.github.com/user/keys", bytes.NewBuffer(reqBody))
	if err != nil {
		return opResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return opResult{}, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		return opResult{detail: "SSH key successfully uploaded to GitHub!"}, nil
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnprocessableEntity && strings.Contains(string(body), "already in use") {
		return opResult{detail: "This SSH key is already associated with your GitHub account."}, nil
	}
	return opResult{}, fmt.Errorf("failed to upload key. Status: %s. Response: %s", resp.Status, string(body))
}

func pushKeyGitLab(profileName, pubKey, token string) (opResult, error) {
	host := "gitlab.com"
	if r, err := git.ListRemotes(); err == nil {
		for _, remote := range r {
			if url, err := git.GetRemoteURL(remote); err == nil {
				if h := detectGitLabHost(url); h != "" {
					host = h
					break
				}
			}
		}
	}
	reqBody, _ := json.Marshal(map[string]string{
		"title": fmt.Sprintf("git-user: %s", profileName),
		"key":   pubKey,
	})
	url := fmt.Sprintf("https://%s/api/v4/user/keys", host)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return opResult{}, err
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return opResult{}, fmt.Errorf("GitLab API request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		return opResult{detail: "SSH key successfully uploaded to GitLab!"}, nil
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(body), "has already been taken") {
		return opResult{detail: "This SSH key is already associated with your GitLab account."}, nil
	}
	return opResult{}, fmt.Errorf("failed to upload key. Status: %s. Response: %s", resp.Status, string(body))
}

func detectGitLabHost(url string) string {
	lower := strings.ToLower(url)
	if !strings.Contains(lower, "gitlab") {
		return ""
	}
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	if strings.Contains(url, "@") {
		parts := strings.SplitN(url, "@", 2)
		if len(parts) == 2 {
			sub := strings.SplitN(parts[1], ":", 2)
			return sub[0]
		}
	}
	parts := strings.SplitN(url, "/", 2)
	return parts[0]
}

func pushKeyBitbucket(profileName, pubKey, username, appPassword string) (opResult, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"label": fmt.Sprintf("git-user: %s", profileName),
		"key":   pubKey,
	})
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/users/%s/ssh-keys", username)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return opResult{}, err
	}
	req.SetBasicAuth(username, appPassword)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return opResult{}, fmt.Errorf("Bitbucket API request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		return opResult{detail: "SSH key successfully uploaded to Bitbucket!"}, nil
	}
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "already exists") || strings.Contains(string(body), "already in use") {
		return opResult{detail: "This SSH key is already associated with your Bitbucket account."}, nil
	}
	return opResult{}, fmt.Errorf("failed to upload key. Status: %s. Response: %s", resp.Status, string(body))
}
