build:
	go build -o bin/agentos .

test:
	go test ./...

fmt:
	gofmt -w .

clean:
	rm -rf bin

.PHONY: build test fmt clean
