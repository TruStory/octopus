PACKAGES=$(shell go list ./...)

build:
	go build -o ../../bin/pushd *.go

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o ../../bin/pushd *.go

run:
	go run *.go

run-docker: build-linux
	touch pushd.env
	docker-compose up pushd

run-gorush:
	touch gorush.env
	docker-compose up -d gorush