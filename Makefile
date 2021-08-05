.PHONY: all build clean test run check lint 

BIN_FILE=calldiff

all: check build run

build:
	@go build -o "${BIN_FILE}"

clean:
	@go clean
	@rm -rf "output/"

test:
	@go test

check:
	@go fmt ./
	@go vet ./

run:
	@./${BIN_FILE}

lint:
	golangci-lint run --enable-all