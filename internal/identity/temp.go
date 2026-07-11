package identity

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type TempService struct {
	tempDir        string
	activeKeys     map[string]*TempKeyInfo
	orphanDetector *OrphanDetector
	stateFile      string
}

type TempKeyInfo struct {
	PrivateKeyPath string
	PublicKeyPath  string
	Fingerprint    string
	CreatedAt      time.Time
	IdentityName   string
}

func NewTempService() (*TempService, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	tempDir := filepath.Join(home, ".ssh")
	stateFile := filepath.Join(home, ".git-users", "temp-state.json")

	gitUsersDir := filepath.Join(home, ".git-users")
	if err := os.MkdirAll(gitUsersDir, 0700); err != nil {
		return nil, fmt.Errorf("creating .git-users directory: %w", err)
	}

	service := &TempService{
		tempDir:    tempDir,
		activeKeys: make(map[string]*TempKeyInfo),
		stateFile:  stateFile,
	}

	service.orphanDetector = NewOrphanDetector(stateFile)

	if err := service.loadState(); err != nil {
	}

	return service, nil
}

func (ts *TempService) ValidateTempDirectory() error {
	info, err := os.Stat(ts.tempDir)
	if err != nil {
		return fmt.Errorf("temp directory not accessible: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("temp path is not a directory: %s", ts.tempDir)
	}

	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		return fmt.Errorf("insecure temp directory permissions: %o (expected 0700)", mode)
	}

	return nil
}

func (ts *TempService) GetTempDir() string {
	return ts.tempDir
}

func (ts *TempService) GetActiveKeys() map[string]*TempKeyInfo {
	return ts.activeKeys
}

func (ts *TempService) GetOrphanDetector() *OrphanDetector {
	return ts.orphanDetector
}

func (ts *TempService) AddKey(identityName string, keyInfo *TempKeyInfo) error {
	ts.activeKeys[identityName] = keyInfo
	return ts.saveState()
}

func (ts *TempService) RemoveKey(identityName string) error {
	delete(ts.activeKeys, identityName)
	return ts.saveState()
}

func (ts *TempService) GetKey(identityName string) (*TempKeyInfo, bool) {
	keyInfo, exists := ts.activeKeys[identityName]
	return keyInfo, exists
}

func (ts *TempService) loadState() error {
	state, err := LoadTempState(ts.stateFile)
	if err != nil {
		return err
	}

	for _, meta := range state.ActiveKeys {
		keyInfo := &TempKeyInfo{
			PrivateKeyPath: meta.KeyPath,
			PublicKeyPath:  meta.KeyPath + ".pub",
			Fingerprint:    meta.Fingerprint,
			CreatedAt:      meta.CreatedAt,
			IdentityName:   meta.IdentityName,
		}
		ts.activeKeys[meta.IdentityName] = keyInfo
	}

	return nil
}

func (ts *TempService) saveState() error {
	state := &TempStateFile{
		ActiveKeys: make([]TempKeyMetadata, 0, len(ts.activeKeys)),
		LastUpdate: time.Now(),
	}

	for _, keyInfo := range ts.activeKeys {
		meta := TempKeyMetadata{
			KeyPath:      keyInfo.PrivateKeyPath,
			IdentityName: keyInfo.IdentityName,
			CreatedAt:    keyInfo.CreatedAt,
			ProcessPID:   os.Getpid(),
			Fingerprint:  keyInfo.Fingerprint,
		}
		state.ActiveKeys = append(state.ActiveKeys, meta)
	}

	return SaveTempState(ts.stateFile, state)
}
