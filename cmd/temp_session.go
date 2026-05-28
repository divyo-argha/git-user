package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

type tempSession struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	KeyPath     string `json:"key_path"`
	PrevName    string `json:"prev_name"`
	PrevEmail   string `json:"prev_email"`
	PrevSSHKey  string `json:"prev_ssh_key"`
}

func tempSessionPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".git-users", "temp_session.json")
}

func loadTempSession() (*tempSession, error) {
	data, err := os.ReadFile(tempSessionPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ts tempSession
	if err := json.Unmarshal(data, &ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

func saveTempSession(ts *tempSession) error {
	path := tempSessionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func removeTempSessionFile() {
	os.Remove(tempSessionPath())
}

func startTempSession(args []string) error {
	name, email, ttl, err := parseTempSessionArgs(args)
	if err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if err := ensureSSHAgent(); err != nil {
		return err
	}

	existing, err := loadTempSession()
	if err == nil && existing != nil {
		if isSSHKeyLoaded(existing.KeyPath) {
			ui.Warn(fmt.Sprintf("A temporary session for %q is already active", existing.Name))
			ui.Info("Run: git-user session stop to end it first")
			return fmt.Errorf("temp session already active")
		}
		cleanupTempSession(existing)
	}

	home, _ := os.UserHomeDir()
	keyPath := filepath.Join(home, ".ssh", fmt.Sprintf("git_tmp_%s", name))

	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0700); err != nil {
		return fmt.Errorf("creating .ssh directory: %w", err)
	}

	ui.Banner("TEMPORARY SESSION: " + name)
	fmt.Println()
	ui.Info("This identity will NOT be saved. Keys are deleted when the session ends.")
	fmt.Println()
	ui.Info(fmt.Sprintf("Generating temporary SSH key at %s...", keyPath))
	ui.Info("You will be prompted to set a passphrase for the key.")

	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-keygen failed: %w", err)
	}
	ui.Success("Temporary SSH key generated!")

	pubKeyBytes, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		os.Remove(keyPath)
		return fmt.Errorf("reading public key: %w", err)
	}

	fmt.Println()
	ui.Divider()
	ui.Banner("📋 YOUR TEMPORARY PUBLIC KEY")
	fmt.Println()
	fmt.Println(string(pubKeyBytes))
	ui.Divider()
	fmt.Println()
	ui.Info("Add this key to your Git platform:")
	fmt.Println("  GitHub:    Settings → SSH and GPG keys → New SSH key")
	fmt.Println("  GitLab:    Preferences → SSH Keys → Add new key")
	fmt.Println("  Bitbucket: Personal settings → SSH keys → Add key")
	fmt.Println()
	ui.Warn("Remember to REMOVE this key from your platform when done!")
	fmt.Println()

	_, _ = ui.Prompt("Press Enter once you've added the key...")

	fmt.Println()
	ui.Info("Loading key into ssh-agent...")
	if err := addSSHKey(keyPath, ttl); err != nil {
		os.Remove(keyPath)
		os.Remove(keyPath + ".pub")
		ui.Error("Failed to load key into agent")
		return err
	}

	prevName := git.CurrentName()
	prevEmail := git.CurrentEmail()
	prevSSHKey := ""
	if sshCmd, err := exec.Command("git", "config", "--global", "core.sshCommand").Output(); err == nil {
		prevSSHKey = strings.TrimSpace(string(sshCmd))
	}

	if err := git.Apply(name, email); err != nil {
		removeSSHKey(keyPath)
		os.Remove(keyPath)
		os.Remove(keyPath + ".pub")
		return fmt.Errorf("applying git config: %w", err)
	}
	if err := git.ConfigureSSH(keyPath); err != nil {
		removeSSHKey(keyPath)
		os.Remove(keyPath)
		os.Remove(keyPath + ".pub")
		git.Apply(prevName, prevEmail)
		return fmt.Errorf("configuring SSH: %w", err)
	}

	ts := &tempSession{
		Name:       name,
		Email:      email,
		KeyPath:    keyPath,
		PrevName:   prevName,
		PrevEmail:  prevEmail,
		PrevSSHKey: prevSSHKey,
	}
	if err := saveTempSession(ts); err != nil {
		ui.Warn("Could not save temp session state — cleanup may be manual")
	}

	fmt.Println()
	ui.Success("Temporary session started!")
	ui.Info(fmt.Sprintf("Identity: %s (%s)", name, email))
	if ttl != "" {
		ui.Info(fmt.Sprintf("Session timeout: %s", ttl))
	}
	fmt.Println()
	ui.Warn("This session is temporary. To end it:")
	ui.Info("  git-user session stop")
	ui.Warn("This will delete the SSH key files and restore your previous identity.")

	return nil
}

func cleanupTempSession(ts *tempSession) {
	if isSSHKeyLoaded(ts.KeyPath) {
		removeSSHKey(ts.KeyPath)
	}

	os.Remove(ts.KeyPath)
	os.Remove(ts.KeyPath + ".pub")

	if ts.PrevName != "" || ts.PrevEmail != "" {
		git.Apply(ts.PrevName, ts.PrevEmail)
	}
	if ts.PrevSSHKey != "" {
		exec.Command("git", "config", "--global", "core.sshCommand", ts.PrevSSHKey).Run()
	} else {
		git.RemoveSSHConfig()
	}

	removeTempSessionFile()
}

func autoCleanupExpiredTempSession() {
	ts, err := loadTempSession()
	if err != nil || ts == nil {
		return
	}
	if !isSSHKeyLoaded(ts.KeyPath) {
		cleanupTempSession(ts)
	}
}

func parseTempSessionArgs(args []string) (name, email, ttl string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ttl", "-t":
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("--ttl requires a duration")
			}
			ttl = args[i+1]
			i++
		default:
			if args[i] != "" && args[i][0] == '-' {
				return "", "", "", fmt.Errorf("unknown option %s", args[i])
			}
			if name == "" {
				name = args[i]
			} else if email == "" {
				email = args[i]
			} else {
				return "", "", "", fmt.Errorf("unexpected argument: %s", args[i])
			}
		}
	}
	if name == "" {
		return "", "", "", fmt.Errorf("usage: git-user session start --temp <name> <email> [--ttl <duration>]")
	}
	if email == "" {
		return "", "", "", fmt.Errorf("usage: git-user session start --temp <name> <email> [--ttl <duration>]")
	}
	return name, email, ttl, nil
}
