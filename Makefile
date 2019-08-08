PACKAGES=$(shell go list ./...)

default: check_deps check_lint build

check_deps:
	@echo "--> Checking deps"
	@go mod download

check_lint:
	@echo "--> Running golangci"
	@golangci-lint run --verbose

build:
	make -C services/push build
	make -C services/uploader build
	make -C services/spotlight build
	make -C services/truapi build

build-linux:
	make -C services/push build-linux
	make -C services/uploader build-linux
	make -C services/spotlight build-linux
	make -C services/truapi build-linux

install_tools_macos:
	brew install golangci/tap/golangci-lint

db_init:
	@go run ./services/db/migrations/*.go init

db_version:
	@go run ./services/db/migrations/*.go version

db_migrate:
	@go run ./services/db/migrations/*.go

db_migrate_down:
	@go run ./services/db/migrations/*.go down

db_reset:
	@go run ./services/db/migrations/*.go reset

start-truapi:
    ./bin/truapid start --home ~/.octopus --chain-id betanet-1
