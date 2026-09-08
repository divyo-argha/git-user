package keyring

import (
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

const keychainService = "git-user"

// httpsTokenService is a distinct keyring service from keychainService so an
// identity's SSH key passphrase and its HTTPS personal-access-token (used by
// git-user token, for hosts/proxies where SSH isn't available) never collide
// under the same (service, account) keyring entry.
const httpsTokenService = "git-user-https-token"

var (
	KeyringGet    = keyring.Get
	KeyringSet    = keyring.Set
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

// SetHTTPSToken stores a personal-access-token (or app password) for an
// identity, used as the HTTPS credential when SSH isn't available (corporate
// proxies, some CI runners). See internal/cli/token.go.
func SetHTTPSToken(profileName, token string) error {
	err := KeyringSet(httpsTokenService, profileName, token)
	return formatHeadlessError(err)
}

func GetHTTPSToken(profileName string) (string, error) {
	val, err := KeyringGet(httpsTokenService, profileName)
	if err == keyring.ErrNotFound {
		return val, err
	}
	return val, formatHeadlessError(err)
}

func DeleteHTTPSToken(profileName string) error {
	err := KeyringDelete(httpsTokenService, profileName)
	if err == keyring.ErrNotFound {
		return nil
	}
	return formatHeadlessError(err)
}

// HasHTTPSToken reports whether an identity has a stored HTTPS token, without
// exposing the token itself. Used by gitenv/doctor to decide whether to wire
// up core.askpass — errors (including "not found") are treated as false.
func HasHTTPSToken(profileName string) bool {
	val, err := KeyringGet(httpsTokenService, profileName)
	return err == nil && val != ""
}
