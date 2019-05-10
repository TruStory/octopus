PACKAGES=$(shell go list ./...)

build:
	go build -o ../../bin/apid *.go

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o ../../bin/apid *.go

run:
	go run *.go

run-docker: build-linux
	docker-compose up apid
