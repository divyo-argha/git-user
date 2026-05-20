package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runSession(args []string) error {
	if len(args) < 1 {
		ui.Error("usage: git-user session <start|stop|status> [name] [--ttl <duration>] [--all]")
		fmt.Println()
		ui.Info("Session management:")
		fmt.Println("  start  - Start authenticated session for current identity")
		fmt.Println("  stop   - End current identity session")
		fmt.Println("  status - Check if session is active")
		return fmt.Errorf("missing subcommand")
	}

	subcommand := args[0]

	switch subcommand {
	case "start":
		return startSession(args[1:])
	case "stop":
		return stopSession(args[1:])
	case "status":
		return sessionStatus()
	default:
		ui.Errorf("unknown session subcommand: %s", subcommand)
		return fmt.Errorf("unknown subcommand")
	}
}

func startSession(args []string) error {
	ui.Banner("START SESSION")
	fmt.Println()

	name, ttl, err := parseSessionStartArgs(args)
	if err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if err := ensureSSHAgent(); err != nil {
		return err
	}
	ui.Success("ssh-agent is running")

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := selectedSessionUser(store, name)
	if user == nil {
		if name != "" {
			ui.Errorf("identity %q not found", name)
			return fmt.Errorf("identity not found")
		}
		ui.Warn("No active identity")
		ui.Info("Run: git-user switch <name>")
		return fmt.Errorf("no active identity")
	}

	if user.SSHKey == "" {
		ui.Warn(fmt.Sprintf("Identity %q has no SSH key", user.Name))
		ui.Info(fmt.Sprintf("Run: git-user bind %s", user.Name))
		return fmt.Errorf("no ssh key")
	}

	if isSSHKeyLoaded(user.SSHKey) {
		ui.Success(fmt.Sprintf("Session already active for %q", user.Name))
		return nil
	}

	ui.Info(fmt.Sprintf("Adding SSH key for %q...", user.Name))
	if ttl != "" {
		ui.Info(fmt.Sprintf("Session timeout: %s", ttl))
	}
	fmt.Println()

	if err := addSSHKey(user.SSHKey, ttl); err != nil {
		ui.Error("Failed to add SSH key")
		ui.Info("Make sure your key has a passphrase: ssh-keygen -p -f " + user.SSHKey)
		return err
	}

	fmt.Println()
	ui.Success("Session started!")
	ui.Info(fmt.Sprintf("Identity: %s (%s)", user.Name, user.Email))
	ui.Info("You can now push without entering passphrase")
	fmt.Println()
	ui.Info("To end session: git-user session stop")
	ui.Info("Or just close your terminal")

	return nil
}

