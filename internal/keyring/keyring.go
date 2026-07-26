package keyring

import (
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

const keychainService = "git-user"

var (
	KeyringGet = keyring.Get
	KeyringSet = keyring.Set
	KeyringDelete = keyring.Delete
)

func formatHeadlessError(err error) error {
	if err == nil {
		return nil
	}
	
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "dbus") || strings.Contains(errStr, "secret") || strings.Contains(errStr, "unsupported") {
		return fmt.Errorf("%w (headless/SSH session? GUI keychain may be unavailable)", err)
	}
	return err
}

func SetKeychainPassphrase(profileName, passphrase string) error {
	err := KeyringSet(keychainService, profileName, passphrase)
	return formatHeadlessError(err)
}

func GetKeychainPassphrase(profileName string) (string, error) {
	val, err := KeyringGet(keychainService, profileName)
	if err == keyring.ErrNotFound {
		return val, err
	}
	return val, formatHeadlessError(err)
}

func DeleteKeychainPassphrase(profileName string) error {
	err := KeyringDelete(keychainService, profileName)
	if err == keyring.ErrNotFound {
		return nil
	}
	return formatHeadlessError(err)
}
