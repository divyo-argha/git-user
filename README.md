# git-user

> Manage and switch between multiple Git identities — including SSH keys and platform accounts — in one command.

```
git user switch work      # switches name, email, SSH key, and ssh config in one shot
```

---

## Table of contents

- [Why git-user](#why-git-user)
- [Installation](#installation)
- [Phase 1 — Identity switching](#phase-1--identity-switching)
- [Phase 2 — SSH keys and platform accounts](#phase-2--ssh-keys-and-platform-accounts)
  - [How SSH handling works](#how-ssh-handling-works)
  - [Security practices](#security-practices)
  - [Full workflow walkthrough](#full-workflow-walkthrough)
- [Command reference](#command-reference)
- [Migration from Phase 1](#migration-from-phase-1)
- [Project structure](#project-structure)

---

## Why git-user

| Problem | git-user solution |
|---|---|
| Multiple people share one machine | Named identities — switch in one command |
| Office vs personal accounts | Add both, switch globally or per-repo |
| Different SSH keys per platform account | Keyed per-user, applied automatically on switch |
| Managing ~/.ssh/config by hand | Managed blocks written and updated automatically |
| Forgetting which account is active | git user current cross-checks everything |

---

## Installation

### Prerequisites

- Go 1.21+ (https://go.dev/dl/)
- git on PATH
- ssh / ssh-keygen on PATH (standard on macOS and Linux; use Git Bash or WSL on Windows)

### Build and install

```bash
git clone https://github.com/yourname/git-user
cd git-user
make install          # /usr/local/bin/git-user  (may need sudo)
make install-local    # ~/bin/git-user             (no sudo)
```

---

## Phase 1 — Identity switching (unchanged)

```bash
git user add work  alice@company.com
git user add home  alice@gmail.com
git user list
git user switch work
git user current
git user edit home  newmail@proton.me
git user remove home
```

**All Phase 1 configs are fully backward-compatible.** SSH/platform fields are omitempty.

---

## Phase 2 — SSH keys and platform accounts

### How SSH handling works

When you run `git user switch work`, two things happen:

1. **Git identity** — sets global user.name / user.email (same as Phase 1).
2. **SSH config** — for each platform account linked to the user, writes a managed Host block into ~/.ssh/config:

```
# git-user:begin github-work
Host github-work
    HostName github.com
    User git
    IdentityFile /home/alice/.ssh/id_ed25519_gituser_work
    IdentitiesOnly yes
    AddKeysToAgent yes
# git-user:end github-work
```

**Safety rules:**
- Only blocks delimited by git-user:begin / git-user:end markers are ever touched.
- All other content in ~/.ssh/config is preserved byte-for-byte.
- Writes are atomic (temp file + rename) — a crash cannot corrupt the file.
- File permissions are enforced at 0600.

You then clone using the alias instead of the real hostname:

```bash
git clone git@github-work:org/repo.git      # authenticates as work account
git clone git@github-home:alice/dotfiles.git # authenticates as home account
```

### Security practices

| Practice | Implementation |
|---|---|
| No plaintext secrets | Only the path to the private key is stored; the key never moves |
| Key permission check | ValidateKeyFile refuses keys wider than 0600 |
| Atomic writes | Temp file + rename — no partial-write window |
| IdentitiesOnly yes | Prevents SSH trying other keys by accident |
| Empty passphrase warning | keygen tells you how to add one afterward |
| No API tokens stored | Phase 2 does not store OAuth tokens or PATs |

### Full workflow walkthrough

```bash
# 1. Add identities
git user add work  alice@company.com
git user add home  alice@gmail.com

# 2. Generate SSH key pairs
git user keygen work     # creates ~/.ssh/id_ed25519_gituser_work{,.pub}
git user keygen home     # creates ~/.ssh/id_ed25519_gituser_home{,.pub}
# (outputs the public key — paste it into your platform settings)

# 3. Map platform accounts
git user platform add work github  alice-corp
git user platform add home github  alice-personal
git user platform add home gitlab  alice-personal

# 4. Test connectivity
git user ssh-test work
# ✔ github     Hi alice-corp! You've successfully authenticated.

git user ssh-test home
# ✔ github     Hi alice-personal! You've successfully authenticated.
# ✔ gitlab     Welcome to GitLab, @alice-personal!

# 5. Switch identities
git user switch work
# ✔ Switched to "work" (alice@company.com)
# ℹ SSH config updated: Host github-work → github.com

# 6. Clone and push
git clone git@github-work:myorg/backend.git
git clone git@github-home:alice/dotfiles.git

# Self-hosted / custom host
git user platform add work custom alice --host git.mycompany.com
git clone git@custom-work:alice/repo.git
```

---

## Command reference

### Identity
| Command | Description |
|---|---|
| git user add n email | Add identity |
| git user list (ls) | List all identities with SSH/platform details |
| git user switch n (sw) | Switch active identity + apply SSH config |
| git user current | Show active identity, SSH key, platforms, sync check |
| git user remove n [--force] (rm) | Remove identity |
| git user edit n email | Update email |

### SSH keys
| Command | Description |
|---|---|
| git user keygen n [--type ed25519|rsa] [--path file] | Generate key pair and link |
| git user ssh-key set n path [comment] | Link existing key |
| git user ssh-key unset n | Unlink key + remove SSH config blocks |
| git user ssh-key show n | Show key path, type, public key comment |

### Platforms
| Command | Description |
|---|---|
| git user platform add n platform handle [--alias a] [--host h] | Map account |
| git user platform remove n platform | Remove mapping + SSH config block |
| git user platform list [name] | List all mappings |

### Diagnostics
| Command | Description |
|---|---|
| git user ssh-test n [platform] | Test SSH connectivity via host alias |

---

## Migration from Phase 1

No migration required. Run these to opt existing users into Phase 2:

```bash
git user keygen work
git user platform add work github  your-handle
git user ssh-test work
```

---

## Project structure

```
git-user/
├── main.go
├── go.mod
├── Makefile
├── install.sh
├── README.md
├── cmd/
│   ├── root.go          dispatcher + usage
│   ├── add.go
│   ├── list.go          updated: shows SSH key + platforms
│   ├── switch.go        updated: applies SSH config on switch
│   ├── current.go       updated: shows SSH key + platform info
│   ├── remove.go
│   ├── edit.go
│   ├── sshkey.go        ssh-key set|unset|show
│   ├── platform.go      platform add|remove|list
│   ├── keygen.go        keygen (wraps ssh-keygen)
│   └── sshtest.go       ssh-test
└── internal/
    ├── config/config.go   User{SSHKey, Accounts}, PlatformAccount, store CRUD
    ├── git/git.go         git config --global wrapper
    ├── sshconf/sshconf.go atomic ~/.ssh/config block manager
    └── ui/ui.go           colour output helpers
```

## License

MIT
