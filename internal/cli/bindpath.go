package cli

import (
	"fmt"
	"path/filepath"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
	"github.com/divyo-argha/git-user/internal/validate"
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

	if err := validate.BindPath(path, true); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	// Resolve absolute path
	expanded := validate.ExpandPath(path)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		ui.Errorf("invalid path: %v", err)
		return err
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
