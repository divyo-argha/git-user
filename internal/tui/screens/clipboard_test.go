package screens

import (
	"testing"
)

func TestClipboardWrite(t *testing.T) {
	// Must not panic on write attempts
	_ = ClipboardWrite("git-user test clipboard text")
}
