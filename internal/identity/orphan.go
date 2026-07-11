package identity

import (
	"fmt"
	"os"
	"time"
)

// OrphanDetector finds and cleans up abandoned temporary files
type OrphanDetector struct {
	stateFile string
}

// OrphanedKey represents a temporary key that was abandoned
type OrphanedKey struct {
	KeyPath      string
	IdentityName string
	CreatedAt    time.Time
	Age          time.Duration
	ProcessPID   int
}

// NewOrphanDetector creates a new orphan detector
func NewOrphanDetector(stateFile string) *OrphanDetector {
	return &OrphanDetector{
		stateFile: stateFile,
	}
}

// Scan finds orphaned temporary keys
func (od *OrphanDetector) Scan() ([]OrphanedKey, error) {
	// Load state file
	state, err := LoadTempState(od.stateFile)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	var orphans []OrphanedKey
	now := time.Now()

	for _, meta := range state.ActiveKeys {
		// Check if the key file still exists
		if _, err := os.Stat(meta.KeyPath); os.IsNotExist(err) {
			// Key file doesn't exist, but it's in state - this is an orphan entry
			// We'll clean it from state but not report it as an orphan to delete
			continue
		}

		// Check if the process is still running
		isRunning := isProcessRunning(meta.ProcessPID)

		// Consider it orphaned if:
		// 1. Process is not running, OR
		// 2. Key is older than 24 hours (safety threshold)
		age := now.Sub(meta.CreatedAt)
		if !isRunning || age > 24*time.Hour {
			orphans = append(orphans, OrphanedKey{
				KeyPath:      meta.KeyPath,
				IdentityName: meta.IdentityName,
				CreatedAt:    meta.CreatedAt,
				Age:          age,
				ProcessPID:   meta.ProcessPID,
			})
		}
	}

	return orphans, nil
}

// CleanupOrphans securely deletes orphaned keys and updates state
func (od *OrphanDetector) CleanupOrphans(orphans []OrphanedKey) error {
	var errors []error

	for _, orphan := range orphans {
		// Secure delete the key pair
		if err := SecureDeleteKeyPair(orphan.KeyPath); err != nil {
			errors = append(errors, fmt.Errorf("deleting %s: %w", orphan.KeyPath, err))
			continue
		}

		// Remove from state file
		if err := RemoveKeyFromState(od.stateFile, orphan.KeyPath); err != nil {
			errors = append(errors, fmt.Errorf("removing from state %s: %w", orphan.KeyPath, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// On Unix systems, we can check if a process exists by sending signal 0
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 (null signal) to check if process exists
	err = process.Signal(os.Signal(nil))
	if err != nil {
		// Process doesn't exist or we don't have permission
		return false
	}

	return true
}

// CleanupOrphanedStateEntries removes state entries for keys that no longer exist
func (od *OrphanDetector) CleanupOrphanedStateEntries() error {
	state, err := LoadTempState(od.stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// Filter out entries where key file doesn't exist
	filtered := make([]TempKeyMetadata, 0, len(state.ActiveKeys))
	for _, meta := range state.ActiveKeys {
		if _, err := os.Stat(meta.KeyPath); err == nil {
			// Key file exists, keep the entry
			filtered = append(filtered, meta)
		}
	}

	// Update state if we removed any entries
	if len(filtered) != len(state.ActiveKeys) {
		state.ActiveKeys = filtered
		if err := SaveTempState(od.stateFile, state); err != nil {
			return fmt.Errorf("saving cleaned state: %w", err)
		}
	}

	return nil
}
