# git-user Roadmap — v4.3.5 → v4.5.0

Current version: **v4.3.5**. This plan reflects what was completed between v4.1.0 and v4.3.5, audits what is still pending from the old plan, and lays out the forward roadmap in a revised release schedule.

---

## What Was Accomplished (v4.1.0 → v4.3.3)

```mermaid
graph LR
    A["v4.1.0\nBaseline"] --> B["v4.1.x\nBug fixes & stability"]
    B --> C["v4.2.0\nMilestone"]
    C --> D["v4.3.3\nCurrent"]
```

### ✅ Completed Items (from old plan)

| Item | File |
|------|------|
| Makefile build target fixed (`./cmd/git-user`) | `Makefile` L10 |
| `install.sh` uses `grep -i` for case-insensitive tarball matching | `install.sh` L30 |
| `edit.go` excludes current user from `IsEmailTaken` on email update | `edit.go` L38 |
| `doctor.go` command string typos fixed (`git-user` used consistently) | `doctor.go` |
| Stale `npm/git-userhub-3.1.2.tgz` artifact removed from repo | `npm/` |
| `npm/package.json` aligned to v4.3.3 with all 6 platform optionalDeps | `npm/package.json` |
| `bindpath.go` tilde/path expansion via `expandPath()` in bind + unbind | `bindpath.go` L34, L92 |
| `update.go` resolves symlinks, handles npm installs, semantic semver compare | `update.go` |
| TUI launched for unknown args via `handleUnknownArg` with fuzzy suggest | `tui.go` L247–273 |
| SSH agent status + loaded key count shown in TUI header | `header.go` L84–105 |
| Sync command with encrypted bundle push/pull fully implemented | `sync.go` |
| `remove.go` cleanly filters slice and calls `keyring.DeleteKeychainPassphrase` | `remove.go` |
| `config.go` `RemoveUser` uses safe filtered-slice pattern | `config.go` L214 |
| `clone.go` identity selection, local config apply, and `--bind` flag | `clone.go` |
| `passphrase.go` full set/remove/verify/mode flow with keyring integration | `passphrase.go` |
| Tests added for `export/import`, `clone`, `stats`, `bind`, `remove`, `edit`, `switch` | `internal/cli/*_test.go` |
| TUI screens directory and component architecture fully structured | `internal/tui/screens/`, `components/` |

---

## Current Coverage Baseline (v4.3.3)

| Package | Coverage |
|---------|----------|
| `internal/bundle` | **77.1%** |
| `internal/cli` | **45.9%** |
| `internal/config` | **48.9%** |
| `internal/git` | **45.2%** |
| `internal/tui/screens` | **46.0%** |
| `internal/tui/components` | **29.4%** |
| `internal/tui` (app) | **9.1%** |
| `internal/identity` | **31.3%** |
| `internal/ui` | **32.0%** |
| `internal/keyring` | **0.0%** ⚠️ |
| `internal/ssh` | **0.0%** ⚠️ |
| `internal/tui/core` | **0.0%** ⚠️ |

---

## Still Outstanding — Verified Open Bugs

These items were in the old plan but are **not yet fixed** after a full code audit of v4.3.3:

### 🔴 Critical / High Priority

