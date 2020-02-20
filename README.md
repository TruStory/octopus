# 🐙

## TruStory Go Monorepo

This is a Go monorepo for all non-TruChain related services for TruStory. It is based on, but not a fork of: https://github.com/flowerinthenight/golang-monorepo. It allows for multiple services to co-exist in a single repo while only building services that are updated. Pull in features from [golang-monorepo](https://github.com/flowerinthenight/golang-monorepo) as needed.

### Actions

- Metrics: runs metrics on user data

### Services

- [TruAPI Light Client](./services/truapi/README.md)
- [AWS S3 Uploader](./services/uploader/README.md)
- [Push Notification Service](./services/push/README.md)
- Spotlight

#### Running

```
# Build all binaries
make

# Start the TruAPI light client
./bin/truapid start --home ~/.octopus --chain-id truchain
```

#### Migrating DB

`make db_migrate`
