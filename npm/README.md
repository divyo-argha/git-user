<div align="center">
  <br />
  <img src="https://raw.githubusercontent.com/divyo-argha/git-user/main/img/git-user-logo-clean.png" alt="git-user" width="100" height="100" style="border-radius:28px" />
  <h1>git-user</h1>

  <p>
    <strong>Switch Git identities in one command.</strong><br />
    No config editing. No SSH key chaos. No wrong-account commits.
  </p>

  <p>
    <a href="https://www.npmjs.com/package/git-userhub"><img src="https://img.shields.io/npm/v/git-userhub?style=flat&color=CB3837&logo=npm&logoColor=white" alt="npm version" /></a>
    <a href="https://www.npmjs.com/package/git-userhub"><img src="https://img.shields.io/npm/dt/git-userhub?style=flat&color=CB3837&logo=npm&logoColor=white&label=total%20downloads" alt="total downloads" /></a>
    <a href="https://github.com/divyo-argha/git-user/releases"><img src="https://img.shields.io/github/v/release/divyo-argha/git-user?style=flat&color=00FFAA&label=latest" alt="Latest Release" /></a>
    <a href="https://github.com/divyo-argha/git-user/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-22c55e?style=flat" alt="MIT" /></a>
  </p>

  <img src="https://img.shields.io/badge/GitHub-supported-181717?style=flat&logo=github&logoColor=white" alt="GitHub" />
  <img src="https://img.shields.io/badge/GitLab-supported-FC6D26?style=flat&logo=gitlab&logoColor=white" alt="GitLab" />
  <img src="https://img.shields.io/badge/Bitbucket-supported-0052CC?style=flat&logo=bitbucket&logoColor=white" alt="Bitbucket" />
  <img src="https://img.shields.io/badge/macOS-supported-000000?style=flat&logo=apple&logoColor=white" alt="macOS" />
  <img src="https://img.shields.io/badge/Linux-supported-FCC624?style=flat&logo=linux&logoColor=black" alt="Linux" />
  <img src="https://img.shields.io/badge/Windows-supported-0078D4?style=flat&logo=windows&logoColor=white" alt="Windows" />

  <br /><br />

</div>

---

> **Note on the package name:** This package is published as `git-userhub` on npm. After installation, the CLI command is `git-user`. The npm name is just the registry identifier — everything you run is `git-user`.

---

## Install

```bash
npm install -g git-userhub
```

That's it. The `git-user` command is now available in your terminal.

---

## The Problem It Solves

You have a work account, a personal account, maybe a freelance client or two. Every few weeks you push commits with the wrong email. Your personal address ends up in a client's repo history. Your work email leaks onto your public GitHub profile.

git-user fixes this permanently. Register your identities once, switch with one command.

---

## Quick Start

```bash
# Register your identities (guided, takes ~2 minutes each)
git-user register   # name: work,     email: you@company.com
git-user register   # name: personal, email: you@gmail.com

# Switch between them instantly
git-user switch work
git-user switch personal

# See what's active
git-user current

# Create and switch in one step
git-user switch -c freelance me@freelance.com
```

---

## What It Does on Switch

```
git-user switch work
        │
        ▼
  1. Reads "work" from ~/.git-users/config.json
  2. Sets ~/.gitconfig  →  user.name, user.email
  3. Sets ~/.gitconfig  →  core.sshCommand (your SSH key)
  4. Verifies SSH connection
  5. ✅ Done — under one second
```

---

## Key Features

| Feature | Description |
|---------|-------------|
| 🔑 **Identity switching** | Name + email + SSH key as one atomic unit |
| 🔐 **SSH key management** | Auto-generate ed25519 keys, bind existing keys, `pubkey` shows active key only |
| 🔒 **Temporary sessions** | Use an identity on a shared machine — zero trace left behind |
| 🛡️ **Security audit** | `git-user security` checks permissions and passphrase protection |
| 🔑 **Keychain integration** | Secure system keychain integration for auto-unlocking passphrase keys |
| 🎨 **Terminal prompt** | Dynamic prompt indicator (installer command: `git-user prompt install`) |
| 🚀 **HTTPS → SSH** | `git-user fix-remote` converts remotes for passwordless push |
| 🪝 **Pre-commit hooks** | Block commits if the wrong identity is active |
| 📦 **Export/import** | Move all identities to a new machine, AES-256 encrypted |
| 🖥️ **TUI** | Interactive menu for everything |
| 🐚 **Shell completions** | bash, zsh, fish |

---

## Temporary Sessions

Working on a borrowed machine? Don't want to leave SSH keys behind?

```bash
# Start a temporary session — nothing is saved permanently
git-user session start --temp alice me@work.com --ttl 2h

# When done — key files deleted, previous identity restored
git-user session stop
```

---

## All Commands

```
register                    Create new identity (guided)
switch <name>               Switch to an identity
switch -c <name> [email]    Create and switch in one step
list                        Show all identities
current                     Show active identity
remove <name>               Delete an identity
edit <name> <email>         Update email
bind <name>                 Link an SSH key
pubkey                      Show public key of active identity
passphrase                  Add, change, or remove (--remove) passphrase for active identity
rekey <name>                Rotate SSH key
fix-remote                  Convert HTTPS remotes to SSH
session start [--ttl <d>]   Load SSH key into ssh-agent
session start --temp ...    Temporary session (nothing saved)
session stop                Unload key / end temp session
session status              Show agent status
security                    Audit all identities
export --all                Export encrypted bundle
import <file>               Import from bundle
doctor                      Full health check
tui                         Interactive menu
completion <shell>          Shell completions
hook install|uninstall      Pre-commit identity guard
--update                    Update to latest version
--version / -v              Show version
```

**Aliases:** `ls` → `list` · `sw` → `switch` · `rm` → `remove`

---

## What Gets Modified

```
~/.git-users/config.json     ← your identities (never auto-deleted)
~/.gitconfig                 ← updated on every switch
~/.ssh/git_<name>            ← private key (stays on your machine)
~/.ssh/git_tmp_<name>        ← temp session key (deleted on stop)
```

Your repositories are never touched.

---

## Full Documentation

**→ [github.com/divyo-argha/git-user](https://github.com/divyo-argha/git-user)**

---

## License

MIT

---

<div align="center">

[![GitHub](https://img.shields.io/badge/Star%20on%20GitHub-181717?style=flat&logo=github&logoColor=white)](https://github.com/divyo-argha/git-user)

<sub>If git-user saved you from a wrong-account commit, consider giving it a ⭐</sub>

</div>
