# Contributing

Issues and pull requests are welcome. If something's broken, open an issue. If something's confusing — even just "I didn't understand what this command does" — that's worth filing too.

## Development

```bash
git clone https://github.com/divyo-argha/git-user.git
cd git-user
make build   # build binary
make test    # run tests
```

Requirements:

- Go 1.25+ (see `go.mod`)
- Node.js (for the npm packaging scripts)

## Project layout

```
cmd/git-user/          entrypoint (version, orphan-key cleanup, dispatch)
internal/cli/          subcommands (register, switch, list, …)
internal/tui/          interactive terminal UI
internal/config/       identity store (~/.git-users/config.json)
internal/identity/     manager, temp sessions, orphan detection
internal/ssh/          ssh-agent integration, key handling
internal/git/          git command wrappers
internal/bundle/       encrypted export/import format
internal/keyring/      OS keychain integration
npm/                   npm packaging for git-userhub
```

## Running tests

```bash
make test              # test runner with coverage table
go test ./...          # plain Go test suite
go test -race ./...    # with the race detector
```

## Quality gates

Before submitting a pull request, make sure these all pass:

```bash
gofmt -l cmd internal logo test   # must be empty
go vet ./...
go build ./...
go test ./...
staticcheck ./...                 # if you have it installed
```

Keep the working tree free of accidental artifacts (`coverage.out`, `logs.zip`, `dist/`) — they are gitignored.

## Releases

Releases are cut from a `v*` tag pushed to `main`. GoReleaser builds and uploads
platform binaries; the npm packages are built and published by the
`npm-publish` workflow. No version bump in code is required — the version comes
from the git tag (see `.goreleaser.yaml` and `internal/version/version.go`).

## Security

See [SECURITY.md](SECURITY.md) for the security policy and how to report a
vulnerability.
