# Packaging with nFPM

This document describes how to build Debian packages for debian-composer-go using [nFPM](https://nfpm.goreleaser.com/).

## Overview

The project uses **nFPM** (Not FPM), a simple packager written in Go, to create `.deb` packages. This approach:

- Requires no external dependencies (no Ruby, no tar)
- Uses a single YAML configuration file (`nfpm.yaml`)
- Supports cross-platform builds
- Can generate multiple package formats (deb, rpm, apk)

## Prerequisites

### Install nFPM

```bash
# Install using Go
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest

# Verify installation
nfpm version
```

Make sure `$GOPATH/bin` is in your `PATH`:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

## Building Packages

### Build Debian Package

```bash
# Build binary and create package
make nfpm-package

# Or manually
go build -o bin/debian-composer-go ./cmd/debian-composer
nfpm package --packager deb --target debian-composer.deb
```

### Build Multi-Arch Packages

```bash
# Build for amd64 and arm64
make nfpm-package-all
```

### Package Output

Packages are created in the `dist/` directory:

```
dist/
├── debian-composer_v1.0.0_linux_amd64.deb
└── debian-composer_v1.0.0_linux_arm64.deb
```

## Installation

### Install the Package

```bash
# Install using dpkg
sudo dpkg -i dist/debian-composer_*.deb

# Or using apt (handles dependencies)
sudo apt install ./dist/debian-composer_*.deb
```

### Verify Installation

```bash
# Check package info
dpkg -l | grep debian-composer

# List installed files
dpkg -L debian-composer

# Run the tool
debian-composer --version
```

### Uninstall

```bash
# Remove package (keep config)
sudo dpkg -r debian-composer

# Purge package (remove config too)
sudo dpkg -P debian-composer
```

## Package Contents

The package includes:

| Path | Description |
|------|-------------|
| `/usr/bin/debian-composer` | Main binary |
| `/usr/share/debian-composer/kitchen/` | Pre-configured recipes |
| `/var/lib/debian-composer/` | State directory (created on install) |
| `/workspace/kitchen` | Symlink to kitchen (created on install) |

## Configuration

### nfpm.yaml

The package is configured in [`nfpm.yaml`](../nfpm.yaml):

```yaml
name: "debian-composer"
version: "${VERSION}"
maintainer: "Debian Composer Team <team@debian-composer.org>"
description: |
  Debian Composer - Bring composability to the Debian ecosystem
depends:
  - apt
  - ca-certificates
  - curl
  - gnupg
  - jq
```

### Maintainer Scripts

- **postinst**: Runs after installation (creates directories, symlinks)
- **prerm**: Runs before removal (cleans up state)

Located in [`scripts/`](../scripts/).

## CI/CD Integration

The GitLab CI pipeline automatically builds packages:

- `package-deb-nfpm`: Builds amd64 `.deb` package
- `package-multiarch-nfpm`: Builds amd64 and arm64 packages
- `package-rpm-nfpm`: Builds RPM package (optional)
- `lintian-check`: Validates Debian policy compliance

Packages are uploaded as artifacts and available for tagged releases.

## Troubleshooting

### nFPM not found

```bash
# Install nFPM
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest

# Add to PATH
export PATH=$PATH:$(go env GOPATH)/bin
```

### Build fails

```bash
# Clean and rebuild
make clean
make nfpm-package
```

### Package validation errors

```bash
# Check package info
dpkg-deb -I dist/debian-composer_*.deb

# List contents
dpkg -c dist/debian-composer_*.deb

# Run lintian
lintian dist/debian-composer_*.deb
```

## Advanced Usage

### Custom Version

```bash
export VERSION=1.0.0
make nfpm-package
```

### Build RPM Package

```bash
nfpm package --packager rpm --target debian-composer.rpm
```

### Build APK Package (Alpine)

```bash
nfpm package --packager apk --target debian-composer.apk
```

## References

- [nFPM Documentation](https://nfpm.goreleaser.com/)
- [nFPM GitHub](https://github.com/goreleaser/nfpm)
- [Debian Policy Manual](https://www.debian.org/doc/debian-policy/)
