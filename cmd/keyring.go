package cmd

import (
	"github.com/zalando/go-keyring"
)

const keychainService = "git-user"

var (
	keyringGet    = keyring.Get
	keyringSet    = keyring.Set
	keyringDelete = keyring.Delete
)

func setKeychainPassphrase(profileName, passphrase string) error {
	return keyringSet(keychainService, profileName, passphrase)
}

func getKeychainPassphrase(profileName string) (string, error) {
	return keyringGet(keychainService, profileName)
}

func deleteKeychainPassphrase(profileName string) error {
	err := keyringDelete(keychainService, profileName)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}
