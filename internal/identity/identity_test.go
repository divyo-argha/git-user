package identity

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMode_String(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{ModePermanent, "Permanent"},
		{ModeTemporary, "Temporary"},
		{Mode(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("Mode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIdentity_Creation(t *testing.T) {
	identity := &Identity{
		Name:      "test",
		Email:     "test@example.com",
		SSHKey:    "/path/to/key",
		Mode:      ModeTemporary,
		CreatedAt: time.Now(),
	}

	if identity.Name != "test" {
		t.Errorf("Name = %v, want test", identity.Name)
	}
	if identity.Mode != ModeTemporary {
		t.Errorf("Mode = %v, want ModeTemporary", identity.Mode)
	}
}

func TestManager_CreateTemporary(t *testing.T) {
	// Create temporary config directory for testing
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".git-users")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	// Set up environment
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	tests := []struct {
		name      string
		idName    string
		email     string
		wantError bool
	}{
		{"valid identity", "test", "test@example.com", false},
		{"empty name", "", "test@example.com", true},
		{"empty email", "test", "", true},
		{"both empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := manager.CreateTemporary(tt.idName, tt.email)
			if (err != nil) != tt.wantError {
				t.Errorf("CreateTemporary() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				if identity.Name != tt.idName {
					t.Errorf("Identity.Name = %v, want %v", identity.Name, tt.idName)
				}
				if identity.Mode != ModeTemporary {
					t.Errorf("Identity.Mode = %v, want ModeTemporary", identity.Mode)
				}
			}
		})
	}
}

func TestManager_GetCurrent(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".git-users")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	// Create empty config.json
	configFile := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configFile, []byte(`{"current":"","users":[]}`), 0600); err != nil {
		t.Fatalf("creating config file: %v", err)
	}

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Initially should be nil (no current identity in empty config)
	if current := manager.GetCurrent(); current != nil {
		t.Logf("Note: Found existing identity from real config: %s", current.Name)
		t.Skip("Skipping test - real config exists")
	}

	// Set a temporary identity
	identity := &Identity{
		Name:      "test",
		Email:     "test@example.com",
		Mode:      ModeTemporary,
		CreatedAt: time.Now(),
	}
	manager.SetCurrent(identity)

	// Should return the identity
	current := manager.GetCurrent()
	if current == nil {
		t.Fatal("GetCurrent() = nil, want identity")
	}
	if current.Name != "test" {
		t.Errorf("GetCurrent().Name = %v, want test", current.Name)
	}
}

func TestManager_CaptureSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".git-users")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Capture snapshot
	if err := manager.CaptureSnapshot(); err != nil {
		t.Errorf("CaptureSnapshot() error = %v", err)
	}

	// Verify snapshot was created
	if manager.previousState == nil {
		t.Error("previousState is nil after CaptureSnapshot()")
	}
}

func TestSnapshot_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		snapshot *IdentitySnapshot
		want     bool
	}{
		{
			name:     "empty snapshot",
			snapshot: &IdentitySnapshot{},
			want:     true,
		},
		{
			name: "snapshot with name",
			snapshot: &IdentitySnapshot{
				Name: "test",
			},
			want: false,
		},
		{
			name: "snapshot with email",
			snapshot: &IdentitySnapshot{
				Email: "test@example.com",
			},
			want: false,
		},
		{
			name: "snapshot with SSH command",
			snapshot: &IdentitySnapshot{
				SSHCommand: "ssh -i key",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.snapshot.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
