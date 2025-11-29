.PHONY: build test clean install

BINARY_NAME=imgcompress
BUILD_DIR=./build

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/imgcompress

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

install:
	go install ./cmd/imgcompress

lint:
	golangci-lint run
