<div align="center">
  <br />
  <img src="logo/gu_logo.png" alt="git-user" width="130" height="130" style="border-radius:26px" />
  <pre style="line-height: 1.25; font-weight: bold; background: none; border: none; padding: 0; display: inline-block; text-align: left; font-family: monospace;">
  <span style="color: #F97316;">▄▀▀ █ ▀█▀</span>       <span style="color: #E2E8F0;">█ █ ▀▀▀ █▀▀ █▀▄</span>
  <span style="color: #F97316;">█ ▄ █  █</span>   <span style="color: #94A3B8;">▄▄▄</span>  <span style="color: #E2E8F0;">█ █ ▀▀▄ █▀  █▀▀</span>
  <span style="color: #F97316;">▀▀▀ ▀  ▀</span>        <span style="color: #E2E8F0;">▀▀▀ ▀▀▀ ▀▀▀ ▀ ▀</span>
  </pre>

  <p>
    <strong>Switch Git identities in one command.</strong><br />
    No config editing. No SSH key chaos. No wrong-account commits.
  </p>

  <p>
    <a href="https://www.npmjs.com/package/git-userhub"><img src="https://img.shields.io/npm/v/git-userhub?style=flat&color=CB3837&logo=npm&logoColor=white" alt="npm version" /></a>
    <a href="https://github.com/divyo-argha/git-user"><img src="https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/divyo-argha/git-user/main/badges/total-downloads.json" alt="Total Downloads" /></a>
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
git-user switch -c freelance -e me@freelance.com
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
| 🛡️ **Security audit** | `git-user audit` checks permissions and passphrase protection |
| 🖋️ **Commit signing** | `git-user sign` configures automatic SSH commit signing |
| 🔑 **Keychain integration** | Secure system keychain integration for auto-unlocking passphrase keys |
| 🎨 **Terminal prompt** | Dynamic prompt indicator (installer command: `git-user prompt install`) |
| 📂 **Auto-switching** | `bind-path` / `unbind-path` — per-directory identity via git `includeIf` |
| 🚀 **HTTPS → SSH** | `git-user fix-remote` converts remotes for passwordless push |
| 🪝 **Pre-commit hooks** | Block commits if the wrong identity is active |
| 📦 **Export/import** | Move all identities to a new machine, AES-256 encrypted |
| 🖥️ **TUI** | Interactive menu for everything |
| 🐚 **Shell completions** | bash, zsh, fish |

---

## All Commands

```
register                    Create new identity (guided)
switch <name> [--local]     Switch identity (global or repository-local override)
switch -c <name> [-e <email>] Create and switch in one step
switch --original           Import and switch to the original pre-git-user identity
list                        Show all identities
current                     Show active identity
prompt                      Output active identity for terminal integration
remove <name>               Delete an identity
edit <name> <email>         Update email
rename <old> <new>          Rename an identity
bind-key <name>              Link an SSH key
bind-path <name> <path>     Bind a directory to an identity
unbind-path <name> <path>   Remove a directory binding
pubkey                      Show public key of active identity
pubkey publish [platform]   Publish the key to GitHub, GitLab, or Bitbucket
passphrase                  Add, change, or remove (--remove) passphrase for active identity
sign <name> [--on|--off]    Enable/disable automatic Git commit signing
rekey <name>                Rotate SSH key
fix-remote                  Convert HTTPS remotes to SSH
logout                      Sign out and clear the active identity
audit                       Audit all identities
export --all                Export encrypted bundle
export <name> [name...]     Export specific identities
switch --original [name]    Import and switch to the original pre-git-user identity
import <file>               Import from bundle
clone <url> [dir]           Clone and auto-configure the local identity
stats                       Audit commit author identity stats
config <list|set|unset>      Manage custom git configurations
sync                        Synchronize identities via a private repository
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
```

Your repositories are never touched.

---

## Full Documentation

**→ [github.com/divyo-argha/git-user](https://github.com/divyo-argha/git-user)**

---

## License

MIT

---
