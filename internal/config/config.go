package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// User represents a stored Git identity.
type User struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	SSHKey string `json:"ssh_key,omitempty"`
}

// Store is the top-level config persisted to disk.
type Store struct {
	Current string `json:"current"` // username key (matches User.Name)
	Users   []User `json:"users"`
}

var configPath string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("cannot determine home directory: " + err.Error())
	}
	configPath = filepath.Join(home, ".git-users", "config.json")
}

// ConfigPath returns the path to the config file (useful for diagnostics).
func ConfigPath() string { return configPath }

// Load reads the config from disk. Returns an empty store if not found.
func Load() (*Store, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Store{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &s, nil
}

// Save writes the store to disk, creating directories as needed.
func Save(s *Store) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// FindUser returns the user with the given name, or nil.
func (s *Store) FindUser(name string) *User {
	for i := range s.Users {
		if s.Users[i].Name == name {
			return &s.Users[i]
		}
	}
	return nil
}

// AddUser appends a new user. Returns error on duplicate name.
func (s *Store) AddUser(name, email string) error {
	if name == "" || email == "" {
		return errors.New("name and email must not be empty")
	}
	if s.FindUser(name) != nil {
		return fmt.Errorf("user %q already exists", name)
	}
	s.Users = append(s.Users, User{Name: name, Email: email})
	return nil
}

// RemoveUser deletes the user by name. Refuses to remove the active user
// unless force is true.
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

// UpdateUser edits the email (and optionally the name key) of an existing user.
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

// BindSSHKey associates an SSH key path with an identity.
func (s *Store) BindSSHKey(name, keyPath string) error {
	u := s.FindUser(name)
	if u == nil {
		return fmt.Errorf("user %q not found", name)
	}
	u.SSHKey = keyPath
	return nil
}

// SetCurrent records the active user name (must already exist).
func (s *Store) SetCurrent(name string) error {
	if s.FindUser(name) == nil {
		return fmt.Errorf("user %q not found", name)
	}
	s.Current = name
	return nil
}

// CurrentUser returns the active User or nil.
func (s *Store) CurrentUser() *User {
	if s.Current == "" {
		return nil
	}
	return s.FindUser(s.Current)
}
