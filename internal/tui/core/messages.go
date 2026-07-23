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

// ── Generic Action Result ─────────────────────────────────────────────────────

// ActionResultMsg is a generic result for any action.
type ActionResultMsg struct {
	Kind    string
	Name    string // identity name if applicable
	Success bool
	Message string
	Err     error
}
