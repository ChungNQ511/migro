# 🔨 Build & Release Documentation

## 📋 **Quick Reference**

```bash
# Development
make dev              # Run in development mode
make build            # Build local binary
make install          # Install to system

# Release
make build-all        # Build for all platforms
make release          # Create release packages
make clean            # Clean build artifacts

# Testing
make test             # Run tests
make deps             # Download dependencies
```

## 🚀 **Complete Build Process**

### **1. Development Build**

```bash
# Quick development run (no build)
make dev

# Build local binary for your platform
make build
# Output: build/migro

# Install locally
make install          # System-wide (/usr/local/bin)
make install-user     # User-only (~/.local/bin)
make go-install       # Using go install

# Test local build
./build/migro --help
```

### **2. Multi-Platform Build**

```bash
# Build for all platforms
make build-all

# Manual build for specific platforms
GOOS=linux GOARCH=amd64 go build -o build/migro-linux-amd64 .
GOOS=darwin GOARCH=amd64 go build -o build/migro-darwin-amd64 .
GOOS=darwin GOARCH=arm64 go build -o build/migro-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o build/migro-windows-amd64.exe .
```

**Output files:**
```
build/
├── migro-linux-amd64
├── migro-darwin-amd64
├── migro-darwin-arm64
└── migro-windows-amd64.exe
```

### **3. Release Package Creation**

```bash
# Create complete release packages
make release

# What it does:
# 1. Clean previous builds
# 2. Download dependencies
# 3. Build for all platforms
# 4. Create archives with documentation
```

**Output packages:**
```
build/release/
├── migro-linux-amd64.tar.gz
├── migro-darwin-amd64.tar.gz
├── migro-darwin-arm64.tar.gz
├── migro-windows-amd64.zip
├── README.md
├── migro.example.yaml
└── LICENSE
```

## 📦 **Release Process**

### **Step 1: Prepare Release**

```bash
# 1. Ensure all changes are committed
git status
git add .
git commit -m "✨ feat: your changes description"

# 2. Update version (if needed)
# Edit go.mod, README.md, or version files

# 3. Build and test
make clean
make build
make test
./build/migro --help  # Test locally
```

### **Step 2: Create Release Build**

```bash
# Create release packages
make release

# Verify packages
ls -la build/release/
```

### **Step 3: Git Tagging**

```bash
# Create version tag (semantic versioning)
git tag -a v1.2.0 -m "v1.2.0: Your release description"

# Push tag
git push origin v1.2.0

# List all tags
git tag -l
```

### **Step 4: GitHub Release**

#### **Option A: Manual (GitHub Web)**

1. Go to: `https://github.com/ChungNQ511/migro/releases`
2. Click **"Create a new release"**
3. **Tag version**: `v1.2.0`
4. **Release title**: `v1.2.0: Release Description`
5. **Description**:
```markdown
## 🎉 What's New
- ✨ Feature 1
- 🐛 Bug fix 2
- ⚡ Performance improvement 3

## 🛠️ Installation
```bash
# Quick install
curl -sSL https://raw.githubusercontent.com/ChungNQ511/migro/main/install.sh | bash

# Go install
go install github.com/ChungNQ511/migro@latest
```
```

6. **Upload files** from `build/release/`:
   - Drag & drop all `.tar.gz` and `.zip` files
7. Click **"Publish release"**

#### **Option B: GitHub CLI**

```bash
# Install GitHub CLI
brew install gh  # macOS
# or
sudo apt install gh  # Ubuntu

# Login
gh auth login

# Create release
gh release create v1.2.0 \
  build/release/*.tar.gz \
  build/release/*.zip \
  --title "v1.2.0: Release Description" \
  --notes "✨ Your release notes here"

# List releases
gh release list
```

## 🔧 **Build Configuration**

### **Makefile Variables**

```makefile
BINARY_NAME=migro              # Output binary name
BUILD_DIR=build                # Build output directory
INSTALL_DIR=/usr/local/bin     # System install location
LDFLAGS=-ldflags "-s -w"       # Build flags (strip symbols)
```

### **Build Flags**

```bash
# Production build (smaller size)
go build -ldflags "-s -w" -o migro .

# Debug build (with symbols)
go build -o migro .

# With version info
go build -ldflags "-X main.version=v1.2.0" -o migro .
```

### **Cross-Compilation**

