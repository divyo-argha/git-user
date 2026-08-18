package version

import (
	"testing"
)

func TestVersion_NotEmpty(t *testing.T) {
	if Version == "" {
		t.Fatal("Version should not be empty")
	}
}

func TestVersion_Format(t *testing.T) {
	if Version[0] != 'v' {
		t.Errorf("Version should start with 'v', got %q", Version)
	}
}

func TestSetVersion(t *testing.T) {
	origVer := Version
	origBuild := BuildVersion
	t.Cleanup(func() {
		Version = origVer
		BuildVersion = origBuild
	})

	SetVersion("v9.9.9")
	if GetVersion() != "v9.9.9" {
		t.Errorf("GetVersion() = %q, want %q", GetVersion(), "v9.9.9")
	}

	SetVersion("")
	if GetVersion() != "v9.9.9" {
		t.Errorf("GetVersion() after empty SetVersion = %q, want %q", GetVersion(), "v9.9.9")
	}
}
