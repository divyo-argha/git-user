package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "downloaded.tar.gz")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return p
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestVerifyChecksumSuccess(t *testing.T) {
	content := "fake-release-tarball-bytes"
	hash := sha256Hex(content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(hash + "  git-user_linux_x86_64.tar.gz\n"))
	}))
	defer srv.Close()

	filePath := writeTempFile(t, content)

	if err := verifyChecksum(filePath, srv.URL, "git-user_linux_x86_64.tar.gz"); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
}

func TestVerifyChecksumMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sha256Hex("different-content") + "  git-user_linux_x86_64.tar.gz\n"))
	}))
	defer srv.Close()

	filePath := writeTempFile(t, "fake-release-tarball-bytes")

	err := verifyChecksum(filePath, srv.URL, "git-user_linux_x86_64.tar.gz")
	if err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}
}

func TestVerifyChecksumMissingEntry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sha256Hex("something") + "  git-user_darwin_arm64.tar.gz\n"))
	}))
	defer srv.Close()

	filePath := writeTempFile(t, "fake-release-tarball-bytes")

	err := verifyChecksum(filePath, srv.URL, "git-user_linux_x86_64.tar.gz")
	if err == nil {
		t.Fatal("expected error for missing checksum entry, got nil")
	}
}

func TestVerifyChecksumUnreachableURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	filePath := writeTempFile(t, "fake-release-tarball-bytes")

	err := verifyChecksum(filePath, srv.URL, "git-user_linux_x86_64.tar.gz")
	if err == nil {
		t.Fatal("expected error for 404 checksums.txt, got nil")
	}
}

func TestFindChecksumsURL(t *testing.T) {
	withChecksums := githubRelease{
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{Name: "git-user_linux_x86_64.tar.gz", BrowserDownloadURL: "https://example.com/tarball"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
		},
	}
	if got := findChecksumsURL(withChecksums); got != "https://example.com/checksums.txt" {
		t.Errorf("expected checksums.txt URL, got %q", got)
	}

	withoutChecksums := githubRelease{
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{Name: "git-user_linux_x86_64.tar.gz", BrowserDownloadURL: "https://example.com/tarball"},
		},
	}
	if got := findChecksumsURL(withoutChecksums); got != "" {
		t.Errorf("expected empty string when no checksums.txt asset present, got %q", got)
	}
}

func TestFindAssetURL(t *testing.T) {
	rel := githubRelease{
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{Name: "git-user_linux_x86_64.tar.gz", BrowserDownloadURL: "https://example.com/linux-amd64"},
			{Name: "git-user_darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/darwin-arm64"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
		},
	}

	if got := findAssetURL(rel, "linux", "x86_64"); got != "https://example.com/linux-amd64" {
		t.Errorf("expected linux/x86_64 asset URL, got %q", got)
	}
	if got := findAssetURL(rel, "darwin", "arm64"); got != "https://example.com/darwin-arm64" {
		t.Errorf("expected darwin/arm64 asset URL, got %q", got)
	}
	if got := findAssetURL(rel, "windows", "x86_64"); got != "" {
		t.Errorf("expected no match for windows/x86_64, got %q", got)
	}
}
