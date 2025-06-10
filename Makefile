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
	@go test -v ./...

# Tidy: format and vet the code
.PHONY: tidy
tidy:
	@go fmt $(PKGS)
	@go vet $(PKGS)
	@go mod tidy

# Install golangci-lint only if it's not already installed
.PHONY: lint-install
lint-install:
	@if ! [ -x "$(GOLANGCI_LINT)" ]; then \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi

# Lint the code using golangci-lint
# todo reuse var if possible
.PHONY: lint
lint: lint-install
	$(shell which golangci-lint) run

# Install pho globally
.PHONY: install
install:
	@echo "Installing pho"
	@go install $(CMD_DIR)
