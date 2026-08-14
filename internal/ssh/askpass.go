package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Environment variables carrying passphrases to the askpass helper script
// (see runViaAskpass). ssh-keygen/ssh-add are run with SSH_ASKPASS pointed at
// a small generated script instead of ever taking a passphrase as a command-
// line argument, since argv is visible to any other local user for the
// process's lifetime (via `ps` or /proc/<pid>/cmdline); environment
// variables of a child process are not.
const (
	EnvPassphrase    = "GIT_USER_PASSPHRASE"
	EnvOldPassphrase = "GIT_USER_OLD_PASSPHRASE"
	EnvNewPassphrase = "GIT_USER_NEW_PASSPHRASE"
)

// askpassScript answers ssh-keygen/ssh-add passphrase prompts (passed as
// argv[1]) from the environment variables above. It carries no secret
// material itself, so it is harmless even if briefly readable by others.
const askpassScript = `#!/bin/sh
case "$1" in
  *[Oo]ld*|*[Cc]urrent*) printf '%s\n' "$` + EnvOldPassphrase + `" ;;
  *[Nn]ew*|*[Aa]gain*|*[Cc]onfirm*) printf '%s\n' "$` + EnvNewPassphrase + `" ;;
  *)
    if [ -n "$` + EnvPassphrase + `" ]; then
      printf '%s\n' "$` + EnvPassphrase + `"
    elif [ -n "$` + EnvOldPassphrase + `" ]; then
      printf '%s\n' "$` + EnvOldPassphrase + `"
    else
      printf '%s\n' "$` + EnvNewPassphrase + `"
    fi
    ;;
esac
`

// runViaAskpass runs name(args...) with secrets supplied to a freshly
// generated SSH_ASKPASS helper script rather than on the command line.
func runViaAskpass(name string, args []string, secrets map[string]string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "git-user-askpass-*")
	if err != nil {
		return nil, fmt.Errorf("creating askpass helper: %w", err)
	}
	defer os.RemoveAll(dir)

	scriptPath := filepath.Join(dir, "askpass.sh")
	if err := os.WriteFile(scriptPath, []byte(askpassScript), 0o700); err != nil {
		return nil, fmt.Errorf("writing askpass helper: %w", err)
	}

	env := os.Environ()
	for k, v := range secrets {
		env = append(env, k+"="+v)
	}
	env = append(env, "SSH_ASKPASS="+scriptPath, "SSH_ASKPASS_REQUIRE=force")

	hasDisplay := false
	for _, e := range env {
		if strings.HasPrefix(e, "DISPLAY=") {
			hasDisplay = true
			break
		}
	}
	if !hasDisplay {
		env = append(env, "DISPLAY=dummy:0")
	}

	cmd := exec.Command(name, args...)
	cmd.Env = env
	return cmd.CombinedOutput()
}

// ChangeKeyPassphrase sets, changes, or removes (when newPassphrase is empty)
// the passphrase protecting the private key at keyPath, without ever placing
// a passphrase on the ssh-keygen command line.
func ChangeKeyPassphrase(keyPath, oldPassphrase, newPassphrase string) error {
	out, err := runViaAskpass("ssh-keygen", []string{"-p", "-f", keyPath}, map[string]string{
		EnvOldPassphrase: oldPassphrase,
		EnvNewPassphrase: newPassphrase,
	})
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// GenerateKey creates a new ed25519 key pair at keyPath with the given
// comment, protected by passphrase (which may be empty). A non-empty
// passphrase is supplied via SSH_ASKPASS rather than the command line.
func GenerateKey(keyPath, comment, passphrase string) error {
	args := []string{"-t", "ed25519", "-C", comment, "-f", keyPath}

	if passphrase == "" {
		out, err := exec.Command("ssh-keygen", append(args, "-N", "")...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	out, err := runViaAskpass("ssh-keygen", args, map[string]string{
		EnvPassphrase:    passphrase,
		EnvNewPassphrase: passphrase,
	})
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// verifyPassphraseViaAskpass asks ssh-keygen to attempt decrypting keyPath
// with passphrase, without placing the passphrase on the command line. It is
// used as a fallback for OpenSSH-format keys whose KDF golang's crypto/ssh
// does not support.
func verifyPassphraseViaAskpass(keyPath, passphrase string) ([]byte, error) {
	return runViaAskpass("ssh-keygen", []string{"-y", "-f", keyPath}, map[string]string{
		EnvPassphrase: passphrase,
	})
}
