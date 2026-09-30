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

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		remote, current string
		want            bool
	}{
		{"v4.18.8", "v4.18.7", true},
		{"v4.19.0", "v4.18.9", true},
		{"v5.0.0", "v4.99.99", true},
		{"v4.18.7", "v4.18.7", false},
		{"v4.18.6", "v4.18.7", false},
		{"4.18.8", "v4.18.7", true}, // npm tags have no "v"
		{"v4.19.0", "v4.19.0-rc.1", true},
		{"v4.19.0-rc.1", "v4.19.0", false},
		{"v4.19.0-rc.2", "v4.19.0-rc.1", true},
		{"v4.19.0-rc.10", "v4.19.0-rc.9", true},
		{"v4.19.0-rc.1", "v4.19.0-beta.5", true},
		{"v4.19.0-rc.1", "v4.19.0-rc.1", false},
		{"v4.19.0-rc.1+build5", "v4.19.0-rc.1", false},
		{"v4.19.0-rc.1.1", "v4.19.0-rc.1", true},
		{"v4.19.0+meta", "v4.19.0", false},
		{"garbage", "v4.18.7", false},
		{"v4.18.7", "garbage", false},
		{"", "v4.18.7", false},
		{"v4.x.1", "v4.0.0", false},
		{"v4.18", "v4.17.9", true},
		{"v4.18.7.9", "v4.18.7", false},
	}
	for _, tt := range tests {
		if got := IsNewerVersion(tt.remote, tt.current); got != tt.want {
			t.Errorf("IsNewerVersion(%q, %q) = %v, want %v", tt.remote, tt.current, got, tt.want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in              string
		major, min, pat int
	}{
		{"v1.2.3", 1, 2, 3},
		{"V1.2.3", 1, 2, 3},
		{"1.2.3-rc.1+b", 1, 2, 3},
		{"1.2", 1, 2, 0},
		{"1", 1, 0, 0},
		{"garbage", 0, 0, 0},
		{"1.x.3", 0, 0, 0},
		{"", 0, 0, 0},
	}
	for _, tt := range tests {
		a, b, c := ParseVersion(tt.in)
		if a != tt.major || b != tt.min || c != tt.pat {
			t.Errorf("ParseVersion(%q) = %d.%d.%d, want %d.%d.%d", tt.in, a, b, c, tt.major, tt.min, tt.pat)
		}
	}
}

func TestIsValid(t *testing.T) {
	for in, want := range map[string]bool{"v1.2.3": true, "1.2.3-rc.1": true, "garbage": false, "": false, "v": false, "v-1.0.0": false} {
		if got := IsValid(in); got != want {
			t.Errorf("IsValid(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestSetVersion_Normalizes(t *testing.T) {
	origVer, origBuild := Version, BuildVersion
	t.Cleanup(func() { Version, BuildVersion = origVer, origBuild })

	SetVersion(" 4.20.0 ")
	if got := GetVersion(); got != "v4.20.0" {
		t.Errorf("GetVersion() = %q, want v4.20.0", got)
	}
	SetVersion("vv4.21.0")
	if got := GetVersion(); got != "v4.21.0" {
		t.Errorf("GetVersion() = %q, want v4.21.0", got)
	}
}

func TestSetVersion_Concurrent(t *testing.T) {
	origVer, origBuild := Version, BuildVersion
	t.Cleanup(func() { Version, BuildVersion = origVer, origBuild })

	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			for j := 0; j < 200; j++ {
				SetVersion("v1.0.0")
				_ = GetVersion()
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}
