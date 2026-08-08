package identity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateManagement(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "temp-state.json")

	// 1. Load empty state (doesn't exist yet)
	state, err := LoadTempState(stateFile)
	if err != nil {
		t.Fatalf("LoadTempState failed: %v", err)
	}
	if len(state.ActiveKeys) != 0 {
		t.Errorf("Expected 0 active keys, got %d", len(state.ActiveKeys))
	}

	// 2. Save state
	meta := TempKeyMetadata{
		KeyPath:      "/path/to/key1",
		IdentityName: "work",
		CreatedAt:    time.Now(),
		ProcessPID:   1234,
		Fingerprint:  "fingerprint1",
	}
	state.ActiveKeys = append(state.ActiveKeys, meta)

	err = SaveTempState(stateFile, state)
	if err != nil {
		t.Fatalf("SaveTempState failed: %v", err)
	}

	// 3. Load populated state
	state, err = LoadTempState(stateFile)
	if err != nil {
		t.Fatalf("LoadTempState failed: %v", err)
	}
	if len(state.ActiveKeys) != 1 || state.ActiveKeys[0].IdentityName != "work" {
		t.Errorf("Unexpected active keys state: %+v", state.ActiveKeys)
	}

	// 4. Add key to state
	meta2 := TempKeyMetadata{
		KeyPath:      "/path/to/key2",
		IdentityName: "home",
		CreatedAt:    time.Now(),
		ProcessPID:   5678,
		Fingerprint:  "fingerprint2",
	}
	err = AddKeyToState(stateFile, meta2)
	if err != nil {
		t.Fatalf("AddKeyToState failed: %v", err)
	}

	// Add existing key should update
	meta2.Fingerprint = "fingerprint2-updated"
	err = AddKeyToState(stateFile, meta2)
	if err != nil {
		t.Fatalf("AddKeyToState update failed: %v", err)
	}

	state, _ = LoadTempState(stateFile)
	if len(state.ActiveKeys) != 2 || state.ActiveKeys[1].Fingerprint != "fingerprint2-updated" {
		t.Errorf("Expected 2 active keys with updated fingerprint, got %d and %+v", len(state.ActiveKeys), state.ActiveKeys)
	}

	// 5. Remove key from state
	err = RemoveKeyFromState(stateFile, "/path/to/key1")
	if err != nil {
		t.Fatalf("RemoveKeyFromState failed: %v", err)
	}

	state, _ = LoadTempState(stateFile)
	if len(state.ActiveKeys) != 1 || state.ActiveKeys[0].KeyPath != "/path/to/key2" {
		t.Errorf("Expected 1 active key after removal, got %d", len(state.ActiveKeys))
	}

	// 6. Test Corrupted file recovery
	err = os.WriteFile(stateFile, []byte("invalid-json"), 0600)
	if err != nil {
		t.Fatalf("Failed to corrupt state file: %v", err)
	}
	state, err = LoadTempState(stateFile)
	if err == nil {
		t.Error("Expected parsing error for invalid JSON")
	}
	if len(state.ActiveKeys) != 0 {
		t.Errorf("Expected empty active keys after recovery from corruption, got %d", len(state.ActiveKeys))
	}
}

func TestTempService(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock HOME to isolate test operations
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create directories to satisfy paths
	sshDir := filepath.Join(tmpDir, ".ssh")
	err := os.MkdirAll(sshDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create mock ssh dir: %v", err)
	}

	// 1. Create Service
	service, err := NewTempService()
	if err != nil {
		t.Fatalf("NewTempService failed: %v", err)
	}

	if service.GetTempDir() != sshDir {
		t.Errorf("Unexpected temp dir path: %q", service.GetTempDir())
	}

	// 2. Validate Temp Directory
	err = service.ValidateTempDirectory()
	if err != nil {
		t.Errorf("ValidateTempDirectory failed: %v", err)
	}

	// Test non-existent path
	service.tempDir = "/nonexistent/directory"
	err = service.ValidateTempDirectory()
	if err == nil {
		t.Error("Expected validation error for non-existent directory")
	}

	// Restore
	service.tempDir = sshDir

	// Test insecure permissions
	err = os.Chmod(sshDir, 0777)
	if err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}
	err = service.ValidateTempDirectory()
	if err == nil {
		t.Error("Expected validation error for insecure directory permissions")
	}
	// Restore permissions
	_ = os.Chmod(sshDir, 0700)

	// 3. Add, Get, Remove Key operations
	keyInfo := &TempKeyInfo{
		PrivateKeyPath: "/path/to/private",
		PublicKeyPath:  "/path/to/public",
		Fingerprint:    "fingerprint",
		CreatedAt:      time.Now(),
		IdentityName:   "work-temp",
	}

	err = service.AddKey("work-temp", keyInfo)
	if err != nil {
		t.Fatalf("AddKey failed: %v", err)
	}

	if len(service.GetActiveKeys()) != 1 {
		t.Errorf("Expected 1 active key, got %d", len(service.GetActiveKeys()))
	}

	retrieved, exists := service.GetKey("work-temp")
	if !exists || retrieved.Fingerprint != "fingerprint" {
		t.Errorf("Failed to retrieve key, got exists=%v", exists)
	}

	err = service.RemoveKey("work-temp")
	if err != nil {
		t.Fatalf("RemoveKey failed: %v", err)
	}

	_, exists = service.GetKey("work-temp")
	if exists {
		t.Error("Expected key to be removed")
	}

	if service.GetOrphanDetector() == nil {
		t.Error("Expected non-nil OrphanDetector")
	}
}

