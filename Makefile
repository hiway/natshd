.PHONY: test run debug build clean tidy

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=natshd
BINARY_PATH=./cmd/natshd

# Build the binary
build:
	$(GOBUILD) -o $(BINARY_NAME) $(BINARY_PATH)

# Run tests
test:
	$(GOTEST) -v ./...

# Run the application
run:
	$(GOCMD) run $(BINARY_PATH)

# Debug run with verbose logging
debug:
	$(GOCMD) run $(BINARY_PATH) --log-level debug

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Tidy up dependencies
tidy:
	$(GOMOD) tidy

# Download dependencies
deps:
	$(GOMOD) download

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Install development dependencies
dev-deps:
	$(GOGET) -t ./...
