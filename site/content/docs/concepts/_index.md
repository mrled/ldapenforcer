+++
title = "Concepts"
weight = 10
+++

Currently, LDAPEnforcer expects to run against 389 Directory Server,
so all of the examples and concepts are discussed with a bias to that system.
Other LDAP servers may do things very differently.
(Support for other systems is desired, feel free to send a PR!)

## User accounts

In this documentation,
a `person` is a user account for a human being,
a `svcacct` (or "service account") is a user account for a program,
and a `user account` is a generic term for either.

## Username and POSIX UID

389 Directory Server uses the `uid` attribute and attribute type in the Distinguished Name as the **username**.
It uses the `uidNumber` attribute for the numeric POSIX user ID number.
Here's an example LDIF that shows these:

```ldif
dn: uid=testuser,ou=enforced,ou=people,dc=micahrl,dc=me
objectClass: inetOrgPerson
objectClass: posixAccount
objectClass: top
uid: testuser
uidNumber: 42069
gidNumber: 10101
```

* The LDAP UID is found in the dn (`uid=testuser`) and attribute list (`uid: testuser`)
* The POSIX UID number is found in the attribute ilst (`uidNumber: 42069`)

In LDAPEnforcer documentation, we use "username" to mean the LDAP UID,
and "POSIX UID" to mean the POSIX UID number.

## Directory layout

Many LDAP directories use separate OUs for services and people.
A simple layout might look like this:

People
:   `ou=people,dc=example,dc=com`

Service accounts
:   `ou=services,dc=example,dc=com`

Groups
:   `ou=groups,dc=example,dc=com`

Depending on your needs, you may not have this separation.
If your directory combines people and service accounts,
you can treat service accounts as people in the LDAPEnforcer configuration with no ill effect.

## Creating POSIX users and groups

User accounts and groups may be designated as POSIX accounts.
To do so, set a POSIX UID and primary GID on a user account,
or a GID on a group.

```toml
[ldapenforcer.person.bobert]
cn = "Bob R Robert"
mail = "bobert@example.com"
posix = [20069, 20101] # uid,gid pair

[ldapenforcer.group.employees]
description = "Regular user accounts here at ACME CORP"
posixGidNumber = 10200
people = ["bobert"]
```

## Nested groups

When a group defines another group as a member,
LDAPEnforcer adds all the *user accounts* in that group as members in the LDAP server.

Example: if the config files have:

* `research` with user members `carol` and `dave`
* `development` with user members `alice` and `bob`
* `rad` with group members `research` and `development`

... then LDAPEnforcer will make LDAP groups like this:

* `research` with user members `carol` and `dave`
* `development` with user members `alice` and `bob`
* `rad` with group members `carol`, `dave`, `alice`, and `bob`

The server won't know that `rad` is composed of `research` and `development`,
just that it contains all of their members.

Why?
Actually, there is no way to request all recursive group membership from an LDAP server;
clients who want to support nested groups must
recursively walk all groups that are members of other groups.
For this reason, support is spotty ---
SSSD can enable support,
but many apps that can autheticate to LDAP don't support it at all.
