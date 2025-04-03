# LDAPEnforcer

`ldapenforcer` declaratively manages users and groups in an LDAP server.

It keeps the definitions of users, groups, and group membership
in plain text files that can be committed to git.
The synchronization process can be run repeatedly to no ill effect ---
unlike applying LDIFs, which will only work once for some operations like add or delete.

Currently it assumes it's talking to an instance of
[389 Directory Server](https://www.port389.org/)
with the [MemberOf plugin](https://www.port389.org/docs/389ds/design/memberof-plugin.html) enabled.

* [Announcement blog post](https://me.micahrl.com/blog/ldapenforcer-alpha/)
* [Full documentation](https://pages.micahrl.com/ldapenforcer)

## Installation

* Download from [GitHub Releases](https://github.com/mrled/ldapenforcer/releases)
* Use the [Docker image](https://github.com/mrled/ldapenforcer/pkgs/container/ldapenforcer)

## Usage

Continuously synchronize users and groups from a config file:

```sh
ldapenforcer sync --poll --config /etc/ldapenforcer.toml
```

Example config file:

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

See complete documentation and examples at
<https://pages.micahrl.com/ldapenforcer>.

## Development

Build from source:

```bash
git clone https://github.com/mrled/ldapenforcer.git
cd ldapenforcer
go build -o ldapenforcer ./cmd/ldapenforcer
```

External requirements:

* `golangci-lint`: for linting
* `goreleaser` for making releases
* `hugo` for the documentation site

### Building the documentation site

* `go run ./cmd/docgen -m site/content/docs/command`
* `cd ./site && hugo`

### Creating a new release

```sh
version=$(go run ./cmd/ldapenforcer version -r); git tag v"$version" && git push origin master v"$version"
```

We use goreleaser in GitHub actions.
To run it locally for testing:

```sh
brew install goreleaser/tap/goreleaser

# Build like the master branch
goreleaser build --snapshot --clean

# Build like the full release
goreleaser release --snapshot --clean
```
