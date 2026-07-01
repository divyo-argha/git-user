package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

func runBindPath(args []string) error {
	if len(args) < 2 {
		ui.Error("usage: git-user bind-path <name> <path>")
		return fmt.Errorf("missing arguments")
	}

	name := args[0]
	path := args[1]

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := store.FindUser(name)
	if user == nil {
		ui.Errorf("identity %q not found", name)
		return fmt.Errorf("user not found")
	}

	// Resolve absolute path
	expanded := expandPath(path)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		ui.Errorf("invalid path: %v", err)
		return err
	}

	// Verify target directory exists
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			ui.Errorf("directory %q does not exist", path)
			return err
		}
		ui.Errorf("error reading directory: %v", err)
		return err
	}
	if !info.IsDir() {
		ui.Errorf("path %q is a file, not a directory", path)
		return fmt.Errorf("path is not a directory")
	}

	if err := store.BindPathToUser(name, abs); err != nil {
		ui.Errorf("binding path: %v", err)
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Bound directory %q to identity %q", abs, name))
	return nil
}

func runUnbindPath(args []string) error {
	if len(args) < 2 {
		ui.Error("usage: git-user unbind-path <name> <path>")
		return fmt.Errorf("missing arguments")
	}

	name := args[0]
	path := args[1]

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	user := store.FindUser(name)
	if user == nil {
		ui.Errorf("identity %q not found", name)
		return fmt.Errorf("user not found")
	}

	// Resolve absolute path
	expanded := expandPath(path)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		ui.Errorf("invalid path: %v", err)
		return err
	}

	if err := store.UnbindPathFromUser(name, abs); err != nil {
		ui.Errorf("unbinding path: %v", err)
		return err
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	ui.Success(fmt.Sprintf("Unbound directory %q from identity %q", abs, name))
	return nil
}
