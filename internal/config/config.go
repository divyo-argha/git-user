package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type User struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	SSHKey string `json:"ssh_key,omitempty"`
}

type Store struct {
	Current string `json:"current"`
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

func ConfigPath() string { return configPath }

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

func (s *Store) FindUser(name string) *User {
	for i := range s.Users {
		if s.Users[i].Name == name {
			return &s.Users[i]
		}
	}
	return nil
}

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