func TestOrphanDetector(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "temp-state.json")

	// Write mock state with:
	// - One key whose file doesn't exist (should be cleaned from state but not reported as orphan)
	// - One key whose process doesn't exist and key file exists (should be flagged as orphan)
	// - One key whose process exists (mocked using current PID) and key file exists (should NOT be flagged)

	key1Path := filepath.Join(tmpDir, "key1")
	key2Path := filepath.Join(tmpDir, "key2")
	key3Path := filepath.Join(tmpDir, "key3")

	err := os.WriteFile(key2Path, []byte("key2-content"), 0600)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	err = os.WriteFile(key3Path, []byte("key3-content"), 0600)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	state := &TempStateFile{
		ActiveKeys: []TempKeyMetadata{
			{
				KeyPath:      key1Path, // missing file
				IdentityName: "missing",
				CreatedAt:    time.Now(),
				ProcessPID:   999999,
			},
			{
				KeyPath:      key2Path, // exists, defunct pid
				IdentityName: "defunct",
				CreatedAt:    time.Now(),
				ProcessPID:   999999, // very unlikely to be active
			},
			{
				KeyPath:      key3Path, // exists, current pid
				IdentityName: "active",
				CreatedAt:    time.Now(),
				ProcessPID:   os.Getpid(),
			},
		},
	}

	data, _ := json.Marshal(state)
	_ = os.WriteFile(stateFile, data, 0600)

	detector := NewOrphanDetector(stateFile)

	// Scan
	orphans, err := detector.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should report defunct key
	if len(orphans) != 1 || orphans[0].IdentityName != "defunct" {
		t.Errorf("Expected 1 orphan ('defunct'), got %d: %+v", len(orphans), orphans)
	}

	// Cleanup Orphaned State Entries (removes missing file entries)
	err = detector.CleanupOrphanedStateEntries()
	if err != nil {
		t.Fatalf("CleanupOrphanedStateEntries failed: %v", err)
	}

	// Verify that 'missing' was removed from state file
	newState, _ := LoadTempState(stateFile)
	for _, k := range newState.ActiveKeys {
		if k.IdentityName == "missing" {
			t.Error("Expected 'missing' entry to be cleaned up from state")
		}
	}

	// CleanupOrphans (deletes defunct files and state entries)
	// We'll mock CleanupOrphans by calling it with the scanned orphan
	err = detector.CleanupOrphans(orphans)
	if err != nil {
		t.Fatalf("CleanupOrphans failed: %v", err)
	}

	// Verify defunct file was deleted
	if _, err := os.Stat(key2Path); !os.IsNotExist(err) {
		t.Error("Expected defunct key file to be deleted")
	}

	// Verify defunct entry was removed from state
	newState, _ = LoadTempState(stateFile)
	for _, k := range newState.ActiveKeys {
		if k.IdentityName == "defunct" {
			t.Error("Expected 'defunct' entry to be removed from state")
		}
	}
}

func TestIsProcessRunning(t *testing.T) {
	if isProcessRunning(-1) {
		t.Error("Expected false for negative PID")
	}
	if isProcessRunning(0) {
		t.Error("Expected false for PID 0")
	}
	if !isProcessRunning(os.Getpid()) {
		t.Error("Expected true for current process PID")
	}
}
