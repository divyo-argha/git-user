# Phase 1 Quick Start Guide

## What's New in Phase 1?

### 1. Contextual Prompt (Git Repo Aware)
The shell prompt now intelligently shows your active git-user identity **only** when you're inside a git repository.

```bash
# Outside a git repo - no prompt
$ cd /tmp
$ # (no git-user prompt shown)

# Inside a git repo - shows identity
$ cd ~/my-project
$ # (git-user prompt shows: 👤 myidentity)
```

### 2. Void State (Sign Out)
You can now sign out and enter a "void" state where you cannot commit or push.

```bash
# Sign out
$ git user current --sign-out
✔ Signed out. You are now in the void state.
ℹ You cannot commit or push until you switch to an active identity.

# Check current state
$ git user current
⚠ No active identity set.
ℹ Run 'git-user switch <n>' to activate one.

# Re-activate an identity
$ git user switch myidentity
✔ Switched to "myidentity" (email@example.com)
```

## Common Workflows

### Workflow 1: Daily Development
```bash
# Start your day
$ git user switch work

# Work on projects (prompt shows "work" in git repos)
$ cd ~/project1
$ # (prompt shows: 👤 work)

# Switch to personal project
$ git user switch personal
$ cd ~/personal-project
$ # (prompt shows: 👤 personal)

# End of day - sign out
$ git user current --sign-out
```

### Workflow 2: Shared Machine
```bash
# User A signs in
$ git user switch alice
$ # (can commit and push)

# User A signs out
$ git user current --sign-out

# User B signs in
$ git user switch bob
$ # (can commit and push)
```

### Workflow 3: Security-Conscious Development
```bash
# Work on sensitive project
$ git user switch secure-project

# When done, immediately sign out
$ git user current --sign-out

# Credentials are cleared from git config
$ git config --global user.name
# Output: <void-no-user>
```

## Key Features

| Feature | Behavior |
|---------|----------|
| **Contextual Prompt** | Only shows in git repositories |
| **Void State** | Prevents commits/pushes with invalid git config |
| **Sign-Out** | Clears all credentials and signing keys |
| **Persistence** | Void state is saved to config file |
| **Re-activation** | Can switch back to any identity anytime |

## Commands Reference

```bash
# Show current identity
git user current

# Sign out (enter void state)
git user current --sign-out

# Switch to an identity
git user switch <name>

# List all identities
git user list

# Add a new identity
git user add <name> <email>

# Show prompt (for shell integration)
git user prompt
```

## Shell Integration

The prompt automatically integrates with your shell. It will:
- Show your active identity when inside a git repo
- Hide when outside a git repo
- Display a checkmark (✔) if you have a signing key configured

### Setup (if not already done)
```bash
git user setup-prompt
```

### Remove
```bash
git user remove-prompt
```

## Security Notes

1. **Void State Username:** `<void-no-user>` - This invalid format prevents accidental commits
2. **Credential Clearing:** Signing keys and SSH configs are cleared on sign-out
3. **Contextual Display:** Prompt only shows in git repos, reducing credential exposure
4. **Persistent State:** Void state is saved, so you won't accidentally use old credentials

## Troubleshooting

### Prompt not showing in git repo?
```bash
# Verify you're in a git repo
git rev-parse --git-dir

# Verify an identity is active
git user current

# Reload shell integration
git user reload
```

### Can't commit after sign-out?
This is expected! Sign-out puts you in void state. Re-activate an identity:
```bash
git user switch <name>
```

### Git config shows `<void-no-user>`?
This is correct! You're in void state. Switch to an active identity:
```bash
git user switch <name>
```

## What's Next?

Phase 2 will add:
- `--remember` and `--forget` modes for credential persistence
- Automatic credential cleanup on profile switches
- Enhanced authentication workflows
