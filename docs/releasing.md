# Releasing a new version

```sh
version=$(go run ./... version -r); git tag v"$version" && git push origin master v"$version"
```

## goreleaser

We use goreelaser in GitHub actions.
To run it locally for testing:

```sh
brew install goreleaser/tap/goreleaser

# Build like the master branch
goreleaser build --snapshot --clean

# Build like the full release
goreleaser release --snapshot --clean
```
