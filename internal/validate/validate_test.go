package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIdentityName(t *testing.T) {
	valid := []string{
		"work",
		"personal",
		"client-a",
		"corp_dev",
		"dev.team",
		"user123",
		"A",
		"a-b_c.d",
		"my-long-identity-name-with-numbers-123456",
	}
	for _, name := range valid {
		if err := IdentityName(name); err != nil {
			t.Errorf("IdentityName(%q) expected valid, got error: %v", name, err)
		}
	}

	invalid := []struct {
		name   string
		reason string
	}{
		{"", "empty"},
		{"   ", "whitespace only"},
		{"work personal", "contains space"},
		{"work\npersonal", "contains newline"},
		{"-work", "starts with hyphen"},
		{".work", "starts with dot"},
		{"_work", "starts with underscore"},
		{"work-", "ends with hyphen"},
		{"work.", "ends with dot"},
		{"work/personal", "contains slash"},
		{"work\\personal", "contains backslash"},
		{"work:personal", "contains colon"},
		{"work;personal", "contains semicolon"},
		{"work*personal", "contains asterisk"},
		{"work?personal", "contains question mark"},
		{"work<personal", "contains angle bracket"},
		{"work>personal", "contains angle bracket"},
		{"work|personal", "contains pipe"},
		{"work\"personal", "contains double quote"},
		{"work'personal", "contains single quote"},
		{"work..personal", "contains consecutive dots"},
		{"../etc/passwd", "path traversal"},
		{strings.Repeat("a", 65), "too long (>64)"},
		{"switch", "reserved keyword"},
		{"register", "reserved keyword"},
		{"help", "reserved keyword"},
		{"version", "reserved keyword"},
		{"tui", "reserved keyword"},
		{"git-user", "reserved keyword"},
		{"gu", "reserved keyword"},
		{"logout", "reserved keyword"},
	}

	for _, tc := range invalid {
		if err := IdentityName(tc.name); err == nil {
			t.Errorf("IdentityName(%q) [%s] expected error, got nil", tc.name, tc.reason)
		}
	}
}

func TestEmail(t *testing.T) {
	valid := []string{
		"user@example.com",
		"user.name+tag@sub.domain.org",
		"first_last@company.co.uk",
		"dev-team@open-source.io",
		"a@b.cd",
		"123@numbers.com",
	}
	for _, email := range valid {
		if err := Email(email); err != nil {
			t.Errorf("Email(%q) expected valid, got error: %v", email, err)
		}
	}

	invalid := []struct {
		email  string
		reason string
	}{
		{"", "empty"},
		{"   ", "whitespace only"},
		{"notanemail", "no @"},
		{"user@", "no domain"},
		{"@domain.com", "no local part"},
		{"user@domain", "no TLD"},
		{"user@domain.", "trailing dot"},
		{"user@.domain.com", "leading dot in domain"},
		{".user@domain.com", "leading dot in local"},
		{"user.@domain.com", "trailing dot in local"},
		{"user..name@domain.com", "consecutive dots in local"},
		{"user@domain..com", "consecutive dots in domain"},
		{"user@domain.c", "single letter TLD"},
		{"user name@domain.com", "space in local"},
		{"user@domain name.com", "space in domain"},
		{"user\n@domain.com", "newline in email"},
		{strings.Repeat("a", 65) + "@example.com", "local part too long"},
		{strings.Repeat("a", 250) + "@b.co", "total length too long"},
	}

	for _, tc := range invalid {
		if err := Email(tc.email); err == nil {
			t.Errorf("Email(%q) [%s] expected error, got nil", tc.email, tc.reason)
		}
	}
}

func TestSSHKeyPath(t *testing.T) {
	tmpDir := t.TempDir()
	validKeyFile := filepath.Join(tmpDir, "id_ed25519")
	if err := os.WriteFile(validKeyFile, []byte("fake-key-content"), 0600); err != nil {
		t.Fatal(err)
	}

	// Valid existing file
	if err := SSHKeyPath(validKeyFile, true); err != nil {
		t.Errorf("SSHKeyPath(validKeyFile, true) unexpected error: %v", err)
	}

	// Valid path without existence check
	if err := SSHKeyPath(filepath.Join(tmpDir, "non_existent_key"), false); err != nil {
		t.Errorf("SSHKeyPath(..., false) unexpected error: %v", err)
	}

	// Directory when file expected
	if err := SSHKeyPath(tmpDir, true); err == nil {
		t.Error("SSHKeyPath(dir, true) expected error, got nil")
	}

	// Non-existent file when mustExist is true
	if err := SSHKeyPath(filepath.Join(tmpDir, "missing_key"), true); err == nil {
		t.Error("SSHKeyPath(missing, true) expected error, got nil")
	}

	// Empty path
	if err := SSHKeyPath("", false); err == nil {
		t.Error("SSHKeyPath(\"\", false) expected error, got nil")
	}

	// Null byte
	if err := SSHKeyPath("path\x00with\x00null", false); err == nil {
		t.Error("SSHKeyPath with null byte expected error, got nil")
	}
}

