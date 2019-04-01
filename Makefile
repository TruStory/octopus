PACKAGES=$(shell go list ./...)

build:
	make -C services/push build
	make -C services/uploader build

check:
	golangci-lint run

install_tools_macos:
	brew install golangci/tap/golangci-lint

