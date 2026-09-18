.PHONY: all build test run clean

BINARY_NAME=bin/syscheck

all: test build

build:
	go build -o $(BINARY_NAME) ./cmd/syscheck

test:
	go test -v -race ./...

run: build
	./$(BINARY_NAME)

clean:
	rm -rf bin/
