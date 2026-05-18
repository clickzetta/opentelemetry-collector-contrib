# =============================================================================
# cz-otel CLI — Build and Packaging Makefile
# =============================================================================
#
# Targets:
#   build       — Build CLI binary for current platform
#   collector   — Build collector binary using OCB
#   package     — Create tar.gz for current platform
#   package-all — Create tar.gz for all supported platforms
#   test        — Run all Go tests
#   lint        — Run linter (placeholder)
#   clean       — Remove build artifacts
#
# Usage:
#   make build                    # Build CLI for current OS/ARCH
#   make package VERSION=1.0.0   # Package with custom version
#   make package-all             # Build packages for all platforms
#   make clean                   # Remove dist/ directory
# =============================================================================

# --- Configuration -----------------------------------------------------------

# Version defaults; override via environment or command line:
#   make build VERSION=1.2.3
VERSION          ?= 0.1.0
COLLECTOR_VERSION ?= 0.120.0

# Binary names
BINARY_NAME      = cz-otel
COLLECTOR_NAME   = otelcol-clickzetta

# Output directory for all build artifacts
DIST_DIR         = dist

# Detect current platform
OS   := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)

# Package naming convention: cz-otel-<version>-<os>-<arch>
PACKAGE_NAME     = $(BINARY_NAME)-$(VERSION)-$(OS)-$(ARCH)

# Supported platforms for cross-compilation
PLATFORMS        = darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows-amd64

# Linker flags for version injection
LDFLAGS          = -X main.version=$(VERSION) -X main.collectorVersion=$(COLLECTOR_VERSION)

# --- Targets -----------------------------------------------------------------

.PHONY: build collector package package-all test lint clean

## build: Compile the CLI binary for the current platform
build:
	@echo "==> Building $(BINARY_NAME) $(VERSION) for $(OS)/$(ARCH)..."
	@mkdir -p $(DIST_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME) ./

## collector: Build the collector binary using OCB (OpenTelemetry Collector Builder)
##
## Prerequisites:
##   Install OCB if not available:
##     go install go.opentelemetry.io/collector/cmd/builder@latest
##   Ensure builder-config.yaml exists in the project root or adjust the path below.
collector:
	@echo "==> Building collector binary using OCB..."
	@mkdir -p $(DIST_DIR)
	ocb --config=builder-config.yaml
	@echo "==> Collector binary built at $(DIST_DIR)/$(COLLECTOR_NAME)"

## package: Assemble a tar.gz distribution for the current platform
##   Depends on build and collector targets.
package: build collector
	@echo "==> Packaging $(PACKAGE_NAME).tar.gz..."
	@rm -rf $(DIST_DIR)/$(PACKAGE_NAME)
	@mkdir -p $(DIST_DIR)/$(PACKAGE_NAME)/bin
	@# Copy binaries
	@cp $(DIST_DIR)/$(BINARY_NAME) $(DIST_DIR)/$(PACKAGE_NAME)/bin/$(BINARY_NAME)
	@cp $(DIST_DIR)/$(COLLECTOR_NAME) $(DIST_DIR)/$(PACKAGE_NAME)/bin/$(COLLECTOR_NAME)
	@# Copy install script (Unix or Windows)
ifeq ($(OS),windows)
	@cp scripts/install.bat $(DIST_DIR)/$(PACKAGE_NAME)/install.bat
else
	@cp scripts/install.sh $(DIST_DIR)/$(PACKAGE_NAME)/install.sh
	@chmod +x $(DIST_DIR)/$(PACKAGE_NAME)/install.sh
endif
	@# Copy README
	@cp scripts/README.md $(DIST_DIR)/$(PACKAGE_NAME)/README.md
	@# Create tar.gz archive
	@cd $(DIST_DIR) && tar -czf $(PACKAGE_NAME).tar.gz $(PACKAGE_NAME)
	@# Clean up staging directory
	@rm -rf $(DIST_DIR)/$(PACKAGE_NAME)
	@echo "==> Package created: $(DIST_DIR)/$(PACKAGE_NAME).tar.gz"

## package-all: Cross-compile and package for all supported platforms
package-all:
	@echo "==> Building packages for all platforms..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		PLAT_OS=$$(echo $$platform | cut -d- -f1); \
		PLAT_ARCH=$$(echo $$platform | cut -d- -f2); \
		PLAT_PKG=$(BINARY_NAME)-$(VERSION)-$$PLAT_OS-$$PLAT_ARCH; \
		echo "==> Building $$PLAT_PKG..."; \
		SUFFIX=""; \
		if [ "$$PLAT_OS" = "windows" ]; then SUFFIX=".exe"; fi; \
		GOOS=$$PLAT_OS GOARCH=$$PLAT_ARCH \
			go build -ldflags "$(LDFLAGS)" \
			-o $(DIST_DIR)/$$PLAT_PKG/bin/$(BINARY_NAME)$$SUFFIX ./; \
		echo "  [NOTE] Collector binary for $$PLAT_OS/$$PLAT_ARCH must be built natively or via CI."; \
		mkdir -p $(DIST_DIR)/$$PLAT_PKG/bin; \
		if [ "$$PLAT_OS" = "windows" ]; then \
			cp scripts/install.bat $(DIST_DIR)/$$PLAT_PKG/install.bat; \
		else \
			cp scripts/install.sh $(DIST_DIR)/$$PLAT_PKG/install.sh; \
			chmod +x $(DIST_DIR)/$$PLAT_PKG/install.sh; \
		fi; \
		cp scripts/README.md $(DIST_DIR)/$$PLAT_PKG/README.md; \
		cd $(DIST_DIR) && tar -czf $$PLAT_PKG.tar.gz $$PLAT_PKG && cd ..; \
		rm -rf $(DIST_DIR)/$$PLAT_PKG; \
		echo "==> Created $(DIST_DIR)/$$PLAT_PKG.tar.gz"; \
	done
	@echo "==> All packages built."

## test: Run all Go tests
test:
	@echo "==> Running tests..."
	go test ./...

## lint: Run linter (placeholder — configure golangci-lint or similar)
lint:
	@echo "==> Linting..."
	@echo "    (placeholder: install and run golangci-lint)"
	@# golangci-lint run ./...

## clean: Remove all build artifacts
clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf $(DIST_DIR)
	@echo "==> Clean complete."
