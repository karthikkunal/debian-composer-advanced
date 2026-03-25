# User Guide

Debian Composer is a post-installation configuration toolkit for Debian systems.

## Installation

```bash
# From source
git clone https://github.com/debian-composer/debian-composer-go.git
cd debian-composer-go
make build
sudo make install
```

## Quick Start

```bash
# Show help
debian-composer --help

# Install a recipe
debian-composer install my-recipe

# Install with variables
debian-composer install my-recipe --var=theme=dark

# Dry run (preview changes)
debian-composer install my-recipe --dry-run
```

## Recipe Concepts

### Recipes
YAML files defining software packages, configurations, and hooks.

### Blends
Curated recipes for specific use cases (education, science, gaming).

### Personas
User-type templates (developer, student, creative professional).

### Mixins
Specialization overlays that add features to existing setups.

### Components
Reusable building blocks (desktop environments, editors, services).

## Common Commands

| Command | Description |
|---------|-------------|
| `install <recipe>` | Install a recipe |
| `remove <recipe>` | Remove an installed recipe |
| `list` | List installed recipes |
| `gallery` | Browse available recipes |
| `blend <name>` | Install a Debian Pure Blend |
| `persona <name>` | Install a persona |
| `probe-hardware` | Detect hardware |
| `system` | Configure system security |
| `validate <recipe>` | Validate recipe syntax |
| `hooks install` | Install git hooks |

## Recipe Composition

Recipes can include other recipes:

```yaml
includes:
  - base-desktop    # Include component
  - dev-tools       # Include component

extends: parent-recipe  # Inherit from parent

layers:
  - overlay-1       # Apply overlay
  - overlay-2

packages:
  - git
  - curl
```

## Variables

Define variables with defaults and choices:

```yaml
variables:
  theme:
    type: string
    default: light
    choices: [light, dark]
    description: "UI theme"
  editor:
    type: string
    default: vim
    description: "Default editor"
```

Use variables in conditions:

```yaml
categories:
  dark-theme:
    condition: "theme == dark"
    packages:
      - gnome-themes-extra
```

## Hooks

Recipes can define pre/post install scripts:

```yaml
install:
  pre:
    - "echo 'Starting installation...'"
  post:
    - "systemctl enable sshd"

configure:
  commands:
    - "update-alternatives --set editor /usr/bin/vim.basic"
```

## Safety Features

- **Dry-run mode**: Preview changes without applying
- **Snapshot integration**: Create Snapper snapshots before changes
- **Conflict detection**: Prevents conflicting recipe installations
- **Dependency resolution**: Automatically installs dependencies

## Configuration Files

Dotfile management:

```bash
# Apply dotfiles
debian-composer config apply

# List dotfile status
debian-composer config list

# Show differences
debian-composer config diff .bashrc
```

## Advanced Usage

### Multi-package-manager Support

Packages can be prefixed with package manager:

```yaml
packages:
  - git                    # APT (or Nala if preferred)
  - flatpak:org.gimp.GIMP  # Flatpak
  - snap:code              # Snap
  - pip:requests           # Python pip
  - npm:typescript         # Node npm
```

When Nala is installed and set as the preferred package manager (via `preferred_package_manager: nala`), APT operations use Nala’s parallel downloads, colored output, and transaction history. You can configure Nala’s behavior with variables like `nala_parallel_downloads`, `nala_fetch_timeout`, etc. (see `essential-tools.yaml`).

### Conditional Installation

```yaml
categories:
  laptop-tools:
    condition: "hardware.is_laptop"
    packages:
      - tlp
      - powertop
```

### Hardware-aware Recipes

```yaml
requirements:
  min_ram: 8192
  gpu: true
```

## Troubleshooting

### Permission Errors
Run with `sudo` for system-wide installation.

### Package Installation Fails
Check apt cache: `sudo apt update` or `sudo nala update` (if using Nala)

### Recipe Validation Errors
Run `debian-composer validate <recipe>` for details.

## Getting Help

- Documentation: `docs/README.md`
- Architecture: `docs/ARCHITECTURE.md`
- Examples: `kitchen/recipes/`
