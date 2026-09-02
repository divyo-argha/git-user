package ssh

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/divyo-argha/git-user/internal/ui"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func EnsureSSHAgent() error {
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		return nil
	}
	// On Windows, OpenSSH agent uses a named pipe — SSH_AUTH_SOCK won't be set
	// but ssh-add may still work. Let it try rather than failing early.
	if runtime.GOOS == "windows" {
		return nil
	}
	ui.Warn("ssh-agent is not running in this shell")
	ui.Info("Start it with:")
	fmt.Println(`  eval "$(ssh-agent -s)"`)
	ui.Info("Then try again.")
	return fmt.Errorf("ssh-agent not running")
}

func GetAgentClient() (agent.Agent, net.Conn, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, nil, fmt.Errorf("SSH_AUTH_SOCK not set")
	}
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, nil, err
	}
	return agent.NewClient(conn), conn, nil
}

func IsSSHKeyLoaded(keyPath string) bool {
	target, err := SSHKeyFingerprint(keyPath)
	if err != nil {
		return false
	}

	loaded, err := LoadedSSHKeyFingerprints()
	if err != nil {
		return false
	}

	for _, fingerprint := range loaded {
		if fingerprint == target {
			return true
		}
	}
	return false
}

func SSHKeyFingerprint(keyPath string) (string, error) {
	pubKeyPath := keyPath + ".pub"
	data, err := os.ReadFile(pubKeyPath)
	if err != nil {
		if _, errStat := os.Stat(pubKeyPath); errStat != nil {
			return "", errStat
		}
		output, errCmd := exec.Command("ssh-keygen", "-lf", pubKeyPath).Output()
		if errCmd != nil {
			return "", errCmd
		}
		return ParseSSHKeyFingerprint(string(output))
	}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		output, errCmd := exec.Command("ssh-keygen", "-lf", pubKeyPath).Output()
		if errCmd != nil {
			return "", errCmd
		}
		return ParseSSHKeyFingerprint(string(output))
	}

	return ssh.FingerprintSHA256(pubKey), nil
}

func LoadedSSHKeyFingerprints() ([]string, error) {
	client, conn, err := GetAgentClient()
	if err == nil {
		defer conn.Close()
		keys, errList := client.List()
		if errList == nil {
			fingerprints := make([]string, 0, len(keys))
			for _, key := range keys {
				fingerprints = append(fingerprints, ssh.FingerprintSHA256(key))
			}
			return fingerprints, nil
		}
	}

	output, errCmd := exec.Command("ssh-add", "-l").CombinedOutput()
	if errCmd != nil {
		outStr := strings.ToLower(string(output))
		if strings.Contains(outStr, "no identities") || strings.Contains(outStr, "empty") {
			return []string{}, nil
		}
		return nil, errCmd
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	fingerprints := make([]string, 0, len(lines))
	for _, line := range lines {
		fingerprint, errParse := ParseSSHKeyFingerprint(line)
		if errParse == nil {
			fingerprints = append(fingerprints, fingerprint)
		}
	}
	return fingerprints, nil
}

func ParseSSHKeyFingerprint(line string) (string, error) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 {
		return "", fmt.Errorf("missing fingerprint")
	}
	return fields[1], nil
}

// AddSSHKeyWithPassphrase adds the SSH key to the agent using the provided passphrase.
// It tries in-process parsing and loading first, and falls back to a secure SSH_ASKPASS execution.
func AddSSHKeyWithPassphrase(keyPath, passphrase string) error {
	if runtime.GOOS == "darwin" {
		_ = EnsureMacOSKeychainConfigured()
	}

	data, err := os.ReadFile(keyPath)
	if err == nil {
		var privKey interface{}
		var errParse error
		if passphrase == "" {
			privKey, errParse = ssh.ParseRawPrivateKey(data)
		} else {
			privKey, errParse = ssh.ParseRawPrivateKeyWithPassphrase(data, []byte(passphrase))
		}

		if errParse == nil {
			client, conn, errDial := GetAgentClient()
			if errDial == nil {
				defer conn.Close()
				errAdd := client.Add(agent.AddedKey{
					PrivateKey: privKey,
					Comment:    keyPath,
				})
				if errAdd == nil {
					if runtime.GOOS == "darwin" {
						_ = addKeyToMacOSKeychain(keyPath, passphrase)
					}
					return nil
				}
			}
		}
	}

	// Try with --apple-use-keychain / -K on macOS
	args := []string{keyPath}
	if runtime.GOOS == "darwin" {
		args = []string{"--apple-use-keychain", keyPath}
	}

	secrets := map[string]string{EnvPassphrase: passphrase}
	if _, err := runViaAskpass("ssh-add", args, secrets); err != nil {
		// Fallback to standard ssh-add keyPath
		outFallback, errFallback := runViaAskpass("ssh-add", []string{keyPath}, secrets)
		if errFallback != nil {
			return fmt.Errorf("ssh-add failed: %v, output: %s", errFallback, string(outFallback))
		}
	}
	if runtime.GOOS == "darwin" {
		_ = addKeyToMacOSKeychain(keyPath, passphrase)
	}
	return nil
}

