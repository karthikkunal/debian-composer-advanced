# Debian Composer

> **Compose your ideal Debian system with modular, reusable recipes.**

[![Go Version](https://img.shields.io/github/go-mod/go-version/debian-composer/debian-composer-go)](https://golang.org/)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/debian-composer/debian-composer-go)](https://github.com/debian-composer/debian-composer-go/releases)

## 🚀 Overview

**Debian Composer** is a powerful post-install configuration and recipe manager for Debian-based systems. Create modular, composable recipes that define your perfect system setup — from development environments to production servers — and share them across the entire Debian ecosystem.

Whether you're managing a fleet of servers, standardizing team workstations, or building custom distro spins, Debian Composer gives you the tools to automate, version, and reproduce system configurations with ease.

## 💡 Why Debian Composer?

### The Problem
The Debian ecosystem spans dozens of derivatives — Ubuntu, Linux Mint, elementary OS, Kali Linux, Pop!_OS, and many more. Each distro team builds incredible tools, configurations, and content. But traditionally, these innovations are **locked within their respective ecosystems**.

System administrators and power users face:
- ❌ Repetitive manual setup across machines
- ❌ No standard way to share configurations
- ❌ Fragmented knowledge across distro communities
- ❌ Difficult to reproduce exact system states

### The Solution
**Debian Composer changes this.** It enables:

✅ **Cross-distro collaboration** — A tool built by one distro team can be used by others  
✅ **Modular configurations** — Desktop configurations from one derivative can enhance another  
✅ **Shared knowledge** — Development environment setups from specialized distros benefit the broader community  
✅ **Reproducible systems** — Define, version, and recreate exact system states  

## ✨ Key Features

### 📦 Recipe Management
- **YAML-based Recipes** — Declarative, human-readable system definitions
- **Recipe Composition** — Powerful multi-level composition with `include`, `extend`, and `layer` directives
- **Deep Merging** — Stackable layers that can override and augment any configuration
- **Import/Export** — Easy sharing with standalone bundling that includes all dependencies
- **Dependency Resolution** — Automatic handling of recipe dependencies in correct order

### 🔒 Reliability & Safety
- **Schema Validation** — Built-in JSON Schema validation catches errors before deployment
- **SQLite State Management** — Tracks installed recipes, versions, and installation timestamps
- **Snapper Integration** — Create system snapshots before installations for easy rollback
- **Dry-run Mode** — Preview changes without applying them
- **Conflict Detection** — Automatically detect and prevent conflicting recipe installations

### 🎯 Advanced Capabilities
- **Conditional Installation** — Install packages based on system state, hardware, or user choices
- **Pre-configured Stacks** — Ready-to-use combinations for common use cases (dev, server, desktop)
- **Phase-based Execution** — Separate install, configure, and verify phases for granular control
- **Verification Hooks** — Post-install validation to ensure correct configuration
- **User & Group Management** — Automated user and group creation during configuration

### 🛠 Package Manager Integration
- **Nala Support** — Modern apt frontend with parallel downloads, colored output, and history tracking
- **Multi-backend** — Support for apt, nala, Flatpak, and Snap packages
- **Batch Operations** — Efficient package installation with dependency resolution

### 🌐 Cross-distro Compatibility
Works seamlessly on any Debian-based distribution:
- Debian
- Ubuntu and all official flavors
- Linux Mint
- elementary OS
- Kali Linux
- Pop!_OS
- MX Linux
- And many more...

## 🎯 Use Cases

### For System Administrators
- **Standardize workstation setups** across your organization
- **Automate server provisioning** with version-controlled recipes
- **Maintain consistency** between development, staging, and production

### For Distro Teams
- **Share innovations** with the broader Debian ecosystem
- **Provide official recipes** for common use cases (gaming, development, multimedia)
- **Enable community contributions** without fragmenting your base system

### For Developers
- **Reproducible dev environments** — onboard new team members in minutes
- **Project-specific tooling** — define exact tools needed per project
- **Easy switching** — switch between different development configurations

### For Power Users
- **Document your setup** — turn your perfect configuration into shareable recipes
- **Experiment safely** — try new configurations with snapshot support
- **Mix and match** — combine recipes from different sources

## 🚀 Quick Start

### Installation

#### From Source
```bash
git clone https://github.com/debian-composer/debian-composer-go.git
cd debian-composer-go
make build
sudo cp bin/debian-composer /usr/local/bin/
```

#### Using Make
```bash
make build
make install  # Installs to /usr/local/bin
```

### Basic Commands

```bash
# Show help
debian-composer --help

# List available recipes
debian-composer available

# Install a recipe
debian-composer install my-recipe

# Install with variables
debian-composer install my-recipe --var "editor=code" --var "theme=dark"

# Install using a pre-configured stack
debian-composer install dev-workstation --stack developer

# Install specific categories only
debian-composer install my-recipe --category editors,terminals

# Apply security hardening
debian-composer install server --apply-security

# Create snapshot before installation (requires Snapper)
debian-composer install my-recipe --with-snapshot

# Preview without applying changes
debian-composer install my-recipe --dry-run

# Import an external recipe
debian-composer import path/to/recipe.yaml

# Export a recipe as a standalone bundle
debian-composer export my-recipe ./standalone.yaml --standalone

# Validate a recipe against schema
debian-composer validate my-recipe

# Validate all recipes in kitchen
debian-composer validate --all

# List installed recipes
debian-composer list

# Show recipe details
debian-composer info my-recipe

# Remove a recipe
debian-composer remove my-recipe

# Switch between recipes
debian-composer switch different-recipe
```

## 📖 Recipe Example

```yaml
name: dev-workstation
version: 1.0.0
description: Complete development workstation setup

variables:
  editor:
    default: "code"
    description: "Preferred code editor (code, vim, neovim)"
  theme:
    default: "dark"
    description: "UI theme preference"

packages:
  - git
  - curl
  - wget
  - build-essential
  - "{{ .editor }}"

categories:
  - name: editors
    packages:
      - code
      - vim
      - neovim

  - name: terminals
    packages:
      - alacritty
      - tmux

install:
  pre:
    - echo "Setting up development environment..."
  post:
    - systemctl enable docker

configure:
  commands:
    - echo "Configuring user preferences..."
  userGroups:
    - developers

verify:
  commands:
    - git --version
    - docker --version

conflicts:
  - minimal-workstation
```

## 📚 Documentation

- **[Architecture Guide](docs/ARCHITECTURE.md)** — Deep dive into Kitchen, Recipes, and Layers
- **[Full Documentation](docs/README.md)** — Complete user and developer documentation
- **[Scripts](scripts/README.md)** — Build and utility scripts reference
- **[Tests](tests/README.md)** — Test cases and examples

## 🏗 Project Structure

```
debian-composer-go/
├── cmd/debian-composer/    # CLI entry point
├── internal/
│   ├── apt/                 # Package manager integration (apt, nala, flatpak, snap)
│   ├── cli/                 # CLI commands (install, remove, import, export, etc.)
│   ├── recipe/              # Recipe parsing, resolution, merging, dependency handling
│   ├── state/               # SQLite state management
│   ├── hook/                # Hook execution engine
│   ├── snapper/             # System snapshot integration
│   └── types/               # Type definitions
├── kitchen/                 # Recipe storage (default location)
├── docs/                    # Documentation
├── scripts/                 # Build and utility scripts
└── tests/                   # Test files and examples
```

## 📋 Requirements

- **Go** 1.26+ (for building from source)
- **Operating System**: Debian-based Linux distribution
- **Package Manager**: APT (Nala optional but recommended)
- **Optional**: Snapper for system snapshots

## 🤝 Contributing

We welcome contributions from the community! Whether it's bug reports, feature requests, documentation improvements, or code contributions — every contribution helps make Debian Composer better.

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute.

### Ways to Contribute
- 🐛 Report bugs and suggest features
- 📝 Improve documentation
- 🍴 Submit recipe examples
- 💻 Contribute code improvements
- 🌍 Help with translations (future feature)

## 🗺 Roadmap

See [ROADMAP.md](ROADMAP.md) for planned features and development direction.

**Upcoming Features:**
- Remote recipe repositories
- Recipe gallery and discovery
- Enhanced conditional logic
- Multi-distro testing framework
- Plugin system for custom package managers

## 📜 Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and recent changes.

**Recent Improvements:**
- Native YQ resolver (no external binary needed)
- Go-based git hooks (replaced shell scripts)
- Enhanced dependency resolution
- Improved UX with Charm libraries

## 📄 License

This project is licensed under the **GNU General Public License v3.0 (GPL-3.0)** — see the [LICENSE](LICENSE) file for details.

The GPL-3.0 license ensures that Debian Composer remains free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation. This copyleft license guarantees that derivative works must also be distributed under the same license terms, protecting the freedom of all users.

## 🙏 Acknowledgments

Built with amazing open-source libraries:
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [Charm](https://github.com/charmbracelet) — Beautiful TUI components
- [SQLx](https://github.com/jmoiron/sqlx) — Database utilities
- [JSONSchema](https://github.com/kaptinlin/jsonschema) — Schema validation

---

**Ready to compose your perfect system?** Get started with `debian-composer --help` or explore example recipes in the documentation.