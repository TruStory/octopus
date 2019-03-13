PACKAGES=$(shell go list ./...)

check:
	golangci-lint run

install_tools_macos:
	brew install golangci/tap/golangci-lint