func EnsureMacOSKeychainConfigured() error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	sshDir := filepath.Join(home, ".ssh")
	_ = os.MkdirAll(sshDir, 0700)
	configPath := filepath.Join(sshDir, "config")

	content := ""
	if data, err := os.ReadFile(configPath); err == nil {
		content = string(data)
	}

	if strings.Contains(content, "UseKeychain yes") && strings.Contains(content, "Host *") {
		return nil
	}

	block := "Host *\n  AddKeysToAgent yes\n  UseKeychain yes\n\n"
	newContent := block + content

	// Write via a temp file + rename instead of os.WriteFile directly: a
	// failure partway through a direct write (e.g. disk full) would truncate
	// the user's existing ~/.ssh/config, since WriteFile opens with O_TRUNC.
	// rename() is atomic, so the original file is only ever replaced whole.
	tmp, err := os.CreateTemp(sshDir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp ssh config: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(newContent); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp ssh config: %w", err)
	}
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("setting temp ssh config permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp ssh config: %w", err)
	}
	return os.Rename(tmpPath, configPath)
}

func addKeyToMacOSKeychain(keyPath, passphrase string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	for _, flag := range []string{"--apple-use-keychain", "-K"} {
		var err error
		if passphrase != "" {
			_, err = runViaAskpass("ssh-add", []string{flag, keyPath}, map[string]string{EnvPassphrase: passphrase})
		} else {
			_, err = exec.Command("ssh-add", flag, keyPath).CombinedOutput()
		}
		if err == nil {
			return nil
		}
	}
	return nil
}

func RemoveSSHKey(keyPath string) error {
	pubKeyPath := keyPath + ".pub"
	data, err := os.ReadFile(pubKeyPath)
	if err == nil {
		pubKey, _, _, _, errParse := ssh.ParseAuthorizedKey(data)
		if errParse == nil {
			client, conn, errDial := GetAgentClient()
			if errDial == nil {
				defer conn.Close()
				errRemove := client.Remove(pubKey)
				if errRemove == nil {
					return nil
				}
			}
		}
	}

	if _, err := os.Stat(pubKeyPath); err != nil {
		return fmt.Errorf("public key not found at %s", pubKeyPath)
	}
	cmd := exec.Command("ssh-add", "-d", pubKeyPath)
	return cmd.Run()
}

func VerifyPassphrase(keyPath, passphrase string) bool {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return false
	}
	if passphrase == "" {
		_, errParse := ssh.ParseRawPrivateKey(data)
		return errParse == nil
	}

	_, errParse := ssh.ParseRawPrivateKeyWithPassphrase(data, []byte(passphrase))
	if errParse == nil {
		return true
	}

	// Fall back to ssh-keygen validation for OpenSSH format keys with unsupported kdf.
	// The passphrase is supplied via SSH_ASKPASS (see verifyPassphraseViaAskpass),
	// never as a command-line argument, since argv is visible to other local
	// users via `ps`/`/proc/<pid>/cmdline`.
	out, errCmd := verifyPassphraseViaAskpass(keyPath, passphrase)
	if errCmd == nil && len(out) > 0 {
		outStr := strings.TrimSpace(string(out))
		if strings.HasPrefix(outStr, "ssh-") || strings.HasPrefix(outStr, "ecdsa-") || strings.HasPrefix(outStr, "sk-") {
			return true
		}
		if _, _, _, _, errPub := ssh.ParseAuthorizedKey(out); errPub == nil {
			return true
		}
	}

	return false
}

// IsPassphraseProtected reports whether the private key at keyPath is
// encrypted with a passphrase. This is the single source of truth for that
// check — internal/cli and internal/tui both call it instead of maintaining
// their own copies, so behavior can't silently drift between the two.
func IsPassphraseProtected(keyPath string) (bool, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return false, err
	}

	_, err = ssh.ParseRawPrivateKey(data)
	if err == nil {
		return false, nil
	}

	var passphraseErr *ssh.PassphraseMissingError
	if errors.As(err, &passphraseErr) {
		return true, nil
	}

	return false, err
}
