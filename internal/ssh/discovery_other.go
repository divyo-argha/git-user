//go:build !windows

package ssh

// EnsureSSHBinariesOnPath is a no-op outside Windows, where ssh-keygen/
// ssh-add/ssh are near-universally already on PATH. See discovery_windows.go
// for why Windows needs this.
func EnsureSSHBinariesOnPath() {}
