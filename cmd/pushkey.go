package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runPubkeyPush(args []string) error {
	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := store.CurrentUser()
	if user == nil {
		ui.Error("No active identity is set. Please switch to an identity first.")
		return fmt.Errorf("no active identity")
	}

	if user.SSHKey == "" {
		ui.Errorf("No SSH key is bound to the active identity %q.", user.Name)
		ui.Info("You can bind an existing key with: git-user bind " + user.Name + " --ssh-key <path>")
		return fmt.Errorf("no SSH key bound")
	}

	pubKeyPath := user.SSHKey + ".pub"
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		ui.Errorf("Could not read public key file %s: %v", pubKeyPath, err)
		return err
	}
	pubKey := strings.TrimSpace(string(pubKeyBytes))

	platform := ""
	customGitLabHost := "gitlab.com"

	if len(args) > 0 {
		platform = strings.ToLower(args[0])
	} else {
		// Auto-detect from git remotes
		platform, customGitLabHost = detectPlatformFromRemotes()
	}

	if platform != "github" && platform != "gitlab" && platform != "bitbucket" {
		idx, err := ui.Select("Select the Git platform to push your SSH key to:", []string{
			"GitHub",
			"GitLab",
			"Bitbucket",
			"Cancel",
		})
		if err != nil || idx == 3 {
			ui.Info("Cancelled")
			return nil
		}
		platforms := []string{"github", "gitlab", "bitbucket"}
		platform = platforms[idx]
	}

	switch platform {
	case "github":
		return pushToGitHub(user.Name, pubKey, pubKeyPath)
	case "gitlab":
		return pushToGitLab(user.Name, pubKey, pubKeyPath, customGitLabHost)
	case "bitbucket":
		return pushToBitbucket(user.Name, pubKey)
	default:
		return fmt.Errorf("unsupported platform")
	}
}

func detectPlatformFromRemotes() (string, string) {
	defaultHost := "gitlab.com"
	if !git.IsInRepo() {
		return "", defaultHost
	}
	remotes, err := git.ListRemotes()
	if err != nil {
		return "", defaultHost
	}
	for _, r := range remotes {
		url, err := git.GetRemoteURL(r)
		if err != nil {
			continue
		}
		urlLower := strings.ToLower(url)
		if strings.Contains(urlLower, "github.com") {
			return "github", defaultHost
		}
		if strings.Contains(urlLower, "bitbucket.org") || strings.Contains(urlLower, "bitbucket.com") {
			return "bitbucket", defaultHost
		}
		if strings.Contains(urlLower, "gitlab.com") {
			return "gitlab", "gitlab.com"
		}
		// Custom GitLab instance check (heuristic: URLs with "gitlab" in other domain names)
		if strings.Contains(urlLower, "gitlab") {
			// Extract host
			host := extractHost(url)
			if host != "" {
				return "gitlab", host
			}
		}
	}
	return "", defaultHost
}

func extractHost(remoteURL string) string {
	remoteURL = strings.TrimPrefix(remoteURL, "https://")
	remoteURL = strings.TrimPrefix(remoteURL, "http://")
	// If SSH format (git@host:org/repo.git)
	if strings.Contains(remoteURL, "@") {
		parts := strings.SplitN(remoteURL, "@", 2)
		if len(parts) == 2 {
			subParts := strings.SplitN(parts[1], ":", 2)
			return subParts[0]
		}
	}
	// HTTPS format (host/org/repo.git)
	parts := strings.SplitN(remoteURL, "/", 2)
	return parts[0]
}

