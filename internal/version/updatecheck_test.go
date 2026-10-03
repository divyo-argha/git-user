package version

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestUpdateCheckDisabled(t *testing.T) {
	for val, want := range map[string]bool{"": false, "0": false, "false": false, "off": false, "1": true, "true": true, "yes": true} {
		t.Setenv(NoUpdateCheckEnv, val)
		if got := UpdateCheckDisabled(); got != want {
			t.Errorf("%s=%q: got %v, want %v", NoUpdateCheckEnv, val, got, want)
		}
	}
}

func TestUpdateCache_RoundTripAndExpiry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "latest.json")
	now := time.Now()

	if _, ok := loadCachedLatest(path, now); ok {
		t.Fatal("missing file should be a miss")
	}
	saveCachedLatest(path, "v4.19.0", now)
	if got, ok := loadCachedLatest(path, now.Add(time.Hour)); !ok || got != "v4.19.0" {
		t.Fatalf("fresh cache: got %q, %v", got, ok)
	}
	if _, ok := loadCachedLatest(path, now.Add(UpdateCheckTTL+time.Second)); ok {
		t.Error("expired cache should be a miss")
	}
	if _, ok := loadCachedLatest(path, now.Add(-time.Hour)); ok {
		t.Error("cache from the future should be a miss")
	}
}

func TestUpdateCache_RejectsBadData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "latest.json")
	now := time.Now()

	saveCachedLatest(path, "not-a-version", now)
	if _, ok := loadCachedLatest(path, now); ok {
		t.Error("an invalid version must not be cached")
	}
	if err := os.WriteFile(path, []byte("{corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := loadCachedLatest(path, now); ok {
		t.Error("corrupt file should be a miss")
	}
}

func TestUpdateCache_ConcurrentWritesLeaveOneValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "latest.json")
	now := time.Now()

	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				saveCachedLatest(path, "v4.19.0", now)
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}

	if got, ok := loadCachedLatest(path, now); !ok || got != "v4.19.0" {
		t.Errorf("cache should be valid after concurrent writes, got %q, %v", got, ok)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("no temp files should be left behind, found %d entries", len(entries))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("cache file stat: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("cache should be private (0600), got %v", info)
	}
}
