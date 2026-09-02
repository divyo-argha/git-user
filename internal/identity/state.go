package identity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TempStateFile persists minimal metadata to detect orphaned files
type TempStateFile struct {
	ActiveKeys []TempKeyMetadata `json:"active_keys"`
	LastUpdate time.Time         `json:"last_update"`
}

// TempKeyMetadata tracks a temporary key across process restarts
type TempKeyMetadata struct {
	KeyPath      string    `json:"key_path"`
	IdentityName string    `json:"identity_name"`
	CreatedAt    time.Time `json:"created_at"`
	ProcessPID   int       `json:"process_pid"`
	Fingerprint  string    `json:"fingerprint,omitempty"`
}

// LoadTempState reads the temp-state.json file
func LoadTempState(stateFile string) (*TempStateFile, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty state on first run
			return &TempStateFile{
				ActiveKeys: []TempKeyMetadata{},
				LastUpdate: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var state TempStateFile
	if err := json.Unmarshal(data, &state); err != nil {
		// If state file is corrupted, return empty state and log warning
		// This allows recovery from corrupted state
		return &TempStateFile{
			ActiveKeys: []TempKeyMetadata{},
			LastUpdate: time.Now(),
		}, fmt.Errorf("parsing state file (recovered): %w", err)
	}

	return &state, nil
}

// SaveTempState writes the temp-state.json file
func SaveTempState(stateFile string, state *TempStateFile) error {
	// Update timestamp
	state.LastUpdate = time.Now()

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}

	// Write to temporary file first (atomic write)
	tmpFile := stateFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("writing temp state file: %w", err)
	}

	// Rename to actual file (atomic operation)
	if err := os.Rename(tmpFile, stateFile); err != nil {
		os.Remove(tmpFile) // Clean up temp file on error
		return fmt.Errorf("saving state file: %w", err)
	}

	return nil
}

// DefaultTempStateFile returns the path of the crash-safety state file used
// to detect orphaned temporary identity keys left behind by an unclean exit.
func DefaultTempStateFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".git-users", "temp-state.json"), nil
}

// RecordTempKey registers a newly created temporary identity's key in the
// crash-safety state file, so a later orphan scan (from any process, e.g. the
// next `git-user` invocation) can find and securely delete it if this
// process dies before the normal switch/logout cleanup path runs. Without
// this, OrphanDetector.Scan never has anything to find — creating a temp key
// is the one place that must record it.
func RecordTempKey(name, keyPath string) error {
	stateFile, err := DefaultTempStateFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(stateFile), 0700); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}
	return AddKeyToState(stateFile, TempKeyMetadata{
		KeyPath:      keyPath,
		IdentityName: name,
		CreatedAt:    time.Now(),
		ProcessPID:   os.Getpid(),
	})
}

// ForgetTempKey removes a temporary identity's key from the crash-safety
// state file once it has been cleaned up normally (switch-away or logout),
// so a later orphan scan doesn't carry a stale entry for an already-deleted
// key indefinitely.
func ForgetTempKey(keyPath string) error {
	stateFile, err := DefaultTempStateFile()
	if err != nil {
		return err
	}
	return RemoveKeyFromState(stateFile, keyPath)
}

// AddKeyToState adds a key to the state file
func AddKeyToState(stateFile string, meta TempKeyMetadata) error {
	state, err := LoadTempState(stateFile)
	if err != nil {
		return err
	}

	// Check if key already exists (update if so)
	found := false
	for i, existing := range state.ActiveKeys {
		if existing.KeyPath == meta.KeyPath {
			state.ActiveKeys[i] = meta
			found = true
			break
		}
	}

	// Add if not found
	if !found {
		state.ActiveKeys = append(state.ActiveKeys, meta)
	}

	return SaveTempState(stateFile, state)
}

// RemoveKeyFromState removes a key from the state file
func RemoveKeyFromState(stateFile, keyPath string) error {
	state, err := LoadTempState(stateFile)
	if err != nil {
		return err
	}

	// Filter out the key
	filtered := make([]TempKeyMetadata, 0, len(state.ActiveKeys))
	for _, meta := range state.ActiveKeys {
		if meta.KeyPath != keyPath {
			filtered = append(filtered, meta)
		}
	}

	state.ActiveKeys = filtered
	return SaveTempState(stateFile, state)
}
