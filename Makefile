PACKAGES=$(shell go list ./...)

default: check_deps check_lint build


check_deps:
	@echo "--> Checking deps"
	@go mod download

check_lint:
	@echo "--> Running golangci"
	@golangci-lint run --verbose

build:
	make -C services/push build-linux
	make -C services/uploader build

install_tools_macos:
	brew install golangci/tap/golangci-lint

