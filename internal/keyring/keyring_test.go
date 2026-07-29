package keyring

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestFormatHeadlessError(t *testing.T) {
	errNone := formatHeadlessError(nil)
	if errNone != nil {
		t.Errorf("Expected nil, got %v", errNone)
	}

	errRegular := errors.New("something went wrong")
	errFormatted1 := formatHeadlessError(errRegular)
	if errFormatted1 != errRegular {
		t.Errorf("Expected unchanged error, got %v", errFormatted1)
	}

	errDbus := errors.New("dbus connection failed")
	errFormatted2 := formatHeadlessError(errDbus)
	if !errors.Is(errFormatted2, errDbus) || errFormatted2.Error() == errDbus.Error() {
		t.Errorf("Expected decorated headless error, got %v", errFormatted2)
	}
}

func TestSetGetKeychainPassphrase(t *testing.T) {
	// Simple mock database
	mockStore := make(map[string]string)

	// Override package variables
	KeyringGet = func(service, user string) (string, error) {
		val, ok := mockStore[user]
		if !ok {
			return "", keyring.ErrNotFound
		}
		return val, nil
	}
	KeyringSet = func(service, user, password string) error {
		mockStore[user] = password
		return nil
	}
	KeyringDelete = func(service, user string) error {
		_, ok := mockStore[user]
		if !ok {
			return keyring.ErrNotFound
		}
		delete(mockStore, user)
		return nil
	}

	// Restore original function pointers after test
	defer func() {
		KeyringGet = keyring.Get
		KeyringSet = keyring.Set
		KeyringDelete = keyring.Delete
	}()

	// Test Set
	err := SetKeychainPassphrase("test-profile", "secret123")
	if err != nil {
		t.Fatalf("Failed to set passphrase: %v", err)
	}

	// Test Get
	pass, err := GetKeychainPassphrase("test-profile")
	if err != nil {
		t.Fatalf("Failed to get passphrase: %v", err)
	}
	if pass != "secret123" {
		t.Errorf("Expected secret123, got %q", pass)
	}

	// Test Get missing
	_, err = GetKeychainPassphrase("missing")
	if err != keyring.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}

	// Test Delete
	err = DeleteKeychainPassphrase("test-profile")
	if err != nil {
		t.Fatalf("Failed to delete passphrase: %v", err)
	}

	// Test Delete missing should return nil
	err = DeleteKeychainPassphrase("missing")
	if err != nil {
		t.Errorf("Expected nil error for deleting missing, got %v", err)
	}
}
