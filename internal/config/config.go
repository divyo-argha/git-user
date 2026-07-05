package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type User struct {
	Name         string   `json:"name"`
	Email        string   `json:"email"`
	SSHKey       string   `json:"ssh_key,omitempty"`
	SignKey      string   `json:"sign_key,omitempty"`
	SignFormat   string   `json:"sign_format,omitempty"` // "ssh" or "gpg"
	SignDisabled bool     `json:"sign_disabled,omitempty"`
	Source       string            `json:"source,omitempty"` // "original" or empty (manual)
	BindPaths    []string          `json:"bind_paths,omitempty"`
	CustomConfig map[string]string `json:"custom_config,omitempty"`
	IsTemporary  bool              `json:"-"`
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

type SyncConfig struct {
	RepoURL    string `json:"repo_url,omitempty"`
	DeviceName string `json:"device_name,omitempty"`
}

type Store struct {
	Current  string          `json:"current"`
	Users    []User          `json:"users"`
	Original *OriginalConfig `json:"original,omitempty"`
	Sync     *SyncConfig     `json:"sync,omitempty"`
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
	return syncIncludeIfs(s)
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

func (s *Store) BindPathToUser(name, path string) error {
	u := s.FindUser(name)
	if u == nil {
		return fmt.Errorf("user %q not found", name)
	}
	for _, p := range u.BindPaths {
		if p == path {
			return nil
		}
	}
	for _, other := range s.Users {
		if other.Name != name {
			for _, p := range other.BindPaths {
				if p == path {
					return fmt.Errorf("path %q is already bound to identity %q", path, other.Name)
				}
			}
		}
	}
	u.BindPaths = append(u.BindPaths, path)
	return nil
}

func (s *Store) UnbindPathFromUser(name, path string) error {
	u := s.FindUser(name)
	if u == nil {
		return fmt.Errorf("user %q not found", name)
	}
	found := false
	filtered := u.BindPaths[:0]
	for _, p := range u.BindPaths {
		if p == path {
			found = true
		} else {
			filtered = append(filtered, p)
		}
	}
	if !found {
		return fmt.Errorf("path %q not bound to identity %q", path, name)
	}
	u.BindPaths = filtered
	return nil
}

func syncIncludeIfs(s *Store) error {
	_, err := exec.LookPath("git")
	if err != nil {
		return nil
	}

	configDir := filepath.Dir(configPath)

	// Clean up old profile snippet files
	files, err := os.ReadDir(configDir)
	if err == nil {
		for _, f := range files {
			if !f.IsDir() && strings.HasPrefix(f.Name(), "profile-") && strings.HasSuffix(f.Name(), ".gitconfig") {
				profileName := strings.TrimSuffix(strings.TrimPrefix(f.Name(), "profile-"), ".gitconfig")
				user := s.FindUser(profileName)
				if user == nil || len(user.BindPaths) == 0 {
					_ = os.Remove(filepath.Join(configDir, f.Name()))
				}
			}
		}
	}

	// Write snippet files for users who have BindPaths
	for _, u := range s.Users {
		if len(u.BindPaths) == 0 {
			continue
		}
		snippetPath := filepath.Join(configDir, fmt.Sprintf("profile-%s.gitconfig", u.Name))
		var sb strings.Builder
		sb.WriteString("# Generated by git-user. DO NOT EDIT.\n")
		sb.WriteString("[user]\n")
		sb.WriteString(fmt.Sprintf("\tname = %s\n", u.Name))
		sb.WriteString(fmt.Sprintf("\temail = %s\n", u.Email))
		if u.SSHKey != "" {
			sb.WriteString("[core]\n")
			sb.WriteString(fmt.Sprintf("\tsshCommand = ssh -i %q -o IdentitiesOnly=yes\n", u.SSHKey))
		}
		if !u.SignDisabled && u.SignKey != "" {
			if u.SignFormat == "ssh" {
				sb.WriteString("[gpg]\n")
				sb.WriteString("\tformat = ssh\n")
			}
			sb.WriteString("[user]\n")
			sb.WriteString(fmt.Sprintf("\tsigningkey = %s\n", u.SignKey))
			sb.WriteString("[commit]\n")
			sb.WriteString("\tgpgsign = true\n")
		}

		err := os.WriteFile(snippetPath, []byte(sb.String()), 0600)
		if err != nil {
			return fmt.Errorf("writing snippet file %s: %w", snippetPath, err)
		}

		if len(u.CustomConfig) > 0 {
			for k, v := range u.CustomConfig {
				_ = exec.Command("git", "config", "--file", snippetPath, k, v).Run()
			}
		}
	}

	// 1. Get existing includeIf configurations managed by git-user
	existingKeys := make(map[string]string)
	cmd := exec.Command("git", "config", "--global", "--get-regexp", `includeif\..*\.path`)
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				k := parts[0]
				v := parts[1]
				if strings.Contains(v, "profile-") && strings.HasSuffix(v, ".gitconfig") {
					existingKeys[k] = v
				}
			}
		}
	}

	// 2. Build the map of desired includeIf configuration keys and values
	desiredKeys := make(map[string]string)
	for _, u := range s.Users {
		if len(u.BindPaths) == 0 {
			continue
		}
		snippetPath := filepath.Join(configDir, fmt.Sprintf("profile-%s.gitconfig", u.Name))

		for _, p := range u.BindPaths {
			normPath := normalizeBindPath(p)
			k := fmt.Sprintf("includeif.gitdir/i:%s.path", normPath)
			desiredKeys[k] = snippetPath
		}
	}

	// 3. Sync configs: Remove undesired/incorrectly valued keys
	for k, v := range existingKeys {
		desiredVal, found := desiredKeys[k]
		if !found || desiredVal != v {
			_ = exec.Command("git", "config", "--global", "--unset-all", k).Run()
		}
	}

	// 4. Sync configs: Set missing or changed keys
	for k, v := range desiredKeys {
		existingVal, found := existingKeys[k]
		if !found || existingVal != v {
			_ = exec.Command("git", "config", "--global", k, v).Run()
		}
	}

	return nil
}

func normalizeBindPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}
