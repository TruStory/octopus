PACKAGES=$(shell go list ./...)

build:
	go build -o ../../bin/spotlightd *.go

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o ../../bin/spotlightd *.go

run:
	go run *.go

build-docker:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0  go build -o bin/spotlightd 
	docker build -t spotlightd .

run-docker:
	docker rm spotlight -f || true
	docker run -d --name spotlight \
		-eSPOTLIGHT_GRAPHQL_ENDPOINT=$(SPOTLIGHT_GRAPHQL_ENDPOINT) \
		-v $(PWD)/storage:/usr/spotlight/storage \
		-v $(PWD)/story.html:/usr/spotlight/story.html \
		-p 54448:54448 spotlightd