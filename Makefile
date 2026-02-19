VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

.PHONY: build test lint clean docker

build:
	CGO_ENABLED=1 go build $(LDFLAGS) -o notifyd ./cmd/notifyd

test:
	go test -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run ./...

clean:
	rm -f notifyd coverage.out
	rm -rf dist/

docker:
	docker build -t notifyd:$(VERSION) .

run: build
	./notifyd --debug --db /tmp/notifyd.db
