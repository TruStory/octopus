GO_BIN ?= go
PACKAGES=$(shell go list ./...)

tidy:
	$(GO_BIN) mod tidy

deps:
	$(GO_BIN) get -u github.com/gobuffalo/packr/v2/packr2
	packr2 clean
	make tidy

build:
	make deps
	packr2
	go build -o ../../../bin/spotlightd cmd/main.go
	packr2 clean

build-linux:
	make deps
	packr2
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../../../bin/spotlightd cmd/main.go
	packr2 clean
	
run:
	go run cmd/main.go

build-docker:
	make deps
	packr2
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../../../bin/spotlightd cmd/main.go
	packr2 clean
	docker build -t spotlightd .

run-docker:
	docker rm spotlight -f || true
	docker run -d --name spotlight \
		-eSPOTLIGHT_GRAPHQL_ENDPOINT=$(SPOTLIGHT_GRAPHQL_ENDPOINT) \
		-p 54448:54448 spotlightd

deps-darwin:
	brew install librsvg

test:
	go test github.com/TruStory/octopus/services/spotlight -v