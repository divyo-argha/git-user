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
