VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -X github.com/plinth-dev/cli/internal/cli.Version=$(VERSION) \
           -X github.com/plinth-dev/cli/internal/cli.Commit=$(COMMIT)

.PHONY: build test vet install clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/plinth ./cmd/plinth

test:
	go test -race -cover ./...

vet:
	go vet ./...

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/plinth

clean:
	rm -rf bin/ coverage.txt coverage.html
