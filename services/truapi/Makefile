PACKAGES=$(shell go list ./...)

VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=truapi \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

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
