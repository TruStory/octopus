PACKAGES=$(shell go list ./...)

build:
	go build -o ../../bin/seed *.go

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o ../../bin/seed *.go

run:
	go run *.go