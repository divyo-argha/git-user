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
- **`internal/policyops`** — `BuildPolicyFile`/`WritePolicy` (`.git-user-policy` content), `ResolveSignerFromIdentity`/`ResolveSignerFromEmail`/`UpsertSigner`/`RemoveSigner` (`.allowed-signers` management), `WireAllowedSignersConfig(repoRoot) (changed bool, err error)` (auto-wires local `gpg.ssh.allowedSignersFile` config — used by `hook install`, `policy signers add`, and used by the TUI's hook-install path since Phase 7).

**Also touched** (small, in-place, not new packages): `internal/git/git.go` gained `ApplyIdentitySSHConfig(sshCommand, sshKey string, local bool) error` and `ConvertRemotesToSSH() ([]RemoteConversionResult, error)` — both were duplicated logic between `cli/switch.go`/`cli/fixremote.go` and the new `diagnostics` package; now single-sourced in `internal/git`. `cli/switch.go`'s `applyUserSSHConfig` and `cli/fixremote.go`'s `runFixRemote` are now thin wrappers over these.

**Known, accepted, minor deviation** (documented so it's not mistaken for a bug later): `doctor --fix`'s HTTPS-remotes-conversion path used to print each remote's conversion line as a side effect of calling `fix-remote`'s own printer; it now prints a single summary line instead (`git.ConvertRemotesToSSH` is silent, `diagnostics` builds its own summary). Functionally equivalent, not byte-identical in that one sub-case only. Everything else was verified line-by-line against the original `doctor.go` output including exact indentation of detail/fix-hint lines.

**Test file touched**: `internal/cli/doctor_test.go` — `tokenExpiryWarning`/`tokenExpiryWarnDays` references updated to `diagnostics.TokenExpiryMessage`/`diagnostics.TokenExpiryWarnDays`. No other test files needed changes; `internal/cli/hook_test.go`'s `TestHookInstallUninstall` passed unmodified against the new `hookops`-backed implementation.

## Before you continue

This work lives on a dedicated branch, `tui-redesign` (currently 1 commit ahead of `origin/main`). **Phase 1 is committed** — commit `8b46b2c` ("feat: add SSH configuration management and Git hooks") contains all of Phase 1's changes (`internal/cli/{doctor,hook,policy,policy_signers,rekey,sign,switch,fixremote}.go` + the 5 new `internal/{diagnostics,hookops,policyops,rekeyops,signing}` packages + this progress file's Phase-1-and-earlier content). **Phases 2–7 are NOT committed yet** — `git status` shows them all as working-tree changes (modified: `app_actions.go`, `app_confirm.go`, `app_forms.go`, `app_options.go`, `components/action_menu.go` (+its test), `components/identity_list.go`, `ops_extra_test.go`, `ops_hook.go`, `ops_report.go`, `screens/dashboard.go`, `screens/detail.go`, this progress file; new: `components/token_badge_test.go`, `ops_policy.go` (+test), `screens/{health,policy,signers,verify}.go` (+tests each), `theme/icons.go`). Decide how to slice these into commits (one per phase, mirroring Phase 1's precedent, is a reasonable default) before doing anything else with the branch.

## Phase 2 — DONE (icon system)

`internal/tui/theme/icons.go` created with the status glyphs (`IconPass="✓"`, `IconWarn="△"`, `IconFail="✕"`, `IconInfo="·"`, `IconSkipped="○"`) plus 4 item glyphs needed to replace real emoji: `IconKey="⚿"`, `IconClipboard="▤"`, `IconPublish="↑"`, `IconWindow="▣"`.

A broader emoji sweep (regex over the full pictographic Unicode ranges, not just the ones spotted by eye) found the true offenders were only: `🔑` `🪟` `📋` `🚀` `🔒` `🔗` in `internal/tui/screens/detail.go` and `internal/tui/app_actions.go`. Important scoping call made here: `⚠✓✕✔✖✦⚓⚡✎✉⚙▲` were **deliberately left alone** even though some fall in the same Unicode block — they're already text-presentation/monochrome in virtually every terminal font (not colorful pictographs), they're used consistently by the CLI side too (`internal/ui`'s `Warn`/`Success`/`Error` all use `⚠`/`✓`/`✕`-family glyphs), and changing them would make the TUI and CLI visually inconsistent with each other for no benefit. Only genuine multi-color pictographic emoji were replaced:
- `🔑` (Bind/Change SSH key file) → `⚿`
- `📋` (Show public key) → `▤`
- `🚀` (Publish SSH key to platform) → `↑`
- `🪟` (Open in new terminal window, both `detail.go` and `app_actions.go`'s multi-account picker) → `▣`
- `🔒` (Manage passphrase) and `🔗` (Manage HTTPS token) → icon dropped entirely rather than inventing an awkward-fit replacement glyph; several existing menu items already have no icon prefix, so this is consistent, not a regression.

No test files referenced the old emoji labels, so no test updates were needed. `go build`/`go vet`/`go test ./...` all green.

## Phase 3 — DONE (Health & Security screen)

New `internal/tui/screens/health.go` (`Health` screen) built directly on `internal/diagnostics.Run` — same package the CLI's `doctor` now uses, so they can't disagree. Replaces the old plain-text `opDoctor`/`Report` dump entirely; `opDoctor` was deleted from `internal/tui/ops_report.go` (confirmed via grep it had no other callers) along with its now-unused `os/exec`/`path/filepath` imports.

- Security score rendered as a colored pill in the header (`PillActive`/`PillWarning`/`PillDanger` — green ≥90%, amber 60–89%, red <60%), plus an issue count line.
- Checks grouped by `Category` with section headers, and `Subject`-scoped checks (per-profile) nested under a "Profile: <name>" subsection row — a real visual hierarchy instead of one flat line per check.
- Cursor navigates only over check rows (`↑/↓`/`j`/`k`); section/subsection rows are just static separators, not selectable — implemented via a parallel `selectable []int` index array into the full `rows` slice (same shape idea as `ActionMenu`'s selectable-skipping, but simpler since Health has no per-row actions, just expand/collapse).
- `Enter`/`Space` toggles inline expansion of a check's `Detail` lines and `FixHint` — no nested modal, matching the plan's "single navigable list" requirement.
- `f` re-runs `diagnostics.Run(store, Options{Fix:true, ...})` in place (the TUI equivalent of `doctor --fix`); `r` refreshes without fixing. Both reuse the same background-load-with-spinner pattern as `screens/stats.go` (`healthLoadedMsg`, `loadCmd`, `startLoad`).
- SSH connectivity checking (the one piece `diagnostics.Options.VerifySSH` needs injected, since it's CLI-interactive on the CLI side) is satisfied here by a small `verifySSHForHealth` closure using the already-shared `internal/ssh.CheckPlatformConnection`/`DefaultPlatforms` — the same primitives Detail's own platform-connectivity checks already use, so no new SSH logic was invented.
- Dashboard's `doctor`/`security` action (`internal/tui/app_actions.go`) now does `pushCmd(screens.NewHealth(a.store, a.theme))` instead of `runTaskCmd(... opDoctor ...)`.
- New `internal/tui/screens/health_test.go` (3 tests) follows house style exactly: construct `Health` directly, feed it a synthetic `diagnostics.Report` via `healthLoadedMsg`, drive with `.Update(tea.KeyMsg{...})`, assert on struct state (`rows`, `selectable`, `cursor`, `expanded`) and emitted `core.*Msg` (`ScreenPopMsg` on Esc).

`go build`/`go vet`/`go test ./...` all green. **Not done**: an actual interactive visual smoke-test in a real terminal — Bubble Tea's alt-screen program can't be meaningfully driven/verified from this non-interactive shell environment, so this screen's real-terminal rendering has only been verified via the unit tests + code review, not eyes-on. Worth a quick manual `git-user` launch + `d`/doctor action check next time there's a real terminal in the loop.

## Phase 4 — DONE (Policy & Signers screens)

New `internal/tui/screens/policy.go` (`PolicyScreen`) and `internal/tui/screens/signers.go` (`SignersScreen`) — both self-contained "sub-ActionMenu screens" following the exact same architectural pattern as the existing `passphrase_menu.go` (own `components.ActionMenu`, own `refreshActions()`, simple in-screen toggles handled locally, anything needing text/confirm input emitted upward as `core.ActionResultMsg{Kind: ...}` for `app_actions.go`/`app_confirm.go`/`app_forms.go`/`app_options.go` to handle via the established Confirm→Form chaining pattern — screens never push other screens directly, they only emit messages the app layer turns into pushes).

- `components.ActionMenu`'s `SystemActions` signature changed to `SystemActions(th, showFixRemote, inRepo bool)` — a "▤ Repository policy" entry now appears in the Health & Security section only when `git.IsInRepo()` (`dashboard.go`'s one call site updated; 4 test call sites in `action_menu_test.go` updated via `sed`).
- `PolicyScreen` shows: whether `.git-user-policy` exists, `require_signing`, `allowed_email_domains`, whether the enforcing hook is installed (`hookInstalledMarker`, reading the same `.git/hooks/pre-commit` marker prefix `diagnostics`/`hookops` use), and the `.allowed-signers` entry count — all read **fresh on every `View()` call** rather than cached at construction, since these are small local file reads (not the expensive git-log/SSH work Health/Stats justify backgrounding) — this also sidesteps needing any special "refresh me after a child form closed" plumbing beyond the `core.StoreRefreshedMsg` handler already added for good measure.
- Actions: "Create/edit policy" → `Confirm` ("require signing?") → **always** continues into a `Form` (domains) regardless of yes/no answer — implemented as a special-cased branch in `app_confirm.go` *before* the generic `if !msg.Confirmed { Cancelled }` gate, exactly mirroring how the existing `ssh-sign` in-flow confirm already works. "Install enforcing hook" reuses `opHook("install")` — at the time Phase 4 landed this was still the old pre-commit-only implementation (intentionally not blocked on Phase 7, since Phase 7's job was exactly to upgrade what this button calls); Phase 7 has since landed, so this button now installs all three hooks with full policy enforcement, with no changes needed on the Policy screen's side.
- `SignersScreen` lists `.allowed-signers` entries as removable items (`✕ Remove <principals>`), plus "Add from a local identity" (pushes an `Options` picker over `store.Users`, with a `"__active__"` sentinel for "use the active identity" — deliberately **not** an empty-string key, since the app-wide convention treats `Choice == ""` as Cancel) and "Add by email + public key file" (`Form`, two fields).
- New `internal/tui/ops_policy.go`: thin op wrappers (`opPolicyWrite`, `opSignerAddIdentity`, `opSignerAddEmail`, `opSignerRemove`) over `internal/policyops`, run via the existing `a.runTaskCmd` background-task pattern — `opSignerAddIdentity`/`opSignerAddEmail` also call `policyops.WireAllowedSignersConfig` after a successful add, so the TUI path auto-wires local git config exactly like the CLI's `policy signers add` already does.
- Tests: `screens/policy_test.go`, `screens/signers_test.go` (behavioral, house style) plus `internal/tui/ops_policy_test.go` — the latter is a genuine functional test using a temp git repo (`withTempRepo` helper, same `os.Chdir`-based pattern as `cli/hook_test.go`) that exercises the real `policyops` calls end-to-end: write a policy file and read it back, add a signer from a local identity and confirm both the `.allowed-signers` content *and* that `git config gpg.ssh.allowedSignersFile` actually got set, then remove it and confirm it's gone. This is more thorough than the screen-level tests alone (which only check emitted messages, not that the underlying operation works) and was worth the extra time given `internal/policyops` had zero test coverage before this.

**Important safety note for future test-writing in this package**: `go test` for `internal/tui` runs with its process `cwd` still inside the real `git-user` checkout (`testutil.Sandbox` only isolates `HOME`/config env vars, it does **not** `os.Chdir`) — so any test that calls something touching `git.RepoRoot()`/`git.IsInRepo()` without explicitly `os.Chdir`-ing into a temp repo first (via `withTempRepo`) would silently operate against *this actual repository* instead of a sandbox. Screen-construction-only tests (like the ones in `policy_test.go`/`signers_test.go`) are safe since they never call the write-side `policyops`/`hookops` functions, only check `repoErr`/emitted messages — but any new ops-level test must use `withTempRepo`.

`go build`/`go vet`/`go test ./...` all green.

## Phase 5 — DONE (Verify screen)

New `internal/tui/screens/verify.go` (`VerifyScreen`), modeled directly on `screens/stats.go`'s background-load-with-spinner pattern, calling `stats.VerifyRange` directly — the one place shared logic already worked correctly before this redesign even started, so this phase just replicates a proven pattern rather than inventing one.

- Default range mirrors `cli/verify.go`'s `resolveDefaultRange` exactly (`HEAD~50..HEAD`, falling back to full history — displayed as `"full history"` — on a repo with fewer than 50 commits); the small helper is duplicated locally as `resolveDefaultVerifyRange` rather than shared, same judgment call already made for `hookops.resolveDefaultMergeRange` (screens can't import `internal/cli`, and it's 4 lines).
- Per-author signature-status coloring is written directly in `View()` (not shared from `cli/stats.go`'s `formatSignatureStatus`), matching the existing precedent that `screens/stats.go` already renders its own colored status text rather than importing CLI presentation code.
- `e` opens a range-edit `Form` (`Skippable()`, blank = auto) via the same "emit `ActionResultMsg`, let the app layer push the screen" pattern as Policy/Signers; `r` refreshes in place.
- **New plumbing point**: the range-edit form needs to hand its result back to the *specific* `VerifyScreen` instance still sitting on the stack underneath it (not a generic op call) — solved via `a.activeScreen().(*screens.VerifyScreen)` type assertion in `app_forms.go`'s new `"verify-range"` case, then calling the screen's own exported `SetRange(string) tea.Cmd` method. This is the first place in the redesign where the app layer reaches back into a specific screen instance rather than just dispatching to an op function; worth remembering as the pattern if a similar "form result must update the screen still underneath it" need comes up in a later phase.
- `SystemActions` gains "✓ Verify commit signatures" alongside "▤ Repository policy", both gated on `inRepo`.
- Tests: `screens/verify_test.go` (4 behavioral tests: loaded-state population, `e` emits the right `ActionResultMsg`, `SetRange` starts a reload with the right state, Esc pops).

`go build`/`go vet`/`go test ./...` all green.

## Phase 6 — DONE (Dashboard/Detail status redesign)

**Dashboard identity rows** (`internal/tui/components/identity_list.go`): `IdentityItem` gained `HasToken`/`TokenExpiring`/`TokenExpired` and `PolicyChecked`/`PolicyOK`.
- `TokenBadgeState(expiresAt string) (hasToken, expiring, expired bool)` (exported — also reused by `detail.go`, see below) derives the token badge **purely from the `HTTPSTokenExpiresAt` config string**, not a real OS-keychain lookup — deliberately, since this runs for every row on every periodic refresh tick, and a real `keyring.HasHTTPSToken` call per row per tick would be needless I/O for a cosmetic badge. This mirrors `doctor`'s own per-profile audit, which already only checks the expiry-metadata string for non-active profiles rather than hitting the keychain for each one.
- `computeActivePolicyStatus` does the repo-scoped check (`git.RepoRoot`, `config.LoadRepoPolicy`) **once per full rebuild**, not once per row — folded into `buildIdentityItems` as a post-step after the row loop rather than threaded through `Dashboard.Update` separately as the original plan sketch suggested; same outcome (one I/O call, not N), simpler wiring, noted here as a deliberate deviation from the plan's exact suggested call site.
- `renderIdentityLine` adds a `TOKEN` pill (green/amber/red for healthy/expiring/expired) and, active-identity-only, a `POLICY` pill (green/red) when a policy check applies.

**Detail overview** (`internal/tui/screens/detail.go`):
- Signing promoted out of the "SSH & SECURITY" bullet list into its own "SIGNING" section, driven by `internal/signing.CurrentStatus(user)` — the exact same function the CLI's `sign` command (once Phase 1 lands, already does) and a future Detail "toggle-sign" rewire would use, so wording can't drift between them. Now also shows the signing key's basename, which can legitimately differ from the general SSH auth key and wasn't visible anywhere before.
- New "HTTPS & TOKEN" section: token presence is a **real `keyring.HasHTTPSToken` check** here (unlike the Dashboard row badge) — but computed once inside `refreshActions()` (which already re-runs on `core.StoreRefreshedMsg`, i.e. after any task completes, including setting a token) and cached in a new `d.hasToken` field, not re-queried on every `View()` call. This distinction matters: Dashboard's periodic-refresh badge needed the free metadata-only proxy to stay cheap at N-rows-times-per-tick, but Detail is a single-identity, less-frequently-reconstructed screen where one real keychain check per meaningful state change is fine — reuses `components.TokenBadgeState` for the expiry-coloring part.
- New "REPOSITORY POLICY" section — active identity only, only rendered when in a repo with a policy file that has a requirement, listing specific non-compliance reasons (mirrors `diagnostics.checkRepoPolicy`'s logic, re-expressed inline here since `screens` can't/shouldn't reach into a per-check `diagnostics.Check` for this narrow a use).

Tests: `internal/tui/components/token_badge_test.go` (new — `TokenBadgeState` boundary cases + `buildIdentityItems` wiring). No existing test in `identity_list_test.go`/`detail_test.go` asserted on full rendered line content, so nothing needed updating there (checked first, per the plan's explicit warning about this).

`go build`/`go vet`/`go test ./...` all green.

## Phase 7 — DONE (hook parity fix)

`internal/tui/ops_hook.go` fully rewritten: `opHook`/`opHookInstall`/`opHookUninstall`/`opHookCheck` now call `hookops.Install`/`Uninstall`/`CheckIdentity` — the same package `cli/hook.go` uses — instead of the old bespoke pre-commit-only logic. This closes the exact gap Phase 7 was scoped for: the TUI's hook action now installs `pre-commit` **and** `pre-push` **and** `post-merge`, and `hook check` now enforces `.git-user-policy` (`require_signing`/`allowed_email_domains`) via `hookops.CheckIdentity`, none of which the old TUI implementation did at all.

- v1 overwrite policy (as planned): `hookops.Install(nil)` — a `nil` `confirmOverwrite` callback means "never overwrite a foreign hook," reported back as `OutcomeSkippedExisting` in the result text rather than silently clobbering or blocking entirely. A per-hook interactive Confirm loop remains a reasonable fast-follow, not required now.
- `opHookInstall` also wires `.allowed-signers` via `policyops.WireAllowedSignersConfig` after a successful install, matching the CLI's `installHook` behavior — this was previously entirely absent from the TUI path.
- Deleted the now-dead `isGitUserHook`/`hookGitDir`/`stringsTrimNewline` helpers. `ops_extra_test.go`'s `TestHookHelpers` (which tested those helpers directly) was replaced with `TestOpHookInstallUninstall`, a real functional test using the `withTempRepo` pattern that installs all three hooks, confirms idempotent re-install, then uninstalls and confirms removal — a strictly better test than the old one since it exercises the actual `opHook` entry point instead of two internal string-matching helpers.
- New `internal/tui/ops_policy_test.go`'s `TestOpHookCheckEnforcesRepoPolicy`: sets up a temp repo with `require_signing=true`, confirms `opHook("check")` **fails** while signing is disabled for the active identity, then confirms it **passes** once `SetSigningKey` is applied — this is the concrete, verified proof that the parity gap is actually closed, not just that the code compiles.

`go build`/`go vet`/`go test ./...` all green.

## Phase 8 — investigation only, already resolved (no code changes)

This was evaluated back when the CLI catalog was first built (before Phase 1 even started) and the judgment call was already made and holds:
- `exec`/`env`: shell-integration primitives already conceptually covered by the existing multi-account/new-terminal-window menu — no direct TUI action needed.
- `current`: redundant with Dashboard's always-visible active-identity display — no action needed.
- `edit`: confirmed (read `cli/edit.go`) functionally identical to the TUI's existing per-identity "email" action — already covered, no new surface needed.
- `init`/`completion`: shell-setup/dev-tooling, not identity-management — out of scope for a TUI.

## All 8 phases are now complete.

Remaining open items, none blocking, all previously flagged inline above:
1. **Real-terminal visual smoke test** — every new screen (`Health`, `PolicyScreen`, `SignersScreen`, `VerifyScreen`) and the Dashboard/Detail badge additions have only been verified via unit tests + code review, never actually looked at in a live terminal. Worth a `git-user` launch in a real TTY before considering this fully done.
2. **Icon glyph choices** — `⚿`/`▤`/`↑`/`▣` (replacing `🔑`/`📋`/`🚀`/`🪟`) were picked for being safely text-presentation, not for having been visually confirmed to render well in every common terminal font. Worth a glance in whatever terminal is actually used day-to-day.
3. **Per-hook interactive overwrite** — Phase 7's `hookops.Install(nil)` policy (skip+report on a foreign hook, never prompt) is a deliberate v1 simplification; a Confirm-screen loop for interactive per-hook overwrite in the TUI is a reasonable fast-follow if it turns out to matter in practice.
4. **Nothing in this whole effort has been `git commit`ed** — see "Before you continue" above, still applicable to everything through Phase 7.

## Verification pattern to keep using

- After Phase 1 (done): golden-output style — ran existing/updated tests plus manual scratch-repo runs of every touched CLI command, diffed against known-good prior output line by line (indentation included).
- For each TUI phase: `go build ./...`, `go vet ./...`, `go test ./internal/tui/...`, plus new screens get behavioral tests following the existing pattern — construct screen directly with an in-memory `*config.Store`, drive via `.Update(tea.KeyMsg{...})`, assert on struct state + emitted `core.*Msg` (see `dashboard_test.go`/`detail_test.go`/`stats_test.go`). No snapshot/pixel testing.
- Manual pass per phase: launch the actual TUI (`git-user` with no args in a TTY) in a scratch config to confirm real rendering/navigation.

---

## Full original plan text (Phases 2–8 detail, as approved)

The complete original plan — written before Phase 1 implementation started — is kept at:
`/home/sbh/.claude-profiles/fugitive/plans/https-arxiv-org-pdf-2604-14014-read-this-flickering-music.md`

That file has the full struct/package designs (written before Phase 1 landed — Phase 1's actual final shapes, documented above, are very close but occasionally more precise than that draft, e.g. `rekeyops.Rotate` takes an injected `generateKey` callback rather than a `passphrase string`, since the CLI and TUI generate keys via genuinely different mechanisms). Read this progress file first; fall back to that plan file only for phases 2–8's design detail not yet summarized here.
