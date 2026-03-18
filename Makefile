BINARY_NAME=finops
VERSION=$(shell cat VERSION)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint generate clean install

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) .

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

generate:
	sqlc generate

clean:
	rm -rf bin/ dist/

install: build
	cp bin/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
