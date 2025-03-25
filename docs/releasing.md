# Releasing a new version

```sh
version=$(go run ./... version -r); git tag "$version" && git push origin "$version"
```
