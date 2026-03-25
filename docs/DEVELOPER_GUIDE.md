# Developer Guide

## Getting Started

### Prerequisites

- Go 1.21+
- SQLite3
- Git

### Setup

```bash
# Clone the repository
git clone https://github.com/debian-composer/debian-composer-go.git
cd debian-composer-go

# Install dependencies
go mod download

# Build
make build

# Run tests
make test
```

### Project Structure

```
debian-composer-go/
├── cmd/
│   └── debian-composer/    # Main application entry point
├── internal/
│   ├── apt/               # Package manager integration (apt, nala, flatpak, snap, etc.)
│   ├── cli/               # CLI commands (Cobra)
│   ├── component/         # Component registry
│   ├── conditions/        # Condition evaluation
│   ├── config/            # Configuration management
│   ├── desktop/           # Desktop environment support
│   ├── hardware/          # Hardware detection
│   ├── hook/              # Hook execution
│   ├── hooks/             # Git hooks
│   ├── pkg/               # Package utilities
│   ├── pkgmgr/            # Multi-package-manager support
│   ├── profile/           # System profiles
│   ├── recipe/            # Recipe parsing and resolution
│   ├── service/           # Systemd service management
│   ├── snapper/           # Snapper integration
│   ├── state/             # SQLite state management
│   ├── sysconfig/         # System configuration (SSH, UFW, fail2ban)
│   ├── tui/               # Terminal UI
│   ├── types/             # Type definitions
│   ├── ux/                # User experience utilities
│   └── validation/        # Schema validation
├── kitchen/               # Default recipe kitchen
├── docs/                  # Documentation
├── hooks/                 # Git hooks
└── tests/                 # Test files
```

## Adding a New Command

1. Create a new file in `internal/cli/`:

```go
package cli

import (
    "fmt"
    "github.com/spf13/cobra"
)

func init() {
    myCmd := &cobra.Command{
        Use:   "mycommand [args]",
        Short: "Description of command",
        RunE:  runMyCommand,
    }
    rootCmd.AddCommand(myCmd)
}

func runMyCommand(cmd *cobra.Command, args []string) error {
    fmt.Println("Running my command!")
    return nil
}
```

## Adding a New Package Manager

1. Add package type in `internal/pkgmgr/pkgmgr.go`:

```go
const PackageMyManager PackageType = "mymanager"
```

2. Implement Install/Remove/IsInstalled methods:

```go
func (m *Manager) installMyManager(pkg Package) error {
    // Implementation
    return nil
}
```

3. Update `APT.Install()` to handle the new type.

## Adding a New Desktop Environment

1. Add to `internal/desktop/desktop.go`:

```go
var Environments = map[string]Environment{
    "myde": {
        Name:        "myde",
        Description: "My Desktop Environment",
        MinRAMMB:    2048,
        MinDiskGB:   10,
        RequiresGPU: false,
        Packages:    []string{"myde-desktop"},
    },
}
```

## Testing

```bash
# Run all tests
go test ./...

# Run specific package
go test ./internal/recipe -v

# Run with coverage
go test -cover ./...

# Run pre-push checks
./hooks/pre-push
```

## Code Style

- Use `go fmt` for formatting
- Use `go vet` for static analysis
- Add tests for new functionality
- Follow standard Go conventions

## Building

```bash
# Build binary
make build

# Build with version info
make build VERSION=1.0.0

# Build deb package
make deb
```

## Git Hooks

The project uses Go-based git hooks for fast, reliable checks.

### Pre-commit Hook
Runs on `git commit`:
- `gofmt` - Go code formatting (blocking)
- `gobuild` - Go compilation (blocking)
- `govet` - Go static analysis
- `permissions` - File permissions
- `yamllint` - YAML linting (blocking)

### Pre-push Hook
Runs on `git push`:
- Go build, test, vet, fmt
- StaticCheck (if installed)
- Binary smoke test
- Test coverage (70% threshold)
- YAML linting
- Blends validation
- Documentation presence

### Hook Configuration

Disable specific checks:
```bash
ENABLE_YAMLLINT_CHECK=false git commit -m "message"
```

Enable debug mode:
```bash
DEBUG=true git push
```

Use custom tool paths:
```bash
YAMLLINT_BIN=/usr/local/bin/yamllint git commit -m "message"
```

Set minimum coverage:
```bash
MIN_COVERAGE=80 git push
```

Disable colors:
```bash
NO_COLOR=1 git commit -m "message"
```

## Committing

1. Pre-commit hooks run automatically on `git commit`

2. Run pre-push checks manually:
   ```bash
   ./hooks/pre-push
   ```

3. Ensure all tests pass:
   ```bash
   make test
   ```

## Documentation

- Add docstrings to public functions
- Update relevant markdown docs in `docs/`
- Update this guide for significant changes
