# 🐙
## TruStory Go Monorepo

This is a Go monorepo for all non-TruChain related services for TruStory. It is based on, but not a fork of: https://github.com/flowerinthenight/golang-monorepo. It allows for multiple services to co-exist in a single repo while only building services that are updated. Pull in features from [golang-monorepo](https://github.com/flowerinthenight/golang-monorepo) as needed.

### Actions
* Seeder: seeds the chain with fake data

### Services

* [TruAPI Light Client](./services/truapi/README.md)
* [AWS S3 Uploader](./services/uploader/README.md)
* [Push Notification Service](./services/push/README.md)
* Spotlight

#### Setup

Since most of these repos use TruChain as a dependency, your Git config has to be setup to work with a private repo. For that you need to add the following to your `.gitconfig`:

```
[url "git@github.com:"]
    insteadOf = https://github.com/
```

This can be run on the command-line with: 
```
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

This forces Git to use `ssh` instead of `https` when pulling repos with `go mod`.

#### Linting

```sh
make check
```
