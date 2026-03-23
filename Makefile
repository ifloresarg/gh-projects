BINARY_NAME=gh-projects
BUILD_DIR=./dist

.PHONY: build test lint install clean

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/gh-projects

test:
	go test ./...

lint:
	golangci-lint run

install:
	gh extension install .

clean:
	rm -rf $(BUILD_DIR)