```bash
# Available platforms
go tool dist list

# Common platforms
GOOS=linux GOARCH=amd64     # Linux 64-bit
GOOS=darwin GOARCH=amd64    # macOS Intel
GOOS=darwin GOARCH=arm64    # macOS Apple Silicon
GOOS=windows GOARCH=amd64   # Windows 64-bit
GOOS=freebsd GOARCH=amd64   # FreeBSD 64-bit
```

## 🧪 **Testing Builds**

### **Local Testing**

```bash
# Test all platforms (if possible)
./build/migro-linux-amd64 --help
./build/migro-darwin-amd64 --help
./build/migro-darwin-arm64 --help
./build/migro-windows-amd64.exe --help  # On Windows or Wine

# Test installation
make install-user
migro --help
```

### **Docker Testing**

```bash
# Build Docker image
make docker-build

# Test in container
make docker-run

# Test different OS
docker run --rm -v $(pwd):/workspace ubuntu:20.04 /workspace/build/migro-linux-amd64 --help
```

## 🗂️ **File Structure**

```
migro/
├── build/                    # Build outputs (gitignored)
│   ├── migro                # Local build
│   ├── migro-*-*            # Platform builds
│   └── release/             # Release packages
├── cmd/                     # Source code
├── main.go                  # Entry point
├── go.mod                   # Go module
├── Makefile                 # Build automation
├── install.sh               # Installation script
├── README.md                # Main documentation
├── BUILD.md                 # This file
└── migro.example.yaml       # Example config
```

## 📝 **Version Management**

### **Semantic Versioning**

```
v1.2.3
│ │ │
│ │ └── Patch: Bug fixes
│ └──── Minor: New features (backward compatible)
└────── Major: Breaking changes
```

### **Version Examples**

```bash
v1.0.0    # Initial release
v1.0.1    # Bug fix
v1.1.0    # New feature
v2.0.0    # Breaking change
```

### **Pre-release Versions**

```bash
v1.2.0-alpha.1    # Alpha release
v1.2.0-beta.1     # Beta release
v1.2.0-rc.1       # Release candidate
```

## 🚨 **Troubleshooting**

### **Common Issues**

```bash
# Clean everything and rebuild
make clean
go clean -cache
go mod download
make build

# Permission issues
chmod +x build/migro*

# Missing dependencies
make deps
go mod tidy

# Cross-compilation issues
go env GOOS GOARCH
export CGO_ENABLED=0  # Disable CGO for cross-compilation
```

### **Build Errors**

```bash
# Module issues
go mod verify
go mod download

# Import path issues
go list -m all

# Build constraint issues
go build -v .  # Verbose output
```

## 📋 **Release Checklist**

### **Pre-Release**

- [ ] All tests pass: `make test`
- [ ] Code committed and pushed
- [ ] Version updated (if needed)
- [ ] README.md updated
- [ ] Dependencies updated: `go mod tidy`

### **Build**

- [ ] Clean build: `make clean`
- [ ] Multi-platform build: `make build-all`
- [ ] Release packages: `make release`
- [ ] Test local binary: `./build/migro --help`

### **Release**

- [ ] Git tag created: `git tag v1.2.0`
- [ ] Tag pushed: `git push origin v1.2.0`
- [ ] GitHub release created
- [ ] Release files uploaded
- [ ] Release notes written
- [ ] Install script tested

### **Post-Release**

- [ ] Verify install script works
- [ ] Test `go install github.com/ChungNQ511/migro@latest`
- [ ] Update documentation if needed
- [ ] Announce release (if applicable)

## 🔄 **Automation Ideas**

### **GitHub Actions** (Future)

```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags:
      - 'v*'
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
      - run: make release
      - uses: softprops/action-gh-release@v1
        with:
          files: build/release/*
```

### **Automated Versioning**

```bash
# Bump version automatically
git tag $(date +v1.%m.%d)

# Or use tools like:
# - semantic-release
# - goreleaser
# - conventional-changelog
```

---

## 💡 **Pro Tips**

1. **Always test locally** before releasing
2. **Use semantic versioning** consistently  
3. **Keep release notes** detailed and user-friendly
4. **Test install script** on clean systems
5. **Automate repetitive tasks** with Makefile
6. **Version your documentation** along with code
7. **Keep build artifacts** in gitignore
8. **Use consistent naming** for releases

---

**Last Updated**: $(date)
**Project**: github.com/ChungNQ511/migro 