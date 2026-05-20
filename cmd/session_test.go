package cmd

import (
	"testing"

	"github.com/divyo-argha/git-user/internal/config"
)

func TestParseSessionStartArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantID  string
		wantTTL string
		wantErr bool
	}{
		{name: "empty uses current identity", args: nil},
		{name: "named identity", args: []string{"work"}, wantID: "work"},
		{name: "ttl only", args: []string{"--ttl", "4h"}, wantTTL: "4h"},
		{name: "name and ttl", args: []string{"work", "--ttl", "1h"}, wantID: "work", wantTTL: "1h"},
		{name: "short ttl", args: []string{"-t", "30m", "personal"}, wantID: "personal", wantTTL: "30m"},
		{name: "missing ttl value", args: []string{"--ttl"}, wantErr: true},
		{name: "unknown option", args: []string{"--forever"}, wantErr: true},
		{name: "too many names", args: []string{"work", "personal"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotTTL, err := parseSessionStartArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseSessionStartArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotID != tt.wantID {
				t.Fatalf("identity = %q, want %q", gotID, tt.wantID)
			}
			if gotTTL != tt.wantTTL {
				t.Fatalf("ttl = %q, want %q", gotTTL, tt.wantTTL)
			}
		})
	}
}

func TestParseSessionStopArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantID  string
		wantAll bool
		wantErr bool
	}{
		{name: "empty uses current identity", args: nil},
		{name: "named identity", args: []string{"work"}, wantID: "work"},
		{name: "all keys explicit", args: []string{"--all"}, wantAll: true},
		{name: "all and identity conflict", args: []string{"work", "--all"}, wantErr: true},
		{name: "unknown option", args: []string{"--ttl"}, wantErr: true},
		{name: "too many names", args: []string{"work", "personal"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotAll, err := parseSessionStopArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseSessionStopArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotID != tt.wantID {
				t.Fatalf("identity = %q, want %q", gotID, tt.wantID)
			}
			if gotAll != tt.wantAll {
				t.Fatalf("all = %v, want %v", gotAll, tt.wantAll)
			}
		})
	}
}

func TestSelectedSessionUser(t *testing.T) {
	store := &config.Store{}
	if err := store.AddUser("work", "work@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddUser("personal", "me@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := store.SetCurrent("work"); err != nil {
		t.Fatal(err)
	}

	if got := selectedSessionUser(store, ""); got == nil || got.Name != "work" {
		t.Fatalf("current user = %#v, want work", got)
	}
	if got := selectedSessionUser(store, "personal"); got == nil || got.Name != "personal" {
		t.Fatalf("named user = %#v, want personal", got)
	}
	if got := selectedSessionUser(store, "missing"); got != nil {
		t.Fatalf("missing user = %#v, want nil", got)
	}
}

func TestParseSSHKeyFingerprint(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    string
		wantErr bool
	}{
		{
			name: "ssh keygen output",
			line: "256 SHA256:abc123 user@example.com (ED25519)",
			want: "SHA256:abc123",
		},
		{
			name: "ssh add output",
			line: "4096 SHA256:def456 /home/user/.ssh/id_rsa (RSA)",
			want: "SHA256:def456",
		},
		{
			name:    "empty line",
			line:    "",
			wantErr: true,
		},
		{
			name:    "missing fingerprint",
			line:    "256",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSSHKeyFingerprint(tt.line)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseSSHKeyFingerprint() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("fingerprint = %q, want %q", got, tt.want)
			}
		})
	}
}
