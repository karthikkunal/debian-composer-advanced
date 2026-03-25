# Debian Composer

> **The aim of this project is to bring composability to the Debian ecosystem — enabling true cross-distro innovation.**

## Why Debian Composer?

The Debian ecosystem spans dozens of derivatives — from Ubuntu and Linux Mint to elementary OS, Kali Linux, and many more. Each distro team builds incredible tools, configurations, enhancements, and content. But traditionally, these innovations are locked within their respective ecosystems.

**Debian Composer changes this.**

We believe that a great tool built by one distro team should benefit the entire Debian ecosystem. Whether it's a cutting-edge desktop environment configuration from one derivative, a security hardening setup from another, or a specialized development environment from a third — Debian Composer makes these portable, reusable, and composable across any Debian-based distribution.

## Core Philosophy

### Composability First

At its core, Debian Composer treats system configurations as **composable building blocks**. Instead of monolithic configurations, you write modular recipes that can:

- **Include** other recipes as base layers
- **Extend** existing configurations with customization
- **Layer** multiple concerns on top of each other
- **Mix and match** components from different sources

### Cross-Distro Innovation

With Debian Composer:

- A tool built by the Kali team can be used by security researchers on Ubuntu
- A desktop configuration perfected by the elementary OS team can enhance a Debian system
- A development environment setup from a specialized distro can be shared with the broader community

This isn't about cloning another distro — it's about **borrowing the best ideas** and **composing** your ideal system from proven components.

## Key Features

- **YAML-based Recipes** — Declarative, human-readable system definitions
- **Recipe Composition** — Include, extend, and layer multiple recipes
- **Conditional Installation** — Install packages based on system state or user choices
- **Pre-configured Stacks** — Ready-to-use combinations for common use cases
- **Verification Hooks** — Ensure your system is configured correctly
- **Cross-distro Compatible** — Works on any Debian-based distribution
- **Hardware Detection** — Automatic hardware detection for optimal driver selection
- **Snapper Integration** — System snapshots before changes with rollback support
- **Debian Pure Blends** — Full support for official Debian Pure Blends
- **Opinionated Distros** — Pre-configured setups for common use cases
- **Persona-based Installation** — User type-based package selection
- **Recipe Validation** — Schema validation for recipe files
- **Service Management** — Systemd service control
- **Progress Indicators** — Beautiful progress bars and spinners

## Getting Started

```bash
# Build from source
make build

# Run the binary
./bin/debian-composer-go --help
```

## Commands

### Basic Commands

```bash
# List available recipes
debian-composer available

# Install a recipe
debian-composer install debian-developer

# List installed recipes
debian-composer list

# Remove a recipe
debian-composer remove debian-developer

# Remove but keep config files
debian-composer remove debian-developer --keep-config
```

### Distros and Blends

```bash
# Install an opinionated distro
debian-composer distro debian-developer
debian-composer distro debian-developer --lang=rust

# List available distros
debian-composer distros

# Install a Debian Pure Blend
debian-composer blend debian-edu

# List available blends
debian-composer blends
```

### Personas and Mixins

```bash
# Install for a user persona
debian-composer persona developer

# Add a mixin for specialization
debian-composer persona developer --mixin=ai-ml

# List available personas and mixins
debian-composer personas
debian-composer mixins
```

### Hardware Detection

```bash
# Detect and display hardware
debian-composer probe-hardware

# Show package recommendations based on hardware
debian-composer probe-hardware --recommend
```

### Safety Features (Snapper)

```bash
# Create a snapshot before installation
debian-composer install my-recipe --with-snapshot

# List available snapshots
debian-composer snapshots

# Rollback to a previous snapshot
debian-composer rollback

# Rollback to latest recipe snapshot
debian-composer rollback --recipe
```

### Validation

```bash
# Validate a recipe file
debian-composer validate my-recipe

# Validate all recipes
debian-composer validate-all

# Strict mode (warnings are errors)
debian-composer validate my-recipe --strict
```

### Service Management

```bash
# Check service status
debian-composer service status nginx

# Enable a service
debian-composer service enable nginx

# Start a service
debian-composer service start nginx

# List services
debian-composer service list
debian-composer service list "docker*"
```

## Architecture

The project uses a modular architecture focused on composability. For a detailed guide on how the kitchen system works, see the [Architecture Guide](ARCHITECTURE.md).

- **Recipe Parser** — Parses YAML recipe definitions with caching
- **Resolver** — Resolves includes, extends, layers, variables, and categories
- **Merger** — Combines multiple recipe layers (pure Go, yaml.v3-based anchor expansion)
- **Validator** — Formal JSON Schema validation for all recipes
- **Package Manager Integration** — Handles package installation with batch support (apt, nala, flatpak, snap, etc.)
- **State Manager** — SQLite-based state persistence
- **Hardware Detection** — Parallel hardware detection using system tools
- **Snapper Manager** — System snapshot management with timeline and compare
- **Service Manager** — Systemd service control via taigrr/systemctl
- **CLI** — Cobra-based command-line interface

## Internal Packages

| Package | Description |
|---------|-------------|
| `internal/apt` | Package manager integration (apt, nala, flatpak, snap, etc.) |
| `internal/cli` | CLI command definitions (Cobra) |
| `internal/hardware` | Hardware detection (lshw, lspci, dmidecode) |
| `internal/recipe` | Recipe parsing and resolution |
| `internal/service` | Systemd service management |
| `internal/state` | SQLite state persistence |
| `internal/snapper` | Snapper snapshot management |
| `internal/types` | Data type definitions |
| `internal/ux` | Progress bars, spinners, logging |
| `internal/validation` | Recipe schema validation |

## Dependencies

### FOSS Libraries Used

| Library | Purpose | License |
|---------|---------|---------|
| `github.com/spf13/cobra` | CLI framework | Apache 2.0 |
| `github.com/mattn/go-sqlite3` | SQLite driver | MIT |
| `github.com/jmoiron/sqlx` | Ergonomic SQL (state layer) | MIT |
| `gopkg.in/yaml.v3` | YAML parsing and anchor expansion | MIT |
| `github.com/kaptinlin/jsonschema` | JSON Schema validation | MIT |
| `github.com/taigrr/systemctl` | Systemd bindings | MIT |
| `github.com/charmbracelet/bubbles` | Spinner and progress UI components | MIT |
| `github.com/charmbracelet/bubbletea` | TUI framework | MIT |
| `github.com/charmbracelet/lipgloss` | Terminal styling | MIT |
| `github.com/bluet/syspkg` | Package manager abstraction | MIT |

## Join the Movement

Every Debian derivative has something valuable to contribute. With Debian Composer, your team's innovations aren't limited to your users — they can enrich the entire ecosystem.

Build something great? Share it. Find something useful from another distro? Use it. That's the power of composability.

---

*Debian Composer — Compose your perfect system from the best of the Debian ecosystem.*
