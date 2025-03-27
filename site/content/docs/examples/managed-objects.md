+++
title = "Managed objects"
+++

Example config file entries and LDIFs that represent the same thing.

User config file example:

```toml
[[user.micahrl]]
cn = "Micah R Ledbetter"
givenName = "Micah" # optional
sn = "Ledbetter" # optional
mail = "me@micahrl.com" # optional
posix = [10069, 10101] # UID number, GID number; optional, indicates POSIX
```

User LDIF example:

```ldif
dn: uid=micahrl,ou=enforced,ou=people,dc=micahrl,dc=me
objectClass: inetOrgPerson # required
objectClass: posixAccount # only if posix
# objectClass: account # only if not posix
objectClass: top # required
uid: micahrl # required
cn: Micah R Ledbetter
givenName: Micah # only if set
sn: Ledbetter # if not set in the config, use the uid
mail: me@micahrl.com # only if set
uidNumber: 10069 # only if posix
gidNumber: 10101 # only if posix
homeDirectory: /home/micahrl # created from /home/{username}, only if posix
gecos: Micah R Ledbetter # set from cn, only if posix
```

Service account config file example:

```toml
[[svcacct.authenticator]]
cn = "Authenticator"
description = "A service account for authenticating users" # required
#mail = "" # optional
#posix = [] # GID number; optional, indicates POSIX
```

Service account LDIF example:

```ldif
dn: uid=authenticator,ou=enforced,ou=svcacct,dc=micahrl,dc=me
objectClass: inetOrgPerson
# objectClass: posixAccount # only if posix
objectClass: account # only if not posix
objectClass: top
uid: authenticator
cn: Authenticator
sn: authrenticator # use the uid
description: A service account for authenticating users
```

Group config file example:

```toml
[[group.patricii]]
description = "Accounts with administrative privileges" # required
posixGidNumber = 10100 # only if posix
users = ["mrladmin"] # list of users
services = [""] # list of service accounst
groups = [""] # list of groups
```

Group LDIF example:

```ldif
dn: cn=patricii,ou=enforced,ou=groups,dc=micahrl,dc=me
objectClass: groupOfNames
objectClass: posixGroup # only if posix
objectClass: top
cn: patricii
description: Accounts with administrative privileges
gidNumber: 10100 # only if posix
```

LDAP group membership LDIF example:

```ldif
# for every membership entry
# applicable to both users and services
dn: cn=datadump-admin,ou=enforced,ou=groups,dc=micahrl,dc=me
changetype: modify
add: member
member: uid=mrladmin,ou=enforced,ou=people,dc=micahrl,dc=me
```

POSIX group membership LDIF example (always assume POSIX membership implies LDAP group membership too):

```ldif
# only if both group and user are posix
# applicable to both users and services
dn: cn=patricii,ou=enforced,ou=groups,dc=micahrl,dc=me
changetype: modify
add: member
member: uid=mrladmin,ou=enforced,ou=people,dc=micahrl,dc=me
-
add: memberUid
memberUid: mrladmin
```
