package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ssh"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runCheckSSH(args []string) error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	var name string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") && name == "" {
			name = arg
		}
	}

	if name == "" {
		name = store.Current
	}

	if name == "" {
		if ui.IsPlainOutput(args) || ui.IsJSONOutput(args) {
			return fmt.Errorf("no active identity")
		}
		ui.Error("No active identity is set. Please specify one: git-user check-ssh <name>")
		return fmt.Errorf("no active identity")
	}

	user := store.FindUser(name)
	if user == nil {
		ui.Errorf("identity %q not found", name)
		return fmt.Errorf("user not found")
	}

	if user.SSHKey == "" {
		if ui.IsJSONOutput(args) {
			_ = json.NewEncoder(os.Stdout).Encode(struct {
				Name             string               `json:"name"`
				Email            string               `json:"email"`
				SSHKey           string               `json:"ssh_key"`
				Connections      []ssh.PlatformResult `json:"connections"`
				ConnectedCount   int                  `json:"connected_count"`
				NothingConnected bool                 `json:"nothing_connected"`
			}{
				Name:             user.Name,
				Email:            user.Email,
				SSHKey:           "",
				Connections:      []ssh.PlatformResult{},
				ConnectedCount:   0,
				NothingConnected: true,
			})
			return nil
		}
		if ui.IsPlainOutput(args) {
			fmt.Println("Nothing connected")
			return nil
		}
		ui.Warn(fmt.Sprintf("Identity %q has no SSH key configured.", user.Name))
		ui.Info("Status: Nothing connected")
		ui.Info(fmt.Sprintf("Bind or generate a key with: git-user bind-key %s", user.Name))
		return nil
	}

	if _, err := os.Stat(user.SSHKey); err != nil {
		ui.Errorf("SSH key file not found: %s", user.SSHKey)
		return err
	}

	// Unlock key if protected
	if err := ensureKeyUnlocked(user.SSHKey); err != nil {
		ui.Warn(fmt.Sprintf("Could not unlock key: %v", err))
	}

	results := ssh.CheckAllPlatforms(user.SSHKey)
	connectedCount := 0
	var connectedNames []string

	for _, res := range results {
		if res.Status == "connected" {
			connectedCount++
			if res.Username != "" {
				connectedNames = append(connectedNames, fmt.Sprintf("%s (%s)", res.Platform, res.Username))
			} else {
				connectedNames = append(connectedNames, res.Platform)
			}
		}
	}

	if ui.IsJSONOutput(args) {
		_ = json.NewEncoder(os.Stdout).Encode(struct {
			Name             string               `json:"name"`
			Email            string               `json:"email"`
			SSHKey           string               `json:"ssh_key"`
			Connections      []ssh.PlatformResult `json:"connections"`
			ConnectedCount   int                  `json:"connected_count"`
			NothingConnected bool                 `json:"nothing_connected"`
		}{
			Name:             user.Name,
			Email:            user.Email,
			SSHKey:           user.SSHKey,
			Connections:      results,
			ConnectedCount:   connectedCount,
			NothingConnected: connectedCount == 0,
		})
		return nil
	}

	if ui.IsPlainOutput(args) {
		for _, res := range results {
			if res.Status == "connected" && res.Username != "" {
				fmt.Printf("%s: connected (%s)\n", res.Platform, res.Username)
			} else {
				fmt.Printf("%s: %s\n", res.Platform, res.Status)
			}
		}
		if connectedCount == 0 {
			fmt.Println("Nothing connected")
		}
		return nil
	}

	ui.Banner(fmt.Sprintf("PLATFORM CONNECTIONS — %s (%s)", user.Name, user.Email))
	fmt.Println()
	ui.Info(fmt.Sprintf("SSH Key: %s", user.SSHKey))
	fmt.Println()

	for _, res := range results {
		switch res.Status {
		case "connected":
			if res.Username != "" {
				ui.Success(fmt.Sprintf("  • %-10s : Connected ✓ (%s)", res.Platform, res.Username))
			} else {
				ui.Success(fmt.Sprintf("  • %-10s : Connected ✓", res.Platform))
			}
		case "network_error":
			ui.Error(fmt.Sprintf("  • %-10s : Network error (could not connect)", res.Platform))
		case "not_added":
			ui.Warn(fmt.Sprintf("  • %-10s : Not connected (key not added)", res.Platform))
		default:
			ui.Warn(fmt.Sprintf("  • %-10s : Not connected", res.Platform))
		}
	}

	fmt.Println()
	ui.Divider()
	if connectedCount == 0 {
		ui.Warn("Nothing connected — this SSH key has not been added to GitHub, GitLab, or Bitbucket.")
		fmt.Println()
		ui.Info("To add this key to your platform:")
		ui.Info("  • View & copy public key : git-user pubkey")
		ui.Info("  • Or push it directly    : git-user pubkey publish")
	} else {
		ui.Success(fmt.Sprintf("Connected to %d platform(s): %s", connectedCount, strings.Join(connectedNames, ", ")))
	}
	ui.Divider()

	return nil
}
