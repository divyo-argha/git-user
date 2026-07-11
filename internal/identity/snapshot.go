package identity

import "time"

// IdentitySnapshot captures state for restoration
type IdentitySnapshot struct {
	Name         string
	Email        string
	SSHCommand   string
	WasTemporary bool
	CapturedAt   time.Time
}

// IsEmpty returns true if the snapshot has no meaningful data
func (s *IdentitySnapshot) IsEmpty() bool {
	return s.Name == "" && s.Email == "" && s.SSHCommand == ""
}
