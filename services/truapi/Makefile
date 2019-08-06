GO_BIN ?= go
PACKAGES=$(shell go list ./...)

VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=truapi \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)'

tidy:
	$(GO_BIN) mod tidy

deps:
	$(GO_BIN) get -u github.com/gobuffalo/packr/v2/packr2
	packr2 clean
	make tidy

build:
	make deps
	packr2 build $(BUILD_FLAGS) -o ../../bin/truapid *.go
	packr2 clean

build-linux:
	make deps
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 packr2 build $(BUILD_FLAGS) -o ../../bin/spotlightd *.go
	packr2 clean

run:
	go run *.go

run-docker: build-linux
	touch truapid.env
	docker-compose up truapid
