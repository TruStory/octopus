PACKAGES=$(shell go list ./...)

# //  go build -X github.com/cosmos/cosmos-sdk/version.Name=dapp \
# //   -X github.com/cosmos/cosmos-sdk/version.Version=1.0 \
# //   -X github.com/cosmos/cosmos-sdk/version.Commit=f0f7b7dab7e36c20b757cebce0e8f4fc5b95de60 \
# //   -X "github.com/cosmos/cosmos-sdk/version.BuildTags=linux darwin amd64"

# VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=truapi \
	-X github.com/cosmos/cosmos-sdk/version.Version=0.5 \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \

BUILD_FLAGS := -ldflags '$(ldflags)'

build:
	go build $(BUILD_FLAGS) -o ../../bin/truapid *.go

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0  /usr/local/go/bin/go build $(BUILD_FLAGS) -o ../../bin/truapid *.go

run:
	go run *.go

run-docker: build-linux
	touch truapid.env
	docker-compose up truapid
