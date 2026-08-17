package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// ── Keymap Tests ────────────────────────────────────────────────────────────

func TestKeyConstants(t *testing.T) {
	if KeyUp != "up" {
		t.Errorf("KeyUp = %q, want %q", KeyUp, "up")
	}
	if KeyDown != "down" {
		t.Errorf("KeyDown = %q, want %q", KeyDown, "down")
	}
	if KeyEnter != "enter" {
		t.Errorf("KeyEnter = %q, want %q", KeyEnter, "enter")
	}
	if KeyEsc != "esc" {
		t.Errorf("KeyEsc = %q, want %q", KeyEsc, "esc")
	}
	if KeyQuit != "q" {
		t.Errorf("KeyQuit = %q, want %q", KeyQuit, "q")
	}
	if KeyHelp != "?" {
		t.Errorf("KeyHelp = %q, want %q", KeyHelp, "?")
	}
}

func TestIsEscKey(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
		want bool
	}{
		{"esc type", tea.KeyMsg{Type: tea.KeyEsc}, true},
		{"escape type", tea.KeyMsg{Type: tea.KeyEscape}, true},
		{"esc string", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("esc")}, true},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}, false},
		{"q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEscKey(tt.msg)
			if got != tt.want {
				t.Errorf("IsEscKey(%+v) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

// ── Help Text Tests ─────────────────────────────────────────────────────────

func TestHelpTexts_NotEmpty(t *testing.T) {
	tests := []struct {
		name string
		fn   func() string
	}{
		{"DashboardHelp", DashboardHelp},
		{"DetailHelp", DetailHelp},
		{"FormHelp", FormHelp},
		{"ConfirmHelp", ConfirmHelp},
		{"FilterHelp", FilterHelp},
		{"ImportExportHelp", ImportExportHelp},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got == "" {
				t.Errorf("%s() returned empty string", tt.name)
			}
		})
	}
}

func TestHelpTexts_ContainKeyReferences(t *testing.T) {
	helpers := []struct {
		name string
		fn   func() string
	}{
		{"DashboardHelp", DashboardHelp},
		{"DetailHelp", DetailHelp},
		{"FormHelp", FormHelp},
		{"ConfirmHelp", ConfirmHelp},
		{"FilterHelp", FilterHelp},
		{"ImportExportHelp", ImportExportHelp},
	}
	for _, h := range helpers {
		t.Run(h.name, func(t *testing.T) {
			got := h.fn()
			if !strings.Contains(got, "Enter") && !strings.Contains(got, "Esc") {
				t.Errorf("%s() should mention key names: %q", h.name, got)
			}
		})
	}
}

func TestHelpTexts_EndWithoutNewline(t *testing.T) {
	helpers := []func() string{
		DashboardHelp,
		DetailHelp,
		FormHelp,
		ConfirmHelp,
		FilterHelp,
		ImportExportHelp,
	}
	for i, fn := range helpers {
		got := fn()
		if strings.HasSuffix(got, "\n") {
			t.Errorf("help text %d should not end with newline: %q", i, got)
		}
	}
}

func TestHelpTexts_AreUnique(t *testing.T) {
	helpers := []func() string{
		DashboardHelp,
		DetailHelp,
		FormHelp,
		ConfirmHelp,
		FilterHelp,
		ImportExportHelp,
	}
	seen := make(map[string]string)
	for _, fn := range helpers {
		key := fn()
		if prev, ok := seen[key]; ok {
			t.Errorf("Duplicate help text: %q and %q both return %q", prev, key, key)
		}
		seen[key] = key
	}
}

// ── Message Type Tests ──────────────────────────────────────────────────────

func TestMessageTypes_AreStructs(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.Msg
	}{
		{"AnimTickMsg", AnimTickMsg(time.Now())},
		{"StoreRefreshedMsg", StoreRefreshedMsg{}},
		{"AgentStatusMsg", AgentStatusMsg{}},
		{"IdentitySwitchedMsg", IdentitySwitchedMsg{}},
		{"IdentityRemovedMsg", IdentityRemovedMsg{}},
		{"ToastMsg", ToastMsg{}},
		{"ToastExpiredMsg", ToastExpiredMsg{}},
		{"ScreenPushMsg", ScreenPushMsg{}},
		{"ScreenPopMsg", ScreenPopMsg{}},
		{"ConfirmResultMsg", ConfirmResultMsg{}},
		{"FormResultMsg", FormResultMsg{}},
		{"PlatformConnectionMsg", PlatformConnectionMsg{}},
		{"ActionResultMsg", ActionResultMsg{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.msg == nil {
				t.Errorf("%s should be a valid tea.Msg", tt.name)
			}
		})
	}
}

func TestAnimTickMsg(t *testing.T) {
	now := time.Now()
	msg := AnimTickMsg(now)
	if time.Time(msg).IsZero() {
		t.Error("AnimTickMsg should wrap a non-zero time")
	}
}

