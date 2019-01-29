PACKAGES=$(shell go list ./...)

MODULES = models

CHAIN_DIR = ./.uploader

define \n


endef

benchmark:
	@go test -bench=. $(PACKAGES)

buidl: build

build: build_cli build_daemon

br: build_daemon run_daemon

bwr: build_daemon wipe_chain run_daemon

check:
	gometalinter ./...

install_tools_macos:
	brew install dep && brew upgrade dep
	brew tap alecthomas/homebrew-tap && brew install gometalinter

go_test:
	@go test $(PACKAGES)
