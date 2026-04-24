# Kitchen

This directory contains all YAML-based recipe definitions for debian-composer.

## Directory Structure

```
kitchen/
├── distros/         # Opinionated named system configurations (--distro=NAME)
│                    # Start here — the Omarchy-style tasteful entry point
│   ├── debian-developer.yaml   # Developer workstation
│   ├── debian-creator.yaml     # Creative workstation
│   ├── debian-scientist.yaml   # Scientific/research workstation
│   ├── debian-homelab.yaml    # Self-hosted homelab server
│   └── debian-devuan.yaml       # Devuan/systemd-free workstation
│
├── blends/          # Debian Pure Blend recipes (--blend=NAME)
│                    # Official Debian domain-specific software collections
│   ├── debian-astro.yaml
│   ├── debian-edu.yaml
│   ├── debian-gis.yaml
│   ├── debian-med.yaml
│   └── debian-science.yaml
│
├── recipes/         # Individual named recipes (--recipe=NAME)
│   └── template-example.yaml   # Config file templating example
│
├── personas/        # User archetype recipes (--persona=NAME)
│   ├── base/        # Base persona definitions (developer, student, sysadmin)
│   └── mixins/      # Persona specializations (--mixin=NAME)
│
├── components/      # Reusable sub-units (referenced via includes:)
│   ├── bases/       # System foundation recipes (desktop, server, minimal)
│   ├── devuan-backend.yaml  # Devuan APT backend for sysvinit systems
│   ├── nginx.yaml
│   ├── postgresql.yaml
│   ├── ollama.yaml
│   └── ...
│
├── stacks/          # Pre-configured category combinations
│
├── fragments/       # YAML anchor/template files (not standalone recipes)
│   ├── base-anchors.yaml       # Package list anchors
│   ├── service-templates.yaml  # Service configuration templates
│   ├── devuan-init.yaml       # SysVinit/OpenRC/Runit service templates
│   └── blend-anchors.yaml      # Pure Blend-specific anchors
│
├── configs/         # Custom configuration overlays (--config=NAME)
│
└── examples/        # Example recipes for reference
```

## Where to Start

**For users:** Start with `--distro` for an opinionated complete setup.

```bash
# List everything available
debian-composer --list-available

# Get details on a distro
debian-composer --info=debian-developer

# Install a distro
sudo debian-composer --distro=debian-developer
sudo debian-composer --distro=debian-scientist --var=field=astronomy
sudo debian-composer --distro=debian-homelab
```

**For contributors:** See `docs/developer/recipe-authoring-guide.md` for how to write recipes, when to use `includes:` vs YAML anchors, and the full component reference.

## YAML Anchors & `includes:`

This project uses two complementary reuse mechanisms:

- **`includes:`** — runtime merge of another YAML file into the current recipe. Use for cross-file reuse (components, fragments).
- **YAML anchors** (`&name` / `*name`) — parse-time alias within a single file. Use for within-file deduplication.

See `docs/developer/recipe-authoring-guide.md` for full documentation.

### Fragment Anchors Quick Reference

```yaml
includes:
  - fragments/base-anchors.yaml      # Common package lists
  - fragments/service-templates.yaml  # Service templates

packages: *blend_essentials          # git, curl, wget, ca-certificates, gnupg
packages: *dev_packages              # essentials + build-essential, make
packages: *python_base               # python3, python3-pip, python3-venv

categories:
  nginx:
    <<: *service_template            # Standard service category template
  mysql:
    <<: *mysql_service               # Pre-configured MySQL template
```

## Recipe Structure

All recipes follow a standardized YAML structure:

```yaml
name: recipe-name
version: 1.0.0
description: "Human-readable description"
author: Debian Composer Team
license: MIT
tags: [tag1, tag2]

includes:
  - components/bases/desktop.yaml    # Optional: pull in components
  - fragments/base-anchors.yaml      # Optional: pull in YAML anchors

variables:
  my_var:
    type: string
    default: value
    choices: [a, b, c]
    description: "What this controls"

packages:
  - always-installed-package

categories:
  category-name:
    condition: my_var == a           # Optional: conditional installation
    description: "What this category does"
    install:
      packages:
        - package-name
    configure:
      service:
        name: service-name
        enable: true
    verify:
      commands:
        - command --version

stacks:
  default:
    description: "Most common setup"
    categories:
      - category-name

requirements:
  min_ram: 2048   # MB
  min_disk: 10    # GB

post_install:
  - "Key info the user needs right now"
```

## Devuan / Systemd-Free Systems

Debian Composer supports Devuan and other systemd-free Debian derivatives as first-class targets.

```bash
# Install Devuan workstation
sudo debian-composer --distro=debian-devuan

# With nala APT frontend (Devuan 5+)
sudo debian-composer --distro=debian-devuan --stack=devuan-nala

# Server (no desktop)
sudo debian-composer --distro=debian-devuan --stack=minimal-server
```

The `debian-devuan` distro recipe includes:
- `components/devuan-backend.yaml` — Devuan APT sources and apt.conf
- `fragments/devuan-init.yaml` — SysVInit/OpenRC/Runit service templates
- Automatic `init_system` detection via `internal/initsys.Detect()`

Conditionals in recipes:

```yaml
variables:
  init_system:
    type: string
    default: sysvinit

categories:
  nginx:
    condition: init_system == sysvinit or init_system == openrc or init_system == runit
    description: "Nginx (sysvinit compatible)"
    configure:
      hooks:
        post_install:
          - type: command
            command: "update-rc.d nginx defaults 2>/dev/null || true"
```

See `docs/DEVuan.md` for full documentation.

## Config File Templating

Use the `template:` hook type to generate config files with Go `text/template` at installation time:

```yaml
hooks:
  post_install:
    - type: template
      command: |
        cat > /etc/nginx/sites-available/{{ .domain }}.conf << 'EOF'
        server {
            listen 80;
            server_name {{ .domain }};
            root {{ .webroot }};
        }
        EOF
```

Available template functions (Sprig library):
- **String:** `upper`, `lower`, `title`, `trim`, `quote`, `default`
- **Math:** `add`, `sub`, `mul`, `div`, `mod`
- **Date:** `now`, `date` (e.g., `{{ now | date "2006-01-02" }}`)
- **Collection:** `list`, `first`, `last`, `reverse`, `sort`
- **Encoding:** `b64enc`, `sha256`, `md5`
- **Advanced:** `toJson`, `toYaml`, `fromJson`

See `recipes/template-example.yaml` for full examples including conditionals, range iteration, and nested variables.

## Cross-Blend Composition

Pure Blend recipes use `include/extend/layer` for composability:

```yaml
name: science-workstation
extends: debian-science     # Inherit all science packages
layers:
  - components/gpu-compute  # Add GPU support
  - components/ai          # Add AI/ML tools

variables:
  enable_gpu: true
```
