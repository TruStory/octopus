# 🐙
## TruStory Go Services

This is a Go monorepo for all non-TruChain related services for TruStory. It is based on, but not a fork of: https://github.com/flowerinthenight/golang-monorepo. It allows for multiple services to co-exist in a single repo while only building services that are updated. Pull in features from [golang-monorepo](https://github.com/flowerinthenight/golang-monorepo) as needed.

## Services

* [AWS S3 Uploader](./services/uploader/README.md)
* [Push Notification Service](./services/push/README.md)

## Installing

Requires Go 1.11+ since it uses Go modules for dependency management.

```sh
go mod vendor
```

### Linting

```sh
make check
```
