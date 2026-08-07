package cli

import (
	"os"
	"os/exec"
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ui"
)

// forceTTY makes ui.IsTTY report true for the duration of the test.
func forceTTY(t *testing.T) {
	t.Helper()
	old := ui.IsTTYFn
	ui.IsTTYFn = func() bool { return true }
	t.Cleanup(func() { ui.IsTTYFn = old })
}

func setGitConfig(name, email string) {
	_ = exec.Command("git", "config", "--global", "user.name", name).Run()
	_ = exec.Command("git", "config", "--global", "user.email", email).Run()
}

func TestShouldPromptFirstRunImport(t *testing.T) {
	// Interactive CLI setup commands should be allowed to prompt.
	for _, sub := range []string{"register", "reg", "switch", "sw"} {
		if !shouldPromptFirstRunImport(sub) {
			t.Errorf("shouldPromptFirstRunImport(%q) = false, want true", sub)
		}
	}

	// The TUI launcher must NOT prompt on the plain terminal — the import
	// question is asked inside the TUI itself.
	for _, sub := range []string{"", "tui", "-i", "--interactive"} {
		if shouldPromptFirstRunImport(sub) {
			t.Errorf("shouldPromptFirstRunImport(%q) = true, want false (TUI asks internally)", sub)
		}
	}

	// Read-only/system commands must never prompt.
	for _, sub := range []string{"list", "current", "prompt", "completion", "--help", "-h", "help", "--version", "-v", "version", "--update", "update", "import", "import-original", "sync", "doctor", "security", "stats", "logout"} {
		if shouldPromptFirstRunImport(sub) {
			t.Errorf("shouldPromptFirstRunImport(%q) = true, want false", sub)
		}
	}
}

func TestMaybePromptFirstRunImport_Decline(t *testing.T) {
	setupTestEnv(t)
	forceTTY(t)
	setGitConfig("alice", "alice@example.com")

	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		return false
	}

	if err := maybePromptFirstRunImport(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ := config.Load()
	if len(store.Users) != 0 {
		t.Fatalf("expected no identities to be imported, got %d", len(store.Users))
	}
	if !store.ImportPrompted {
		t.Error("expected ImportPrompted to be set after declining")
	}
	if store.Original != nil {
		t.Error("expected no original snapshot when declining")
	}
}

func TestMaybePromptFirstRunImport_ImportsWithChosenName(t *testing.T) {
	setupTestEnv(t)
	forceTTY(t)
	setGitConfig("alice", "alice@example.com")
	_ = exec.Command("git", "config", "--global", "core.sshCommand", "ssh -i ~/.ssh/id_alice -o IdentitiesOnly=yes -o SomeFlag").Run()

	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		return true
	}
	ui.PromptFn = func(label string) (string, error) {
		return "my-main-account", nil
	}

	if err := maybePromptFirstRunImport(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ := config.Load()
	u := store.FindUser("my-main-account")
	if u == nil {
		t.Fatalf("expected identity %q to be imported", "my-main-account")
	}
	if u.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %q", u.Email)
	}
	if u.Source != "original" {
		t.Errorf("expected source original, got %q", u.Source)
	}
	if u.SSHCommand != "ssh -i ~/.ssh/id_alice -o IdentitiesOnly=yes -o SomeFlag" {
		t.Errorf("expected SSHCommand to be preserved exactly, got %q", u.SSHCommand)
	}
	if u.SSHKey != "~/.ssh/id_alice" {
		t.Errorf("expected SSHKey extracted from command, got %q", u.SSHKey)
	}
	if store.Original == nil || store.Original.Email != "alice@example.com" {
		t.Error("expected original snapshot to be saved")
	}
	if !store.ImportPrompted {
		t.Error("expected ImportPrompted to be set after importing")
	}
}

func TestMaybePromptFirstRunImport_UsesDefaultName(t *testing.T) {
	setupTestEnv(t)
	forceTTY(t)
	setGitConfig("alice", "alice@example.com")

	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		return true
	}
	ui.PromptFn = func(label string) (string, error) {
		return "", nil // user pressed enter → use suggestion
	}

	if err := maybePromptFirstRunImport(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ := config.Load()
	if store.FindUser("alice") == nil {
		t.Error("expected identity to be named after git user.name when nothing typed")
	}
}

