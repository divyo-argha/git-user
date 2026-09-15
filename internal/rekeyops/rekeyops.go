package rekeyops

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ssh"
)

type Result struct {
	OldKeyPath     string
	NewKeyPath     string
	BackupPath     string
	HadOldKey      bool
	SignKeyCarried bool
}

// ResolveOldKeyPath returns the SSH key path currently bound to name,
// falling back to the identity's default key path if none is bound.
func ResolveOldKeyPath(store *config.Store, name string) (string, error) {
	user := store.FindUser(name)
	if user == nil {
		return "", fmt.Errorf("identity not found")
	}
	if user.SSHKey != "" {
		return user.SSHKey, nil
	}
	return config.DefaultSSHKeyPath(name)
}

// Rotate performs the mechanical parts of SSH key rotation shared by every
// caller: making room for the new key, backing up the old one, invoking
// generateKey to actually create the new key pair (the caller decides how —
// interactively via a terminal-attached ssh-keygen, or non-interactively
// with a supplied passphrase), restoring the backup on failure, rebinding
// the identity to the new key, and carrying SignKey forward if it pointed at
// the key being rotated out (the fix for the bug where rotation silently
// left commit signing pointed at a deleted key).
func Rotate(store *config.Store, name, newKeyPath string, generateKey func(newKeyPath string) error) (Result, error) {
	oldKeyPath, err := ResolveOldKeyPath(store, name)
	if err != nil {
		return Result{}, err
	}
	user := store.FindUser(name)
	signKeyWasOldKey := user.SignFormat == "ssh" && user.SignKey == oldKeyPath

	if newKeyPath == "" {
		newKeyPath = oldKeyPath
	}

	sshDir := filepath.Dir(newKeyPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return Result{}, fmt.Errorf("creating .ssh directory: %w", err)
	}
	if newKeyPath != oldKeyPath {
		if _, err := os.Stat(newKeyPath); err == nil {
			return Result{}, fmt.Errorf("a key already exists at %s", newKeyPath)
		}
	}

	backupPath := oldKeyPath + ".backup"
	hasOldKey := false
	if _, err := os.Stat(oldKeyPath); err == nil {
		hasOldKey = true
		if ssh.IsSSHKeyLoaded(oldKeyPath) {
			_ = ssh.RemoveSSHKey(oldKeyPath)
		}
		if err := os.Rename(oldKeyPath, backupPath); err != nil {
			return Result{}, fmt.Errorf("backing up key: %w", err)
		}
		if _, err := os.Stat(oldKeyPath + ".pub"); err == nil {
			os.Rename(oldKeyPath+".pub", backupPath+".pub")
		}
	}

	if err := generateKey(newKeyPath); err != nil {
		if hasOldKey {
			os.Rename(backupPath, oldKeyPath)
			os.Rename(backupPath+".pub", oldKeyPath+".pub")
		}
		return Result{OldKeyPath: oldKeyPath, BackupPath: backupPath, HadOldKey: hasOldKey}, fmt.Errorf("generating SSH key: %w", err)
	}

	if err := store.BindSSHKey(name, newKeyPath); err != nil {
		return Result{}, fmt.Errorf("binding new SSH key: %w", err)
	}

	signKeyCarried := false
	if signKeyWasOldKey {
		if err := store.SetSigningKey(name, newKeyPath, "ssh"); err == nil {
			signKeyCarried = true
		}
	}

	return Result{
		OldKeyPath:     oldKeyPath,
		NewKeyPath:     newKeyPath,
		BackupPath:     backupPath,
		HadOldKey:      hasOldKey,
		SignKeyCarried: signKeyCarried,
	}, nil
}
