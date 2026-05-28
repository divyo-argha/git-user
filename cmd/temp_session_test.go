package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseTempSessionArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantName  string
		wantEmail string
		wantTTL   string
		wantErr   bool
	}{
		{
			name:      "name and email",
			args:      []string{"alice", "alice@example.com"},
			wantName:  "alice",
			wantEmail: "alice@example.com",
		},
		{
			name:      "name email and ttl",
			args:      []string{"alice", "alice@example.com", "--ttl", "4h"},
			wantName:  "alice",
			wantEmail: "alice@example.com",
			wantTTL:   "4h",
		},
		{
			name:      "ttl before name and email",
			args:      []string{"--ttl", "1h", "bob", "bob@example.com"},
			wantName:  "bob",
			wantEmail: "bob@example.com",
			wantTTL:   "1h",
		},
		{name: "missing email", args: []string{"alice"}, wantErr: true},
		{name: "empty args", args: nil, wantErr: true},
		{name: "missing ttl value", args: []string{"alice", "a@b.com", "--ttl"}, wantErr: true},
		{name: "unknown flag", args: []string{"alice", "a@b.com", "--foo"}, wantErr: true},
		{name: "extra arg", args: []string{"alice", "a@b.com", "extra"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotEmail, gotTTL, err := parseTempSessionArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseTempSessionArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotName != tt.wantName {
				t.Errorf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotEmail != tt.wantEmail {
				t.Errorf("email = %q, want %q", gotEmail, tt.wantEmail)
			}
			if gotTTL != tt.wantTTL {
				t.Errorf("ttl = %q, want %q", gotTTL, tt.wantTTL)
			}
		})
	}
}

func TestTempSessionPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	ts := &tempSession{
		Name:       "alice",
		Email:      "alice@example.com",
		KeyPath:    filepath.Join(tmpDir, ".ssh", "git_tmp_alice"),
		PrevName:   "bob",
		PrevEmail:  "bob@example.com",
		PrevSSHKey: "",
	}

	if err := saveTempSession(ts); err != nil {
		t.Fatalf("saveTempSession: %v", err)
	}

	info, err := os.Stat(tempSessionPath())
	if err != nil {
		t.Fatalf("temp session file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("temp session file permissions = %o, want 0600", info.Mode().Perm())
	}

	loaded, err := loadTempSession()
	if err != nil {
		t.Fatalf("loadTempSession: %v", err)
	}
	if loaded == nil {
		t.Fatal("loadTempSession returned nil")
	}
	if loaded.Name != ts.Name || loaded.Email != ts.Email {
		t.Errorf("loaded = %+v, want %+v", loaded, ts)
	}

	removeTempSessionFile()

	loaded2, err := loadTempSession()
	if err != nil {
		t.Fatalf("loadTempSession after remove: %v", err)
	}
	if loaded2 != nil {
		t.Error("expected nil after remove")
	}
}

func TestLoadTempSessionCorrupt(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	path := tempSessionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadTempSession()
	if err == nil {
		t.Error("expected error for corrupt JSON")
	}
}

func TestTempSessionJSONFields(t *testing.T) {
	ts := &tempSession{
		Name:       "test",
		Email:      "test@example.com",
		KeyPath:    "/tmp/key",
		PrevName:   "prev",
		PrevEmail:  "prev@example.com",
		PrevSSHKey: "ssh -i /tmp/prev",
	}
	data, err := json.Marshal(ts)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"name", "email", "key_path", "prev_name", "prev_email", "prev_ssh_key"} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing JSON field %q", key)
		}
	}
}
