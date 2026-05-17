# git-user

> Tired of juggling multiple Git accounts? Switch between them in one command. That's it.

---

## The Problem

Managing multiple Git identities is a pain:
- **Freelancers**: Each client has their own GitHub org and wants commits under their email
- **Open source maintainers**: Personal projects, company projects, and community projects all need different identities
- **Contractors**: Working across multiple companies with different SSH key requirements
- **Team leads**: Need to switch between personal, company, and team lead accounts

Without a tool, you're either:
- Manually changing git config every time
- Forgetting which account is active and committing with the wrong email
- Managing SSH keys manually
- Copying SSH config blocks around

**git-user** fixes this. One command, everything switches. No mistakes.

---

## Install (Copy & Paste)

```bash
curl -sSfL https://raw.githubusercontent.com/divyo-argha/git-user/main/install.sh | bash
```

Then reload your shell:
```bash
source ~/.zshrc  # or ~/.bashrc if you use bash
```

Done. That's the whole install.

### What just happened?
- Checked if Go is installed (and installed it if needed)
- Downloaded and built the tool
- Installed it to `/usr/local/bin`
- Added it to your PATH
- Updated your shell config

If something goes wrong, see [Troubleshooting](#troubleshooting) below.

---

## Quick Start (2 minutes)

```bash
# Create your first identity
git-user register

# You'll be asked:
# - Name (e.g., "work")
# - Email (e.g., "you@company.com")
# - Generate SSH key? (yes/no)

# If you said yes, it'll show you the public key
# Copy it and add it to GitHub/GitLab/Bitbucket
# Then press Enter

# Done! Now switch to it:
git-user switch work

# Check what's active:
git-user current

# See all your identities:
git-user list
```

---

## Common Tasks

### Add another identity
```bash
git-user register
```

### Switch between identities
```bash
git-user switch work
git-user switch personal
```

### Rotate an SSH key (it expired or you want a new one)
```bash
git-user rekey work
```

### Check if everything's working
```bash
git-user doctor
```
This checks:
- Is an identity active?
- Does your SSH key exist?
- Can you connect to GitHub?
- Is git installed?

### Use the interactive menu
```bash
git-user -i
# or
git-user tui
```
Arrow keys to navigate, Enter to select.

---

## What You Need to Know

**SSH Keys**: Think of it as a digital key that proves you're you to GitHub. Each account needs its own key.

**Identity**: A profile with your name, email, and SSH key. You can have as many as you want.

**Global Config**: Git's default name and email. `git-user` updates this when you switch identities.

---

## All Commands

| Command | What it does |
|---------|-------------|
| `git-user register` | Create a new identity (guided setup) |
| `git-user list` | Show all your identities |
| `git-user switch <name>` | Activate an identity |
| `git-user current` | See which identity is active |
| `git-user rekey <name>` | Generate a new SSH key for an identity |
| `git-user bind <name> --ssh-key <path>` | Link an existing SSH key |
| `git-user remove <name>` | Delete an identity |
| `git-user edit <name> <email>` | Change an identity's email |
| `git-user doctor` | Check your setup for issues |
| `git-user -i` | Open interactive menu |

---

## Real Use Cases

### Freelancer with Multiple Clients
Each client has their own GitHub org, their own SSH key, and wants commits under their email. You work on all three in a single day.

```bash
git-user register
# Name: acme-corp, Email: dev@acme.com, Generate key: yes

git-user register
# Name: startup-xyz, Email: you@startup-xyz.io, Generate key: yes

git-user register
# Name: agency-work, Email: contractor@agency.com, Generate key: yes

# Before working on ACME project:
git-user switch acme-corp

# Before switching to Startup project:
git-user switch startup-xyz

# All commits go to the right account. No mistakes.
```

### Open Source Maintainer with Multiple Orgs
You maintain projects under your personal GitHub, your company's org, and a community foundation. Each needs a different identity.

```bash
git-user register
# Name: personal, Email: you@gmail.com

git-user register
# Name: company, Email: you@company.com

git-user register
# Name: foundation, Email: maintainer@foundation.org

# Switch based on which project you're working on
git-user switch personal    # Your side projects
git-user switch company     # Work projects
git-user switch foundation  # Community projects
```

### Team Lead Managing Multiple Accounts
You have a personal account, a company account, and sometimes need to commit as the team lead account for releases.

```bash
git-user register
# Name: personal, Email: you@gmail.com

git-user register
# Name: company, Email: you@company.com

git-user register
# Name: team-lead, Email: team-lead@company.com

# Each identity has its own SSH key
# Switch instantly without manual config changes
git-user switch team-lead
```

### Contractor Working Across Multiple Companies
You work with 3 different companies. Each has their own GitHub, their own SSH key requirements, and their own email domain.

```bash
git-user register
# Name: client-1, Email: contractor@client1.com

git-user register
# Name: client-2, Email: contractor@client2.com

git-user register
# Name: client-3, Email: contractor@client3.com

# One command to switch everything
git-user switch client-1
```

---

## Troubleshooting

### "git-user: command not found"
Reload your shell:
```bash
source ~/.zshrc  # or ~/.bashrc
```

### "Permission denied" during install
The installer needs sudo. It'll ask for your password.

### SSH verification failed
This means your SSH key isn't added to GitHub yet. The tool will tell you exactly what to do.

### Want to uninstall?
```bash
sudo rm /usr/local/bin/git-user
```

### Manual install (if the one-liner doesn't work)
```bash
git clone https://github.com/divyo-argha/git-user
cd git-user
go build -o git-user
sudo cp git-user /usr/local/bin/
git-user --help
```

---

## How It Works

When you run `git-user switch work`:
1. Updates your global git config (name + email)
2. Sets up your SSH key in `~/.ssh/config`
3. Tests the SSH connection to make sure it works
4. Saves which identity is active

Everything is stored in `~/.git-users/config.json`. You can back it up or move it between machines.

---

## Security

- Your private SSH keys never leave your machine
- Keys are stored with `0600` permissions (only you can read them)
- The tool checks key permissions automatically
- No plaintext secrets anywhere

---

## Features Coming Soon

These are in development:
- Platform mapping (link your GitHub/GitLab/Bitbucket usernames)
- SSH host aliases (clone with `git@github-work:org/repo.git`)
- Key expiration tracking
- Export/import identities

---

## Need Help?

```bash
git-user --help
git-user doctor
```

Or check the [GitHub issues](https://github.com/divyo-argha/git-user/issues).

---

## License

MIT. Do whatever you want with it.

---

<p align="center">
  Made for developers who just want their Git to work.
</p>
