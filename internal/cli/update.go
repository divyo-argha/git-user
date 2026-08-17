package cli

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/divyo-argha/git-user/internal/ui"
	"github.com/divyo-argha/git-user/internal/version"
)

// githubRelease is the subset of the GitHub Releases API response RunUpdate
// needs to locate a platform binary asset and its checksums.txt.
type githubRelease struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// findAssetURL returns the browser_download_url of the release asset whose
// name matches the given OS/arch, or "" if none matches.
func findAssetURL(rel githubRelease, osName, archName string) string {
	for _, asset := range rel.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, osName) && strings.Contains(name, strings.ToLower(archName)) {
			return asset.BrowserDownloadURL
		}
	}
	return ""
}

// findChecksumsURL returns the browser_download_url of the release's
// checksums.txt asset, or "" if the release has none.
func findChecksumsURL(rel githubRelease) string {
	for _, asset := range rel.Assets {
		if strings.EqualFold(asset.Name, "checksums.txt") {
			return asset.BrowserDownloadURL
		}
	}
	return ""
}

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

	var release githubRelease
	var downloadURL string

	// 1. Try fetching latest release from /releases/latest
	latestReq, _ := http.NewRequest("GET", "https://api.github.com/repos/divyo-argha/git-user/releases/latest", nil)
	latestReq.Header.Set("User-Agent", "git-user-updater")
	if resp, err := http.DefaultClient.Do(latestReq); err == nil {
		if resp.StatusCode == http.StatusOK {
			var rel githubRelease
			if err := json.NewDecoder(resp.Body).Decode(&rel); err == nil {
				if url := findAssetURL(rel, osName, archName); url != "" {
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
						if url := findAssetURL(rel, osName, archName); url != "" {
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
		ui.PrintUpdateCurrent(version.GetVersion())
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

	ui.Info(fmt.Sprintf("Updating git-user: %s → %s (%s %s)", version.GetVersion(), release.TagName, goos, goarch))
	ui.Info(fmt.Sprintf("Downloading git-user %s from GitHub releases...", release.TagName))
	if err := downloadFile(downloadURL, tmpPath); err != nil {
		return fmt.Errorf("downloading binary: %w", err)
	}

	// Verify the download against the release's checksums.txt before this
	// binary is ever extracted or executed. This is the actual security
	// boundary for self-update: without it, a tampered or substituted release
	// asset would be installed (and, when the install directory requires
	// sudo, run as root) with nothing to catch it.
	checksumsURL := findChecksumsURL(release)
	if checksumsURL == "" {
		return fmt.Errorf("release %s has no checksums.txt — refusing to install an unverified binary", release.TagName)
	}
	if err := verifyChecksum(tmpPath, checksumsURL, path.Base(downloadURL)); err != nil {
		return fmt.Errorf("verifying download integrity: %w", err)
	}
	ui.Success("Checksum verified")

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

	// Cosmetic confirmation only (the checksum check above is the actual
	// security boundary): run the new binary and check its printed version.
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

	ui.PrintUpdateSuccess(version.GetVersion(), release.TagName, versionOK)
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

// verifyChecksum downloads a GoReleaser-style checksums.txt from checksumsURL
// and confirms filePath's SHA-256 matches the entry for assetName. It errors
// if the checksums file can't be fetched, has no entry for assetName, or the
// computed hash doesn't match — the caller must treat any error as "do not
// install this file".
func verifyChecksum(filePath, checksumsURL, assetName string) error {
	req, _ := http.NewRequest("GET", checksumsURL, nil)
	req.Header.Set("User-Agent", "git-user-updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading checksums.txt: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading checksums.txt: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading checksums.txt: %w", err)
	}

	expected := ""
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == assetName {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("no checksum entry for %s in checksums.txt", assetName)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing downloaded file: %w", err)
	}
	actual := hex.EncodeToString(h.Sum(nil))

	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", assetName, expected, actual)
	}
	return nil
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

	targetVersion := "latest"
	req, _ := http.NewRequest("GET", "https://registry.npmjs.org/git-userhub/latest", nil)
	req.Header.Set("User-Agent", "git-user-updater")
	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var npmPkg struct {
			Version string `json:"version"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&npmPkg); err == nil && npmPkg.Version != "" {
			targetVersion = npmPkg.Version
			if !isNewerVersion(npmPkg.Version, version.GetVersion()) {
				ui.PrintUpdateCurrent(version.GetVersion())
				return nil
			}
		}
	}

	ui.Info(fmt.Sprintf("Updating git-userhub: %s → %s via npm...", version.GetVersion(), targetVersion))

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

	npmCmd := exec.Command("npm", "install", "-g", "git-userhub@latest")
	if _, err := npmCmd.CombinedOutput(); err != nil {
		ui.Warn("Automatic npm update could not be completed.")
		ui.Info("To update manually, run: npm install -g git-userhub@latest")
		return nil
	}

	ui.PrintUpdateSuccess(version.GetVersion(), targetVersion, true)
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
	return version.ParseVersion(v)
}

func isNewerVersion(remoteTag, currentVersion string) bool {
	return version.IsNewerVersion(remoteTag, currentVersion)
}

