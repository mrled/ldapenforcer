+++
title = "LDAPEnforcer"
+++

`ldapenforcer` idempotently manages users and groups in an LDAP server.

It keeps the definitions of users, groups, and group membership
in plain text files that can be committed to git.
The synchronization process can be run repeatedly to no ill effect ---
unlike applying LDIFs, which will only work once for some operations like add or delete.

## Open source

The source code is available [on GitHub](https://github.com/mrled/ldapenforcer).

## Install

* [Binaries](https://github.com/mrled/ldapenforcer/releases)
* [Container images](https://github.com/mrled/ldapenforcer/pkgs/container/ldapenforcer)

## Example

```sh
ldapenforcer sync --poll --config /etc/ldapenforcer.toml
```

This will run indefinitely, polling the config file(s) for changes every 10s,
and checking the LDAP server every 24h even if no config changes happen in that period
to ensure that it hasn't drifted from the enforced configuration.

Here's a sample `ldapenforcer.toml`:

```toml
[ldapenforcer]
uri = "ldap://localhost:389"
bind_dn = "cn=Directory Manager"
password = "P@ssw0rd"

# Directory Structure
enforced_people_ou = "ou=enforced,ou=people,dc=micahrl,dc=me"
enforced_svcacct_ou = "ou=enforced,ou=services,dc=micahrl,dc=me"
enforced_group_ou = "ou=enforced,ou=groups,dc=micahrl,dc=me"

[ldapenforcer.person.bobert]
cn = "Bob R Robert"
mail = "bobert@example.com"
posix = [20069, 20101]

[ldapenforcer.group.employees]
description = "Regular user accounts here at ACME CORP"
posixGidNumber = 10200
people = ["bobert"]
```

Now if the config file changes,
`ldapenforcer` will notice the change and automatically apply it to the LDAP server.
By default, it polls the config file every 10s,
because polling a file's modification time is cheap.

But it will also enforce the configuration in case the LDAP server changes.
If an administrator manually gives different membership to a user,
`ldapenforcer` will notice and revert it.
By default, it polls the LDAP server every 24h,
because reading all managed users from the LDAP server can be expensive on large servers.

See [Configuration file]({{< ref "docs/configuration/file" >}})
for a complete list of configuration options.

## Enforced OUs

LDAPEnforcer will only modify objects in its **enforced OUs**.

The names of the enforced OUs are set in the configuration.
From the example above:

```toml
enforced_people_ou = "ou=enforced,ou=people,dc=micahrl,dc=me"
enforced_svcacct_ou = "ou=enforced,ou=services,dc=micahrl,dc=me"
enforced_group_ou = "ou=enforced,ou=groups,dc=micahrl,dc=me"
```

The LDAP server may have users and groups in other OUs,
for instance some default users might exist in `ou=people,dc=micahrl,dc=me`,
or perhaps there another OU entirely in `ou=migrated,dc=micahrl,dc=me=`.
LDAPEnforcer will never modify objects outside of the enforced OUs.

Depending on your needs, you may be able to enforce your entire directory,
or you may want to only enforce part of it.

## Synchronization

```sh
# Run synchronization once and then exit
ldapenforcer sync

# Run synchronization continually, polling the config file and LDAP server for changes
ldapenforcer sync --poll

# Show what would be synced without modifying the LDAP directory
ldapenforcer sync --dry-run

# Verify whether the LDAP server matches the configuration
ldapenforcer verify
```

### Synchronization steps

1. **Organizational Units**: First, LDAPEnforcer ensures that the required organizational units (OUs) exist in the LDAP directory.
2. **People**: Creates or updates person entries in LDAP with the configured attributes.
3. **Service Accounts**: Creates or updates service account entries in LDAP with the configured attributes.
4. **Groups**: Creates or updates group entries in LDAP, including appropriate members and nested group memberships.

## Limitations

It's not designed to "move" objects between OUs; it will simply delete and recreate them.
If unmanaged attributes are set on objects in the LDAP directory,
including passwords, profile images, physical addresses, phone numbers, or any other attribute,
moving them in this way will destroy them.
It's designed to complement other user/group management tools,
and it isn't intended to make an LDAP server fully stateless.

Currently it assumes it's talking to an instance of
[389 Directory Server](https://www.port389.org/)
with the [MemberOf plugin](https://www.port389.org/docs/389ds/design/memberof-plugin.html) enabled.
