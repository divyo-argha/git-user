# 📋 git-user CLI Command Reference

This document provides a comprehensive reference for all `git-user` command-line subcommands, flags, and options.

> 💡 **Tip:** Running `git-user` without arguments on an interactive terminal opens the **TUI Dashboard**, where all identity, SSH key, signing, and diagnostics operations can be performed visually without needing to memorize commands.

---

## 📑 Table of Contents

- [Identities](#identities)
- [Terminal Sessions & Isolation](#terminal-sessions--isolation)
- [SSH & Keys](#ssh--keys)
- [Repos & Portability](#repos--portability)
- [System, Health & Security](#system-health--security)
- [Shell Completion](#shell-completion)
- [Command Reference Table](#command-reference-table)
- [Aliases & Flags](#aliases--flags)
- [Environment Variables](#environment-variables)

---

## Identities

### `register`
Create a new identity via guided interactive setup.
```bash
git-user register
git-user register -n work -e me@work.com
git-user register --temp -n guest -e guest@corp.com
```
* **Flags:**
  * `-n, --name <name>`: Identity profile name
  * `-e, --email <email>`: Email address
  * `-t, --temp`: Create as a temporary identity (automatically deleted on logout)

### `switch`
Switch the active Git identity.
```bash
# Global switch (all repositories)
git-user switch work

# Local switch (only inside current git repository)
git-user switch work --local

# Session switch (only for current terminal shell)
git-user switch work --session

# Create and switch in one command
git-user switch -c work -e me@work.com

# Create temporary identity and switch
git-user switch -c guest -e guest@corp.com --temp

# Switch without SSH setup
git-user switch -c work --skip-ssh

# Switch back to pre-git-user original identity
git-user switch --original
```
* **Flags:**
  * `-c <name>`: Create new identity and switch to it
  * `-e, --email <email>`: Email address (used with `-c`)
  * `-t, --temp`: Create temporary identity (used with `-c`)
  * `--skip-ssh`: Skip SSH key setup (used with `-c`)
  * `-l, --local`: Switch only for the current repository
  * `-s, --session`: Switch for current terminal session only
  * `--original`: Import and switch to original pre-git-user config

### `list`
List all registered identities.
```bash
git-user list
git-user list --plain
git-user list --json
```
* **Flags:**
  * `--plain`: Plain, machine-readable output (`name <email> [# active]`)
  * `--json`: Structured JSON array output

### `current`
Show the currently active identity.
```bash
git-user current
git-user current --plain
git-user current --json
```

### `prompt`
Output the active identity for terminal prompt integration.
```bash
git-user prompt
git-user prompt --icon
git-user prompt install [shell]
git-user prompt uninstall [shell|all]
```
* **Flags:**
  * `-i, --icon`: Include profile icon prefix
  * `-a, --always`: Output profile even outside git repositories
  * `-p, --plain`: Output plain profile name without badges

### `remove`
Delete an identity profile and its configuration.
```bash
git-user remove work
git-user remove work --force
```

### `edit`
Update an identity's email address.
```bash
git-user edit work new-email@company.com
```

### `rename`
Rename an identity. Automatically updates git config if active.
```bash
git-user rename old-work new-work
```

### `logout`
Sign out and clear the active identity, returning git config to a clean "void" state.
```bash
git-user logout
git-user logout --yes
```

---

## Terminal Sessions & Isolation

### `env`
Output shell export or unset statements for per-terminal session isolation.
```bash
eval "$(git-user env work)"
eval "$(git-user env --unset)"
```

### `shell`
Launch an isolated subshell locked to a Git identity.
```bash
git-user shell work
```

### `exec`
Execute a single command using a Git identity's environment without affecting other terminals.
```bash
git-user exec work -- git push origin main
git-user exec personal -- gh repo list
```

### `init`
Generate shell integration hook function for seamless auto-switching.
```bash
eval "$(git-user init)"
```

---

## SSH & Keys

### `pubkey`
Display the public SSH key for the active identity.
```bash
git-user pubkey
git-user pubkey publish [github|gitlab|bitbucket]
git-user pubkey test [name]
```

### `connections`
Test and report which Git platforms have accepted an identity's SSH key.
```bash
git-user connections
git-user connections work
```
* **Aliases:** `check-ssh`, `check`, `test-ssh`

### `bind-key`
Attach or link an existing SSH key to an identity profile.
```bash
git-user bind-key work --ssh-key ~/.ssh/custom_id_ed25519
```

### `bind-path` / `unbind-path`
Bind or unbind a directory path to an identity for automatic switching when `cd`ing.
```bash
git-user bind-path work ~/Projects/work
git-user unbind-path work ~/Projects/work
```

### `passphrase`
Manage passphrase protection for the active, unlocked identity.
```bash
git-user passphrase --set
git-user passphrase --remove
git-user passphrase --mode persistent|login|everytime
git-user passphrase --ttl 4h
git-user passphrase --confirm-on-use
git-user passphrase --harden
```

### `token`
Manage HTTPS personal-access-tokens (stored securely in the OS keychain).
```bash
git-user token work --set
git-user token work --remove
git-user token work --expires 2026-12-31
```

### `rekey`
Rotate the SSH key for an identity with automatic backup and rollback safety.
```bash
git-user rekey work
git-user rekey work --force
```

### `sign`
Manage GPG/SSH commit signing for an identity.
```bash
git-user sign work
```

---

## Repos & Portability

### `fix-remote`
Convert HTTPS git remotes in the current repository to SSH.
```bash
git-user fix-remote
```

### `clone`
Clone a repository and auto-configure local identity and SSH key.
```bash
git-user clone git@github.com:org/repo.git
```

### `export`
Bundle identities and keys into an AES-256 encrypted archive.
```bash
git-user export --all
git-user export work personal
```

### `import`
Restore identities and keys from an encrypted bundle archive.
```bash
git-user import ~/backup.bundle
git-user import --force ~/backup.bundle
```

### `sync`
Synchronize identities across devices using an encrypted private Git repository.
```bash
git-user sync
```

---

## System, Health & Security

### `doctor`
Diagnose common configuration drift, missing keys, and environment problems.
```bash
git-user doctor
git-user doctor --fix
```

### `refresh`
Fix config conflicts doctor finds by re-syncing live `.gitconfig` to match git-user's stored state.
```bash
git-user refresh
```
* **Alias:** `repair`

### `audit`
Run a security audit on permissions, passphrases, and key protection.
```bash
git-user audit
git-user audit --fix
```
* **Alias:** `security`

### `stats`
Audit commit author identity stats across recent commits.
```bash
git-user stats
```

### `verify`
Verify commit signatures in a revision range (CI-friendly; exits non-zero on unsigned/invalid commits).
```bash
git-user verify
git-user verify --range HEAD~10..HEAD
```

### `policy`
Manage repository `.git-user-policy` and `.allowed-signers` enforcement.
```bash
git-user policy init
git-user policy show
git-user policy signers
```

### `config`
Manage custom git configurations for an identity profile.
```bash
git-user config list work
git-user config set work core.autocrlf input
git-user config unset work core.autocrlf
```

### `hook`
Install or uninstall repository pre-commit and pre-push verification hooks.
```bash
git-user hook install
git-user hook uninstall
git-user hook check
```

### `log`
Show the identity-switch audit log.
```bash
git-user log
git-user log -n 10
git-user log --all
git-user log --plain
```
* **Alias:** `history`

### `uninstall`
Completely remove git-user, keys, and configurations, restoring original git settings.
```bash
git-user uninstall
git-user uninstall --yes
```
* **Alias:** `purge`

### `install-git`
Securely install official Git via verified package manager (winget on Windows).
```bash
git-user install-git
```

### `tui`
Open the interactive terminal user interface.
```bash
git-user tui
git-user -i
# Running `git-user` with no arguments also opens the TUI
```

---

## Shell Completion

Generate shell autocompletion scripts for your shell:
```bash
# Bash
git-user completion bash > /etc/bash_completion.d/git-user

# Zsh
git-user completion zsh > "${fpath[1]}/_git-user"

# Fish
git-user completion fish > ~/.config/fish/completions/git-user.fish

# PowerShell
git-user completion powershell | Out-String | Invoke-Expression
```

---

## Command Reference Table

| Command | Description |
|---------|-------------|
| `register` | Create a new identity (guided setup with SSH) |
| `switch <name> [--local]` | Switch to an identity (globally, or locally in repository config) |
| `switch <name> --session` | Switch identity for current terminal session only |
| `switch -c <name> [-e <email>]` | Create and switch in one command |
| `switch -c <name> --temp` | Create temporary identity removed on logout |
| `switch -c <name> --skip-ssh` | Create and switch, skipping SSH key setup |
| `switch --original` | Import and switch to original pre-git-user identity |
| `list` | Show all identities (`--plain`, `--json`) |
| `current` | Show active identity (`--plain`, `--json`) |
| `prompt` | Output active identity for prompt integration |
| `remove <name>` | Delete an identity |
| `edit <name> <email>` | Update email |
| `rename <old> <new>` | Rename an identity |
| `pubkey` | Show public key of active identity |
| `connections [name]` | Check SSH connections to GitHub, GitLab, and Bitbucket |
| `bind-key <name>` | Link an SSH key to an identity |
| `bind-path <name> <path>` | Bind a directory path to an identity for auto-switching |
| `unbind-path <name> <path>` | Unbind a directory path from an identity |
| `passphrase` | Manage passphrase and agent TTL for active identity |
| `token <name>` | Manage HTTPS personal-access-tokens |
| `sign <name>` | Manage commit signing for an identity |
| `rekey <name>` | Rotate SSH key with rollback safety |
| `fix-remote` | Convert HTTPS remotes to SSH |
| `logout` | Sign out, restoring clean void state |
| `audit [--fix]` | Security audit of keys, permissions, and passphrases |
| `doctor [--fix]` | Full health check and diagnosis |
| `refresh` | Re-sync live `.gitconfig` to match git-user state |
| `log [-n <count>]` | View identity-switch audit log |
| `stats` | Audit and show commit author identity stats |
| `verify` | Verify commit signatures in revision range |
| `policy` | Manage repo signing policy and allowed signers |
| `config` | Manage custom git configurations for an identity |
| `env <name>` | Output shell export/unset statements |
| `shell <name>` | Launch isolated subshell for an identity |
| `exec <name> -- <cmd>` | Execute single command in an identity environment |
| `init [shell]` | Generate shell integration hook |
| `hook` | Manage pre-commit and pre-push git hooks |
| `clone <repo-url>` | Clone repo and configure local identity |
| `export` | Encrypted bundle export of identities and keys |
| `import <file>` | Import identities from encrypted bundle |
| `sync` | Sync identities across devices via private repo |
| `uninstall` | Remove git-user and restore original git config |
| `completion <shell>` | Generate shell completions (bash/zsh/fish/powershell) |
| `install-git` | Install Git via winget on Windows |
| `tui` | Interactive terminal UI dashboard |
| `--update` | Self-update to latest release |
| `--version` / `-v` | Show version |

---

## Aliases & Flags

* **Aliases:**
  * `ls` → `list`
  * `sw` → `switch`
  * `rm` → `remove`
  * `reg` → `register`
  * `whoami` → `current`
  * `lo` / `signout` → `logout`
  * `bind` → `bind-key`
  * `pubkey push` → `pubkey publish`
  * `check-ssh` / `check` / `test-ssh` → `connections`
  * `repair` → `refresh`
  * `history` → `log`
  * `purge` → `uninstall`
  * `security` → `audit`
  * `import-original` → `switch --original`
  * `tui` / `-i` / `--interactive` → interactive menu

* **Dual Flag/Subcommand Convention:** All commands can be run either as subcommands (`git-user current`, `git-user list`) or as standard flags (`git-user --current`, `git-user -c`, `git-user --list`, `git-user -l`, `git-user --switch <name>`, `git-user -s <name>`).

---

## Environment Variables

* `NO_COLOR=1` / `--no-color`: Disable coloured terminal output (also respects `TERM=dumb`).
* `GIT_USER_NO_UPDATE_CHECK=1`: Disable background release update check.
* `GIT_USER_CONFIG=<path>`: Override configuration file path (defaults to `~/.git-users/config.json`).
