package validate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// Reserved identity names that collide with CLI commands or system keywords.
var reservedNames = map[string]bool{
	"active":          true,
	"add":             true,
	"audit":           true,
	"bind":            true,
	"bind-key":        true,
	"bind-path":       true,
	"check":           true,
	"check-ssh":       true,
	"clone":           true,
	"completion":      true,
	"config":          true,
	"connections":     true,
	"current":         true,
	"del":             true,
	"delete":          true,
	"doctor":          true,
	"edit":            true,
	"export":          true,
	"fix-remote":      true,
	"git-user":        true,
	"gu":              true,
	"help":            true,
	"hook":            true,
	"import":          true,
	"import-original": true,
	"list":            true,
	"lo":              true,
	"logout":          true,
	"ls":              true,
	"passphrase":      true,
	"prompt":          true,
	"pubkey":          true,
	"reg":             true,
	"register":        true,
	"rekey":           true,
	"remove":          true,
	"rename":          true,
	"rm":              true,
	"security":        true,
	"sign":            true,
	"signout":         true,
	"stats":           true,
	"sw":              true,
	"switch":          true,
	"sync":            true,
	"tui":             true,
	"unbind-path":     true,
	"update":          true,
	"upgrade":         true,
	"use":             true,
	"version":         true,
	"whoami":          true,
}

// IsReservedName reports whether a given name is a reserved command/subcommand keyword.
func IsReservedName(name string) bool {
	return reservedNames[strings.ToLower(strings.TrimSpace(name))]
}

func isAsciiLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isAsciiDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isAsciiAlphanumeric(r rune) bool {
	return isAsciiLetter(r) || isAsciiDigit(r)
}

// IdentityName validates an identity profile name.
//
// Rules:
//   - Cannot be empty or whitespace-only
//   - Cannot contain spaces, control characters, tabs, or newlines
//   - Max length: 64 characters
//   - Allowed characters: ASCII alphanumeric (a-z, A-Z, 0-9), hyphens (-), underscores (_), and dots (.)
//   - Must start with an ASCII alphanumeric character
//   - Must not end with a dot (.) or hyphen (-)
//   - Cannot contain consecutive dots ('..')
//   - Cannot contain path separators or dangerous characters (/ \ : ; ' " * ? < > | ,)
//   - Cannot be a reserved CLI command keyword
func IdentityName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("identity name cannot be empty")
	}

	if len(name) > 64 {
		return fmt.Errorf("identity name %q is too long (maximum 64 characters, got %d)", name, len(name))
	}

	// Check for whitespace and control chars
	for _, r := range name {
		if unicode.IsSpace(r) {
			return errors.New("identity name cannot contain spaces or whitespace")
		}
		if unicode.IsControl(r) {
			return errors.New("identity name cannot contain control characters")
		}
	}

	// Must start with alphanumeric
	first := rune(name[0])
	if !isAsciiAlphanumeric(first) {
		return fmt.Errorf("identity name %q must start with an ASCII letter or number (got %q)", name, string(first))
	}

	// Must not end with dot or hyphen
	last := rune(name[len(name)-1])
	if last == '.' || last == '-' {
		return fmt.Errorf("identity name %q cannot end with a dot or hyphen", name)
	}

	// Check for allowed characters only
	for _, r := range name {
		if !isAsciiAlphanumeric(r) && r != '-' && r != '_' && r != '.' {
			return fmt.Errorf("identity name %q contains invalid character %q (only ASCII letters, digits, dots, hyphens, and underscores are allowed)", name, string(r))
		}
	}

	// Check for path traversal dots
	if strings.Contains(name, "..") {
		return fmt.Errorf("identity name %q cannot contain consecutive dots ('..')", name)
	}

	// Check for reserved words
	if IsReservedName(name) {
		return fmt.Errorf("identity name %q is a reserved command name — please choose a different name", name)
	}

	return nil
}

// ── Email Validation ──────────────────────────────────────────────────────────

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)
)

// Email validates an email address according to RFC 5322 standard conventions.
//
// Rules:
//   - Cannot be empty
//   - Cannot contain spaces or control characters
//   - Max length: 254 characters (RFC 5321)
//   - Must contain exactly one '@'
//   - Local part cannot start/end with dot, cannot contain consecutive dots
//   - Domain part must contain at least one dot
//   - TLD must be at least 2 alphabetic characters
func Email(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return errors.New("email address cannot be empty")
	}

	if len(email) > 254 {
		return fmt.Errorf("email address is too long (maximum 254 characters, got %d)", len(email))
	}

	for _, r := range email {
		if unicode.IsSpace(r) {
			return errors.New("email address cannot contain spaces")
		}
		if unicode.IsControl(r) {
			return errors.New("email address cannot contain control characters")
		}
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return errors.New("email address must contain exactly one '@' symbol (e.g. user@example.com)")
	}

	local, domain := parts[0], parts[1]

	if local == "" {
		return errors.New("email local part (before '@') cannot be empty")
	}
	if len(local) > 64 {
		return errors.New("email local part (before '@') is too long (maximum 64 characters)")
	}
	if strings.HasPrefix(local, ".") || strings.HasSuffix(local, ".") {
		return errors.New("email local part cannot start or end with a dot")
	}
	if strings.Contains(local, "..") {
		return errors.New("email local part cannot contain consecutive dots ('..')")
	}

	if domain == "" {
		return errors.New("email domain part (after '@') cannot be empty")
	}
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return errors.New("email domain cannot start or end with a dot")
	}
	if strings.Contains(domain, "..") {
		return errors.New("email domain cannot contain consecutive dots ('..')")
	}
	if !strings.Contains(domain, ".") {
		return errors.New("email domain must include a top-level domain (e.g. .com, .org, .io)")
	}

	// Check TLD
	dotIdx := strings.LastIndex(domain, ".")
	tld := domain[dotIdx+1:]
	if len(tld) < 2 {
		return errors.New("email domain extension must be at least 2 characters long (e.g. .com, .io)")
	}
	for _, r := range tld {
		if !unicode.IsLetter(r) {
			return errors.New("email top-level domain extension must contain only letters")
		}
	}

	if !emailRegex.MatchString(email) {
		return errors.New("invalid email address format (expected format: user@example.com)")
	}

	return nil
}

