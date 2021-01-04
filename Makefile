BUILDTIME=$(shell date -u +%Y-%m-%d.%H%M)
REVISION=$(shell git log --oneline | head -1 | cut -d\  -f 1)

LDFLAGS=-ldflags "-X main.gitRevision=$(REVISION) -X main.buildTime=$(BUILDTIME)"

all:
	go build $(LDFLAGS) -o promclient cmd/main.go

check:
	go vet cmd/main.go
	golint cmd/main.go