func TestMaybePromptFirstRunImport_SkipsWhenUsersExist(t *testing.T) {
	setupTestEnv(t)
	forceTTY(t)
	setGitConfig("alice", "alice@example.com")

	store, _ := config.Load()
	_ = store.AddUser("existing", "existing@example.com")
	_ = config.Save(store)

	called := false
	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		called = true
		return true
	}

	if err := maybePromptFirstRunImport(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("prompt should not be shown when identities already exist")
	}
}

func TestMaybePromptFirstRunImport_NotTTY(t *testing.T) {
	setupTestEnv(t)
	// IsTTYFn is not overridden → false in tests.
	setGitConfig("alice", "alice@example.com")

	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		t.Fatal("prompt should not be shown outside a TTY")
		return false
	}

	if err := maybePromptFirstRunImport(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ := config.Load()
	if len(store.Users) != 0 {
		t.Fatalf("expected no identities in non-TTY mode, got %d", len(store.Users))
	}
}

func TestRunImportOriginal_PromptsForName(t *testing.T) {
	setupTestEnv(t)
	setGitConfig("bob", "bob@example.com")

	ui.PromptFn = func(label string) (string, error) {
		return "work-identity", nil
	}

	if err := runImportOriginal(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ := config.Load()
	u := store.FindUser("work-identity")
	if u == nil {
		t.Fatal("expected identity to be imported under the chosen name")
	}
	if u.Source != "original" {
		t.Errorf("expected source original, got %q", u.Source)
	}
}

func TestRunImportOriginal_UsesArgNameWithoutPrompt(t *testing.T) {
	setupTestEnv(t)
	setGitConfig("bob", "bob@example.com")

	ui.PromptFn = func(label string) (string, error) {
		t.Fatal("should not prompt when a name argument is provided")
		return "", nil
	}

	if err := runImportOriginal([]string{"main"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store, _ := config.Load()
	if store.FindUser("main") == nil {
		t.Error("expected identity to be imported as main")
	}
}

func TestExecute_OnboardingDeclinesThenRegister(t *testing.T) {
	setupTestEnv(t)
	forceTTY(t)
	setGitConfig("jane", "jane@example.com")

	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		return false // decline the import prompt
	}
	ui.SelectFn = func(label string, options []string) (int, error) {
		return 2, nil // skip SSH setup during register
	}

	os.Args = []string{"git-user", "register", "--name", "personal", "--email", "personal@example.com"}
	if err := Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	store, _ := config.Load()
	if store.FindUser("personal") == nil {
		t.Fatal("expected registered identity personal")
	}
	if store.FindUser("jane") != nil {
		t.Fatal("original git identity should NOT be imported when the prompt is declined")
	}
	if !store.ImportPrompted {
		t.Error("expected ImportPrompted to be set")
	}
}

func TestExecute_OnboardingImportsWithChosenName(t *testing.T) {
	setupTestEnv(t)
	forceTTY(t)
	setGitConfig("jane", "jane@example.com")

	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		return true // accept the import prompt
	}
	ui.PromptFn = func(label string) (string, error) {
		return "my-personal", nil // choose our own identity name
	}
	ui.SelectFn = func(label string, options []string) (int, error) {
		return 2, nil // skip SSH setup during register
	}

	os.Args = []string{"git-user", "register", "--name", "work", "--email", "work@example.com"}
	if err := Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	store, _ := config.Load()
	imported := store.FindUser("my-personal")
	if imported == nil {
		t.Fatal("expected original identity to be imported under the chosen name")
	}
	if imported.Email != "jane@example.com" {
		t.Errorf("expected imported email jane@example.com, got %q", imported.Email)
	}
	if store.FindUser("work") == nil {
		t.Fatal("expected registered identity work to still be created")
	}
}

func TestExecute_OnboardingDoesNotRunForList(t *testing.T) {
	setupTestEnv(t)
	forceTTY(t)
	setGitConfig("jane", "jane@example.com")

	prompted := false
	ui.ConfirmFn = func(question string, defaultYes bool) bool {
		prompted = true
		return false
	}

	os.Args = []string{"git-user", "list"}
	if err := Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if prompted {
		t.Error("list should never trigger the onboarding prompt")
	}
}
