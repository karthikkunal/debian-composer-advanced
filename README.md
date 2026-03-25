# Debian Composer

> **Bring composability to the Debian ecosystem — enabling true cross-distro innovation.**

[![Go Version](https://img.shields.io/github/go-mod/go-version/debian-composer/debian-composer-go)](https://golang.org/)
[![License](https://img.shields.io/github/license/debian-composer/debian-composer-go)](LICENSE)

## Overview

Debian Composer is a tool for composing, sharing, and reusing system configurations across the Debian ecosystem. It allows distro teams and users to create modular, composable recipes that can be mixed and matched to build custom systems.

## Why Debian Composer?

The Debian ecosystem spans dozens of derivatives — Ubuntu, Linux Mint, elementary OS, Kali Linux, and many more. Each distro team builds incredible tools, configurations, and content. But traditionally, these innovations are locked within their respective ecosystems.

**Debian Composer changes this.** A tool built by one distro team can be used by others. A desktop configuration from one derivative can enhance another. A development environment setup from a specialized distro can be shared with the broader community.

## Features

- **YAML-based Recipes** — Declarative, human-readable system definitions
- **Recipe Composition** — Powerful multi-level composition (Include, Extend, and Layer)
- **Deep Merging** — Stackable layers that can override and augment any configuration
- **Recipe Management** — Easy import and export of recipes with standalone bundling
- **Schema Validation** — Built-in JSON Schema validation for all recipes and components
- **Conditional Installation** — Install packages based on system state or user choices
- **Pre-configured Stacks** — Ready-to-use combinations for common use cases
- **Verification Hooks** — Ensure your system is configured correctly
- **Cross-distro Compatible** — Works on any Debian-based distribution
- **SQLite State Management** — Tracks installed recipes and their states
- **Nala Integration** — Modern apt frontend (default) with parallel downloads, colored output, history, and rollback

## Installation

### From Source

```bash
git clone https://github.com/debian-composer/debian-composer-go.git
cd debian-composer-go
make build
```

### Quick Start

```bash
# Show help
debian-composer --help

# Install a recipe
debian-composer install my-recipe

# Import an external recipe
debian-composer import path/to/recipe.yaml

# Export a recipe as a standalone bundle
debian-composer export my-recipe ./standalone.yaml --standalone

# Validate a recipe
debian-composer validate my-recipe
```

## Documentation

- [Architecture Guide](docs/ARCHITECTURE.md) — Detailed guide on Kitchen, Recipes, and Layers
- [Documentation](docs/README.md) — Project documentation
- [Scripts](scripts/README.md) — Build and utility scripts
- [Tests](tests/README.md) — Test cases and examples

## Project Structure

```
debian-composer-go/
├── cmd/debian-composer/    # CLI entry point
├── internal/
│   ├── apt/                 # Package manager integration (apt, nala, flatpak, snap, etc.)
│   ├── cli/                 # CLI commands
│   ├── recipe/              # Recipe parsing, resolution, merging
│   ├── state/               # SQLite state management
│   └── types/               # Type definitions
├── docs/                    # Documentation
├── scripts/                 # Build scripts
└── tests/                   # Test files
```

## Requirements

- Go 1.26+
- Debian-based Linux distribution
- APT package manager (or Nala as an optional frontend)

## License

MIT License — see LICENSE file for details.

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Roadmap

See [ROADMAP.md](ROADMAP.md) for planned features and development direction.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.