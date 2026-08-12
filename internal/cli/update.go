package cli

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/divyo-argha/git-user/internal/ui"
	"github.com/divyo-argha/git-user/internal/version"
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

	// Handle npm-installed packages
	if isNpmInstall(execPath) {
		return handleNpmUpdate()
	}

	type githubRelease struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

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

	findAssetURL := func(rel githubRelease) string {
		for _, asset := range rel.Assets {
			name := strings.ToLower(asset.Name)
			if strings.Contains(name, osName) && strings.Contains(name, strings.ToLower(archName)) {
				return asset.BrowserDownloadURL
			}
		}
		return ""
	}

	var release githubRelease
	var downloadURL string

	// 1. Try fetching latest release from /releases/latest
	latestReq, _ := http.NewRequest("GET", "https://api.github.com/repos/divyo-argha/git-user/releases/latest", nil)
	latestReq.Header.Set("User-Agent", "git-user-updater")
	if resp, err := http.DefaultClient.Do(latestReq); err == nil {
		if resp.StatusCode == http.StatusOK {
			var rel githubRelease
			if err := json.NewDecoder(resp.Body).Decode(&rel); err == nil {
				if url := findAssetURL(rel); url != "" {
					release = rel
					downloadURL = url
				}
			}
		}
		resp.Body.Close()
	}

	// 2. Fallback to scanning /releases if /releases/latest lacks binary assets for this OS/Arch
	if downloadURL == "" {
		allReq, _ := http.NewRequest("GET", "https://api.github.com/repos/divyo-argha/git-user/releases", nil)
		allReq.Header.Set("User-Agent", "git-user-updater")
		if resp, err := http.DefaultClient.Do(allReq); err == nil {
			if resp.StatusCode == http.StatusOK {
				var releases []githubRelease
				if err := json.NewDecoder(resp.Body).Decode(&releases); err == nil {
					for _, rel := range releases {
						if rel.Draft {
							continue
						}
						if url := findAssetURL(rel); url != "" {
							release = rel
							downloadURL = url
							break
						}
					}
				}
			}
			resp.Body.Close()
		}
	}

	if downloadURL == "" || release.TagName == "" {
		return fmt.Errorf("no binary release found for %s/%s on GitHub", goos, goarch)
	}

	// Compare remote version against currently installed version
	if !isNewerVersion(release.TagName, version.GetVersion()) {
		ui.Success(fmt.Sprintf("git-user is already up to date (%s). Latest available release: %s", version.GetVersion(), release.TagName))
		return nil
	}

	// Download to a temp file. Prefer the install directory so the final swap
	// is a same-filesystem rename; fall back to the system temp dir when the
	// install directory is not writable by this user (e.g. a sudo install of
	// the binary into /usr/local/bin). The replacement logic then promotes
	// itself with sudo.
	tmpDir := filepath.Dir(execPath)
	tmpFile, err := os.CreateTemp(tmpDir, "git-user-update-*")
	if err != nil {
		tmpFile, err = os.CreateTemp("", "git-user-update-*")
		if err != nil {
			return fmt.Errorf("creating temp file: %w", err)
		}
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	tmpFile.Close()

	ui.Info(fmt.Sprintf("Updating git-user %s → %s (%s %s)", version.GetVersion(), release.TagName, goos, goarch))
	ui.Info(fmt.Sprintf("Downloading git-user %s from GitHub releases...", release.TagName))
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

	// Verify the downloaded binary runs and reports the expected version
	// before replacing the installed copy.
	versionOK := false
	if verOut, verErr := exec.Command(newBinary, "--version").Output(); verErr == nil {
		actual := strings.TrimSpace(string(verOut))
		if strings.Contains(actual, strings.TrimPrefix(release.TagName, "v")) ||
			strings.Contains(actual, release.TagName) {
			versionOK = true
			ui.Success(fmt.Sprintf("Download verified: %s", actual))
		} else {
			ui.Warn(fmt.Sprintf("Downloaded binary reports %q — expected %s. Proceeding, but verify with 'git-user --version' after the update.", actual, release.TagName))
		}
	} else {
		ui.Warn(fmt.Sprintf("Could not verify the downloaded binary: %v", verErr))
	}

	// Replace the installed binary (platform-specific: handles running
	// executables and permission escalation).
	msg, err := installBinary(execPath, newBinary)
	if err != nil {
		return err
	}
	if msg != "" {
		fmt.Printf("\n%s\n", msg)
		return nil
	}

	if versionOK {
		fmt.Printf("\n\033[32m✨ git-user updated to %s (verified)\033[0m\n", release.TagName)
	} else {
		fmt.Printf("\n\033[32m✨ git-user updated to %s\033[0m — run 'git-user --version' to confirm\n", release.TagName)
	}
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
			out, err := os.CreateTemp(filepath.Dir(archivePath), "git-user-new-*")
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

// installBinary replaces the installed binary with a freshly downloaded one.
// Implementations are platform-specific (update_unix.go / update_windows.go),
// where a non-empty message is printed instead of the default "updated"
// banner (used when the swap completes after this process exits).
func handleNpmUpdate() error {
	ui.Info("Detected npm installation. Checking registry for updates...")

	req, _ := http.NewRequest("GET", "https://registry.npmjs.org/git-userhub/latest", nil)
	req.Header.Set("User-Agent", "git-user-updater")
	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var npmPkg struct {
			Version string `json:"version"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&npmPkg); err == nil && npmPkg.Version != "" {
			if !isNewerVersion(npmPkg.Version, version.GetVersion()) {
				ui.Success(fmt.Sprintf("git-user is already up to date (%s). Latest npm version: %s", version.GetVersion(), npmPkg.Version))
				return nil
			}
		}
	}

	// On Windows the running executable is locked by the OS, so npm cannot
	// replace it in place. Hand the update to a background process that runs
	// npm once this process has exited.
	if runtime.GOOS == "windows" {
		if err := scheduleNpmUpdateWindows(); err != nil {
			return fmt.Errorf("scheduling npm update: %w", err)
		}
		ui.Success("✨ git-userhub update scheduled — it will finish in the background after this command exits")
		return nil
	}

	ui.Info("Updating git-userhub via npm...")
	npmCmd := exec.Command("npm", "install", "-g", "git-userhub@latest")
	if _, err := npmCmd.CombinedOutput(); err != nil {
		ui.Warn("Automatic npm update could not be completed.")
		ui.Info("To update manually, run: npm install -g git-userhub@latest")
		return nil
	}

	ui.Success("✨ git-userhub updated via npm to latest version")
	return nil
}

func isNpmInstall(execPath string) bool {
	lower := strings.ToLower(execPath)
	return strings.Contains(lower, "node_modules") ||
		strings.Contains(lower, "git-userhub") ||
		strings.Contains(lower, ".npm") ||
		strings.Contains(lower, "nvm")
}

func parseVersion(v string) (int, int, int) {
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	parts := strings.Split(v, ".")
	var major, minor, patch int
	if len(parts) > 0 {
		fmt.Sscanf(parts[0], "%d", &major)
	}
	if len(parts) > 1 {
		fmt.Sscanf(parts[1], "%d", &minor)
	}
	if len(parts) > 2 {
		patchStr := parts[2]
		if idx := strings.IndexAny(patchStr, "-+"); idx != -1 {
			patchStr = patchStr[:idx]
		}
		fmt.Sscanf(patchStr, "%d", &patch)
	}
	return major, minor, patch
}

func isNewerVersion(remoteTag, currentVersion string) bool {
	rMaj, rMin, rPat := parseVersion(remoteTag)
	cMaj, cMin, cPat := parseVersion(currentVersion)

	if rMaj != cMaj {
		return rMaj > cMaj
	}
	if rMin != cMin {
		return rMin > cMin
	}
	return rPat > cPat
}