func TestStoreRefreshedMsg(t *testing.T) {
	msg := StoreRefreshedMsg{Store: nil, Err: nil}
	if msg.Store != nil || msg.Err != nil {
		t.Error("StoreRefreshedMsg zero value should have nil fields")
	}
}

func TestPlatformConnectionMsg(t *testing.T) {
	msg := PlatformConnectionMsg{
		ProfileName: "test",
		Platform:    "GitHub",
		Username:    "@user",
		Status:      "connected",
	}
	if msg.ProfileName != "test" {
		t.Errorf("ProfileName = %q, want %q", msg.ProfileName, "test")
	}
	if msg.Platform != "GitHub" {
		t.Errorf("Platform = %q, want %q", msg.Platform, "GitHub")
	}
	if msg.Username != "@user" {
		t.Errorf("Username = %q, want %q", msg.Username, "@user")
	}
	if msg.Status != "connected" {
		t.Errorf("Status = %q, want %q", msg.Status, "connected")
	}
}

func TestActionResultMsg(t *testing.T) {
	msg := ActionResultMsg{
		Kind:    "switch",
		Name:    "dev",
		Success: true,
		Message: "switched",
		Err:     nil,
	}
	if msg.Kind != "switch" {
		t.Errorf("Kind = %q", msg.Kind)
	}
	if !msg.Success {
		t.Error("Success should be true")
	}
}

// ── Command Tests ───────────────────────────────────────────────────────────

func TestToastTimerCmd(t *testing.T) {
	cmd := ToastTimerCmd(10 * time.Millisecond)
	if cmd == nil {
		t.Fatal("ToastTimerCmd should return a non-nil command")
	}

	msg := cmd()
	if _, ok := msg.(ToastExpiredMsg); !ok {
		t.Errorf("ToastTimerCmd should return ToastExpiredMsg, got %T", msg)
	}
}

func TestShowToastCmd(t *testing.T) {
	cmd := ShowToastCmd("hello", 0, time.Second)
	if cmd == nil {
		t.Fatal("ShowToastCmd should return a non-nil command")
	}

	msg := cmd()
	toast, ok := msg.(ToastMsg)
	if !ok {
		t.Errorf("ShowToastCmd should return ToastMsg, got %T", msg)
	}
	if toast.Text != "hello" {
		t.Errorf("ToastMsg.Text = %q, want %q", toast.Text, "hello")
	}
	if toast.Duration != time.Second {
		t.Errorf("ToastMsg.Duration = %v, want %v", toast.Duration, time.Second)
	}
}

func TestRefreshStoreCmd(t *testing.T) {
	cmd := RefreshStoreCmd()
	if cmd == nil {
		t.Fatal("RefreshStoreCmd should return a non-nil command")
	}

	msg := cmd()
	if _, ok := msg.(StoreRefreshedMsg); !ok {
		t.Errorf("RefreshStoreCmd should return StoreRefreshedMsg, got %T", msg)
	}
}

func TestCheckSyncStatusCmd(t *testing.T) {
	// nil store never runs git commands and reports in sync.
	msg := CheckSyncStatusCmd(nil)()
	if sm, ok := msg.(SyncStatusMsg); !ok || !sm.InSync {
		t.Errorf("nil store: got %#v, want SyncStatusMsg{InSync: true}", msg)
	}

	// Empty store (no active identity) is always in sync.
	msg = CheckSyncStatusCmd(&config.Store{})()
	if sm, ok := msg.(SyncStatusMsg); !ok || !sm.InSync {
		t.Errorf("empty store: got %#v, want SyncStatusMsg{InSync: true}", msg)
	}

	// Active identity without a matching git config reports out of sync.
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	store := &config.Store{}
	if err := store.AddUser("eng", "eng@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetCurrent("eng"); err != nil {
		t.Fatal(err)
	}
	msg = CheckSyncStatusCmd(store)()
	sm, ok := msg.(SyncStatusMsg)
	if !ok {
		t.Fatalf("got %T, want SyncStatusMsg", msg)
	}
	if sm.InSync {
		t.Error("expected out of sync when git config has no matching identity")
	}

	// Once the git config matches the active identity, it reports in sync.
	gitDir := filepath.Join(dir, ".gitconfig")
	cfg := "[user]\n\tname = eng\n\temail = eng@example.com\n"
	if err := os.WriteFile(gitDir, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	msg = CheckSyncStatusCmd(store)()
	sm, ok = msg.(SyncStatusMsg)
	if !ok {
		t.Fatalf("got %T, want SyncStatusMsg", msg)
	}
	if !sm.InSync {
		t.Error("expected in sync when git config matches active identity")
	}
}

func TestShowToastCmd_DifferentStyles(t *testing.T) {
	tests := []struct {
		name  string
		style theme.ToastStyleKind
	}{
		{"success", theme.ToastStyleSuccess},
		{"error", theme.ToastStyleError},
		{"info", theme.ToastStyleInfo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ShowToastCmd("test", tt.style, time.Second)
			msg := cmd()
			toast := msg.(ToastMsg)
			if toast.Style != tt.style {
				t.Errorf("ToastMsg.Style = %d, want %d", toast.Style, tt.style)
			}
		})
	}
}

