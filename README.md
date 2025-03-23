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
```

## Configuration

LDAPEnforcer uses TOML configuration files. A sample configuration file is provided at `config.sample.toml`.

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
user_base_dn = "ou=users,dc=example,dc=com"
service_base_dn = "ou=services,dc=example,dc=com"
group_base_dn = "ou=groups,dc=example,dc=com"
managed_ou = "managed"

# Include other config files
includes = [
  "additional-config.toml",
  "/absolute/path/to/config.toml"
]
```

The `includes` option allows you to include other configuration files. Relative paths are resolved relative to the parent configuration file's directory.

## License

[MIT](LICENSE)