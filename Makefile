# Migro Database Migration Tool Makefile

BINARY_NAME=migro
MAIN_PACKAGE=.
BUILD_DIR=build
INSTALL_DIR=/usr/local/bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOINSTALL=$(GOCMD) install

# Build flags
LDFLAGS=-ldflags "-s -w"
BUILD_FLAGS=-v $(LDFLAGS)

# Default target
.DEFAULT_GOAL := help

.PHONY: help build clean test deps tidy install uninstall run dev release

help: ## Show this help message
	@echo "🚀 Migro Database Migration Tool"
	@echo ""
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m\033[0m"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development
deps: ## Download dependencies
	@echo "📦 Downloading dependencies..."
	$(GOGET) -v ./...

tidy: ## Tidy up go.mod
	@echo "🧹 Tidying up dependencies..."
	$(GOMOD) tidy

test: ## Run tests
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

##@ Build
build: deps ## Build the binary
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "✅ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: deps ## Build for multiple platforms
	@echo "🔨 Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	
	# macOS AMD64  
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	
	@echo "✅ Multi-platform builds completed in $(BUILD_DIR)/"

##@ Installation
install: build ## Install binary to system PATH
	@echo "📥 Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✅ $(BINARY_NAME) installed successfully!"
	@echo "   You can now run: $(BINARY_NAME) --help"

install-user: build ## Install binary to user PATH (~/.local/bin)
	@echo "📥 Installing $(BINARY_NAME) to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)
	@chmod +x ~/.local/bin/$(BINARY_NAME)
	@echo "✅ $(BINARY_NAME) installed to user PATH!"
	@echo "   Make sure ~/.local/bin is in your PATH"
	@echo "   You can now run: $(BINARY_NAME) --help"

go-install: ## Install using go install (requires Go)
	@echo "📥 Installing with go install..."
	$(GOINSTALL) $(MAIN_PACKAGE)
	@echo "✅ $(BINARY_NAME) installed via go install!"

uninstall: ## Uninstall binary from system
	@echo "🗑️  Uninstalling $(BINARY_NAME)..."
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@rm -f ~/.local/bin/$(BINARY_NAME)
	@echo "✅ $(BINARY_NAME) uninstalled!"

##@ Usage
run: build ## Build and run with example command
	@echo "🚀 Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME) --help

dev: ## Run in development mode (no build)
	@echo "🔧 Running in development mode..."
	$(GOCMD) run $(MAIN_PACKAGE) --help

setup-example: ## Set up example configuration
	@echo "⚙️  Setting up example configuration..."
	@if [ ! -f migro.yaml ]; then \
		cp migro.example.yaml migro.yaml; \
		echo "✅ Created migro.yaml from example"; \
		echo "   Please edit migro.yaml with your database credentials"; \
	else \
		echo "⚠️  migro.yaml already exists"; \
	fi

##@ Cleanup
clean: ## Clean build artifacts
	@echo "🧹 Cleaning up..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "✅ Cleanup completed!"

##@ Release
release: clean build-all ## Create release builds
	@echo "📦 Creating release package..."
	@mkdir -p $(BUILD_DIR)/release
	@cp README.md $(BUILD_DIR)/release/
	@cp migro.example.yaml $(BUILD_DIR)/release/
	@cp LICENSE $(BUILD_DIR)/release/ 2>/dev/null || true
	
	# Create archives
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 -C release README.md migro.example.yaml
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 -C release README.md migro.example.yaml  
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 -C release README.md migro.example.yaml
	@cd $(BUILD_DIR) && zip -q release/$(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe -j release/README.md release/migro.example.yaml
	
	@echo "✅ Release packages created in $(BUILD_DIR)/release/"

# Get current version from git tags
CURRENT_VERSION := $(shell git tag -l | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | sort -V | tail -1)
CURRENT_VERSION := $(if $(CURRENT_VERSION),$(CURRENT_VERSION),v0.0.0)

# Extract version components
VERSION_PARTS := $(subst v,,$(CURRENT_VERSION))
VERSION_PARTS := $(subst ., ,$(VERSION_PARTS))
MAJOR := $(word 1,$(VERSION_PARTS))
MINOR := $(word 2,$(VERSION_PARTS))
PATCH := $(word 3,$(VERSION_PARTS))

release-patch: ## Release new patch version (x.x.X)
	$(eval NEW_VERSION := v$(MAJOR).$(MINOR).$(shell echo $$(($(PATCH) + 1))))
	@echo "🏷️  Creating patch release: $(CURRENT_VERSION) → $(NEW_VERSION)"
	@$(MAKE) _do_release VERSION=$(NEW_VERSION)

release-minor: ## Release new minor version (x.X.0)
	$(eval NEW_VERSION := v$(MAJOR).$(shell echo $$(($(MINOR) + 1))).0)
	@echo "🏷️  Creating minor release: $(CURRENT_VERSION) → $(NEW_VERSION)"
	@$(MAKE) _do_release VERSION=$(NEW_VERSION)

release-major: ## Release new major version (X.0.0)
	$(eval NEW_VERSION := v$(shell echo $$(($(MAJOR) + 1))).0.0)
	@echo "🏷️  Creating major release: $(CURRENT_VERSION) → $(NEW_VERSION)"
	@$(MAKE) _do_release VERSION=$(NEW_VERSION)

release-auto: ## Automatic release (CRUD features = minor version)
	$(eval NEW_VERSION := v1.2.0)
	@echo "🚀 Creating automated release for CRUD features: $(NEW_VERSION)"
	@$(MAKE) _do_release VERSION=$(NEW_VERSION) MESSAGE="✨ feat: Add comprehensive CRUD operations\n\n- Add insert, update, select-one, select-many, delete commands\n- Support automatic timestamps and soft delete\n- Add query preview and formatted table results\n- Include table validation from migration files\n- Update documentation with usage examples"

_do_release: ## Internal: perform the actual release
	@echo "🔄 Preparing release $(VERSION)..."
	
	# Check for uncommitted changes
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "📝 Committing changes..."; \
		git add .; \
		git commit -m "$${MESSAGE:-Release $(VERSION)}"; \
	else \
		echo "✅ Working directory is clean"; \
	fi
	
	# Create and push tag
	@echo "🏷️  Creating tag $(VERSION)..."
	@git tag -a $(VERSION) -m "$(VERSION): $${MESSAGE:-Release $(VERSION)}"
	@git push origin main
	@git push origin $(VERSION)
	
	# Build release packages
	@echo "📦 Building release packages..."
	@$(MAKE) release
	
	@echo "✅ Release $(VERSION) completed!"
	@echo ""
	@echo "📋 Next steps:"
	@echo "   1. Go to GitHub: https://github.com/ChungNQ511/migro/releases"
	@echo "   2. Create a new release for tag $(VERSION)"
	@echo "   3. Upload binaries from $(BUILD_DIR)/release/"
	@echo "   4. Publish the release!"

publish-github: ## Publish release to GitHub (requires gh CLI)
	@echo "🚀 Publishing to GitHub..."
	@if ! command -v gh &> /dev/null; then \
		echo "❌ GitHub CLI (gh) not found. Install it first:"; \
		echo "   brew install gh"; \
		echo "   or visit: https://cli.github.com"; \
		exit 1; \
	fi
	@echo "📤 Creating GitHub release..."
	@gh release create $(VERSION) $(BUILD_DIR)/release/*.tar.gz $(BUILD_DIR)/release/*.zip \
		--title "$(VERSION): CRUD Operations Support" \
		--notes "🎉 **Major Feature Release**\n\n## New CRUD Operations\n- ✨ **insert**: Add records with auto timestamps\n- ✨ **update**: Modify records with auto updated_at\n- ✨ **select-one**: Query single records\n- ✨ **select-many**: Query multiple records with pagination\n- ✨ **delete**: Soft delete with deleted_at\n\n## Features\n- 📋 **Query Preview**: Shows SQL before execution\n- 🎨 **Formatted Results**: Beautiful table output\n- 🛡️ **Table Validation**: Checks against migration files\n- 🕒 **Auto Timestamps**: Handles created_at/updated_at\n\n## Installation\n\`\`\`bash\ncurl -sSL https://raw.githubusercontent.com/ChungNQ511/migro/main/install.sh | bash\n\`\`\`\n\nSee README.md for complete usage examples!"
	@echo "✅ GitHub release published!"

release-complete: release-auto publish-github ## Complete automated release with GitHub publishing

##@ Docker
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .
	@echo "✅ Docker image built: $(BINARY_NAME):latest"

docker-run: ## Run in Docker container
	@echo "🐳 Running in Docker..."
	@docker run --rm -it -v $(PWD):/workspace $(BINARY_NAME):latest --help 