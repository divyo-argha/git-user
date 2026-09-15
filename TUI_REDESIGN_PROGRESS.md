# TUI Redesign — Progress & Resume Notes

Working notes for an in-progress multi-session effort. If you're picking this up in a new session, read this whole file first, then re-read the current state of the files listed under "Phase 1 — DONE" before touching anything, since this file is a snapshot and the code is the source of truth.

## Where this came from

Two research papers were read and partly implemented earlier in this project:
1. arXiv:2604.14014 ("Analysis of Commit Signing on GitHub") → led to: signing defaults (`register --no-sign`), `doctor` signing/token-expiry warnings, `verify` command, `.git-user-policy` (`require_signing`, `allowed_email_domains`), `post-merge` hook, `--json` on `doctor`/`verify`.
2. NDSS'25 ("Attributing Open-Source Contributions is Critical but Difficult") → led to: committed `.allowed-signers` file + `policy signers add/list/remove`, auto-wired into local git config on `clone`/`hook install`; a small `stats` improvement (unregistered-author counts). A DNS-based "hijackable domain" audit was designed then explicitly rejected (redundant, unreliable heuristic, scope mismatch for a personal identity tool).

Then: the user asked whether the TUI (`internal/tui/`) had all of this too. It didn't — `opDoctor`/`opHook`/`opRekey` in the TUI are **independent reimplementations** of the CLI logic, not shared code, which is exactly how a `SignKey`-goes-stale-after-key-rotation bug existed in one copy and not the other (found and hand-fixed in both places before the redesign started). That gap became the full TUI redesign effort documented here.

## Locked decisions (do not re-litigate these)

1. **Extract shared business logic** so CLI and TUI call the same underlying functions — not two reimplementations. This is the permanent fix for the drift/bug-duplication class.
2. **Full redesign**, not just bolted-on missing screens — rethink navigation/layout, surface status (signing, token expiry, policy compliance, security score) at a glance, organize actions into well-grouped menus.
3. **No emoji icons** — clean, minimal Unicode status glyphs: `✓` pass, `✕` fail, `△` warn, `○` skipped, `·` info — colored via the existing Tokyo-Night-style theme (`internal/tui/theme/theme.go`), never carrying meaning through the glyph's own color. Discoverability stays **menu-driven** (exhaustive, well-organized contextual menus), explicitly **not** a command palette.

Full original plan (8 phases, with the detailed struct/package designs) is preserved at the bottom of this file under "Full original plan text" — that's the canonical spec for phases 2–8; this file's job above that is just to say what's done vs. not.

## Phase 1 — DONE (shared-logic extraction)

All of this is committed to the working tree (not yet git-committed — see "Before you continue" below) and verified: `go build ./...`, `go vet ./...`, `go test ./...` all green, plus manual scratch-repo end-to-end tests of `doctor` (text + `--json`), `hook install`/`check` (policy blocking), `policy signers add/list`, `sign`, and `rekey` (confirmed `SignKey` still carries forward after rotation).

