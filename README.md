# git-user

> Switch Git accounts in one command. No config editing. No SSH key chaos.

---

I built this because I got tired of the same stupid problem every single day.

I have a work GitHub account, a personal one, and two client freelance accounts. Every time I jumped between projects I'd either forget to change my git config, push a commit with my work email to a personal repo, or spend ten minutes untangling SSH keys. It was embarrassing. There had to be a better way.

`git-user` is that better way. You register your identities once, and from then on switching between them is literally one command.

---

## How it works (the short version)

```bash
git-user register    # set up a new identity — name, email, SSH key, all in one go
git-user switch work # switch to it
git-user current     # check what's active
```

That's it. Everything else — writing your `~/.gitconfig`, managing `~/.ssh/config`, testing the connection — happens automatically in the background.

---

## Installing

```bash
curl -sSfL https://raw.githubusercontent.com/divyo-argha/git-user/main/install.sh | bash
```

Then restart your terminal (or `source ~/.zshrc` / `source ~/.bashrc`).

Prefer to build from source?

```bash
git clone https://github.com/divyo-argha/git-user
cd git-user
go build -o git-user
sudo cp git-user /usr/local/bin/
```

**Requirements:** Go 1.21+, git, ssh-keygen. You almost certainly already have all of these.

---

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
