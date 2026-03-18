# git-user

> Manage and switch between multiple Git identities from a single command.

```
git user switch work      # one command — done.
```

No more `git config --global user.name "..."` every time you sit at a different machine or need to flip between office and personal accounts.

---

## Why git-user?

| Problem | git-user solution |
|---|---|
| Multiple people share one machine | Each person has a named identity — switch in one command |
| Office vs personal accounts | Add both, switch instantly |
| Folder-based `.gitconfig` is fragile | Global config is always in sync with the active identity |
| Forgetting which account is active | `git user current` cross-checks git config for you |

---

## Installation

### Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- `git` on your PATH

### Option A — Makefile (recommended)

```bash
git clone https://github.com/yourname/git-user
cd git-user

# Install system-wide (may need sudo)
make install

# Or install to ~/bin (no sudo)
make install-local
```

### Option B — install script

```bash
bash install.sh
```

### Option C — manual

```bash
go build -o git-user .
sudo mv git-user /usr/local/bin/git-user
```

### Enable as `git user` subcommand

Git automatically delegates `git <subcommand>` to a binary named `git-<subcommand>` if it exists on your PATH. No extra setup needed — once `git-user` is in `/usr/local/bin`, both of these work:

```bash
git-user switch work
git  user  switch work   # identical
```

---

## Commands

| Command | Description |
|---|---|
| `git user add <name> <email>` | Add a new identity |
| `git user list` (`ls`) | List all identities |
| `git user switch <name>` (`sw`) | Switch active identity (updates global git config) |
| `git user current` | Show active identity + git config sync check |
| `git user remove <name>` (`rm`) | Remove an identity |
| `git user edit <name> <email>` | Update an identity's email |

---

## Sample session

```
$ git user add work   alice@company.com
✔ Added user "work" (alice@company.com)
ℹ Run 'git-user switch work' to activate

$ git user add home   alice@gmail.com
✔ Added user "home" (alice@gmail.com)
ℹ Run 'git-user switch home' to activate

$ git user list
ℹ Git Identities
─────────────────────────────────────────
  work                 alice@company.com
  home                 alice@gmail.com
─────────────────────────────────────────
  No active identity — run 'git-user switch <name>'

$ git user switch work
✔ Switched to "work" (alice@company.com)

$ git user current
ℹ Active Identity
─────────────────────────────────────────
  Name  : work
  Email : alice@company.com
─────────────────────────────────────────

$ git user list
ℹ Git Identities
─────────────────────────────────────────
▶ work                 alice@company.com
  home                 alice@gmail.com
─────────────────────────────────────────
  Active: work

$ git user edit home  personal@proton.me
✔ Updated "home" → email is now personal@proton.me

$ git user switch home
✔ Switched to "home" (personal@proton.me)

$ git user remove work
✔ Removed user "work"

$ git user remove home
✖ user "home" is currently active; use --force to remove

$ git user remove home --force
✔ Removed user "home"
⚠ No active identity — run 'git-user switch <name>'
```

---

## Storage

Config lives at `~/.git-users/config.json`:

```json
{
  "current": "work",
  "users": [
    { "name": "work", "email": "alice@company.com" },
    { "name": "home", "email": "alice@gmail.com"   }
  ]
}
```

- File permissions: `0600` (owner read/write only)
- Directory permissions: `0700`

---

## Project structure

```
git-user/
├── main.go                    # Entry point
├── go.mod
├── Makefile
├── install.sh
├── cmd/
│   ├── root.go                # Subcommand dispatcher + usage
│   ├── add.go
│   ├── list.go
│   ├── switch.go
│   ├── current.go
│   ├── remove.go
│   └── edit.go
└── internal/
    ├── config/
    │   ├── config.go          # Storage: load, save, CRUD on Store
    │   └── config_test.go
    ├── git/
    │   └── git.go             # Thin wrapper around `git config --global`
    └── ui/
        └── ui.go              # Colour output helpers
```

**Separation of concerns:**

| Layer | Responsibility |
|---|---|
| `cmd/` | CLI parsing, user-facing messages |
| `internal/config` | Config file I/O, domain rules (no duplicates, active-user guard) |
| `internal/git` | All `git` subprocess calls — isolated for easy mocking |
| `internal/ui` | Colour / formatting — zero business logic |

---

## Running tests

```bash
go test ./...
```

---

## Phase 2 — extension ideas

| Feature | Where to add |
|---|---|
| SSH key binding per identity | Add `SSHKey string` to `User`; `git/git.go` writes `~/.ssh/config` |
| GPG signing key per identity | Add `SigningKey string`; apply via `git config user.signingkey` |
| Per-repo identity override | New command `git user local <name>` — uses `--local` flag in git |
| Shell prompt integration | Expose `git-user current --short` for PS1 scripts |
| Import from existing `.gitconfig` | `git user import` reads current global config and creates an entry |
| Multiple config profiles | Support `GIT_USER_CONFIG` env var to point to alternate config files |
| Interactive TUI | Use `bubbletea` for arrow-key identity selection |

All Phase 2 features are additive — they extend `User`, add new subcommands, or add flags to existing ones. The storage format is forward-compatible (unknown JSON fields are ignored).

---

## License

MIT
