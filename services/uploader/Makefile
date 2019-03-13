PACKAGES=$(shell go list ./...)

MODULES = models

CHAIN_DIR = ./.uploader

define \n


endef

update_deps:
	dep ensure -v

check:
	gometalinter ./...

install_tools_macos:
	brew install dep && brew upgrade dep
	brew tap alecthomas/homebrew-tap && brew install gometalinter
