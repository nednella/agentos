VERSION ?= dev
LDFLAGS := -X github.com/nednella/agentos/internal/version.Version=$(VERSION)

build:
	go build -ldflags '$(LDFLAGS)' -o bin/agentos .

install:
	go install -ldflags '$(LDFLAGS)' .

test:
	go test ./...

fmt:
	gofmt -w .

clean:
	rm -rf bin

.PHONY: build install test fmt clean
