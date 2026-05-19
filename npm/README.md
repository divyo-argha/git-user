# git-user (npm package)

> Switch Git accounts in one command. No config editing. No SSH key chaos.

This is the npm distribution of [git-user](https://github.com/divyo-argha/git-user), a CLI tool for managing multiple Git identities.

## Installation

```bash
npm install -g @divyo-argha/git-user
```

Or use without installing:

```bash
npx @divyo-argha/git-user register
```

## Quick Start

```bash
# Create your first identity
git-user register

# Switch between identities
git-user switch work
git-user switch personal

# List all identities
git-user list

# Check current identity
git-user current
```

## Documentation

Full documentation available at: https://github.com/divyo-argha/git-user

## License

MIT
