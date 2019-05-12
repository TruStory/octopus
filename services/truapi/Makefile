PACKAGES=$(shell go list ./...)

build:
	go build -o ../../bin/truapid *.go

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o ../../bin/truapid *.go

run:
	go run *.go

run-docker: build-linux
	docker-compose up truapid
