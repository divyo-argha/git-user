package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dangerous calls write to the real git config / SSH config / keychain when
// executed outside a sandboxed HOME.
var dangerousCalls = []string{
	"git.Apply(",
	"git.ConfigureSigning(",
	"git.ConfigureSSH(",
	"git.SetConfig(",
	"git.RemoveSigningConfig(",
	"git.ClearIdentityScope(",
	"git.SetRemoteURL(",
	"opSwitch(",
	"opRename(",
	"opAdd(",
	"opBindPath(",
	"opUnbindPath(",
	"runSwitch(",
	"runLogout(",
	"runSign(",
	"runConfig(",
	"runBind(",
	"runBindPath(",
	"runRekey(",
	"runPrompt(",
	"runHook(",
}

// sandboxMarkers prove a test file isolates HOME before running any of the
// dangerous calls above.
var sandboxMarkers = []string{
	"testutil.Sandbox(",
	"withTempConfig(",
	"setupTestEnv(",
	`Setenv("HOME"`,
}

// TestNoTestWritesRealConfig scans every _test.go file in the repo and fails
// if a file calls a git-config-writing function without also setting up an
// isolated HOME. This is the hard guarantee that test runs can never touch
// the developer's real ~/.gitconfig again.
func TestNoTestWritesRealConfig(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}

	var violations []string
	err = filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "vendor") {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(content)
		hasDanger := false
		for _, call := range dangerousCalls {
			if strings.Contains(src, call) {
				hasDanger = true
				break
			}
		}
		if !hasDanger {
			return nil
		}
		sandboxed := false
		for _, marker := range sandboxMarkers {
			if strings.Contains(src, marker) {
				sandboxed = true
				break
			}
		}
		if !sandboxed {
			rel, _ := filepath.Rel(repoRoot, path)
			violations = append(violations, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(violations) > 0 {
		t.Fatalf("test files call git-config-writing functions without isolating HOME (add testutil.Sandbox):\n  %s", strings.Join(violations, "\n  "))
	}
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