**New packages** (all under `internal/`, zero TUI changes in this phase — CLI-only):
- **`internal/diagnostics`** — `doctor.go`'s ~20 checks as data: `Check{ID, Category, Name, Status, Message, Detail []string, FixHint, Fixed, Scored, Subject, IsProgress}`, `Report{Checks, Issues, Fixed, ScoreTotal, ScorePassed}`, `Run(store *config.Store, Options{Fix bool, VerifySSH func(string) error}) (Report, error)`. `TokenExpiryMessage`/`SigningDisabledMessage`/`TokenExpiryWarnDays` also live here now (moved from `doctor.go`'s private helpers). `cli/doctor.go` is now ~110 lines: calls `Run`, then renders `Report.Checks` in order (progress lines via `ui.Info`, `StatusPass→ui.Success`, `StatusWarn`/`StatusNotice→ui.Warn`, `StatusInfo→ui.Info`, `Detail` lines and `FixHint` after each). SSH connectivity checking (`verifySSHConnectionWithKey`, still in `cli/utils.go`) is **injected**, not moved — it's genuinely CLI-interactive (unlocks passphrase-protected keys via `ensureKeyUnlocked`), so `diagnostics.Run` takes it as a callback rather than depending on `internal/cli`.
- **`internal/hookops`** — `Specs`/`Content` (hook script template, moved verbatim), `Install(confirmOverwrite func(spec, existing string) bool) ([]InstallResult, error)`, `Uninstall()`, `CheckIdentity(store) error` (returns typed `*IdentityMismatchError` or `*PolicyViolation` for the caller to format), `WarnOnUnsignedMerge(store) []stats.AuthorStat`, `EnforceRepoPolicy(user) error`. `cli/hook.go` is a thin wrapper; the interactive "hook already exists, overwrite?" prompt is passed in as a closure (`cliConfirmOverwriteHook`) so the TUI can later supply its own (v1 TUI policy per the plan: never silently overwrite, skip + report).
- **`internal/signing`** — `CurrentStatus(user) Status`, `Disable(store, name)`, `Enable(store, name, key, format) (resolvedKey, resolvedFormat string, autoDetected bool, err error)` — same key/format auto-detection `sign.go` always had. Deliberately does **not** touch live git config (`git.ConfigureSigning`/`RemoveSigningConfig`) — that stays caller-side since success/warning messaging differs by caller.
- **`internal/rekeyops`** — `ResolveOldKeyPath(store, name)`, `Rotate(store, name, newKeyPath string, generateKey func(newKeyPath string) error) (Result, error)`. Key generation itself is **injected** (CLI runs interactive `ssh-keygen` with inherited stdio; TUI's existing `opRekey` uses `ssh.GenerateKey` with an explicit passphrase — genuinely different mechanisms, so this is the correct seam) — but backup/restore-on-failure/rebind/**carry `SignKey` forward if it pointed at the old key**/mkdir/collision-check are now the single shared implementation.
- **`internal/policyops`** — `BuildPolicyFile`/`WritePolicy` (`.git-user-policy` content), `ResolveSignerFromIdentity`/`ResolveSignerFromEmail`/`UpsertSigner`/`RemoveSigner` (`.allowed-signers` management), `WireAllowedSignersConfig(repoRoot) (changed bool, err error)` (auto-wires local `gpg.ssh.allowedSignersFile` config — used by `hook install`, `policy signers add`, and will be used by the TUI's hook-install path in Phase 7).

**Also touched** (small, in-place, not new packages): `internal/git/git.go` gained `ApplyIdentitySSHConfig(sshCommand, sshKey string, local bool) error` and `ConvertRemotesToSSH() ([]RemoteConversionResult, error)` — both were duplicated logic between `cli/switch.go`/`cli/fixremote.go` and the new `diagnostics` package; now single-sourced in `internal/git`. `cli/switch.go`'s `applyUserSSHConfig` and `cli/fixremote.go`'s `runFixRemote` are now thin wrappers over these.

**Known, accepted, minor deviation** (documented so it's not mistaken for a bug later): `doctor --fix`'s HTTPS-remotes-conversion path used to print each remote's conversion line as a side effect of calling `fix-remote`'s own printer; it now prints a single summary line instead (`git.ConvertRemotesToSSH` is silent, `diagnostics` builds its own summary). Functionally equivalent, not byte-identical in that one sub-case only. Everything else was verified line-by-line against the original `doctor.go` output including exact indentation of detail/fix-hint lines.

**Test file touched**: `internal/cli/doctor_test.go` — `tokenExpiryWarning`/`tokenExpiryWarnDays` references updated to `diagnostics.TokenExpiryMessage`/`diagnostics.TokenExpiryWarnDays`. No other test files needed changes; `internal/cli/hook_test.go`'s `TestHookInstallUninstall` passed unmodified against the new `hookops`-backed implementation.

## Before you continue

Nothing has been `git commit`ed yet — `git status --short` at the repo root shows everything from Phase 1 still as uncommitted working-tree changes (modified: `doctor.go`, `doctor_test.go`, `fixremote.go`, `hook.go`, `policy.go`, `policy_signers.go`, `rekey.go`, `sign.go`, `switch.go`, `internal/git/git.go`; new dirs: `internal/diagnostics`, `internal/hookops`, `internal/policyops`, `internal/rekeyops`, `internal/signing`). Decide whether to commit Phase 1 before starting Phase 2 — it's a clean, independently-revertable unit (no TUI changes at all), so committing it alone first is reasonable, but that wasn't explicitly asked for yet.

## What's next — Phase 2 onward (NOT STARTED)

In order (each independently shippable — do NOT attempt as one giant change):

- **Phase 2 — Icon system**: new `internal/tui/theme/icons.go` (`IconPass="✓"`, `IconWarn="△"`, `IconFail="✕"`, `IconInfo="·"`, `IconSkipped="○"`, paired with existing `SuccessStyle`/`WarningStyle`/`ErrorStyle`/`Dim`/`InfoStyle`). Sweep real emoji out of `internal/tui/screens/detail.go` (🔑📋🚀) and `internal/tui/app_actions.go` (🪟) — confirmed by grep these are the only actual pictographic emoji in the TUI today (everything else like `✦↓⇪~⌘▲●○▶` is already fine, text-presentation, keep as-is). User confirmed "minimal Unicode symbols" style — no elaborate new per-action-category icon taxonomy, reuse the simple glyph vocabulary already in `action_menu.go`.
- **Phase 3 — Health & Security screen**: new `internal/tui/screens/health.go` built on `internal/diagnostics.Run`, replacing `ops_report.go`'s `opDoctor` (plain-text dump) entirely. Security score as a prominent colored header (≥90% green / 60–89% amber / <60% red — proposed thresholds, not separately confirmed with user). Checks grouped by `Category`, per-profile (`Subject`-scoped) checks nested under profile name. Footer: `f` = fix all now (`diagnostics.Run(store, Options{Fix:true, ...})`), `r` = refresh.
- **Phase 4 — Policy & Signers screens**: new `internal/tui/screens/policy.go`, `internal/tui/screens/signers.go`, built on `internal/policyops`/`internal/hookops`. `components.ActionMenu`'s `SystemActions` gains a repo-gated entry (signature becomes `SystemActions(th, showFixRemote, inRepo bool)`).
- **Phase 5 — Verify screen**: new `internal/tui/screens/verify.go`, modeled directly on the existing `screens/stats.go` (same pattern already works — `internal/stats` is the one place shared logic already worked correctly pre-redesign).
- **Phase 6 — Dashboard/Detail status redesign**: `components/identity_list.go` gains `TOKEN`/`POLICY` pills; `screens/detail.go`'s `renderOverview` gains HTTPS-token, promoted Signing (via `internal/signing.CurrentStatus`), and Repository-Policy sections.
- **Phase 7 — Hook parity fix**: rewrite `internal/tui/ops_hook.go`'s `opHook` to call `hookops.Install`/`Uninstall`/`CheckIdentity` instead of its old pre-commit-only bespoke logic — this is the change that actually closes the original TUI/CLI hook-feature gap (pre-push, post-merge, policy enforcement, allowed-signers wiring, all currently missing from the TUI's hook action).
- **Phase 8 — Remaining gaps, evaluated not blanket-added**: `exec`/`env`/`current`/`edit`/`init`/`completion` all reviewed and judged not to need direct TUI actions (`edit` confirmed functionally redundant with the TUI's existing per-identity "email" action by reading `cli/edit.go`). No code expected here, just a final check that this judgment still holds.

## Verification pattern to keep using

- After Phase 1 (done): golden-output style — ran existing/updated tests plus manual scratch-repo runs of every touched CLI command, diffed against known-good prior output line by line (indentation included).
- For each TUI phase: `go build ./...`, `go vet ./...`, `go test ./internal/tui/...`, plus new screens get behavioral tests following the existing pattern — construct screen directly with an in-memory `*config.Store`, drive via `.Update(tea.KeyMsg{...})`, assert on struct state + emitted `core.*Msg` (see `dashboard_test.go`/`detail_test.go`/`stats_test.go`). No snapshot/pixel testing.
- Manual pass per phase: launch the actual TUI (`git-user` with no args in a TTY) in a scratch config to confirm real rendering/navigation.

---

## Full original plan text (Phases 2–8 detail, as approved)

The complete original plan — written before Phase 1 implementation started — is kept at:
`/home/sbh/.claude-profiles/fugitive/plans/https-arxiv-org-pdf-2604-14014-read-this-flickering-music.md`

That file has the full struct/package designs (written before Phase 1 landed — Phase 1's actual final shapes, documented above, are very close but occasionally more precise than that draft, e.g. `rekeyops.Rotate` takes an injected `generateKey` callback rather than a `passphrase string`, since the CLI and TUI generate keys via genuinely different mechanisms). Read this progress file first; fall back to that plan file only for phases 2–8's design detail not yet summarized here.
