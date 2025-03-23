Write a Go program that manages users and groups in an LDAP server.
Program name: ldapenforcer

Users, groups, and memberships should be managed idempotently.
Use a TOML config file that specifies the LDAP URI and the DN to connect and make changes.
The config file should also support a reference to a separate password file for binding,
which should be read and the contents trimmed of whitespace.
Also settings for user base DN, service base DN, and group base DN.
Also setttings for a MANAGED_OU name, which is an OU inside each of the services, users, and groups DN that indicates a managed object.
specify config on command line, fall back to LDAPENFORCER_CONFIG environment variable, fall back to /etc/ldapenforcer.toml
Can include other config files by specifying a list of files as "includes". Where absolute paths are used directly and relative paths are considered relatiev to the config file's parent directory.

A managed object is fully controlled by this program.
If an object exists in a managed OU in the LDAP server but not in the configuration, the object should be deleted.
And if the object exists in the configuration but not the managed OU, it should be created.
If there are any differences between the object in the configuration and the LDAP server, the LDAP server should be modified to reflect the configuration.

if a group contains a list of other groups as members, all of the MEMBERS of all those child gruops should be added directly as members of the parent group. (A group object should not be set as a member of another group object in the LDAP server.)

if the LDAP server has a managed user or service as a member of a non-managed group, ignore it.
if the LDAP server has a non-managed user or service as a member of a managed group, ignore it.
You can tell whether an object is managed or not by whether its in the managedou under its base.

Handle cases where accounts or groups are changed from POSIX to non-POSIX or vice versa.

When defining membership, if both the user/service and group are POSIX, apply both LDAP and POSIX membership. Otherwise, assign only LDAP group membership.

Handle cyclical group references by assembling all group references first, and raising an exception for groups that are cyclical.
Do the same for config file inclusion that is cyclical.

For POSIX users, automatically assemble the homeDirectory attribute from /home/username.

load entire configuration and check for validity before doing any actions.
if any part of configuration is invalid, refuse to work.
add a command line argument to check config and exit with 0 if successful or 1 if not.

Use the most common way to handle logs on Go command line programs that allows more/less verbose output controlled by commandline paths.

Show the directory structure and all files of the whole project.

Include a mode controlled by a command-line argument to write out LDIFs to make all changes without applying them to the LDAP server.
When talking to the LDAP server, only use LDIFs from the output mode.

Write tests to parse config files and check for validity.
Write a test that creates LDIF files for creating users, services, and groups.
Write a test that creates LDIF files for memberships.
Don't write tests with mock LDAP server responses (we'll do that later).

Assume the use of the Member Of plugin on 389 Directory Server.

Config file is toml

	[ldapenforcer]
	uri = "ldaps://dirsrv:636"
	binddn = "cn=DirectoryManager"
	passfile = "/containeripc/topsecret/ds-dm-password"
	managedou = "enforced"
	people_base = "ou=people,dc=micahrl,dc=me"
	services_base = "ou=services,dc=micahrl,dc=me"
	groups_base = "ou=groups,dc=micahrl,dc=me"
	includes = ["/etc/ldapenforcer/people.toml", "./otherpeople.toml", ...]

User config file example:

	[[user.micahrl]]
	cn = "Micah R Ledbetter"
	givenName = "Micah" # optional
	sn = "Ledbetter" # optional
	mail = "me@micahrl.com" # optional
	posix = [10069, 10101] # optional

User example:

    dn: uid=micahrl,ou=enforced,ou=people,dc=micahrl,dc=me
    objectClass: inetOrgPerson # required
    objectClass: posixAccount # only if posix
    # objectClass: account # only if not posix
    objectClass: top # required
    uid: micahrl # required
    cn: Micah R Ledbetter
    givenName: Micah # only if set
    sn: Ledbetter # required; if not set in config, derive from last word in cn
    mail: me@micahrl.com # only if set
    uidNumber: 10069 # only if posix
    gidNumber: 10101 # only if posix
    homeDirectory: /home/micahrl # only if posix
    gecos: Micah R Ledbetter # set from cn

Service config file example:

	[[service.authenticator]]
	cn = "Authenticator"
	description = "A service account for authenticating users" # required
	#mail = "" # optional
	#posix = [] #optional

Service example:

    dn: uid=authenticator,ou=enforced,ou=services,dc=micahrl,dc=me
    objectClass: inetOrgPerson
    # objectClass: posixAccount # only if posix
    objectClass: account # only if not posix
    objectClass: top
    uid: authenticator
    cn: Authenticator
    sn: authrenticator # just repeat the uid
    description: A service account for authenticating users

Group config file example:

	[[group.patricii]]
	description = "Accounts with administrative privileges" # required
	posixGidNumber = 10100 # only if posix
	users = ["mrladmin"] # list of users
	services = [""] # list of service accounst
	groups = [""] # list of groups

Group example:

    dn: cn=patricii,ou=enforced,ou=groups,dc=micahrl,dc=me
    objectClass: groupOfNames
    objectClass: posixGroup # only if posix
    objectClass: top
    cn: patricii
    description: Accounts with administrative privileges
    gidNumber: 10100 # only if posix

LDAP group membership example:

	# for every membership entry
	# applicable to both users and services
    dn: cn=datadump-admin,ou=enforced,ou=groups,dc=micahrl,dc=me
    changetype: modify
    add: member
    member: uid=mrladmin,ou=enforced,ou=people,dc=micahrl,dc=me

POSIX group membership example:

	# only if both group and user are posix
	# applicable to both users and services
    dn: cn=patricii,ou=enforced,ou=groups,dc=micahrl,dc=me
    changetype: modify
    add: memberUid
    memberUid: mrladmin
