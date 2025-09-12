# The name of the binary
BINARY_NAME=llm-batch
# The top-level directory for build artifacts
BINDIR=bin

# Get version from the latest git tag. Fallback to v0.0.0 if not in a git repo.
# The version is embedded into the binary.
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
LDFLAGS := -ldflags="-X main.version=${VERSION}"

# Platform specific directories
LINUX_DIR = $(BINDIR)/linux-amd64
WINDOWS_DIR = $(BINDIR)/windows-amd64
DARWIN_DIR = $(BINDIR)/darwin-universal

# Default target
all: build

# Build binaries for all supported platforms
build:
	@mkdir -p $(LINUX_DIR) $(WINDOWS_DIR) $(DARWIN_DIR)
	@echo "Building for linux/amd64..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(LINUX_DIR)/$(BINARY_NAME)
	@echo "Building for windows/amd64..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(WINDOWS_DIR)/$(BINARY_NAME).exe
	@echo "Building for macOS universal binary..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINDIR)/$(BINARY_NAME)-darwin-amd64
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINDIR)/$(BINARY_NAME)-darwin-arm64
	@lipo -create -output $(DARWIN_DIR)/$(BINARY_NAME) $(BINDIR)/$(BINARY_NAME)-darwin-amd64 $(BINDIR)/$(BINARY_NAME)-darwin-arm64
	@rm $(BINDIR)/$(BINARY_NAME)-darwin-amd64 $(BINDIR)/$(BINARY_NAME)-darwin-arm64

# Create distribution packages (.tar.gz and .zip)
package: build
	@echo "Creating distribution packages..."
	@tar -czf $(BINDIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(LINUX_DIR) $(BINARY_NAME)
	@tar -czf $(BINDIR)/$(BINARY_NAME)-$(VERSION)-darwin-universal.tar.gz -C $(DARWIN_DIR) $(BINARY_NAME)
	@zip -j $(BINDIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(WINDOWS_DIR)/$(BINARY_NAME).exe

# Clean up the build artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf $(BINDIR)

.PHONY: all build package clean

