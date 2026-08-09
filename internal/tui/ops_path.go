package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/divyo-argha/git-user/internal/config"
)

// ── Path bindings ─────────────────────────────────────────────────────────────

func opBindPath(store *config.Store, name, path string) error {
	expanded := expandPath(path)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory %q does not exist", path)
		}
		return fmt.Errorf("error reading directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %q is a file, not a directory", path)
	}
	if err := store.BindPathToUser(name, abs); err != nil {
		return err
	}
	return config.Save(store)
}

func opUnbindPath(store *config.Store, name, path string) error {
	expanded := expandPath(path)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	if err := store.UnbindPathFromUser(name, abs); err != nil {
		return err
	}
	return config.Save(store)
}

