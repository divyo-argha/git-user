# git-user

> Switch Git accounts in one command. No config editing. No SSH key chaos.

---

I built this because I got tired of the same problem every day: juggling multiple Git accounts (work, personal, clients) and constantly forgetting to switch my git config or SSH keys. 

`git-user` solves this. Register your identities once, then switch between them in one command.

---

## Installation

### One-line install (recommended)

```bash
curl -sSfL https://raw.githubusercontent.com/divyo-argha/git-user/main/install.sh | bash
```

Restart your terminal. PATH is configured automatically.

### Via npm

```bash
npm install -g @divyo-argha/git-user
```

### Via Go

```bash
go install github.com/divyo-argha/git-user@latest
```

**Requirements:** Git, ssh-keygen (optional)

---

## Quick Start

```bash
# Create your first identity
git-user register

# Or create and switch in one command
git-user switch -c work

# Switch between identities
git-user switch work
git-user switch personal

# List all identities
git-user list

# Check what's active
git-user current
```

---

## How It Works

### First Time Setup

When you run `git-user register`, you get a guided setup:

1. **Enter identity name** (e.g., "work", "personal")
2. **Enter email address**
3. **Choose SSH key setup:**
   - **Auto-generate** - Creates key, displays it in terminal, waits for you to add it to GitHub/GitLab
   - **Use existing key** - Provide path to your existing SSH key
   - **Skip** - Set up SSH later

The public key is displayed right in your terminal - just copy and paste it to GitHub/GitLab/Bitbucket.

### Quick Create and Switch

```bash
# Prompts for email and SSH setup, then switches immediately
git-user switch -c work

# Even faster with email
git-user switch -c work me@work.com
```

### Daily Usage

```bash
# Switch identities
git-user switch work      # ✓ Switched to "work" (you@company.com)
git-user switch personal  # ✓ Switched to "personal" (you@gmail.com)

# That's it. Git config and SSH are updated automatically.
```

---

## Commands

| Command | Description |
|---------|-------------|
| `register` | Create new identity (guided setup) |
| `switch <name>` | Switch to an identity |
| `switch -c <name> [email]` | Create and switch in one command |
| `list` | Show all identities |
| `current` | Show active identity |
| `remove <name>` | Delete an identity |
| `edit <name> <email>` | Update email |
| `bind <name> --ssh-key <path>` | Link SSH key to identity |
| `rekey <name>` | Rotate SSH key |
| `fix-remote` | Convert HTTPS remotes to SSH |
| `doctor` | Run health check |
| `tui` | Interactive menu |

**Aliases:** `ls` (list), `sw` (switch), `rm` (remove)

---

## Real-World Examples

### Freelancer with Multiple Clients

```bash
git-user register  # name: client-a, email: you@client-a.com
git-user register  # name: client-b, email: you@client-b.com
git-user register  # name: personal, email: you@gmail.com

# Before each work session
git-user switch client-a
```

### Work vs Personal

```bash
git-user register  # name: work, email: you@company.com
git-user register  # name: personal, email: you@gmail.com

git-user switch work      # at the office
git-user switch personal  # at home
```

### Quick Setup for Multiple Identities

```bash
git-user switch -c work me@work.com
git-user switch -c personal me@gmail.com
git-user switch -c client me@client.com
# Each gets its own SSH key automatically
```

---

## Passwordless Push with SSH

### The Problem

When you `git push` and see this:

```
Username for 'https://github.com': _
```

Your SSH keys are useless because the repository is using HTTPS, not SSH.

### The Solution

`git-user` automatically detects HTTPS remotes and offers to convert them:

```bash
$ git-user switch work
  ✅ Switched to work (you@company.com)
  
  ⚠️  This repo uses HTTPS remotes
      Convert to SSH for passwordless push? [Y/n] y
  
  ✅ origin: https://github.com/company/app.git → git@github.com:company/app.git
  
  Try: git push
```

### Manual Conversion

Already in a repo with HTTPS remotes? Fix it instantly:

```bash
$ git-user fix-remote

  ✅ origin: https://github.com/user/repo.git → git@github.com:user/repo.git
  ✅ upstream: https://github.com/org/repo.git → git@github.com:org/repo.git
  
  Converted 2 remote(s) to SSH
  Try: git push
```

