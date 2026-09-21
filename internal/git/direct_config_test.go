package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirectGitConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ".gitconfig")

	// 1. Set simple key
	if err := setDirectConfig(cfg, "user.name", "Alice"); err != nil {
		t.Fatalf("setDirectConfig failed: %v", err)
	}
	if err := setDirectConfig(cfg, "user.email", "alice@example.com"); err != nil {
		t.Fatalf("setDirectConfig failed: %v", err)
	}

	// 2. Read keys back
	name, err := getDirectConfig(cfg, "user.name")
	if err != nil || name != "Alice" {
		t.Errorf("getDirectConfig(user.name) = %q, want Alice (err: %v)", name, err)
	}
	email, err := getDirectConfig(cfg, "user.email")
	if err != nil || email != "alice@example.com" {
		t.Errorf("getDirectConfig(user.email) = %q, want alice@example.com (err: %v)", email, err)
	}

	// 3. Update existing key
	if err := setDirectConfig(cfg, "user.name", "Bob"); err != nil {
		t.Fatalf("update setDirectConfig failed: %v", err)
	}
	name, err = getDirectConfig(cfg, "user.name")
	if err != nil || name != "Bob" {
		t.Errorf("updated getDirectConfig(user.name) = %q, want Bob", name)
	}

	// 4. Set subsection key (e.g. includeIf)
	incKey := "includeif.gitdir/i:C:/Projects/.path"
	incVal := "C:/Users/Bob/.config/git-user/profile-bob.gitconfig"
	if err := setDirectConfig(cfg, incKey, incVal); err != nil {
		t.Fatalf("setDirectConfig includeif failed: %v", err)
	}
	gotInc, err := getDirectConfig(cfg, incKey)
	if err != nil || gotInc != incVal {
		t.Errorf("getDirectConfig(includeif) = %q, want %q (err: %v)", gotInc, incVal, err)
	}

	// 5. Unset key
	if err := unsetDirectConfig(cfg, "user.email"); err != nil {
		t.Fatalf("unsetDirectConfig failed: %v", err)
	}
	if _, err := getDirectConfig(cfg, "user.email"); err == nil {
		t.Errorf("expected user.email to be unset, but found it")
	}

	// Verify file content structure
	content, _ := os.ReadFile(cfg)
	t.Logf("Generated .gitconfig:\n%s", string(content))
}
