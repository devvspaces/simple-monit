# Makefile for server-monitor

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=server-monitor
BINARY_UNIX=$(BINARY_NAME)_unix
MAIN_PATH=./
CONFIG_EXAMPLE=config.yaml.example
CONFIG_FILE=config.yaml

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build clean test coverage deps tidy fmt lint run install uninstall

all: test build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

# Cross-compilation for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) $(MAIN_PATH)

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

test:
	$(GOTEST) -v ./...

coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

deps:
	$(GOGET) -v -t -d ./...

tidy:
	$(GOMOD) tidy

fmt:
	$(GOCMD) fmt ./...

lint:
	golangci-lint run ./...

run: build
	./$(BINARY_NAME) -config $(CONFIG_FILE)

config:
	@if [ ! -f $(CONFIG_FILE) ]; then \
		cp $(CONFIG_EXAMPLE) $(CONFIG_FILE); \
		echo "Created $(CONFIG_FILE) from example file"; \
	else \
		echo "$(CONFIG_FILE) already exists"; \
	fi

install: build
	mkdir -p $(DESTDIR)/usr/local/bin
	cp $(BINARY_NAME) $(DESTDIR)/usr/local/bin

uninstall:
	rm -f $(DESTDIR)/usr/local/bin/$(BINARY_NAME)

help:
	@echo "Make commands:"
	@echo "  all          - Run tests and build"
	@echo "  build        - Build the binary"
	@echo "  build-linux  - Cross-compile for Linux"
	@echo "  clean        - Remove build artifacts"
	@echo "  test         - Run tests"
	@echo "  coverage     - Generate test coverage report"
	@echo "  deps         - Download dependencies"
	@echo "  tidy         - Tidy go.mod file"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  run          - Build and run the application"
	@echo "  config       - Create config file from example if it doesn't exist"
	@echo "  install      - Install binary to /usr/local/bin"
	@echo "  uninstall    - Remove binary from /usr/local/bin"