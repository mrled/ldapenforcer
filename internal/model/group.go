package model

// Group represents an LDAP group
type Group struct {
	// Description (required)
	Description string `toml:"description"`

	// POSIX GID number (optional)
	// If set, indicates this is a POSIX group
	PosixGidNumber int `toml:"posixGidNumber,omitempty"`

	// List of people in this group
	People []string `toml:"people,omitempty"`

	// List of service accounts in this group
	SvcAccts []string `toml:"svcaccts,omitempty"`

	// List of groups whose members should be included in this group
	Groups []string `toml:"groups,omitempty"`
}

// IsPosix returns true if the group has a POSIX GID number
func (g *Group) IsPosix() bool {
	return g.PosixGidNumber > 0
}
