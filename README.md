# LDAPEnforcer

A command line tool for managing and enforcing policies on LDAP directories.

## Installation

```bash
go install github.com/mrled/ldapenforcer@latest
```

Or build from source:

```bash
git clone https://github.com/mrled/ldapenforcer.git
cd ldapenforcer
go build
```

## Usage

```bash
# Show version
ldapenforcer version

# Show current configuration
ldapenforcer config show --config=config.toml

# Validate configuration
ldapenforcer config validate --config=config.toml

# View configured entities
ldapenforcer config list-people --config=config.toml
ldapenforcer config list-svcaccts --config=config.toml
ldapenforcer config list-groups --config=config.toml
ldapenforcer config show-group groupname --config=config.toml

# Verify LDAP state against configuration
ldapenforcer verify --config=config.toml
ldapenforcer verify verify-person uid --config=config.toml

# Synchronize LDAP with configuration
ldapenforcer sync --config=config.toml
ldapenforcer sync --config=config.toml --dry-run  # Preview changes without applying

# Synchronize specific entities
ldapenforcer sync sync-person uid --config=config.toml
ldapenforcer sync sync-svcacct uid --config=config.toml
ldapenforcer sync sync-group groupname --config=config.toml
```

## Configuration

LDAPEnforcer uses TOML configuration files. A sample configuration file is provided at `config.sample.toml`, and a more comprehensive example is at `examples/full-config.toml`.

You can specify the configuration file using the `--config` flag. Configuration options can also be provided via command-line flags:

```bash
ldapenforcer --ldap-uri="ldap://example.com:389" --bind-dn="cn=admin,dc=example,dc=com" --password="secret"
```

### Configuration File Format

The configuration file uses the following format:

```toml
[ldapenforcer]
# LDAP Connection Settings
uri = "ldap://example.com:389"
bind_dn = "cn=admin,dc=example,dc=com"
password = "admin_password"
# OR
# password_file = "/path/to/password/file.txt"

# Directory Structure
people_base_dn = "ou=people,dc=example,dc=com"
svcacct_base_dn = "ou=svcaccts,dc=example,dc=com"
group_base_dn = "ou=groups,dc=example,dc=com"
managed_ou = "managed"

# Include other config files
includes = [
  "additional-config.toml",
  "/absolute/path/to/config.toml"
]

# Person definitions
[ldapenforcer.person.uid]
cn = "Person's Full Name"
givenName = "Person's First Name"  # optional
sn = "Person's Last Name"  # optional, derived from CN if not provided
mail = "person@example.com"  # optional
posix = [1000, 1000]  # [UID number, GID number], optional

# Service account definitions
[ldapenforcer.svcacct.uid]
cn = "Service Name"
description = "Service description"  # required
mail = "service@example.com"  # optional
posix = [1000, 1000]  # [UID number, GID number], optional

# Group definitions
[ldapenforcer.group.groupname]
description = "Group description"  # required
posixGidNumber = 1000  # optional
people = ["uid1", "uid2"]  # list of people in this group
svcaccts = ["uid1", "uid2"]  # list of service accounts in this group
groups = ["group1", "group2"]  # list of groups whose members should be included
```

The `includes` option allows you to include other configuration files. Relative paths are resolved relative to the parent configuration file's directory.

### Person Configuration

People are defined under the `[ldapenforcer.person.<uid>]` section:

- `cn`: Common Name (full name)
- `givenName`: First name (optional)
- `sn`: Surname/Last name (optional, derived from CN if not provided)
- `mail`: Email address (optional)
- `posix`: POSIX attributes as `[UID number, GID number]` (optional)

If `posix` is provided, the person will be created with the `posixAccount` objectClass.

### Service Account Configuration

Service accounts are defined under the `[ldapenforcer.svcacct.<uid>]` section:

- `cn`: Common Name
- `description`: Description (required)
- `mail`: Email address (optional)
- `posix`: POSIX attributes as `[UID number, GID number]` (optional)

If `posix` is provided, the service account will be created with the `posixAccount` objectClass. Both UID and GID numbers are required for POSIX accounts.

### Group Configuration

Groups are defined under the `[ldapenforcer.group.<groupname>]` section:

- `description`: Description (required)
- `posixGidNumber`: POSIX GID number (optional)
- `people`: List of people UIDs in this group
- `svcaccts`: List of service account UIDs in this group
- `groups`: List of groups whose members should be included

If a group is referenced in another group's `groups` list, only the members of the referenced group are included, not the group itself. This allows for nested groups while avoiding cycles.

Note: The term "user" refers collectively to people and service accounts when discussing both types of entities.

## LDAP Synchronization

LDAPEnforcer can synchronize the directory with your configuration, creating or updating LDAP entries to match your defined entities.

### How Synchronization Works

1. **Organizational Units**: First, LDAPEnforcer ensures that the required organizational units (OUs) exist in the LDAP directory.
2. **People**: Creates or updates person entries in LDAP with the configured attributes.
3. **Service Accounts**: Creates or updates service account entries in LDAP with the configured attributes.
4. **Groups**: Creates or updates group entries in LDAP, including appropriate members and nested group memberships.

### Dry Run Mode

You can perform a dry run to see what changes would be made without actually modifying the LDAP directory:

```bash
ldapenforcer sync --dry-run --config=config.toml
```

### Verifying LDAP State

Before synchronizing, you can verify the current state of your LDAP directory against your configuration:

```bash
ldapenforcer verify --config=config.toml
```

### Selective Synchronization

You can synchronize specific entities instead of the entire directory:

```bash
# Synchronize a specific person
ldapenforcer sync sync-person uid --config=config.toml

# Synchronize a specific service account
ldapenforcer sync sync-svcacct uid --config=config.toml

# Synchronize a specific group
ldapenforcer sync sync-group groupname --config=config.toml
```

## License

[MIT](LICENSE)