#### 1. `root.go` — `--version` / `-v` flag silently fails
> **File:** [`root.go`](file:///Users/bobdylan/Divyo/git-user/internal/cli/root.go)
>
> The `usage` string (L68) documents `git-user --version` but no handler exists in `Execute()`. The flag silently falls through to `handleUnknownArg`.
>
> **Fix:** Add a `--version` / `-v` handler before the `sub := args[0]` switch that prints `version.Version` and exits 0.

#### 2. `clone.go` — extra `git clone` flags are silently dropped
> **File:** [`clone.go`](file:///Users/bobdylan/Divyo/git-user/internal/cli/clone.go#L103-L106)
>
> Line 104 appends only `passArgs[1]` (the destination directory) to `cloneArgs`. Flags like `--depth 1`, `--branch main`, `--single-branch` are silently dropped.
>
> **Fix:** Change `append(cloneArgs, passArgs[1])` → `append(cloneArgs, passArgs[1:]...)`.

#### 3. `passphrase.go` — phantom `git-user session start` reference
> **File:** [`passphrase.go`](file:///Users/bobdylan/Divyo/git-user/internal/cli/passphrase.go#L217)
>
> Line 217 instructs: `"Use 'git-user session start' to unlock this key"` — but no `session` subcommand exists. This is actively misleading.
>
> **Fix:** Replace with the correct hint (e.g. `"Run 'git-user passphrase --mode login' to adjust how your key is unlocked."` or mention `ssh-add`).

#### 4. `pubkey.go` — `access denied` error when passing a non-active identity name
> **File:** [`pubkey.go`](file:///Users/bobdylan/Divyo/git-user/internal/cli/pubkey.go#L30-L35)
>
> `git-user pubkey <name>` where `<name>` is not the active identity returns `"access denied: you can only view the public key of the active identity"`. The security intent is fine but the message is cryptic and unhelpful.
>
> **Fix:** Keep the restriction but rewrite the error message to be friendly: `"To view <name>'s public key, switch first: git-user switch <name>"`.

#### 5. `tui.go` — double confirmation prompt for TUI-initiated `unbind` and `remove`
> **File:** [`tui.go`](file:///Users/bobdylan/Divyo/git-user/internal/cli/tui.go#L131-L139), [`tui.go`](file:///Users/bobdylan/Divyo/git-user/internal/cli/tui.go#L224-L229)
>
> Both `unbind` (L136) and `remove` (L225) in `executeAction` show a `ui.Confirm` prompt even though the TUI modal that triggered the action already required confirmation. Users are prompted twice.
>
> **Fix:** Accept a `fromTUI bool` parameter in `executeAction` (or use a separate codepath) to skip the redundant prompt when the call originates from the TUI.

### 🟡 Medium Priority

#### 6. `git.go` — `ConvertHTTPSToSSH` doesn't strip embedded credentials
> **File:** [`git.go`](file:///Users/bobdylan/Divyo/git-user/internal/git/git.go#L270-L287)
>
> `https://user:token@github.com/foo/bar` is not handled — `parts[0]` would be `user:token@github.com`, producing a malformed SSH URL.
>
> **Fix:** Before splitting, strip everything up to the last `@` from the host segment using `strings.SplitN` or a regex.

#### 7. `SECURITY.md` — supported versions table still references `3.x`
> **File:** [`SECURITY.md`](file:///Users/bobdylan/Divyo/git-user/SECURITY.md#L7-L10)
>
> The table lists `3.x` as "Actively maintained" and `<3.0` as "End of life". The project is at v4.3.3.
>
> **Fix:** Update to `4.x` actively maintained, `< 4.0` end of life.

#### 8. `config.go` — `UnbindPathFromUser` still uses slice header alias
> **File:** [`config.go`](file:///Users/bobdylan/Divyo/git-user/internal/config/config.go#L328)
>
> `filtered := u.BindPaths[:0]` shares the underlying array with `u.BindPaths`, which is a footgun that could cause data loss under some GC patterns. `RemoveUser` was fixed to use a new allocated slice but this function was not.
>
> **Fix:** Use `var filtered []string` instead of `u.BindPaths[:0]`.

#### 9. `sync.go` — no auto-recovery if `syncDir` is missing after setup
> **File:** [`sync.go`](file:///Users/bobdylan/Divyo/git-user/internal/cli/sync.go#L117)
>
> After initial sync setup, if `~/.git-users/sync` is deleted or missing, the pull at L117 fails silently. There is no user-facing diagnostic or automatic re-clone.
>
> **Fix:** Check `os.Stat(syncDir)` before the pull step; if missing and `store.Sync.RepoURL` is set, re-clone the sync repo before proceeding.

#### 10. `update.go` — temp file not in target directory (Windows cross-device rename)
> **File:** [`update.go`](file:///Users/bobdylan/Divyo/git-user/internal/cli/update.go#L93)
>
> `os.CreateTemp("", ...)` uses the OS temp directory, which may be on a different drive/partition from the binary on Windows. `os.Rename` across drives returns an error.
>
> **Fix:** Use `os.CreateTemp(filepath.Dir(execPath), "git-user-update-*")` for both the download archive and the extracted binary.

---

## Revised Release Schedule (v4.3.4 → v4.5.0)

```mermaid
graph TD
    R1["v4.3.4\nCritical Bug Fixes"] --> R2["v4.4.0\nPolish & Robustness"]
    R2 --> R3["v4.4.1\nDocs & Security Sync"]
    R3 --> R4["v4.4.2\nTest Coverage Push"]
    R4 --> R5["v4.5.0\nMilestone Release"]
```

---

## 📦 Detailed Release Breakdown

### ✅ Release 1 (v4.3.4): Critical Bug Fixes — DONE
> **Focus:** Fix the highest-visibility correctness bugs affecting users daily.

- [x] **[root.go](file:///Users/bobdylan/Divyo/git-user/internal/cli/root.go)**: Add `--version` / `-v` handler that prints `version.Version` and exits.
- [x] **[clone.go](file:///Users/bobdylan/Divyo/git-user/internal/cli/clone.go#L103-L106)**: Change `append(cloneArgs, passArgs[1])` → `append(cloneArgs, passArgs[1:]...)` to forward all extra git flags.
- [x] **[passphrase.go](file:///Users/bobdylan/Divyo/git-user/internal/cli/passphrase.go#L217)**: Remove the phantom `git-user session start` hint; replace with accurate `ssh-add` and `--mode` guidance.
- [x] **[pubkey.go](file:///Users/bobdylan/Divyo/git-user/internal/cli/pubkey.go#L30-L35)**: Rewrite access-denied error to a friendly "switch first" message.

---

### ✅ Release 2 (v4.3.5): Polish & Robustness — DONE
> **Focus:** UX friction elimination, Windows stability, slice safety, sync resilience.

- [x] **[tui.go](file:///Users/bobdylan/Divyo/git-user/internal/cli/tui.go#L131-L139)** & **[tui.go](file:///Users/bobdylan/Divyo/git-user/internal/cli/tui.go#L224-L229)**: Skip redundant `ui.Confirm` for `unbind` and `remove` when called from TUI context.
- [x] **[git.go](file:///Users/bobdylan/Divyo/git-user/internal/git/git.go#L270-L290)**: Fix `ConvertHTTPSToSSH` to strip embedded `user:pass@` from URL host before conversion.
- [x] **[sync.go](file:///Users/bobdylan/Divyo/git-user/internal/cli/sync.go#L114)**: Auto-recover missing `syncDir` by re-cloning from `store.Sync.RepoURL`.
- [x] **[update.go](file:///Users/bobdylan/Divyo/git-user/internal/cli/update.go#L92)**: Create temp files in `filepath.Dir(execPath)` to prevent Windows cross-device rename errors.
- [x] **[config.go](file:///Users/bobdylan/Divyo/git-user/internal/config/config.go#L328)**: Fix `UnbindPathFromUser` to allocate a new slice (`var filtered []string`) instead of aliasing.

---

### ✅ Release 3 (v4.4.1): Docs & Security Policy Sync — DONE
> **Focus:** Keep documentation truthful and up to date with the v4.x reality.

- [x] **[SECURITY.md](file:///Users/bobdylan/Divyo/git-user/SECURITY.md#L7-L10)**: Update supported versions table — `3.x` → `4.x` actively maintained, `< 4.0` end of life.
- [x] **[README.md](file:///Users/bobdylan/Divyo/git-user/README.md)**: Audit all command examples and feature descriptions against actual v4.3.x behaviour.
- [x] **[TERMINAL-INTEGRATION.md](file:///Users/bobdylan/Divyo/git-user/TERMINAL-INTEGRATION.md)**: Verify zsh/bash/fish prompt integration snippets still work with current `git-user prompt` output.
- [x] **`internal/keyring`**: Improve fallback error messages for headless Linux / SSH sessions where GUI keychain is unavailable.

---

### Release 4 (v4.4.2): Test Coverage Push
> **Focus:** Raise coverage in the lowest-covered packages. Target: 60%+ across all packages.

| Package | Current | Target |
|---------|---------|--------|
| `internal/keyring` | 0.0% | 50%+ |
| `internal/ssh` | 0.0% | 40%+ |
| `internal/tui` (app) | 9.1% | 30%+ |
| `internal/tui/core` | 0.0% | 30%+ |
| `internal/identity` | 31.3% | 55%+ |
| `internal/ui` | 32.0% | 55%+ |
| `internal/tui/components` | 29.4% | 50%+ |
| `internal/cli` | 45.9% | 60%+ |

- [ ] **`internal/keyring`**: Add tests for `SetKeychainPassphrase`, `GetKeychainPassphrase`, `DeleteKeychainPassphrase` using mock backends.
- [ ] **`internal/ssh`**: Add tests for `VerifyPassphrase`, `IsSSHKeyLoaded`, `AddSSHKeyWithPassphrase` error paths.
- [ ] **`internal/tui`**: Expand `app_test.go` to cover screen push/pop state transitions.
- [ ] **`internal/cli`**: Add edge-case tests for `passphrase` (wrong passphrase, unprotected key, mode switches), `sync` (mock git ops), `clone` extra-args forwarding.
- [ ] **`internal/git`**: Add tests for `ConvertHTTPSToSSH` with credential-embedded and malformed URLs.

---

### Release 5 (v4.5.0): Milestone Release & Final Audit
> **Focus:** Cross-platform verification, binary audit, and major release tag.

- [ ] **Cross-Platform Verification**: Run `make release-test` (goreleaser snapshot) for all 6 target platforms: macOS ARM64/x64, Linux x64/ARM64, Windows x64/ARM64.
- [ ] **Binary Audit**: Verify `-s -w` ldflags are applied, binary size is reasonable, no debug symbols.
- [ ] **npm Release**: Trigger `npm-publish.yml` with Sigstore provenance for all 6 platform packages at `v4.5.0`.
- [ ] **Release Notes**: Write clean changelog covering all fixes from v4.3.3 → v4.5.0.

---

## Verification Plan

### Automated Testing
```bash
go test ./...                            # All unit tests pass
go test ./... -cover                     # Per-package coverage report
make release-test                        # Goreleaser snapshot cross-platform build
node npm/scripts/build-packages.js       # npm package build verification
```

### Manual Verification Checklist
| Scenario | Expected |
|----------|----------|
| `git-user --version` | Prints `v4.3.x` and exits 0 |
| `git-user clone <url> --depth 1 --branch main` | Both flags forwarded to `git clone` |
| `git-user passphrase` (after setting) | No mention of `session start` |
| `git-user pubkey <non-active-name>` | Friendly "switch first" message |
| TUI → remove identity → confirm | Confirmation appears **once only** |
| `git-user fix-remote` with `https://user:token@github.com/foo/bar` | Converts correctly to SSH URL |
| Delete `~/.git-users/sync`, then `git-user sync` | Auto-re-clones from configured remote |
| `git-user update` on Windows (different drive install) | No cross-device rename error |
