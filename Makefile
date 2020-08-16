BUILDTIME=$(shell date -u +%Y-%m-%d.%H%M)
REVISION=$(shell git log --oneline | head -1 | cut -d\  -f 1)

LDFLAGS=-ldflags "-X main.GitRevision=$(REVISION) -X main.BuildTime=$(BUILDTIME)"

all:
	go build $(LDFLAGS) -o promclient main.go
