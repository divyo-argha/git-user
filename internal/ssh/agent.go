package ssh

import (
	"fmt"
	"net"
	"os"
	"os/exec"
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

// addSSHKeyWithPassphrase adds the SSH key to the agent using the provided passphrase.
// It tries in-process parsing and loading first, and falls back to a secure SSH_ASKPASS execution.
func AddSSHKeyWithPassphrase(keyPath, passphrase string) error {
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
					return nil
				}
			}
		}
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	cmd := exec.Command("ssh-add", keyPath)
	env := os.Environ()

	env = append(env, "GIT_USER_ASKPASS_MODE=true")
	env = append(env, "GIT_USER_PASSPHRASE="+passphrase)
	env = append(env, "SSH_ASKPASS="+exe)
	env = append(env, "SSH_ASKPASS_REQUIRE=force")

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

	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-add failed: %v, output: %s", err, string(out))
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
	var errParse error
	if passphrase == "" {
		_, errParse = ssh.ParseRawPrivateKey(data)
	} else {
		_, errParse = ssh.ParseRawPrivateKeyWithPassphrase(data, []byte(passphrase))
	}
	return errParse == nil
}
