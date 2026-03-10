# The targets cannot be run in parallel
.NOTPARALLEL:

VERSION := $(shell git describe --tags --always --match "v*.*.*" 2>/dev/null || echo "v1.2.0")

# Use git in windows since we don't have access to the `date` tool natively
ifeq ($(OS),Windows_NT)
	DATE := $(shell git log -1 --format="%ad" --date=format-local:'%Y-%m-%dT%H:%M UTC' 2>nul || echo "unknown")
	EXECUTABLE_PATH := ./gios.exe
else
	DATE := $(shell date -u '+%Y-%m-%d-%H:%M UTC')
	EXECUTABLE_PATH := ./gios
endif

VERSION_FLAGS := -X "main.AppVersion=$(VERSION)" -X "main.BuildTime=$(DATE)"
LDFLAGS       := -ldflags='-s -w $(VERSION_FLAGS)'

BINARY_NAME := gios
BUILD_DIR   := build

LOCAL_ARCH ?= $(shell uname -m)
ifeq ($(LOCAL_ARCH),x86_64)
    TARGET_ARCH ?= amd64
else ifeq ($(LOCAL_ARCH),amd64)
    TARGET_ARCH ?= amd64
else ifeq ($(LOCAL_ARCH),arm64)
    TARGET_ARCH ?= arm64
else ifeq ($(LOCAL_ARCH),aarch64)
    TARGET_ARCH ?= arm64
else
    $(error This system's architecture $(LOCAL_ARCH) isn't supported)
endif

LOCAL_OS ?= $(shell go env GOOS)
ifeq ($(LOCAL_OS),linux)
    TARGET_OS ?= linux
else ifeq ($(LOCAL_OS),darwin)
    TARGET_OS ?= darwin
else ifeq ($(LOCAL_OS),windows)
    TARGET_OS ?= windows
else
    $(error This system's OS $(LOCAL_OS) isn't supported)
endif

.PHONY: all
all: build test

.PHONY: clean
clean:
	@echo "=> Cleaning..."
	@go clean
	@rm -rf $(BUILD_DIR) $(EXECUTABLE_PATH) ~/.gios/bin/gios

.PHONY: fmt
fmt:
	@echo "=> Formatting code..."
	@go fmt ./...

.PHONY: vet
vet:
	@echo "=> Running go vet..."
	@go vet ./...

.PHONY: lint
lint:
	@echo "=> Running golangci-lint..."
	@golangci-lint run || echo "golangci-lint not installed or found issues"

.PHONY: test
test: vet
	@echo "=> Running tests..."
	@go test -v -race $(LDFLAGS) ./...

.PHONY: build
build:
	@echo "=> Building $(BINARY_NAME) for $(TARGET_OS)/$(TARGET_ARCH)..."
	@GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH) go build $(LDFLAGS) -o $(EXECUTABLE_PATH) ./cmd/gios

.PHONY: install
install: build
	@echo "=> Installing to ~/.gios/bin..."
	@mkdir -p ~/.gios/bin
	@mv $(EXECUTABLE_PATH) ~/.gios/bin/gios
	@echo "=> Installed successfully."

# Cross-compilation targets specifically for CI/CD actions
.PHONY: build-all
build-all: 
	@mkdir -p $(BUILD_DIR)
	@echo "=> Building Linux (amd64, arm64)..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/gios
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/gios
	@echo "=> Building macOS (amd64, arm64)..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/gios
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/gios
	@echo "=> Building Windows (amd64, arm64)..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/gios
	@GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe ./cmd/gios

.PHONY: release-manifest
release-manifest:
	@echo "=> Generating release manifest..."
	@echo "version: $(VERSION)" > $(BUILD_DIR)/release.yml
	@echo "binaries:" >> $(BUILD_DIR)/release.yml
	@for target in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 windows-arm64; do \
		case $$target in \
			windows*) file="$(BINARY_NAME)-$$target.exe" ;; \
			*) file="$(BINARY_NAME)-$$target" ;; \
		esac; \
		hash=$$(sha256sum $(BUILD_DIR)/$$file | awk '{print $$1}'); \
		echo "  - target: $$target" >> $(BUILD_DIR)/release.yml; \
		echo "    executable: $$file" >> $(BUILD_DIR)/release.yml; \
		echo "    sha256: $$hash" >> $(BUILD_DIR)/release.yml; \
	done