// ── SSH Key Path Validation ───────────────────────────────────────────────────

// SSHKeyPath validates an SSH private or public key path.
//
// If mustExist is true, the path is expanded and verified to exist as a regular file.
func SSHKeyPath(path string, mustExist bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("SSH key path cannot be empty")
	}

	for _, r := range path {
		if r == 0 {
			return errors.New("SSH key path contains null bytes")
		}
		if unicode.IsControl(r) {
			return errors.New("SSH key path contains control characters")
		}
	}

	expanded := ExpandPath(path)

	if mustExist {
		info, err := os.Stat(expanded)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("SSH key file not found: %s", path)
			}
			return fmt.Errorf("cannot access SSH key file %s: %w", path, err)
		}
		if info.IsDir() {
			return fmt.Errorf("SSH key path %q is a directory, not a key file", path)
		}
	}

	return nil
}

// ── Bind Path / Directory Validation ──────────────────────────────────────────

// BindPath validates a directory path for auto-switching bindings.
//
// If mustExist is true, the path is expanded and checked to ensure it is an existing directory.
func BindPath(path string, mustExist bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("directory path cannot be empty")
	}

	for _, r := range path {
		if r == 0 {
			return errors.New("directory path contains null bytes")
		}
		if unicode.IsControl(r) {
			return errors.New("directory path contains control characters")
		}
	}

	expanded := ExpandPath(path)

	if mustExist {
		info, err := os.Stat(expanded)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("directory not found: %s", path)
			}
			return fmt.Errorf("cannot access directory %s: %w", path, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("path %q is a file, not a directory", path)
		}
	}

	return nil
}

// ── Git Config Key & Value Validation ─────────────────────────────────────────

// GitConfigKey validates a custom git configuration key name (e.g. "user.name", "core.autocrlf", "url.git@github.com:.insteadOf").
func GitConfigKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("git config key cannot be empty")
	}

	if len(key) > 256 {
		return errors.New("git config key is too long (maximum 256 characters)")
	}

	for _, r := range key {
		if unicode.IsSpace(r) {
			return errors.New("git config key cannot contain spaces")
		}
		if unicode.IsControl(r) || r == 0 {
			return errors.New("git config key contains invalid control characters")
		}
		if r == '=' || r == ';' || r == '#' || r == '"' || r == '\'' {
			return fmt.Errorf("git config key cannot contain character %q", string(r))
		}
	}

	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return fmt.Errorf("git config key %q must follow 'section.key' format (e.g. 'core.autocrlf', 'commit.gpgsign')", key)
	}

	// Section must be alphanumeric + hyphen
	section := parts[0]
	if section == "" {
		return errors.New("git config section cannot be empty")
	}
	for _, r := range section {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' {
			return fmt.Errorf("git config section %q contains invalid character %q", section, string(r))
		}
	}

	// Variable (last part) must be alphanumeric + hyphen
	variable := parts[len(parts)-1]
	if variable == "" {
		return errors.New("git config variable name cannot be empty")
	}
	for _, r := range variable {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' {
			return fmt.Errorf("git config variable %q contains invalid character %q", variable, string(r))
		}
	}

	return nil
}

// GitConfigValue validates a custom git configuration value to prevent multi-line injection.
func GitConfigValue(val string) error {
	for _, r := range val {
		if r == 0 {
			return errors.New("git config value contains null bytes")
		}
		if r == '\n' || r == '\r' {
			return errors.New("git config value cannot contain newlines")
		}
	}
	return nil
}

// ── Repository URL Validation ─────────────────────────────────────────────────

// RepoURL validates a repository URL for cloning and remote operations.
func RepoURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return errors.New("repository URL cannot be empty")
	}

	// Prevent command-line flag injection (e.g. "--upload-pack=...")
	if strings.HasPrefix(rawURL, "-") {
		return errors.New("repository URL cannot start with a hyphen ('-')")
	}

	for _, r := range rawURL {
		if r == 0 || unicode.IsControl(r) {
			return errors.New("repository URL contains invalid control characters")
		}
		if unicode.IsSpace(r) {
			return errors.New("repository URL cannot contain spaces")
		}
	}

	return nil
}

// ── Passphrase Validation ─────────────────────────────────────────────────────

// Passphrase validates an encryption passphrase.
func Passphrase(pass string, minLen int) error {
	if minLen > 0 && len(pass) < minLen {
		return fmt.Errorf("passphrase must be at least %d characters long (got %d)", minLen, len(pass))
	}
	for _, r := range pass {
		if r == 0 {
			return errors.New("passphrase contains invalid null bytes")
		}
	}
	return nil
}

// PassphraseMatch verifies that two entered passphrases match and satisfy minLen.
func PassphraseMatch(p1, p2 string, minLen int) error {
	if p1 != p2 {
		return errors.New("passphrases do not match")
	}
	return Passphrase(p1, minLen)
}

// ── Helper ────────────────────────────────────────────────────────────────────

// ExpandPath expands leading ~ to user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return filepath.Clean(path)
}
