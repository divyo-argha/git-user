<div align="center">
  <br />
  <img src="img/git-user-logo-clean.png" alt="git-user" width="120" height="120" style="border-radius:26px" />
  <!-- <br /><br /> -->
  <h1>git-user</h1>

  <p>
    <strong>One command to rule all your Git identities.</strong><br />
    Stop committing as the wrong person. Stop juggling SSH keys. Stop editing config files.
  </p>

  <p>
    <a href="https://github.com/divyo-argha/git-user/releases"><img src="https://img.shields.io/github/v/release/divyo-argha/git-user?style=flat&color=00FFAA&label=latest" alt="Latest Release" /></a>
    <a href="https://github.com/divyo-argha/git-user/releases"><img src="https://img.shields.io/github/downloads/divyo-argha/git-user/total?style=flat&color=00FFAA&label=gh%20downloads" alt="GitHub Downloads" /></a>
    <a href="https://www.npmjs.com/package/git-userhub"><img src="https://img.shields.io/npm/v/git-userhub?style=flat&color=CB3837&logo=npm&logoColor=white&label=npm" alt="npm" /></a>
    <a href="https://www.npmjs.com/package/git-userhub"><img src="https://img.shields.io/npm/dt/git-userhub?style=flat&color=CB3837&logo=npm&logoColor=white&label=total%20downloads" alt="total downloads" /></a>
    <a href="https://github.com/divyo-argha/git-user"><img src="https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/divyo-argha/git-user/main/badges/total-downloads.json&style=flat" alt="Combined Downloads" /></a>
    <a href="https://pkg.go.dev/github.com/divyo-argha/git-user"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-22c55e?style=flat" alt="MIT" /></a>
  </p>

  <p>
    <a href="#-the-problem">The Problem</a> ·
    <a href="#-install">Install</a> ·
    <a href="#-quick-start">Quick Start</a> ·
    <a href="#-why-git-user">Why git-user</a> ·
    <a href="#-features">Features</a> ·
    <a href="#-commands">Commands</a> ·
    <a href="#-security">Security</a> ·
    <a href="#-contributing">Contributing</a>
  </p>

  <br />

  <img src="https://img.shields.io/badge/GitHub-supported-181717?style=flat&logo=github&logoColor=white" alt="GitHub" />
  <img src="https://img.shields.io/badge/GitLab-supported-FC6D26?style=flat&logo=gitlab&logoColor=white" alt="GitLab" />
  <img src="https://img.shields.io/badge/Bitbucket-supported-0052CC?style=flat&logo=bitbucket&logoColor=white" alt="Bitbucket" />
  <img src="https://img.shields.io/badge/macOS-supported-000000?style=flat&logo=apple&logoColor=white" alt="macOS" />
  <img src="https://img.shields.io/badge/Linux-supported-FCC624?style=flat&logo=linux&logoColor=black" alt="Linux" />
  <img src="https://img.shields.io/badge/Windows-supported-0078D4?style=flat&logo=windows&logoColor=white" alt="Windows" />

  <br /><br />

</div>

---

## 😤 The Problem

You're a developer with multiple lives — work, personal, freelance, open source. Each one has its own Git account, its own SSH key, its own email.

And every few weeks, this happens:

```
# You just pushed 3 commits to your client's repo.
# Then you check the author.

Author: you@personal.com   ← 💀 wrong account. again.

# Or your work email leaked onto your public GitHub profile.
# Or your client can see your personal email in their repo history.
```

You've tried everything:

| Attempt | Result |
|---------|--------|
| Editing `~/.gitconfig` manually | You forget. Every time. |
| Per-repo `.git/config` overrides | Works until you clone a new repo |
| Multiple terminal profiles | Still mix them up |
| SSH config `Host` aliases | Breaks half your existing remotes |
| Remembering which key goes where | Not a real solution |

**git-user is the permanent fix.** Register your identities once. Switch with one command. Everything — git config, SSH key, remote verification — updates automatically in under a second.

---

## 📦 Install

<table>
<tr>
<td width="33%" valign="top">

### One-line
```bash
curl -sSfL https://raw.githubusercontent.com/divyo-argha/git-user/main/install.sh | bash
```
Restart your terminal. PATH is configured automatically.

