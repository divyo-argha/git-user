package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/divyo-argha/git-user/internal/ui"
)

func RunUpdate() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine install path: %w", err)
	}
	// Resolve symlinks (e.g. npm bin wrapper points to real binary)
	if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
		execPath = resolved
	}

	ui.Info(fmt.Sprintf("Updating git-user from %s...", execPath))

	// Fetch latest release info
	releaseURL := "https://api.github.com/repos/divyo-argha/git-user/releases/latest"
	req, _ := http.NewRequest("GET", releaseURL, nil)
	req.Header.Set("User-Agent", "git-user-updater")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching release info: %w", err)
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("parsing release info: %w", err)
	}

	// Map runtime values to release asset naming
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	osName := goos // linux, darwin, windows
	archName := goarch
	if archName == "amd64" {
		archName = "x86_64"
	}

	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}

	// Find matching asset
	var downloadURL string
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, osName) && strings.Contains(name, strings.ToLower(archName)) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no binary found for %s/%s in release %s", goos, goarch, release.TagName)
	}

	// Download to temp file
	tmpFile, err := os.CreateTemp("", "git-user-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	tmpFile.Close()

	ui.Info(fmt.Sprintf("Downloading %s...", release.TagName))
	if err := downloadFile(downloadURL, tmpPath); err != nil {
		return fmt.Errorf("downloading binary: %w", err)
	}

	// Extract binary from tar.gz
	newBinary, err := extractBinary(tmpPath, "git-user"+ext)
	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}
	defer os.Remove(newBinary)

	// Make executable
	if err := os.Chmod(newBinary, 0755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	// Replace: move old binary aside, move new one in
	backupPath := execPath + ".bak"
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("backing up current binary: %w", err)
	}

	if err := os.Rename(newBinary, execPath); err != nil {
		// Rollback
		os.Rename(backupPath, execPath)
		return fmt.Errorf("installing new binary: %w", err)
	}

	// Clean up backup
	os.Remove(backupPath)

	fmt.Printf("\n\033[32m✨ git-user updated to %s\033[0m\n", release.TagName)
	return nil
}

func downloadFile(url, dest string) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "git-user-updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Follow redirects (http.DefaultClient does this, but handle non-200)
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// extractBinary extracts a named file from a .tar.gz archive into a temp file.
// Returns the path to the extracted file.
func extractBinary(archivePath, binaryName string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Match by base name to handle paths like ./git-user or git-user
		if filepath.Base(hdr.Name) == binaryName {
			out, err := os.CreateTemp("", "git-user-new-*")
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				os.Remove(out.Name())
				return "", err
			}
			out.Close()
			return out.Name(), nil
		}
	}

	return "", fmt.Errorf("binary %q not found in archive", binaryName)
}