Now `git push` works without credentials.

### How It Works

- **HTTPS URLs** require username/password or tokens
- **SSH URLs** use your SSH keys (already set up by git-user)
- `fix-remote` converts: `https://github.com/user/repo.git` → `git@github.com:user/repo.git`
- Works with GitHub, GitLab, Bitbucket, and any Git platform

### When to Use

Run `git-user fix-remote` when:
- You cloned a repo via HTTPS (GitHub's default)
- Git asks for credentials when pushing
- You want passwordless authentication

---

## SSH Key Options

### Option 1: Auto-generate (Recommended)

The tool creates a new SSH key and displays it in your terminal:

```
┌─────────────────────────────────────────┐
│     📋 YOUR PUBLIC KEY                  │
└─────────────────────────────────────────┘

ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... you@company.com

Copy the key above and add it to your Git platform:
  GitHub:    Settings → SSH and GPG keys → New SSH key
  GitLab:    Preferences → SSH Keys → Add new key
  Bitbucket: Personal settings → SSH keys → Add key
```

Just copy the key, add it to your platform, press Enter. The tool verifies the connection automatically.

### Option 2: Use Existing Key

Already have SSH keys? Just provide the path:

```bash
# During setup, choose option 2
Path to SSH key: ~/.ssh/id_ed25519
```

### Option 3: Skip

Skip SSH setup and add it later:

```bash
git-user bind work --ssh-key ~/.ssh/id_ed25519
```

---

## What Gets Modified

- **`~/.gitconfig`** - Updates `user.name`, `user.email`, and `core.sshCommand`
- **`~/.git-users/config.json`** - Stores your identities
- **`~/.ssh/git_<name>`** - SSH keys (if auto-generated)

Your repositories are never touched. Only your global Git config changes.

---

## Troubleshooting

```bash
# Run health check
git-user doctor

# Check version
git-user --version

# Get help
git-user --help
```

Common issues:
- **SSH verification failed** - The key may not be added to your platform yet, or needs a few seconds to propagate
- **Command not found** - Restart your terminal or run `source ~/.zshrc` (or `~/.bashrc`)

---

## Uninstall

```bash
# Remove binary
sudo rm /usr/local/bin/git-user

# Remove config (optional)
rm -rf ~/.git-users
```

---

## Contributing

Issues and pull requests welcome! See [CONTRIBUTING.md](CONTRIBUTING.md).

---

## License

MIT License - see [LICENSE](LICENSE)

---

*Made for developers who just want their Git to work.*

## Workflow diagrams

Pictures are worth a thousand man pages. Here's exactly what happens at each step.

---

### Setting up a new identity (first time)

```
┌──────────────────────────────────────────────────────────────────┐
│                       git-user register                          │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
                   ┌─────────────────────┐
                   │  What's your name?  │  ← e.g. "work" or "alice-personal"
                   └──────────┬──────────┘
                              │
                              ▼
                   ┌─────────────────────┐
                   │  What's your email? │  ← e.g. alice@company.com
                   └──────────┬──────────┘
                              │
                              ▼
             ┌────────────────────────────────┐
             │   Generate a new SSH key? Y/n  │
             └────────────┬───────────────────┘
                          │
           ┌──────────────┴──────────────┐
           ▼                             ▼
     ┌──────────┐                ┌──────────────────────┐
     │   YES    │                │          NO          │
     └────┬─────┘                └──────────┬───────────┘
          │                                 │
          ▼                                 ▼
┌──────────────────────┐       ┌────────────────────────────┐
│ Key generated at     │       │ Enter path to your key     │
│ ~/.ssh/git_work      │       │ e.g. ~/.ssh/id_ed25519     │
│                      │       └────────────────────────────┘
│ ┌──────────────────┐ │
│ │  PUBLIC KEY      │ │  ← printed right here in the terminal
│ │  ssh-ed25519 ... │ │     copy it, add it to GitHub/GitLab/Bitbucket
│ │  Fingerprint: .. │ │     (same key works on all platforms!)
│ └──────────────────┘ │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│  Added it to GitHub? │
│  Press Enter...      │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────┐
│  Testing connection  │
│  ssh -T git@github.. │
└──────────┬───────────┘
           │
     ┌─────┴──────┐
     ▼            ▼
  ✅ Works    ❌ Fails
  Identity    Shows you
  saved and   exactly
  activated   what to fix
```

**The whole thing takes under 2 minutes.** The public key is printed directly in your terminal — you don't have to hunt for it in `~/.ssh/`. The tool tests the connection before saving anything, so you never end up with a half-configured identity.

**Note:** One SSH key works on GitHub, GitLab, and Bitbucket. Just add the public key to whichever platforms you use.

---

### Switching between identities

Once you have two or more identities, switching looks like this:

```
You type:  git-user switch personal
                     │
                     ▼
      ┌──────────────────────────────────────────┐
      │           What git-user does             │
      │                                          │
      │  1. Finds "personal" in its store        │
      │     ~/.git-users/config.json             │
      │                                          │
      │  2. Updates ~/.gitconfig                 │
      │     user.name  = Alice                   │
      │     user.email = alice@gmail.com         │
      │                                          │
      │  3. Rewrites ~/.ssh/config block         │
      │     Host github.com                      │
      │       IdentityFile ~/.ssh/git_personal   │
      │       IdentitiesOnly yes                 │
      │                                          │
      │  4. Tests SSH connection                 │
      │     → Hi alice! (github.com)             │
      │                                          │
      │  5. ✅ Switched to personal              │
      └──────────────────────────────────────────┘
                     │
                     ▼
            git push just works.
```

One command. Half a second. You're done.

---

### A real day with multiple accounts

This is what switching actually looks like day-to-day:

```
 9:00 AM — starting work
──────────────────────────────────────────────────
 $ git-user switch work
   ✅ Switched to work (alice@company.com)

 $ cd ~/projects/company-app
 $ git commit -m "fix: null check on user input"
 $ git push                   ← commits as alice@company.com ✓


 1:00 PM — open source, on your lunch break
──────────────────────────────────────────────────
 $ git-user switch personal
   ✅ Switched to personal (alice@gmail.com)

 $ cd ~/oss/my-cool-library
 $ git commit -m "feat: add streaming support"
 $ git push                   ← commits as alice@gmail.com ✓


 5:00 PM — freelance client work
──────────────────────────────────────────────────
 $ git-user switch client-a
   ✅ Switched to client-a (alice@client-a.com)

 $ cd ~/freelance/their-dashboard
 $ git push                   ← commits as alice@client-a.com ✓
```

Each switch takes less time than unlocking your screen. No config editing, no SSH agent juggling, no "wait, which account am I on?"

---

### Rotating an SSH key

Keys expire. Keys get revoked. Sometimes you just want a fresh one. `rekey` handles the whole process:

```
git-user rekey work
        │
        ▼
┌──────────────────────────────────────────┐
│  Generates new ed25519 key pair          │
│  ~/.ssh/git_work (new)                  │
└───────────────────┬──────────────────────┘
                    │
                    ▼
┌──────────────────────────────────────────┐
│  Prints new public key to terminal       │
│  → you copy it, add to GitHub            │
│  → press Enter when done                 │
└───────────────────┬──────────────────────┘
                    │
                    ▼
┌──────────────────────────────────────────┐
│  Tests connection with the new key       │
└───────────────────┬──────────────────────┘
                    │
          ┌─────────┴──────────┐
          ▼                    ▼
     ✅ Works             ❌ Fails
     Old key replaced     Old key kept
     atomically           Nothing breaks
                          Tells you what's wrong
```

If the new key fails verification, you're still on the old one. Nothing breaks. Fix the issue and try again.

---

### Diagnosing problems

Something feels off? This is your first stop:

```
$ git-user doctor

Checking your git-user setup...

  ✅  git installed (2.43.0)
  ✅  ssh-keygen available
  ✅  Active identity: work (alice@company.com)
  ✅  SSH key exists at ~/.ssh/git_work
  ✅  Key permissions OK (0600)
  ✅  GitHub connection verified — Hi alice-corp!
  ──────────────────────────────────────────────
  Everything looks good.
```

When something's wrong, it's specific:

```
$ git-user doctor

  ✅  git installed (2.43.0)
  ✅  ssh-keygen available
  ✅  Active identity: work (alice@company.com)
  ✅  SSH key exists at ~/.ssh/git_work
  ❌  GitHub connection failed
      Your key isn't added to GitHub yet.
      Run: cat ~/.ssh/git_work.pub
      Then go to: github.com/settings/keys
```

It tells you what's wrong and what to do about it. No decoding cryptic SSH errors.

---

## All commands

| Command | What it does |
|---|---|
| `git-user register` | Create a new identity — guided, SSH included |
| `git-user switch <name>` | Activate an identity |
| `git-user list` | See all your identities |
| `git-user current` | Check what's active right now |
| `git-user rekey <name>` | Generate a fresh SSH key for an identity |
| `git-user bind <name> --ssh-key <path>` | Link an SSH key you already have |
| `git-user remove <name>` | Delete an identity |
| `git-user edit <name> <email>` | Update an identity's email |
| `git-user doctor` | Run a health check on everything |
| `git-user -i` | Open the interactive TUI menu |

---

## Real scenarios

### Freelancer with multiple clients

```bash
git-user register   # name: client-a, email: you@client-a.com
git-user register   # name: client-b, email: you@client-b.com
git-user register   # name: client-c, email: you@client-c.com

# Before every session — one command
git-user switch client-b
```

No more accidentally emailing a client with commits from your personal address. No more "wait, which SSH key does this repo need?"

### Work vs personal

```bash
git-user register   # name: work,     email: you@company.com
git-user register   # name: personal, email: you@gmail.com

git-user switch work      # at the office
git-user switch personal  # on your own time
```

Your company email never leaks onto your public GitHub profile. Your personal commits don't show up in your employer's activity.

### Open source maintainer with multiple orgs

```bash
git-user register   # name: personal,   email: you@gmail.com
git-user register   # name: company,    email: you@company.com
git-user register   # name: foundation, email: maintainer@foundation.org

git-user switch personal    # your side projects
git-user switch company     # work projects
git-user switch foundation  # community projects
```

### Already have SSH keys set up

You don't need to regenerate anything. Skip the key generation during `register`, then bind your existing key:

```bash
git-user register   # name: work, email: you@company.com → generate key: no
git-user bind work --ssh-key ~/.ssh/id_ed25519_work
```

### Prefer clicking over typing

```bash
git-user -i   # or: git-user tui
```

Arrow keys to navigate, Enter to select. Does everything the CLI does, just with a beautiful menu.

---

## What's stored where

```
~/.git-users/
  └── config.json         ← your identities (names, emails, key paths)

~/.gitconfig              ← updated on every switch (name + email)
~/.ssh/config             ← updated on every switch (which key to use)
~/.ssh/git_<name>         ← private key (never leaves your machine)
~/.ssh/git_<name>.pub     ← public key (what you add to GitHub/GitLab)
```

`git-user` stores the *path* to your private keys, not the keys themselves. If you back up `~/.git-users/config.json` and move it to a new machine, you just need to re-run `register` for the SSH key part — all the profile info carries over.

---

## Security

- Private keys stay on your machine with `0600` permissions — only you can read them
- Key permissions are validated before use — if something's wrong, you're told immediately
- `IdentitiesOnly yes` in `~/.ssh/config` means SSH only tries the key you assigned, nothing else
- Config writes are atomic (temp file + rename) — a crash mid-write can't leave you in a broken state

---

## Troubleshooting

**"git-user: command not found"**
Restart your terminal, or run `source ~/.zshrc` (bash: `source ~/.bashrc`). If it still doesn't work, check that `/usr/local/bin` is in your `$PATH`.

**"Permission denied" during install**
The installer copies the binary to `/usr/local/bin` which usually needs sudo. It'll ask for your password — that's expected.

**SSH verification failed after register**
The key probably isn't added to GitHub yet. Run `git-user doctor` — it'll show you the public key again and tell you exactly where to paste it.

**Want to uninstall?**
```bash
sudo rm /usr/local/bin/git-user
rm -rf ~/.git-users
```
Your SSH keys and `~/.gitconfig` are not touched. Only the tool itself and its config file are removed.

---

## Contributing

Issues and pull requests are welcome. If something's broken, open an issue. If something's confusing — even just "I didn't understand what this command does" — that's worth filing too. The goal is for this to be usable by someone who's never touched SSH config before.

---

MIT License.

---

*Made for developers who just want their Git to work.*