func stopSession(args []string) error {
	ui.Banner("STOP SESSION")
	fmt.Println()

	name, all, err := parseSessionStopArgs(args)
	if err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if err := ensureSSHAgent(); err != nil {
		return nil
	}

	if all {
		if err := removeAllSSHKeys(); err != nil {
			ui.Warn("Failed to remove keys from agent")
			return err
		}
		ui.Success("All SSH keys removed from agent")
		ui.Info("Session ended")
		return nil
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := selectedSessionUser(store, name)
	if user == nil {
		if name != "" {
			ui.Errorf("identity %q not found", name)
			return fmt.Errorf("identity not found")
		}
		ui.Info("No active identity")
		return nil
	}
	if user.SSHKey == "" {
		ui.Info(fmt.Sprintf("Identity %q has no SSH key", user.Name))
		return nil
	}

	if !isSSHKeyLoaded(user.SSHKey) {
		ui.Info(fmt.Sprintf("No active session for %q", user.Name))
		return nil
	}

	if err := removeSSHKey(user.SSHKey); err != nil {
		ui.Warn("Failed to remove key from agent")
		return err
	} else {
		ui.Success(fmt.Sprintf("SSH key removed for %q", user.Name))
	}

	ui.Info("Session ended")
	ui.Info(fmt.Sprintf("Current identity is still %q", user.Name))
	ui.Info("You'll need to authenticate again on next push")

	return nil
}

func sessionStatus() error {
	ui.Banner("SESSION STATUS")
	fmt.Println()

	if os.Getenv("SSH_AUTH_SOCK") == "" {
		ui.Info("No ssh-agent running")
		ui.Info("Start session: git-user session start")
		showCurrentSessionIdentity()
		return nil
	}

	ui.Success("✓ ssh-agent is running")

	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.Output()

	if err != nil {
		ui.Info("No keys loaded in agent")
		ui.Info("Start session: git-user session start")
		showCurrentSessionIdentity()
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		ui.Success(fmt.Sprintf("✓ %d key(s) loaded", len(lines)))
		fmt.Println()
		ui.Info("Loaded keys:")
		for _, line := range lines {
			fmt.Println("  " + line)
		}
	} else {
		ui.Info("No keys loaded in agent")
		ui.Info("Start session: git-user session start")
	}

	showCurrentSessionIdentity()

	return nil
}

func showCurrentSessionIdentity() {
	store, err := config.Load()
	if err == nil && store.Current != "" {
		fmt.Println()
		ui.Info(fmt.Sprintf("Current identity: %s", store.Current))
		if user := store.CurrentUser(); user != nil && user.SSHKey != "" {
			if isSSHKeyLoaded(user.SSHKey) {
				ui.Success("Current identity key is loaded")
			} else {
				ui.Warn("Current identity key is not loaded")
			}
		}
	}
}

func parseSessionStartArgs(args []string) (name, ttl string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ttl", "-t":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("--ttl requires a duration")
			}
			ttl = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", fmt.Errorf("unknown option %s", args[i])
			}
			if name != "" {
				return "", "", fmt.Errorf("only one identity name can be provided")
			}
			name = args[i]
		}
	}
	return name, ttl, nil
}

func parseSessionStopArgs(args []string) (name string, all bool, err error) {
	for _, arg := range args {
		switch arg {
		case "--all":
			all = true
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, fmt.Errorf("unknown option %s", arg)
			}
			if name != "" {
				return "", false, fmt.Errorf("only one identity name can be provided")
			}
			name = arg
		}
	}
	if all && name != "" {
		return "", false, fmt.Errorf("use either an identity name or --all, not both")
	}
	return name, all, nil
}

func selectedSessionUser(store *config.Store, name string) *config.User {
	if name != "" {
		return store.FindUser(name)
	}
	return store.CurrentUser()
}

func ensureSSHAgent() error {
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		return nil
	}
	ui.Warn("ssh-agent is not running in this shell")
	ui.Info("Start it with:")
	fmt.Println(`  eval "$(ssh-agent -s)"`)
	ui.Info("Then run: git-user session start")
	return fmt.Errorf("ssh-agent not running")
}

func isSSHKeyLoaded(keyPath string) bool {
	target, err := sshKeyFingerprint(keyPath)
	if err != nil {
		return false
	}

	loaded, err := loadedSSHKeyFingerprints()
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

func sshKeyFingerprint(keyPath string) (string, error) {
	pubKeyPath := keyPath + ".pub"
	if _, err := os.Stat(pubKeyPath); err != nil {
		return "", err
	}

	output, err := exec.Command("ssh-keygen", "-lf", pubKeyPath).Output()
	if err != nil {
		return "", err
	}
	return parseSSHKeyFingerprint(string(output))
}

func loadedSSHKeyFingerprints() ([]string, error) {
	output, err := exec.Command("ssh-add", "-l").Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	fingerprints := make([]string, 0, len(lines))
	for _, line := range lines {
		fingerprint, err := parseSSHKeyFingerprint(line)
		if err == nil {
			fingerprints = append(fingerprints, fingerprint)
		}
	}
	return fingerprints, nil
}

func parseSSHKeyFingerprint(line string) (string, error) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 {
		return "", fmt.Errorf("missing fingerprint")
	}
	return fields[1], nil
}

func addSSHKey(keyPath, ttl string) error {
	args := []string{}
	if ttl != "" {
		args = append(args, "-t", ttl)
	}
	args = append(args, keyPath)
	cmd := exec.Command("ssh-add", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func removeSSHKey(keyPath string) error {
	cmd := exec.Command("ssh-add", "-d", keyPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func removeAllSSHKeys() error {
	cmd := exec.Command("ssh-add", "-D")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
