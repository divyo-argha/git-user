package cli

import (
	"fmt"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/ui"
	"github.com/divyo-argha/git-user/internal/validate"
)

func isValidEmail(email string) bool {
	return validate.Email(email) == nil
}

func runRegister(args []string) error {
	var name, email string
	var isTemp bool
	var err error

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name", "-n":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "--email", "-e":
			if i+1 < len(args) {
				email = args[i+1]
				i++
			}
		case "--temp", "-t":
			isTemp = true
		case "--passphrase", "-p":
			ui.Warn("--passphrase is no longer accepted as a CLI argument (it could leak via `ps` or shell history) — you'll be prompted for it interactively instead.")
			if i+1 < len(args) {
				i++ // skip the (now ignored) value
			}
		}
	}

	ui.Banner("CREATE NEW IDENTITY")
	fmt.Println()

	if name == "" {
		name, err = ui.Prompt("Identity name (e.g., 'work', 'personal'):")
		if err != nil {
			return err
		}
	}

	for {
		if err := validate.IdentityName(name); err != nil {
			ui.Warn(err.Error())
			name, err = ui.Prompt("Enter a valid identity name:")
			if err != nil {
				return err
			}
			continue
		}
		break
	}

	if email == "" {
		email, err = ui.Prompt("Email address:")
		if err != nil {
			return err
		}
	}

	for {
		if err := validate.Email(email); err != nil {
			ui.Warn(err.Error())
			email, err = ui.Prompt("Enter a valid email address:")
			if err != nil {
				return err
			}
			continue
		}
		break
	}

	store, err := config.Load()
	if err != nil {
		ui.Errorf("loading config: %v", err)
		return err
	}

	// No active identity at all yet: this registration will become the
	// active one automatically once it's fully set up below, instead of
	// silently creating a correctly-configured identity that git never
	// actually uses until a separate `switch` is remembered.
	activateOnCreate := store.Current == ""

	if store.IsNameTaken(name) {
		ui.Errorf("identity %q already exists", name)
		return fmt.Errorf("user exists")
	}

	if store.IsEmailTaken(email) {
		ui.Errorf("Email already in use — each identity must have a unique email to prevent impersonation.")
		return fmt.Errorf("email exists")
	}

	if err := store.AddUser(name, email); err != nil {
		ui.Errorf("%v", err)
		return err
	}

	if isTemp {
		u := store.FindUser(name)
		if u != nil {
			u.IsTemporary = true
		}
	}

	fmt.Println()
	ui.Info("SSH Key Setup:")

	idx, err := ui.Select("Choose SSH key setup:", []string{
		"Auto-generate (recommended)",
		"Use existing key",
		"Skip for now",
	})
	if err != nil {
		idx = 0 // Default to auto-generate
	}

	var sshKeyPath string

	switch idx {
	case 0: // Auto-generate
		path, err := generateAndDisplayKey(name, email)
		if err != nil {
			ui.Warn("Key generation failed, skipping SSH setup")
			break
		}
		sshKeyPath = path

	case 1: // Use existing key
		keyPath, err := ui.Prompt("Enter path to your SSH private key:")
		if err != nil {
			return err
		}
		if keyPath == "" {
			ui.Error("No path provided")
			return fmt.Errorf("no path")
		}
		if err := validate.SSHKeyPath(keyPath, true); err != nil {
			ui.Error(err.Error())
			return err
		}
		sshKeyPath = validate.ExpandPath(keyPath)
		ui.Success("Using existing key")

	case 2: // Skip
		ui.Info("Skipping SSH key setup")
		ui.Info("You can set up SSH later with: git-user bind-key " + name + " --ssh-key <path>")

	default:
		ui.Warn("Invalid choice, skipping SSH setup")
		ui.Info("You can set up SSH later with: git-user bind-key " + name + " --ssh-key <path>")
	}

	if sshKeyPath != "" {
		if err := store.BindSSHKey(name, sshKeyPath); err != nil {
			ui.Errorf("binding SSH key: %v", err)
		}
		fmt.Println()
		if ui.Confirm("Would you like to sign your Git commits automatically using this identity's SSH key?", true) {
			if err := store.SetSigningKey(name, sshKeyPath, "ssh"); err != nil {
				ui.Warn(fmt.Sprintf("Failed to enable SSH commit signing: %v", err))
			} else {
				ui.Success("Commit signing configured automatically!")
			}
		} else {
			store.ToggleSigning(name, true)
		}
	}

	activated := false
	if activateOnCreate {
		user := store.FindUser(name)
		if user != nil {
			if err := git.Apply(user.Name, user.Email); err != nil {
				ui.Warn(fmt.Sprintf("applying git config: %v", err))
			} else {
				store.Current = name
				activated = true
				if sshKeyPath != "" {
					if err := applyUserSSHConfig(user, false); err != nil {
						ui.Warn(fmt.Sprintf("applying SSH config: %v", err))
					}
				}
				if !user.SignDisabled && user.SignKey != "" {
					if err := git.ConfigureSigning(user.SignKey, user.SignFormat); err != nil {
						ui.Warn(fmt.Sprintf("applying signing config: %v", err))
					}
				}
			}
		}
	}

	if err := config.Save(store); err != nil {
		ui.Errorf("saving config: %v", err)
		return err
	}

	fmt.Println()
	ui.Divider()
	ui.AnimatedSuccess(fmt.Sprintf("Identity created: %s (%s)", name, email))
	if sshKeyPath != "" {
		ui.Success(fmt.Sprintf("SSH key configured: %s", sshKeyPath))
	}
	fmt.Println()
	if activated {
		ui.Success("This is your first identity, so it's now active.")
	} else {
		ui.Info(fmt.Sprintf("Activate with: git-user switch %s", name))
	}
	ui.Divider()

	return nil
}
