<div align="center">
  <br />
  <img src="https://raw.githubusercontent.com/divyo-argha/git-user/main/logo/gu_logo.png" alt="git-user logo" width="120" height="120" style="border-radius:24px" />
  
  # git-user

  <h3>Stop committing with the wrong identity.</h3>
  <p>
    <strong>Switch Git accounts, SSH keys, and commit signing instantly in one command.</strong><br />
    No manual <code>.gitconfig</code> editing. No SSH key chaos. No wrong-account commits.
  </p>

  <p>
    <a href="https://www.npmjs.com/package/git-userhub"><img src="https://img.shields.io/npm/v/git-userhub?style=flat-square&color=CB3837&logo=npm&logoColor=white" alt="npm version" /></a>
    <a href="https://www.npmjs.com/package/git-userhub"><img src="https://img.shields.io/npm/dt/git-userhub?style=flat-square&color=CB3837&logo=npm&logoColor=white&label=npm%20downloads" alt="npm downloads" /></a>
    <a href="https://github.com/divyo-argha/git-user/releases"><img src="https://img.shields.io/github/v/release/divyo-argha/git-user?style=flat-square&color=00FFAA&label=latest" alt="Latest Release" /></a>
    <a href="https://github.com/divyo-argha/git-user/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-22c55e?style=flat-square" alt="MIT" /></a>
  </p>

  <p>
    <img src="https://img.shields.io/badge/GitHub-supported-181717?style=flat-square&logo=github&logoColor=white" alt="GitHub" />
    <img src="https://img.shields.io/badge/GitLab-supported-FC6D26?style=flat-square&logo=gitlab&logoColor=white" alt="GitLab" />
    <img src="https://img.shields.io/badge/Bitbucket-supported-0052CC?style=flat-square&logo=bitbucket&logoColor=white" alt="Bitbucket" />
    <img src="https://img.shields.io/badge/macOS-supported-000000?style=flat-square&logo=apple&logoColor=white" alt="macOS" />
    <img src="https://img.shields.io/badge/Linux-supported-FCC624?style=flat-square&logo=linux&logoColor=black" alt="Linux" />
    <img src="https://img.shields.io/badge/Windows-supported-0078D4?style=flat-square&logo=windows&logoColor=white" alt="Windows" />
  </p>

  <br />
</div>

> 💡 **npm Package Note:** This package is published as `git-userhub` on npm. Once installed, your terminal command is **`git-user`**.

---

## ⚡ Quick Start

### 1. Installation
```bash
npm install -g git-userhub
```

### 2. Register Your Identities
```bash
git-user register
```
*Interactive prompts guide you to set up your work, personal, or freelance profiles with names, emails, and bound SSH keys.*

### 3. Switch Instantly
```bash
# Switch active identity globally
git-user switch work

# Or switch for the current repository only
git-user switch personal --local
```

### 4. Interactive Terminal UI (TUI)
Prefer an interactive visual menu? Launch the keyboard-driven TUI:
```bash
git-user tui
```
*Navigate with <kbd>↑</kbd> / <kbd>↓</kbd> (or <kbd>k</kbd> / <kbd>j</kbd>), switch panes with <kbd>Tab ⇥</kbd>, and select with <kbd>Enter ↵</kbd>.*

---

## 🎯 What It Handles Automatically

When you switch identities, `git-user` updates your environment atomically in **<0.1 seconds**:

* 👤 **Git Identity**: Updates `user.name` and `user.email`.
* 🔑 **SSH Key**: Sets `core.sshCommand` so Git operations automatically use the bound SSH key.
* 🖋️ **Commit Signing**: Configures SSH/GPG commit signing parameters automatically.
* 📂 **Directory Auto-Switching**: Binds workspace folders (`~/work`, `~/personal`) to switch identity automatically via `includeIf`.

---

## 🔥 Key Features & Highlights

