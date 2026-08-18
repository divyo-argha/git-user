package core

import (
	"time"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// ── Animation ─────────────────────────────────────────────────────────────────

// AnimTickMsg is fired on every animation frame (50 ms by default).
// Defined here in core so both the app layer and individual screens can
// type-switch on it without creating a circular import.
type AnimTickMsg time.Time

// ── Store / Data Messages ─────────────────────────────────────────────────────

// StoreRefreshedMsg is sent after config.Load() completes.
type StoreRefreshedMsg struct {
	Store *config.Store
	Err   error
}

// SyncStatusMsg reports whether the active identity matches the resolved git
// config (user.name/user.email). When false, commits would be authored by a
// different identity than the one the user believes is active.
type SyncStatusMsg struct {
	InSync bool
	Err    error
}

// ── Agent Status ──────────────────────────────────────────────────────────────

// AgentStatusMsg is sent after checking SSH agent connectivity.
type AgentStatusMsg struct {
	Connected bool
	KeyCount  int
	Err       error
}

// ── Identity Operations ───────────────────────────────────────────────────────

// IdentitySwitchedMsg is sent after an identity switch completes.
type IdentitySwitchedMsg struct {
	Name    string
	Email   string
	Success bool
	Err     error
}

// IdentityRemovedMsg is sent after removing an identity.
type IdentityRemovedMsg struct {
	Name string
	Err  error
}

// ── Toast / UI Notifications ──────────────────────────────────────────────────

// ToastMsg triggers a transient notification.
type ToastMsg struct {
	Text     string
	Style    theme.ToastStyleKind
	Duration time.Duration
}

// ToastExpiredMsg signals that the toast should be dismissed.
type ToastExpiredMsg struct{}

// ── Navigation ────────────────────────────────────────────────────────────────

// ScreenPushMsg tells the app to push a new screen onto the stack.
type ScreenPushMsg struct {
	Screen Screen
}

// ScreenPopMsg tells the app to pop the current screen.
type ScreenPopMsg struct{}

// ── Confirmation Dialog ───────────────────────────────────────────────────────

// ConfirmResultMsg is returned by the confirmation dialog.
type ConfirmResultMsg struct {
	Confirmed bool
	Context   string // identifies what was being confirmed
}

// FormResultMsg represents the submitted values of a form.
type FormResultMsg struct {
	Context string
	Values  []string
}

// ── Platform Check Messages ──────────────────────────────────────────────────

// PlatformConnectionMsg reports the status of connection checks for a platform.
type PlatformConnectionMsg struct {
	ProfileName string
	Platform    string // "GitHub", "GitLab", "Bitbucket"
	Username    string
	Status      string // "checking", "connected", "not_added", "network_error"
}

// ── Generic Action Result ─────────────────────────────────────────────────────

// ActionResultMsg is a generic result for any action.
type ActionResultMsg struct {
	Kind    string
	Name    string // identity name if applicable
	Success bool
	Message string
	Err     error
}

// ── Options / Choice ──────────────────────────────────────────────────────────

// OptionResultMsg is returned by a generic option/choice screen. An empty
// Choice means the user cancelled (Esc).
type OptionResultMsg struct {
	Context string
	Choice  string
}

// ── In-TUI Background Operations ──────────────────────────────────────────────

// TaskResultMsg is the result of an operation run as a background command
// inside the TUI (switch, logout, export, import, security audit, etc.).
type TaskResultMsg struct {
	Kind       string // action kind that produced this result
	Name       string // identity name if applicable
	Success    bool
	Detail     string // report text (shown on a Report screen when ShowReport)
	ShowReport bool   // push a Report screen with Detail
	Err        error
}

// ── External Process (suspend/resume) ───────────────────────────────────────

// ShellSessionEndedMsg is delivered when a subshell launched via
// tea.ExecProcess (an isolated shell scoped to one identity) exits and the
// TUI regains the terminal.
type ShellSessionEndedMsg struct {
	Name string
	Err  error
}

// ── Version Check ─────────────────────────────────────────────────────────────

// VersionCheckMsg reports the result of the asynchronous remote release check.
type VersionCheckMsg struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
}

