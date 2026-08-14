package tui

import (
	"fmt"
	"github.com/divyo-argha/git-user/internal/bundle"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
			keyPath, err := config.DefaultSSHKeyPath(id.Name)
			if err != nil {
				report += fmt.Sprintf("Skipping %q: %v\n", id.Name, err)
				continue
			}
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
	if !isValidEmail(email) {
		return opResult{}, fmt.Errorf("invalid email format: %q", email)
	}
	if store.IsEmailTaken(email) {
		return opResult{}, fmt.Errorf("email %q is already used by another identity", email)
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
	store.ImportPrompted = true
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
