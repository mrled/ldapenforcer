package model

// Person represents a person in LDAP
type Person struct {
	// Common name (CN)
	CN string `toml:"cn"`

	// Given name (optional)
	GivenName string `toml:"givenName,omitempty"`

	// Surname (optional)
	SN string `toml:"sn,omitempty"`

	// Email address (optional)
	Mail string `toml:"mail,omitempty"`

	// POSIX attributes: UID number, GID number (optional)
	// If set, indicates this is a POSIX person
	Posix []int `toml:"posix,omitempty"`
}

// IsPosix returns true if the person has POSIX attributes
func (p *Person) IsPosix() bool {
	return len(p.Posix) == 2
}

// GetUIDNumber returns the POSIX UID number, or 0 if not set
func (p *Person) GetUIDNumber() int {
	if p.IsPosix() {
		return p.Posix[0]
	}
	return 0
}

// GetGIDNumber returns the POSIX GID number, or 0 if not set
func (p *Person) GetGIDNumber() int {
	if p.IsPosix() {
		return p.Posix[1]
	}
	return 0
}

// GetSN returns the surname, calculating it from CN if not set
func (p *Person) GetSN() string {
	if p.SN != "" {
		return p.SN
	}
	
	// If SN is not set, use the last word in CN
	words := splitWords(p.CN)
	if len(words) > 0 {
		return words[len(words)-1]
	}
	return ""
}