package identity

import (
	"fmt"
	"os"
)

// SecureDelete overwrites file with zeros before deletion
// Uses 3-pass overwrite for paranoid security
func SecureDelete(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, nothing to delete
			return nil
		}
		return fmt.Errorf("stat file: %w", err)
	}

	// Don't try to delete directories
	if info.IsDir() {
		return fmt.Errorf("cannot secure delete directory: %s", path)
	}

	// Open file for writing
	f, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("opening file for overwrite: %w", err)
	}
	defer f.Close()

	// Get file size
	size := info.Size()

	// Perform 3-pass overwrite with zeros
	zeros := make([]byte, 4096) // 4KB buffer
	for pass := 0; pass < 3; pass++ {
		// Seek to beginning
		if _, err := f.Seek(0, 0); err != nil {
			return fmt.Errorf("seeking to start (pass %d): %w", pass+1, err)
		}

		// Overwrite entire file
		remaining := size
		for remaining > 0 {
			toWrite := int64(len(zeros))
			if remaining < toWrite {
				toWrite = remaining
			}

			n, err := f.Write(zeros[:toWrite])
			if err != nil {
				return fmt.Errorf("writing zeros (pass %d): %w", pass+1, err)
			}
			remaining -= int64(n)
		}

		// Sync to disk to ensure data is written
		if err := f.Sync(); err != nil {
			return fmt.Errorf("syncing to disk (pass %d): %w", pass+1, err)
		}
	}

	// Close file before deletion
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing file: %w", err)
	}

	// Finally, unlink the file
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing file: %w", err)
	}

	return nil
}

// SecureDeleteKeyPair securely deletes both private and public key files
func SecureDeleteKeyPair(privateKeyPath string) error {
	publicKeyPath := privateKeyPath + ".pub"

	// Delete private key
	if err := SecureDelete(privateKeyPath); err != nil {
		return fmt.Errorf("deleting private key: %w", err)
	}

	// Delete public key (best effort - may not exist)
	if err := SecureDelete(publicKeyPath); err != nil {
		// Only return error if file exists but deletion failed
		if !os.IsNotExist(err) {
			return fmt.Errorf("deleting public key: %w", err)
		}
	}

	return nil
}
