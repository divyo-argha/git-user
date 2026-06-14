package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type User struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	SSHKey       string `json:"ssh_key,omitempty"`
	SignKey      string `json:"sign_key,omitempty"`
	SignFormat   string `json:"sign_format,omitempty"` // "ssh" or "gpg"
	SignDisabled bool   `json:"sign_disabled,omitempty"`
	Source       string `json:"source,omitempty"` // "original" or empty (manual)
	IsTemporary  bool   `json:"-"`
}

// OriginalConfig holds the gitconfig state that existed before git-user was first used.
type OriginalConfig struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	SSHCommand   string `json:"ssh_command,omitempty"`
	SignKey      string `json:"sign_key,omitempty"`
	SignFormat   string `json:"sign_format,omitempty"`
	CommitGPGSign string `json:"commit_gpgsign,omitempty"`
}

type Store struct {
	Current  string          `json:"current"`
	Users    []User          `json:"users"`
	Original *OriginalConfig `json:"original,omitempty"`
}

var configPath string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("cannot determine home directory: " + err.Error())
	}
	configPath = filepath.Join(home, ".git-users", "config.json")
}

func ConfigPath() string { return configPath }

func SetConfigPath(path string) { configPath = path }

func TempConfigPath() string {
	return filepath.Join(filepath.Dir(configPath), "temp.json")
}

func DeleteTempConfig() {
	_ = os.Remove(TempConfigPath())
}

func Load() (*Store, error) {
	var s Store
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parsing config: %w", err)
		}
	}

	tempData, err := os.ReadFile(TempConfigPath())
	if err == nil {
		var tempUsers []User
		if err := json.Unmarshal(tempData, &tempUsers); err == nil {
			for i := range tempUsers {
				tempUsers[i].IsTemporary = true
			}
			s.Users = append(s.Users, tempUsers...)
		}
	}

	return &s, nil
}

func Save(s *Store) error {
	var permUsers []User
	var tempUsers []User
	for _, u := range s.Users {
		if u.IsTemporary {
			tempUsers = append(tempUsers, u)
		} else {
			permUsers = append(permUsers, u)
		}
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if len(tempUsers) > 0 {
		tempData, _ := json.MarshalIndent(tempUsers, "", "  ")
		_ = os.WriteFile(TempConfigPath(), tempData, 0600)
	} else {
		DeleteTempConfig()
	}

	sToSave := *s
	sToSave.Users = permUsers

	data, err := json.MarshalIndent(&sToSave, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "config-*.json")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("writing config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Chmod(tmpName, 0600); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("setting permissions: %w", err)
	}
	if err := os.Rename(tmpName, configPath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("saving config: %w", err)
	}
	return nil
}

func (s *Store) FindUser(name string) *User {
	for i := range s.Users {
		if s.Users[i].Name == name {
			return &s.Users[i]
		}
	}
	return nil
}

// IsNameTaken returns true if any profile already uses this name.
func (s *Store) IsNameTaken(name string) bool {
	return s.FindUser(name) != nil
}

// IsEmailTaken returns true if any profile already uses this email.
func (s *Store) IsEmailTaken(email string) bool {
	for _, u := range s.Users {
		if u.Email == email {
			return true
		}
	}
	return false
}

func (s *Store) AddUser(name, email string) error {
	if name == "" || email == "" {
		return errors.New("name and email must not be empty")
	}
	if s.IsNameTaken(name) {
		return fmt.Errorf("user %q already exists", name)
	}
	if s.IsEmailTaken(email) {
		return fmt.Errorf("email %q already in use", email)
	}
	s.Users = append(s.Users, User{Name: name, Email: email})
	return nil
}

func (s *Store) RemoveUser(name string, force bool) error {
	if s.FindUser(name) == nil {
		return fmt.Errorf("user %q not found", name)
	}
	if s.Current == name && !force {
		return fmt.Errorf("user %q is currently active; use --force to remove", name)
	}
	filtered := s.Users[:0]
	for _, u := range s.Users {
		if u.Name != name {
			filtered = append(filtered, u)
		}
	}
	s.Users = filtered
	if s.Current == name {
		s.Current = ""
	}
	return nil
}

func (s *Store) UpdateUser(name, newEmail string) error {
	u := s.FindUser(name)
	if u == nil {
		return fmt.Errorf("user %q not found", name)
	}
	if newEmail == "" {
		return errors.New("email must not be empty")
	}
	u.Email = newEmail
	return nil
}

func (s *Store) BindSSHKey(name, keyPath string) error {
	u := s.FindUser(name)
	if u == nil {
		return fmt.Errorf("user %q not found", name)
	}
	u.SSHKey = keyPath
	return nil
}

func (s *Store) SetSigningKey(name, key, format string) error {
	u := s.FindUser(name)
	if u == nil {
		return fmt.Errorf("user %q not found", name)
	}
	u.SignKey = key
	u.SignFormat = format
	u.SignDisabled = false
	return nil
}

func (s *Store) ToggleSigning(name string, disabled bool) error {
	u := s.FindUser(name)
	if u == nil {
		return fmt.Errorf("user %q not found", name)
	}
	u.SignDisabled = disabled
	return nil
}

func (s *Store) SetCurrent(name string) error {
	if s.FindUser(name) == nil {
		return fmt.Errorf("user %q not found", name)
	}
	s.Current = name
	return nil
}

func (s *Store) CurrentUser() *User {
	if s.Current == "" {
		return nil
	}
	return s.FindUser(s.Current)
}

// SnapshotOriginal saves the current gitconfig state as the original, if not already saved.
// Should be called before the first switch.
func (s *Store) SnapshotOriginal(name, email, sshCommand, signKey, signFormat, commitGPGSign string) {
	if s.Original != nil {
		return // already saved, never overwrite
	}
	s.Original = &OriginalConfig{
		Name:         name,
		Email:        email,
		SSHCommand:   sshCommand,
		SignKey:      signKey,
		SignFormat:   signFormat,
		CommitGPGSign: commitGPGSign,
	}
}
