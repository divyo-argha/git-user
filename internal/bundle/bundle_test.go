package bundle_test

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/bundle"
)

var testIdentities = []bundle.Identity{
	{Name: "work", Email: "work@example.com", PrivateKey: []byte("fake-private"), PublicKey: []byte("fake-public")},
	{Name: "personal", Email: "me@gmail.com"},
}

func TestRoundTrip(t *testing.T) {
	passphrase := "correct-horse-battery-staple"

	encrypted, err := bundle.Encrypt(testIdentities, passphrase)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := bundle.Decrypt(encrypted, passphrase)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if len(got) != len(testIdentities) {
		t.Fatalf("expected %d identities, got %d", len(testIdentities), len(got))
	}
	for i, id := range got {
		if id.Name != testIdentities[i].Name || id.Email != testIdentities[i].Email {
			t.Errorf("identity %d mismatch: got %+v", i, id)
		}
	}
	if string(got[0].PrivateKey) != "fake-private" {
		t.Errorf("private key not preserved: %s", got[0].PrivateKey)
	}
}

func TestWrongPassphrase(t *testing.T) {
	encrypted, err := bundle.Encrypt(testIdentities, "right-passphrase")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = bundle.Decrypt(encrypted, "wrong-passphrase")
	if err == nil {
		t.Fatal("expected error with wrong passphrase, got nil")
	}
}

func TestCorruptData(t *testing.T) {
	encrypted, err := bundle.Encrypt(testIdentities, "passphrase")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// flip a byte in the ciphertext region
	encrypted[len(encrypted)-1] ^= 0xFF

	_, err = bundle.Decrypt(encrypted, "passphrase")
	if err == nil {
		t.Fatal("expected error with corrupt data, got nil")
	}
}

func TestTooShort(t *testing.T) {
	_, err := bundle.Decrypt([]byte("tooshort"), "passphrase")
	if err == nil {
		t.Fatal("expected error for too-short input")
	}
}