func pushToGitHub(profileName, pubKey, pubKeyPath string) error {
	// Try gh CLI first
	if _, err := exec.LookPath("gh"); err == nil {
		cmd := exec.Command("gh", "auth", "status")
		if err := cmd.Run(); err == nil {
			ui.Info("GitHub CLI (gh) detected and authenticated. Using gh to add SSH key...")
			addCmd := exec.Command("gh", "ssh-key", "add", pubKeyPath, "--title", fmt.Sprintf("git-user: %s", profileName))
			out, err := addCmd.CombinedOutput()
			if err == nil {
				ui.Success("SSH key successfully added to GitHub via gh CLI!")
				return nil
			}
			ui.Warn(fmt.Sprintf("gh CLI upload failed: %s. Falling back to REST API...", strings.TrimSpace(string(out))))
		}
	}

	// Fallback to PAT
	token, err := ui.Prompt("Enter GitHub Personal Access Token (requires 'write:public_key' scope):")
	if err != nil || token == "" {
		ui.Error("Token required to interact with API.")
		return fmt.Errorf("missing token")
	}

	ui.Info("Pushing key to GitHub API...")
	reqBody, _ := json.Marshal(map[string]string{
		"title": fmt.Sprintf("git-user: %s", profileName),
		"key":   pubKey,
	})

	req, err := http.NewRequest("POST", "https://api.github.com/user/keys", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ui.Errorf("GitHub API request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		ui.Success("SSH key successfully uploaded to GitHub!")
		return nil
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)
	if resp.StatusCode == http.StatusUnprocessableEntity && strings.Contains(bodyStr, "already in use") {
		ui.Warn("This SSH key is already associated with your GitHub account.")
		return nil
	}

	ui.Errorf("Failed to upload key. Status: %s. Response: %s", resp.Status, bodyStr)
	return fmt.Errorf("api upload failed")
}

func pushToGitLab(profileName, pubKey, pubKeyPath, host string) error {
	// Try glab CLI first (works if standard gitlab.com)
	if host == "gitlab.com" {
		if _, err := exec.LookPath("glab"); err == nil {
			cmd := exec.Command("glab", "auth", "status")
			if err := cmd.Run(); err == nil {
				ui.Info("GitLab CLI (glab) detected and authenticated. Using glab to add SSH key...")
				addCmd := exec.Command("glab", "ssh-key", "add", pubKeyPath, "--title", fmt.Sprintf("git-user: %s", profileName))
				out, err := addCmd.CombinedOutput()
				if err == nil {
					ui.Success("SSH key successfully added to GitLab via glab CLI!")
					return nil
				}
				ui.Warn(fmt.Sprintf("glab CLI upload failed: %s. Falling back to REST API...", strings.TrimSpace(string(out))))
			}
		}
	}

	token, err := ui.Prompt(fmt.Sprintf("Enter GitLab (%s) Personal Access Token (requires 'api' or 'write_repository' scope):", host))
	if err != nil || token == "" {
		ui.Error("Token required to interact with API.")
		return fmt.Errorf("missing token")
	}

	ui.Info(fmt.Sprintf("Pushing key to GitLab API (%s)...", host))
	reqBody, _ := json.Marshal(map[string]string{
		"title": fmt.Sprintf("git-user: %s", profileName),
		"key":   pubKey,
	})

	url := fmt.Sprintf("https://%s/api/v4/user/keys", host)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ui.Errorf("GitLab API request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		ui.Success("SSH key successfully uploaded to GitLab!")
		return nil
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)
	if resp.StatusCode == http.StatusBadRequest && strings.Contains(bodyStr, "has already been taken") {
		ui.Warn("This SSH key is already associated with your GitLab account.")
		return nil
	}

	ui.Errorf("Failed to upload key. Status: %s. Response: %s", resp.Status, bodyStr)
	return fmt.Errorf("api upload failed")
}

func pushToBitbucket(profileName, pubKey string) error {
	username, err := ui.Prompt("Enter Bitbucket Username:")
	if err != nil || username == "" {
		ui.Error("Username required.")
		return fmt.Errorf("missing username")
	}

	password, err := readPassphrase("Enter Bitbucket App Password (requires 'ssh:write' scope): ")
	if err != nil || password == "" {
		ui.Error("App password required.")
		return fmt.Errorf("missing app password")
	}

	ui.Info("Pushing key to Bitbucket API...")
	reqBody, _ := json.Marshal(map[string]string{
		"label": fmt.Sprintf("git-user: %s", profileName),
		"key":   pubKey,
	})

	url := fmt.Sprintf("https://api.bitbucket.org/2.0/users/%s/ssh-keys", username)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ui.Errorf("Bitbucket API request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		ui.Success("SSH key successfully uploaded to Bitbucket!")
		return nil
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)
	if strings.Contains(bodyStr, "already exists") || strings.Contains(bodyStr, "already in use") {
		ui.Warn("This SSH key is already associated with your Bitbucket account.")
		return nil
	}

	ui.Errorf("Failed to upload key. Status: %s. Response: %s", resp.Status, bodyStr)
	return fmt.Errorf("api upload failed")
}