func TestSSHKeyFilename(t *testing.T) {
	valid := []string{"git_work", "id_ed25519", "work-key", "my.key_2"}
	for _, name := range valid {
		if err := SSHKeyFilename(name); err != nil {
			t.Errorf("SSHKeyFilename(%q) expected valid, got error: %v", name, err)
		}
	}

	invalid := []struct {
		name   string
		reason string
	}{
		{"", "empty"},
		{"has/slash", "path separator"},
		{"has\\backslash", "path separator"},
		{"../escape", "traversal"},
		{".", "dot"},
		{"..", "dotdot"},
		{".hidden", "leading dot"},
		{"id_rsa.pub", "reserved .pub suffix"},
		{"id_rsa.backup", "reserved .backup suffix"},
		{"known_hosts", "reserved SSH filename"},
		{"config", "reserved SSH filename"},
		{"authorized_keys", "reserved SSH filename"},
		{"has space", "whitespace"},
		{"has\x00null", "control character"},
	}
	for _, tc := range invalid {
		if err := SSHKeyFilename(tc.name); err == nil {
			t.Errorf("SSHKeyFilename(%q) expected error (%s), got nil", tc.name, tc.reason)
		}
	}
}

func TestBindPath(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "regular_file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Valid directory
	if err := BindPath(tmpDir, true); err != nil {
		t.Errorf("BindPath(tmpDir, true) unexpected error: %v", err)
	}

	// File when directory expected
	if err := BindPath(filePath, true); err == nil {
		t.Error("BindPath(file, true) expected error, got nil")
	}

	// Missing directory when mustExist=true
	if err := BindPath(filepath.Join(tmpDir, "missing"), true); err == nil {
		t.Error("BindPath(missing, true) expected error, got nil")
	}

	// Empty path
	if err := BindPath("", false); err == nil {
		t.Error("BindPath(\"\", false) expected error, got nil")
	}
}

func TestGitConfigKeyAndValue(t *testing.T) {
	validKeys := []string{
		"user.name",
		"user.email",
		"core.autocrlf",
		"commit.gpgsign",
		"tag.forcesigntag",
		"url.git@github.com:.insteadof",
	}
	for _, k := range validKeys {
		if err := GitConfigKey(k); err != nil {
			t.Errorf("GitConfigKey(%q) expected valid, got error: %v", k, err)
		}
	}

	invalidKeys := []string{
		"",
		"   ",
		"nosubsection",
		"user.name=bad",
		"user.name;bad",
		"user name.test",
		"user\n.name",
	}
	for _, k := range invalidKeys {
		if err := GitConfigKey(k); err == nil {
			t.Errorf("GitConfigKey(%q) expected error, got nil", k)
		}
	}

	if err := GitConfigValue("valid value 123"); err != nil {
		t.Errorf("GitConfigValue valid unexpected error: %v", err)
	}
	if err := GitConfigValue("value\nwith\nnewline"); err == nil {
		t.Error("GitConfigValue with newline expected error, got nil")
	}
	if err := GitConfigValue("value\x00with\x00null"); err == nil {
		t.Error("GitConfigValue with null byte expected error, got nil")
	}
}

func TestRepoURL(t *testing.T) {
	valid := []string{
		"https://github.com/user/repo.git",
		"git@github.com:user/repo.git",
		"ssh://git@gitlab.com/user/repo.git",
		"http://localgit.dev/repo.git",
		"/path/to/local/repo",
	}
	for _, u := range valid {
		if err := RepoURL(u); err != nil {
			t.Errorf("RepoURL(%q) expected valid, got error: %v", u, err)
		}
	}

	invalid := []string{
		"",
		"--upload-pack=evil",
		"-c core.editor=evil",
		"https://github.com/user/repo with spaces",
		"https://github.com/user\n/repo",
	}
	for _, u := range invalid {
		if err := RepoURL(u); err == nil {
			t.Errorf("RepoURL(%q) expected error, got nil", u)
		}
	}
}

func TestPassphrase(t *testing.T) {
	if err := Passphrase("secret123", 8); err != nil {
		t.Errorf("Passphrase unexpected error: %v", err)
	}
	if err := Passphrase("short", 8); err == nil {
		t.Error("Passphrase expected error for short length, got nil")
	}
	if err := Passphrase("secret\x00123", 4); err == nil {
		t.Error("Passphrase expected error for null byte, got nil")
	}
	if err := PassphraseMatch("pass1", "pass2", 4); err == nil {
		t.Error("PassphraseMatch expected error for mismatched passphrases, got nil")
	}
	if err := PassphraseMatch("pass1234", "pass1234", 8); err != nil {
		t.Errorf("PassphraseMatch unexpected error: %v", err)
	}
}
