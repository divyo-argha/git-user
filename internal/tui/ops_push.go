package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ── Pubkey push ───────────────────────────────────────────────────────────────

// opPushKey attempts to publish the active identity's public key to a platform,
// preferring the platform CLI (gh/glab). It returns ErrNeedsCredential when a
// token / username+password must be entered by the user.
func opPushKey(store *config.Store, platform string) (opResult, error) {
	user := store.CurrentUser()
	if user == nil {
		return opResult{}, fmt.Errorf("no active identity is set")
	}
	if user.SSHKey == "" {
		return opResult{}, fmt.Errorf("no SSH key is bound to the active identity %q", user.Name)
	}
	pubKeyPath := user.SSHKey + ".pub"
	if _, err := os.ReadFile(pubKeyPath); err != nil {
		return opResult{}, fmt.Errorf("could not read public key file %s", pubKeyPath)
	}

	switch platform {
	case "github":
		if _, err := exec.LookPath("gh"); err == nil {
			if _, authErr := runCaptured("", "gh", "auth", "status"); authErr == nil {
				addCmd := exec.Command("gh", "ssh-key", "add", pubKeyPath, "--title", fmt.Sprintf("git-user: %s", user.Name))
				if out, err := addCmd.CombinedOutput(); err == nil {
					return opResult{detail: "SSH key successfully added to GitHub via gh CLI!"}, nil
				} else {
					_ = out
				}
			}
		}
		return opResult{}, ErrNeedsCredential
	case "gitlab":
		if _, err := exec.LookPath("glab"); err == nil {
			if _, authErr := runCaptured("", "glab", "auth", "status"); authErr == nil {
				addCmd := exec.Command("glab", "ssh-key", "add", pubKeyPath, "--title", fmt.Sprintf("git-user: %s", user.Name))
				if out, err := addCmd.CombinedOutput(); err == nil {
					return opResult{detail: "SSH key successfully added to GitLab via glab CLI!"}, nil
				} else {
					_ = out
				}
			}
		}
		return opResult{}, ErrNeedsCredential
	case "bitbucket":
		return opResult{}, ErrNeedsCredential
	}
	return opResult{}, fmt.Errorf("unsupported platform")
}

// opPushKeyWithCredential publishes a key using an API token (or username +
// app password for Bitbucket).
func opPushKeyWithCredential(store *config.Store, platform, username, credential string) (opResult, error) {
	user := store.CurrentUser()
	if user == nil {
		return opResult{}, fmt.Errorf("no active identity is set")
	}
	if user.SSHKey == "" {
		return opResult{}, fmt.Errorf("no SSH key is bound to the active identity %q", user.Name)
	}
	pubKeyPath := user.SSHKey + ".pub"
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return opResult{}, fmt.Errorf("could not read public key file %s", pubKeyPath)
	}
	pubKey := strings.TrimSpace(string(pubKeyBytes))

	switch platform {
	case "github":
		return pushKeyGitHub(user.Name, pubKey, credential)
	case "gitlab":
		return pushKeyGitLab(user.Name, pubKey, credential)
	case "bitbucket":
		return pushKeyBitbucket(user.Name, pubKey, username, credential)
	}
	return opResult{}, fmt.Errorf("unsupported platform")
}

func pushKeyGitHub(profileName, pubKey, token string) (opResult, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"title": fmt.Sprintf("git-user: %s", profileName),
		"key":   pubKey,
	})
	req, err := http.NewRequest("POST", "https://api.github.com/user/keys", bytes.NewBuffer(reqBody))
	if err != nil {
		return opResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return opResult{}, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		return opResult{detail: "SSH key successfully uploaded to GitHub!"}, nil
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnprocessableEntity && strings.Contains(string(body), "already in use") {
		return opResult{detail: "This SSH key is already associated with your GitHub account."}, nil
	}
	return opResult{}, fmt.Errorf("failed to upload key. Status: %s. Response: %s", resp.Status, string(body))
}

func pushKeyGitLab(profileName, pubKey, token string) (opResult, error) {
	host := "gitlab.com"
	if r, err := git.ListRemotes(); err == nil {
		for _, remote := range r {
			if url, err := git.GetRemoteURL(remote); err == nil {
				if h := detectGitLabHost(url); h != "" {
					host = h
					break
				}
			}
		}
	}
	reqBody, _ := json.Marshal(map[string]string{
		"title": fmt.Sprintf("git-user: %s", profileName),
		"key":   pubKey,
	})
	url := fmt.Sprintf("https://%s/api/v4/user/keys", host)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return opResult{}, err
	}
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return opResult{}, fmt.Errorf("GitLab API request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		return opResult{detail: "SSH key successfully uploaded to GitLab!"}, nil
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(body), "has already been taken") {
		return opResult{detail: "This SSH key is already associated with your GitLab account."}, nil
	}
	return opResult{}, fmt.Errorf("failed to upload key. Status: %s. Response: %s", resp.Status, string(body))
}

func detectGitLabHost(url string) string {
	lower := strings.ToLower(url)
	if !strings.Contains(lower, "gitlab") {
		return ""
	}
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	if strings.Contains(url, "@") {
		parts := strings.SplitN(url, "@", 2)
		if len(parts) == 2 {
			sub := strings.SplitN(parts[1], ":", 2)
			return sub[0]
		}
	}
	parts := strings.SplitN(url, "/", 2)
	return parts[0]
}

func pushKeyBitbucket(profileName, pubKey, username, appPassword string) (opResult, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"label": fmt.Sprintf("git-user: %s", profileName),
		"key":   pubKey,
	})
	url := fmt.Sprintf("https://api.bitbucket.org/2.0/users/%s/ssh-keys", username)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return opResult{}, err
	}
	req.SetBasicAuth(username, appPassword)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return opResult{}, fmt.Errorf("bitbucket API request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		return opResult{detail: "SSH key successfully uploaded to Bitbucket!"}, nil
	}
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "already exists") || strings.Contains(string(body), "already in use") {
		return opResult{detail: "This SSH key is already associated with your Bitbucket account."}, nil
	}
	return opResult{}, fmt.Errorf("failed to upload key. Status: %s. Response: %s", resp.Status, string(body))
}
