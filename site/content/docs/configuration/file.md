+++
title = "Configuration file"
+++

## Complete commented configuration file

A complete configuration file example

```toml
[ldapenforcer]
# LDAP Connection Settings
# Standard LDAP connection
uri = "ldap://example.com:389"
# For secure LDAPS connection, use:
# uri = "ldaps://example.com:636"

bind_dn = "cn=admin,dc=example,dc=com"
password = "admin_password"
# OR
# password_file = "/path/to/password/file.txt"  # Absolute path
# password_file = "passwords/ldap.txt"  # Relative to config file
# OR
# password_command = "pass show ldap/admin"  # Execute command to get password
# password_command = "security find-generic-password -s 'LDAP Admin' -w"  # macOS keychain example
# password_command = "secret-tool lookup service ldap user admin"  # GNOME keyring example
# password_command = "kubectl get secret -n directory dirsrv-env -o go-template='{{ index .data \"DS_DM_PASSWORD\" | base64decode}}'"
#
# For commands that need shell features (|, <, >, $, ``, etc.), set password_command_via_shell = true
# Example with kubectl and go-templates (requires password_command_via_shell = true):
# password_command = "somehow-get-password | grep 'password:' | awk '{print $2}'"
# password_command_via_shell = true


# Path to CA certificate file for LDAPS connections (only needed for LDAPS)
# ca_cert_file = "/path/to/ca.crt"  # Absolute path
# ca_cert_file = "certs/ca.crt"  # Relative to config file

# NOTE: All configuration settings can also be set via environment variables.
# Environment variable names are in the format LDAPENFORCER_<SETTING_NAME>,
# for example:
#   - LDAPENFORCER_URI
#   - LDAPENFORCER_BIND_DN
#   - LDAPENFORCER_PASSWORD
#   - LDAPENFORCER_LOG_LEVEL
#   - LDAPENFORCER_LDAP_LOG_LEVEL
#   - LDAPENFORCER_PEOPLE_BASE_DN
# Environment variables take precedence over config file settings.

# Logging configuration
# Main application log level
# Valid values: ERROR, WARN, INFO, DEBUG, TRACE (case-insensitive)
main_log_level = "ERROR"

# LDAP-specific log level - control LDAP protocol logging independently
# Use DEBUG or TRACE level for detailed protocol logging
ldap_log_level = "ERROR"

# Directory Structure
enforced_people_ou = "ou=enforced,ou=people,dc=example,dc=com"
enforced_svcacct_ou = "ou=enforced,ou=svcaccts,dc=example,dc=com"
enforced_group_ou = "ou=enforced,ou=groups,dc=example,dc=com"

# Include files - paths are relative to this config file's directory
# unless they are absolute paths
includes = [
    # "additional-config.toml",
    # "/absolute/path/to/config.toml"
]

# Person definitions
[ldapenforcer.person.micahrl]
cn = "Micah R Ledbetter"
givenName = "Micah"
sn = "Ledbetter"
mail = "me@micahrl.com"
posix = [10069, 10101]   # UID number, GID number

[ldapenforcer.person.jdoe]
cn = "John Doe"
mail = "jdoe@example.com"
posix = [10070, 10102]

# Service account definitions
[ldapenforcer.svcacct.authenticator]
cn = "Authenticator"
description = "A service account for authenticating users"

[ldapenforcer.svcacct.backups]
cn = "Backup Service"
description = "A service account for performing backups"
posix = [10200, 10200] # UID number, GID number (both required for POSIX)

# Group definitions
[ldapenforcer.group.admins]
description = "Administrative users"
posixGidNumber = 10100
people = ["micahrl"]
svcaccts = ["authenticator"]
groups = []

[ldapenforcer.group.users]
description = "Regular users"
posixGidNumber = 10101
people = ["jdoe"]
svcaccts = []
groups = []

[ldapenforcer.group.all]
description = "All users and services"
people = []
svcaccts = []
groups = ["admins", "users", "cycle1"] # Nested groups - members are included

[ldapenforcer.group.cycle1]
description = "Cyclic group 1"
people = []
svcaccts = []
groups = ["cycle2"]

[ldapenforcer.group.cycle2]
description = "Cyclic group 2"
people = []
svcaccts = []
groups = ["cycle1"]
```

## LDAP objects configuration

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

**Empty groups are not permitted by the `groupOfNames` object class**.
If you define an enforced group which has no members at all in the configuration,
it will not be created in the directory.
If all members are removed from an enforced group in the configuration,
it will be deleted from the directory.
