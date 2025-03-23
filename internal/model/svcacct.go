package model

// SvcAcct represents an LDAP service account
type SvcAcct struct {
	// Common name (CN)
	CN string `toml:"cn"`

	// Description (required)
	Description string `toml:"description"`

	// Email address (optional)
	Mail string `toml:"mail,omitempty"`

	// POSIX attributes: UID number, GID number (optional)
	// If set, indicates this is a POSIX account
	Posix []int `toml:"posix,omitempty"`
}

// IsPosix returns true if the service account has POSIX attributes
func (s *SvcAcct) IsPosix() bool {
	return len(s.Posix) == 2
}

// GetUIDNumber returns the POSIX UID number, or 0 if not set
func (s *SvcAcct) GetUIDNumber() int {
	if s.IsPosix() {
		return s.Posix[0]
	}
	return 0
}

// GetGIDNumber returns the POSIX GID number, or 0 if not set
func (s *SvcAcct) GetGIDNumber() int {
	if s.IsPosix() {
		return s.Posix[1]
	}
	return 0
}