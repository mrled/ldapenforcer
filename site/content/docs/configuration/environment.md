+++
title = "Environment variables"
+++

All configuration settings in the `[ldapenforcer]` section can also be set via environment variables using the following format:

```
LDAPENFORCER_<SETTING_NAME>
```

Where `<SETTING_NAME>` is the uppercase name of the configuration setting with underscores. For example:

- `LDAPENFORCER_URI` for the LDAP URI
- `LDAPENFORCER_BIND_DN` for the bind DN
- `LDAPENFORCER_PASSWORD` for the password
- `LDAPENFORCER_PASSWORD_FILE` for the password file path
- `LDAPENFORCER_CA_CERT_FILE` for the CA certificate file
- `LDAPENFORCER_LOG_LEVEL` for the main log level
- `LDAPENFORCER_LDAP_LOG_LEVEL` for the LDAP-specific log level
- `LDAPENFORCER_PEOPLE_BASE_DN` for the people base DN
- `LDAPENFORCER_SVCACCT_BASE_DN` for the service accounts base DN
- `LDAPENFORCER_GROUP_BASE_DN` for the groups base DN
- `LDAPENFORCER_MANAGED_OU` for the managed OU name

For boolean settings like `password_command_via_shell`, the value should be a valid boolean string:
- `LDAPENFORCER_PASSWORD_COMMAND_VIA_SHELL="true"` for true
- `LDAPENFORCER_PASSWORD_COMMAND_VIA_SHELL="false"` for false

For the `includes` setting, the value should be a comma-separated list:
- `LDAPENFORCER_INCLUDES="file1.toml,file2.toml"` or `LDAPENFORCER_INCLUDES="file1.toml, file2.toml"`

Note that users and groups must be configured in TOML config files.
