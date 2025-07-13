PKGS := $(shell go list ./...)

BUILD_DIR := build
CMD_DIR = ./cmd/pho
MAIN_FILE := $(CMD_DIR)/main.go

BINARY_NAME := pho
INSTALL_DIR := $(shell go env GOPATH)/bin

# Default target
all: build

.PHONY: build
build:
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)

# Run the git-undo binary
.PHONY: run
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run tests
.PHONY: test
test:
	@go test ./...

# Tidy: format and vet the code
.PHONY: tidy
tidy:
	@go fmt $(PKGS)
	@go vet $(PKGS)
	@go mod tidy

# Lint the code using golangci-lint
# todo reuse var if possible
.PHONY: lint
lint:
	$(shell which golangci-lint) run

# Install pho globally to GOPATH/bin
.PHONY: install
install:
	@echo "Installing pho to $(INSTALL_DIR)..."
	@go install $(CMD_DIR)
	@echo "✓ Successfully installed pho to $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo "Make sure $(INSTALL_DIR) is in your PATH to use 'pho' from anywhere"

# Uninstall pho from GOPATH/bin
.PHONY: uninstall
uninstall:
	@echo "Uninstalling pho from $(INSTALL_DIR)..."
	@if [ -f "$(INSTALL_DIR)/$(BINARY_NAME)" ]; then \
		rm -f "$(INSTALL_DIR)/$(BINARY_NAME)" && \
		echo "✓ Successfully uninstalled pho from $(INSTALL_DIR)/$(BINARY_NAME)"; \
	else \
		echo "⚠ pho was not found in $(INSTALL_DIR)"; \
	fi

# Show help with available targets
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build      - Build the pho binary to build/ directory"
	@echo "  run        - Build and run the pho binary"
	@echo "  test       - Run all tests with verbose output"
	@echo "  lint       - Run golangci-lint (installs if needed)"
	@echo "  tidy       - Format code, run vet, and tidy modules"
	@echo "  install    - Install pho globally to GOPATH/bin"
	@echo "  uninstall  - Remove pho from GOPATH/bin"
	@echo "  help       - Show this help message"
