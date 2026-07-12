package keyring

import (
	"github.com/zalando/go-keyring"
)

const keychainService = "git-user"

var (
	KeyringGet = keyring.Get
	KeyringSet = keyring.Set
	KeyringDelete = keyring.Delete
)

func SetKeychainPassphrase(profileName, passphrase string) error {
	return KeyringSet(keychainService, profileName, passphrase)
}

func GetKeychainPassphrase(profileName string) (string, error) {
	return KeyringGet(keychainService, profileName)
}

func DeleteKeychainPassphrase(profileName string) error {
	err := KeyringDelete(keychainService, profileName)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}
