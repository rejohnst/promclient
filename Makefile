GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOPATH ?= $(shell go env GOPATH)
BUILDTIME := $(shell date -u +%Y-%m-%d.%H%M)
REVISION := $(shell git log --oneline | head -1 | cut -d\  -f 1)
BINARY := promclient-$(GOOS)-$(GOARCH)

LDFLAGS := -ldflags "-X main.gitRevision=$(REVISION) -X main.buildTime=$(BUILDTIME)"

.PHONY: all build install check clean

all: build

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BINARY) .

install: build
	install -m 0755 $(BINARY) $(GOPATH)/bin/promclient

check:
	test -z "$$(gofmt -l *.go internal)"
	go test ./...
	go vet ./...
	staticcheck ./...

clean:
	rm -f promclient-*
