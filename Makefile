ARCH?=arm64
OS?=darwin
BUILDTIME=$(shell date -u +%Y-%m-%d.%H%M)
REVISION=$(shell git log --oneline | head -1 | cut -d\  -f 1)

LDFLAGS=-ldflags "-X main.gitRevision=$(REVISION) -X main.buildTime=$(BUILDTIME)"

all:
	CGO_ENABLED=0 GOOS= GOARCH=$(ARCH) go build $(LDFLAGS) -o promclient-$(OS)-$(ARCH) main.go

install:
	install -m 0755 promclient-$(OS)-$(ARCH) $(GOPATH)/bin/promclient

check:
	go vet main.go
	golint main.go
