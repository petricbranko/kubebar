BINARY := kubebar
PKG := ./cmd/kubebar

.PHONY: build test lint install clean

build:
	go build -o bin/$(BINARY) $(PKG)

test:
	go test ./...

lint:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }
	go vet ./...
	staticcheck ./...

install:
	go install $(PKG)

clean:
	rm -rf bin
