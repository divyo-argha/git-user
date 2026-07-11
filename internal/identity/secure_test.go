package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecureDelete(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_key")

	// Write some data to the file
	testData := []byte("sensitive SSH private key data that should be securely deleted")
	if err := os.WriteFile(testFile, testData, 0600); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testFile); err != nil {
		t.Fatalf("test file doesn't exist: %v", err)
	}

	// Secure delete the file
	if err := SecureDelete(testFile); err != nil {
		t.Fatalf("SecureDelete() error = %v", err)
	}

	// Verify file no longer exists
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("file still exists after SecureDelete()")
	}
}

func TestSecureDelete_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does_not_exist")

	// Should not error on non-existent file
	if err := SecureDelete(nonExistent); err != nil {
		t.Errorf("SecureDelete() on non-existent file error = %v, want nil", err)
	}
}

func TestSecureDelete_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_dir")

	if err := os.Mkdir(testDir, 0700); err != nil {
		t.Fatalf("creating test directory: %v", err)
	}

	// Should error when trying to delete a directory
	if err := SecureDelete(testDir); err == nil {
		t.Error("SecureDelete() on directory should error, got nil")
	}
}

func TestSecureDeleteKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	privateKey := filepath.Join(tmpDir, "test_key")
	publicKey := privateKey + ".pub"

	// Create both keys
	if err := os.WriteFile(privateKey, []byte("private key data"), 0600); err != nil {
		t.Fatalf("creating private key: %v", err)
	}
	if err := os.WriteFile(publicKey, []byte("public key data"), 0644); err != nil {
		t.Fatalf("creating public key: %v", err)
	}

	// Delete the key pair
	if err := SecureDeleteKeyPair(privateKey); err != nil {
		t.Fatalf("SecureDeleteKeyPair() error = %v", err)
	}

	// Verify both files are gone
	if _, err := os.Stat(privateKey); !os.IsNotExist(err) {
		t.Error("private key still exists after SecureDeleteKeyPair()")
	}
	if _, err := os.Stat(publicKey); !os.IsNotExist(err) {
		t.Error("public key still exists after SecureDeleteKeyPair()")
	}
}

func TestSecureDeleteKeyPair_MissingPublicKey(t *testing.T) {
	tmpDir := t.TempDir()
	privateKey := filepath.Join(tmpDir, "test_key")

	// Create only private key
	if err := os.WriteFile(privateKey, []byte("private key data"), 0600); err != nil {
		t.Fatalf("creating private key: %v", err)
	}

	// Should succeed even if public key doesn't exist
	if err := SecureDeleteKeyPair(privateKey); err != nil {
		t.Errorf("SecureDeleteKeyPair() with missing public key error = %v", err)
	}

	// Verify private key is gone
	if _, err := os.Stat(privateKey); !os.IsNotExist(err) {
		t.Error("private key still exists after SecureDeleteKeyPair()")
	}
}

func TestSecureDelete_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large_file")

	// Create a larger file (16KB)
	largeData := make([]byte, 16*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	if err := os.WriteFile(testFile, largeData, 0600); err != nil {
		t.Fatalf("creating large test file: %v", err)
	}

	// Secure delete should handle large files
	if err := SecureDelete(testFile); err != nil {
		t.Fatalf("SecureDelete() on large file error = %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("large file still exists after SecureDelete()")
	}
}