// ── Screen Interface Tests ──────────────────────────────────────────────────

// screenImplCheck verifies that the Screen interface is correctly defined.
// This is a compile-time check.
func TestScreenInterface(t *testing.T) {
	var s Screen = &testScreen{}
	if s.Title() != "test" {
		t.Errorf("testScreen.Title() = %q, want %q", s.Title(), "test")
	}
}

type testScreen struct{}

func (s *testScreen) Init() tea.Cmd                        { return nil }
func (s *testScreen) Update(msg tea.Msg) (Screen, tea.Cmd) { return s, nil }
func (s *testScreen) View(width, height int) string        { return "" }
func (s *testScreen) ShortHelp() string                    { return "help" }
func (s *testScreen) Title() string                        { return "test" }

func TestScreenInterface_ShortHelp(t *testing.T) {
	s := &testScreen{}
	if s.ShortHelp() != "help" {
		t.Errorf("ShortHelp() = %q, want %q", s.ShortHelp(), "help")
	}
}

func TestScreenInterface_InitReturnsNil(t *testing.T) {
	s := &testScreen{}
	if s.Init() != nil {
		t.Error("Init() should return nil")
	}
}

func TestScreenInterface_ViewDoesNotPanic(t *testing.T) {
	s := &testScreen{}
	_ = s.View(80, 24)
}

// ── KeyPassphraseMsg Tests ──────────────────────────────────────────────────

func TestKeyLoadedMsg(t *testing.T) {
	msg := KeyLoadedMsg{Path: "/tmp/key", Loaded: true}
	if msg.Path != "/tmp/key" {
		t.Errorf("KeyLoadedMsg.Path = %q", msg.Path)
	}
	if !msg.Loaded {
		t.Error("KeyLoadedMsg.Loaded should be true")
	}
}

func TestKeyPassphraseMsg(t *testing.T) {
	msg := KeyPassphraseMsg{Path: "/tmp/key", Protected: true, Err: nil}
	if msg.Path != "/tmp/key" {
		t.Errorf("KeyPassphraseMsg.Path = %q", msg.Path)
	}
	if !msg.Protected {
		t.Error("KeyPassphraseMsg.Protected should be true")
	}
	if msg.Err != nil {
		t.Errorf("KeyPassphraseMsg.Err should be nil, got %v", msg.Err)
	}
}

func TestScreenPopMsg(t *testing.T) {
	msg := ScreenPopMsg{}
	if _, ok := interface{}(msg).(tea.Msg); !ok {
		t.Error("ScreenPopMsg should implement tea.Msg")
	}
}

func TestConfirmResultMsg(t *testing.T) {
	msg := ConfirmResultMsg{Confirmed: true, Context: "delete"}
	if !msg.Confirmed {
		t.Error("ConfirmResultMsg.Confirmed should be true")
	}
	if msg.Context != "delete" {
		t.Errorf("ConfirmResultMsg.Context = %q", msg.Context)
	}
}

func TestFormResultMsg(t *testing.T) {
	msg := FormResultMsg{Context: "register", Values: []string{"dev", "dev@test.com"}}
	if msg.Context != "register" {
		t.Errorf("FormResultMsg.Context = %q", msg.Context)
	}
	if len(msg.Values) != 2 {
		t.Errorf("FormResultMsg.Values should have 2 items, got %d", len(msg.Values))
	}
}

func TestIdentitySwitchedMsg(t *testing.T) {
	msg := IdentitySwitchedMsg{Name: "dev", Email: "dev@test.com", Success: true, Err: nil}
	if msg.Name != "dev" || msg.Email != "dev@test.com" || !msg.Success {
		t.Error("IdentitySwitchedMsg fields mismatch")
	}
}

func TestIdentityRemovedMsg(t *testing.T) {
	msg := IdentityRemovedMsg{Name: "ops", Err: nil}
	if msg.Name != "ops" {
		t.Error("IdentityRemovedMsg.Name mismatch")
	}
}

func TestVersionCheckMsg(t *testing.T) {
	msg := VersionCheckMsg{
		CurrentVersion:  "v4.8.0",
		LatestVersion:   "v5.0.0",
		UpdateAvailable: true,
	}
	if !msg.UpdateAvailable || msg.LatestVersion != "v5.0.0" {
		t.Errorf("VersionCheckMsg fields mismatch: %+v", msg)
	}
}

func TestCheckVersionCmd_Offline(t *testing.T) {
	cmd := CheckVersionCmd("v4.8.0")
	if cmd == nil {
		t.Fatal("CheckVersionCmd returned nil")
	}
}