</td>
<td width="33%" valign="top">

### npm
```bash
npm install -g git-userhub
```
> Published as `git-userhub` on npm.
> After install, the command is `git-user`.

</td>
<td width="33%" valign="top">

### Go
```bash
go install github.com/divyo-argha/git-user@latest
```

### Self-update
```bash
git-user --update
```

</td>
</tr>
</table>

**Requirements:** ![Git](https://img.shields.io/badge/Git-required-F05032?style=flat&logo=git&logoColor=white) · ssh-keygen (optional, for SSH key generation)

---

## ⚡ Quick Start

Two minutes to set up. One second to switch forever after.

```bash
# Step 1 — register your identities (guided, interactive)
git-user register   # → name: work,     email: you@company.com
git-user register   # → name: personal, email: you@gmail.com
git-user register   # → name: client-a, email: you@client.com

# Step 2 — switch
git-user switch work

# Step 3 — push. that's it.
git push   # ← commits as you@company.com ✓
```

```bash
# Create and switch in one command
git-user switch -c freelance me@freelance.com

# Always know who you are
git-user current
```

---

## 🏆 Why git-user?

There are other tools that try to solve this. Here's how git-user is different:

| Feature | git-user | direnv / per-dir config | SSH `Host` aliases | Manual `~/.gitconfig` |
|---------|:--------:|:----------------------:|:------------------:|:---------------------:|
| One command to switch everything | ✅ | ❌ | ❌ | ❌ |
| SSH key managed automatically | ✅ | ❌ | ⚠️ partial | ❌ |
| Works across all repos, not just one | ✅ | ❌ | ✅ | ✅ |
| SSH connection verified on switch | ✅ | ❌ | ❌ | ❌ |
| Clean logout/sign-out to void state | ✅ | ❌ | ❌ | ❌ |
| Encrypted export/import | ✅ | ❌ | ❌ | ❌ |
| Pre-commit identity guard | ✅ | ❌ | ❌ | ❌ |
| Security audit built-in | ✅ | ❌ | ❌ | ❌ |
| Interactive TUI | ✅ | ❌ | ❌ | ❌ |
| Shell completions | ✅ | ❌ | ❌ | ❌ |
| Zero config files to edit manually | ✅ | ❌ | ❌ | ❌ |

> **The key difference:** git-user manages the *whole identity* — name, email, SSH key, and passphrase protection — as a single atomic unit. Other approaches only solve part of the problem, leaving you to manually wire the rest.

---

## ✨ Features

<table>
<tr>
<td width="50%" valign="top">

### 🔑 Identity Management
- Register unlimited identities — name, email, SSH key
- Switch in one command, git config updates instantly
- `switch -c <name>` — create and switch in one step
- Edit email without re-registering
- Remove identities safely, with active-identity guard

</td>
<td width="50%" valign="top">

### 🔐 SSH Key Handling
- Auto-generate ed25519 keys per identity
- `pubkey` — print active identity's public key (add to GitHub, GitLab, Bitbucket)
- Bind any existing key to any identity
- `rekey` rotates keys with automatic backup and rollback
- `IdentitiesOnly yes` — SSH never leaks the wrong key

</td>
</tr>
<tr>
<td width="50%" valign="top">

### 🛡️ Security & Passphrases
- Passphrase-protected keys enforced by default
- `security` audits every identity: permissions, passphrase, key existence
- `passphrase` changes passphrase only for the active, unlocked identity
- All config writes are atomic (temp file + rename) — crash-safe
- All files stored at `0600` permissions

</td>
<td width="50%" valign="top">

### 🔒 Passphrase-Gated Switching
- Gated switch: switching to a passphrase-protected profile requires entering the passphrase to unlock the SSH key
- Seamless ssh-agent management: the SSH key is added automatically on switch
- Security by default: you cannot act as an identity without verifying the passphrase first
- Clean logout: sign out at any time to clear active user config completely

</td>
</tr>
<tr>
<td width="50%" valign="top">

### 🚀 Passwordless Push
- Detects HTTPS remotes on `switch` and offers to convert
- `fix-remote` converts all remotes HTTPS → SSH instantly
- Works with GitHub, GitLab, Bitbucket, and any Git host

</td>
<td width="50%" valign="top">

### 🖥️ Developer Experience
- Interactive TUI menu (`git-user tui`)
- Shell completions for bash, zsh, fish
- Pre-commit hooks to block wrong-identity commits
- `doctor` diagnoses your entire setup in one command
- Encrypted export/import for moving to a new machine

</td>
</tr>
</table>

---

## 🔄 How It Works

### Under the hood — one switch

```
git-user switch work
        │
        ▼
  1. Looks up "work" in ~/.git-users/config.json
  2. Sets ~/.gitconfig  →  user.name, user.email
  3. Sets ~/.gitconfig  →  core.sshCommand (points to your key)
  4. Verifies SSH connection to GitHub/GitLab/Bitbucket
  5. ✅ Switched to "work" (you@company.com)

git push  ← just works, every time
```

### A real day with multiple accounts

```
 9:00 AM — starting work
──────────────────────────────────────────────────────────
 $ git-user switch work
   ✅ Switched to work (you@company.com)
 $ git push                        ← commits as you@company.com ✓

 1:00 PM — open source on lunch break
──────────────────────────────────────────────────────────
 $ git-user switch personal
   ✅ Switched to personal (you@gmail.com)
 $ git push                        ← commits as you@gmail.com ✓

 5:00 PM — freelance client work
──────────────────────────────────────────────────────────
 $ git-user switch client-a
   ✅ Switched to client-a (you@client-a.com)
 $ git push                        ← commits as you@client-a.com ✓
```

Each switch: under one second. No config editing. No SSH juggling.

---

## 🚪 Logout / Void State

When you are done with your work or leaving a shared machine, you can sign out to clear your active Git identity completely:

```bash
git-user logout
```

What happens:
- Unloads the active SSH key from `ssh-agent`
- Clears the global `user.name` and `user.email` from `~/.gitconfig`
- Clears `core.sshCommand` from `~/.gitconfig`
- Puts the terminal into a clean "void" state (no git user configured), preventing accidental commits under your identity by other users.

---

## 📋 Commands

| Command | Description |
|---------|-------------|
| `register` | Create a new identity (guided setup with SSH) |
| `switch <name>` | Switch to an identity |
| `switch -c <name> [email]` | Create and switch in one command |
| `list` | Show all identities |
| `current` | Show active identity |
| `remove <name>` | Delete an identity |
| `edit <name> <email>` | Update email |
| `bind <name> [--ssh-key <path>]` | Link an SSH key to an identity |
| `pubkey` | Show the public key of the active identity |
| `passphrase` | Add or change passphrase for the active, unlocked identity |
| `rekey <name>` | Rotate SSH key (with rollback safety) |
| `fix-remote` | Convert HTTPS remotes to SSH |
| `logout` | Sign out, clearing the active identity and restoring a void state |
| `security` | Audit all identities for security issues |
| `export --all` | Export all identities + SSH keys (AES-256 encrypted) |
| `export <name> [name...]` | Export specific identities |
| `import <file>` | Import from an encrypted bundle |
| `doctor` | Run a full health check |
| `tui` | Interactive menu |
| `completion <shell>` | Shell completions (bash/zsh/fish) |
| `hook <install\|uninstall>` | Pre-commit hook to verify identity |
| `--update` | Update to the latest version |
| `--version` / `-v` | Show version |

**Aliases:** `ls` → `list` · `sw` → `switch` · `rm` → `remove`

---

## 🛡️ Security

<table>
<tr>
<td width="50%" valign="top">

**What git-user does**
- Private keys stay on your machine at `0600` permissions
- Config writes are atomic (temp file + rename) — crash-safe
- `IdentitiesOnly yes` in SSH config — no key leakage
- Passphrase protection audited by `security` command
- Export bundles encrypted with AES-256-GCM, passphrase stretched with scrypt (N=2¹⁷)
- Passphrases are never passed as CLI arguments — entered directly into the terminal
- `pubkey` only shows the active identity's key — other identities' keys are never exposed

</td>
<td width="50%" valign="top">

**What git-user never does**
- Never stores passphrases
- Never sends keys or config anywhere
- Never modifies your repositories
- Never overwrites existing identities on import
- `logout` command cleanly clears all gitconfig references and unloads loaded keys

</td>
</tr>
</table>

### Run a security audit

```bash
git-user security
```

```
✔ Config file permissions OK (0600)

ℹ work (you@company.com)
  ✔ Permissions OK: git_work
  ✔ Passphrase protected

ℹ personal (you@gmail.com)
  ✔ Permissions OK: git_personal
  ⚠ No passphrase detected
    Fix: ssh-keygen -p -f ~/.ssh/git_personal
```

---

## 🚚 Moving to a New Machine

```bash
# On your current machine
git-user export --all
# → ~/git-user-export-2026-05-29.bundle  (AES-256 encrypted)

# Transfer the file, then on the new machine
git-user import ~/git-user-export-2026-05-29.bundle
# ✅ Imported: work (you@company.com) → ~/.ssh/git_work
# ✅ Imported: personal (you@gmail.com) → ~/.ssh/git_personal

git-user switch work
# Ready to push immediately
```

---

## 🔧 Troubleshooting

```bash
git-user doctor
```

```
✅ git installed (2.43.0)
✅ ssh-keygen available
✅ Active identity: work (you@company.com)
✅ SSH key exists at ~/.ssh/git_work
✅ Key permissions OK (0600)
✅ GitHub connection verified — Hi alice-corp!
──────────────────────────────────────────────
Everything looks good.
```

**Common issues:**

| Symptom | Fix |
|---------|-----|
| `git-user: command not found` | Restart terminal or `source ~/.zshrc` |
| SSH verification failed | Key not added to platform yet — run `git-user pubkey` to copy the public key |
| `Permission denied` during install | Expected — installer needs sudo for `/usr/local/bin` |
| Git asks for credentials on push | Run `git-user fix-remote` to convert HTTPS → SSH |

---

## 🐚 Shell Completions

```bash
# Bash
git-user completion bash | sudo tee /etc/bash_completion.d/git-user

# Zsh
git-user completion zsh > "${fpath[1]}/_git-user"

# Fish
git-user completion fish > ~/.config/fish/completions/git-user.fish
```

```bash
git-user sw<TAB>          # → git-user switch
git-user switch <TAB>     # → work  personal  client-a
git-user remove <TAB>     # → your identity names
```

---

## 🎨 Terminal Prompt Integration

You can display your active `git-user` profile directly in your terminal prompt (like Starship, Powerlevel10k, Zsh, or Fish). The `git-user prompt` command is extremely fast and will only output your profile name if you are currently inside a git repository, making it perfect for custom prompt segments!

To avoid automatically modifying your personal shell configurations, we've provided simple, copy-paste instructions for all the popular shells.

👉 **[View the Terminal Integration Guide](./TERMINAL-INTEGRATION.md)**

---

## 🪝 Pre-commit Hooks

```bash
git-user hook install   # in any repo where identity matters
```

```bash
git commit -m "Add feature"

# ✖ Identity mismatch!
#   Expected: work (you@company.com)
#   Git config: you@gmail.com
#   Run: git-user switch work
```

---

## 📁 What Gets Modified

```
~/.git-users/
  └── config.json          ← your identities (names, emails, key paths)

~/.gitconfig               ← updated on every switch/logout (name, email, sshCommand)
~/.ssh/git_<name>          ← private key (never leaves your machine)
~/.ssh/git_<name>.pub      ← public key (what you add to GitHub/GitLab)
```

Your repositories are never touched. Only global git config changes.

---

## 🤝 Contributing

Issues and pull requests are welcome. If something's broken, open an issue. If something's confusing — even just "I didn't understand what this command does" — that's worth filing too.

```bash
git clone https://github.com/divyo-argha/git-user.git
cd git-user
make build   # build binary
make test    # run tests
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

---

## 📄 License

MIT — see [LICENSE](LICENSE).

---

<div align="center">

**Made for developers who just want their Git to work.**

<br />

[![GitHub](https://img.shields.io/badge/Star%20on%20GitHub-181717?style=flat&logo=github&logoColor=white)](https://github.com/divyo-argha/git-user)
[![npm](https://img.shields.io/badge/Install%20via%20npm-CB3837?style=flat&logo=npm&logoColor=white)](https://www.npmjs.com/package/git-userhub)

<br />

<sub>If git-user saved you from a wrong-account commit, consider giving it a ⭐</sub>

</div>
