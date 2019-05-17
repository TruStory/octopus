PACKAGES=$(shell go list ./...)

build:
	go build -o ../../bin/truapid *.go

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0  /usr/local/go/bin/go build -o ../../bin/truapid *.go

run:
	go run *.go

run-docker: build-linux
	touch truapid.env
	docker-compose up truapid