| Feature | Highlight | Documentation |
| :--- | :--- | :--- |
| ⚡ **Atomic Switch** | Swaps name, email, and SSH keys in under 100ms | [Learn More →](https://github.com/divyo-argha/git-user#-%EF%B8%8F-quick-start) |
| 📂 **Auto-Switch Directories** | Bind folders (`git-user bind-path work ~/work/`) to auto-switch | [Learn More →](https://github.com/divyo-argha/git-user#-directory-auto-switching-bind-path) |
| 🪝 **Pre-Commit Guard** | Block accidental commits if the wrong identity is active | [Learn More →](https://github.com/divyo-argha/git-user#-%EF%B8%8F-pre-commit-hooks) |
| 🖋️ **Commit Signing** | Native SSH & GPG commit signing support | [Learn More →](https://github.com/divyo-argha/git-user#-%EF%B8%8F-commit-signing-git-user-sign) |
| 🔄 **Encrypted Sync** | Sync identities across machines via a private Git repo | [Learn More →](https://github.com/divyo-argha/git-user#-%EF%B8%8F-cross-device-sync-git-user-sync) |
| 🖥️ **Rich TUI** | Keyboard-navigable (<kbd>↑</kbd> <kbd>↓</kbd> <kbd>↵</kbd> <kbd>Tab</kbd>) terminal interface | [Learn More →](https://github.com/divyo-argha/git-user#-%EF%B8%8F-interactive-tui-git-user-tui) |
| 🎨 **Terminal Prompt** | Show active Git identity in Zsh, Bash, Fish, or Starship prompt | [Integration Guide →](https://github.com/divyo-argha/git-user/blob/main/TERMINAL-INTEGRATION.md) |

---

## 📌 Command Reference Cheat Sheet

```bash
# Core Operations
git-user register                      # Guided setup for a new identity
git-user switch <name>                 # Switch active global identity
git-user switch <name> --local         # Switch identity for current repository only
git-user switch <name> --session       # Switch identity for current terminal session only
git-user switch -c <name> -e <email>   # Create and switch in one step
git-user current                       # Print currently active identity
git-user list                          # List all registered identities

# Terminal Session Isolation (Multi-Terminal Workflows)
eval "$(git-user init)"                # Enable seamless per-session switching
git-user switch --session <name>       # Lock current terminal tab to an identity
eval "$(git-user env <name>)"          # Export identity env vars directly
git-user shell <name>                  # Launch an isolated subshell for an identity
git-user exec <name> -- <cmd...>       # Run a single command under an identity

# Directory & Auto-Switching
git-user bind-path <name> <dir>        # Bind directory to auto-switch identity
git-user unbind-path <name> <dir>      # Remove directory binding

# Security & SSH Management
git-user pubkey                        # Output public key of active identity
git-user pubkey publish [platform]     # Upload public key to GitHub/GitLab/Bitbucket
git-user sign <name> [--on|--off]      # Toggle automatic SSH commit signing
git-user hook install                  # Install pre-commit guard hook

# Sync & Maintenance
git-user sync                          # Sync identities across devices
git-user doctor                        # Run environment health check
git-user tui                           # Open interactive terminal interface
git-user --update                      # Update to latest version
```

---

## 📚 Full Documentation & Deep Dives

For detailed guides, troubleshooting, terminal prompt customization, and security architecture:

* 📖 **[Main Project Repository & Documentation](https://github.com/divyo-argha/git-user#readme)**
* 🐚 **[Terminal & Shell Prompt Integration Guide](https://github.com/divyo-argha/git-user/blob/main/TERMINAL-INTEGRATION.md)**
* 🔒 **[Security Architecture & Audit Details](https://github.com/divyo-argha/git-user/blob/main/SECURITY.md)**
* 💬 **[Report Issues & Request Features](https://github.com/divyo-argha/git-user/issues)**

---

<div align="center">
  <sub>Built with ❤️ for developers who manage multiple Git identities. Distributed under the MIT License.</sub>
</div>
