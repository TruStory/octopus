PACKAGES=$(shell go list ./...)

build:
	packr2
	go build -o ../../bin/spotlightd *.go
	packr2 clean

build-linux:
	packr2
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o ../../bin/spotlightd *.go
	packr2 clean

run:
	go run *.go

build-docker:
	packr2
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o bin/spotlightd 
	packr2 clean
	docker build -t spotlightd .

run-docker:
	docker rm spotlight -f || true
	docker run -d --name spotlight \
		-eSPOTLIGHT_GRAPHQL_ENDPOINT=$(SPOTLIGHT_GRAPHQL_ENDPOINT) \
		-v $(PWD)/storage:/usr/spotlight/storage \
		-v $(PWD)/claim.svg:/usr/spotlight/claim.svg \
		-p 54448:54448 spotlightd