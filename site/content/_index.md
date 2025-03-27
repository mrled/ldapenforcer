+++
title = "LDAPEnforcer"
+++

ldapenforcer keeps the definitions of users (people and service accounts),
groups, and group membership in plain text files that can be committed to git.
The synchronization process can be run repeatedly to no ill effect ---
unlike applying LDIFs, which will only work once for some operations like add or delete.

## Example

```sh
ldapenforcer sync --poll --config /etc/ldapenforcer.toml
```

This will run indefinitely, polling the config file(s) for changes every 10s,
and checking the LDAP server every 24h even if no config changes happen in that period
to ensure that it hasn't drifted from the enforced configuration.

`ldapenforcer.toml`:

```toml
[ldapenforcer]
uri = "ldap://localhost:389"
bind_dn = "cn=Directory Manager"
password = "P@ssw0rd"

# Directory Structure
enforced_people_ou = "ou=enforced,ou=people,dc=micahrl,dc=me"
enforced_svcacct_ou = "ou=enforced,ou=services,dc=micahrl,dc=me"
enforced_group_ou = "ou=enforced,ou=groups,dc=micahrl,dc=me"

# Service accounts (programmatic users)
[ldapenforcer.svcacct.authenticator]
cn = "Authenticator"
description = "A service account for authenticating users"

# People (human users)
[ldapenforcer.person.bobert]
cn = "Bob R Robert"
mail = "bobert@example.com"
posix = [20069, 20101]

# Groups
[ldapenforcer.group.bots]
description = "Bots here at ACME CORP"
posixGidNumber = 10100
svcaccts = ["authenticator"]

[ldapenforcer.group.employees]
description = "Regular user accounts here at ACME CORP"
posixGidNumber = 10200
people = ["bobert"]

[ldapenforcer.group.everyone]
description = "People and bots together in one big happy ACME CORP family"
groups = ["employees", "bots"]
```

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
