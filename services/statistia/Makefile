PACKAGES=$(shell go list ./...)

build:
	go build -o ../../bin/statistiad *.go

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o ../../bin/statistiad *.go

run:
	go run *.go

seed-today:
	go run *.go today today

seed-historical:
	go run *.go 2019-04-14 today