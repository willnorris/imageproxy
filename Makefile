.PHONY: all build test clean run

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=imageproxy
BINARY_UNIX=$(BINARY_NAME)_unix
BIN_DIR=bin

all: test build

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

build: $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME) -v ./cmd/imageproxy

test: 
	$(GOTEST) -v ./...

clean: 
	$(GOCLEAN)
	rm -rf $(BIN_DIR)

run: build
	./$(BIN_DIR)/$(BINARY_NAME)

# Cross compilation
build-linux: $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BIN_DIR)/$(BINARY_UNIX) -v ./cmd/imageproxy

docker-build:
	docker build -t $(BINARY_NAME) .

# Example commands
run-with-cache: build
	./$(BIN_DIR)/$(BINARY_NAME) -cache memory:100:24h -cache-max-age 24h

run-with-disk-cache: build
	./$(BIN_DIR)/$(BINARY_NAME) -cache ./cache -cache-max-age 24h

# Long-term caching
run-with-5year-cache: build
	./$(BIN_DIR)/$(BINARY_NAME) -cache ./cache -cache-max-age 43800h 