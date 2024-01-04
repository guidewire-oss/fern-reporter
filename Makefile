# Makefile for a Gin-based Golang project using gox for cross-compilation

# Project Name
BINARY_NAME=fern

# Go related variables.
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOPKG=$(GOBASE)

# Go build and run commands
.PHONY: all build run clean cross-compile docker-build docker-run

all: build

build:
	@echo "Building..."
	@GOBIN=$(GOBIN) go build -o $(GOBIN)/$(BINARY_NAME) $(GOPKG)

run:
	@echo "Running..."
	@GOBIN=$(GOBIN) ./bin/$(BINARY_NAME)

clean:
	@echo "Cleaning..."
	@GOBIN=$(GOBIN) go clean
	@rm -f $(GOBIN)/$(BINARY_NAME)

# Cross-compilation with gox
cross-compile:
	@echo "Cross compiling for Linux and Mac..."
	@gox -osarch="linux/amd64 darwin/amd64" -output "$(GOBIN)/$(BINARY_NAME)_{{.OS}}_{{.Arch}}" $(GOPKG)

# Testing
test:
	@echo "Testing..."
	@go test ./...

docker-build: cross-compile
	@echo "Building Docker image..."
	@docker build -t fern-app .

docker-run: docker-build
	@echo "Running application in Docker..."
	@docker run -p 8080:8080 fern-app


