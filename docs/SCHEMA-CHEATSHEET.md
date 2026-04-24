# Recipe Schema Cheatsheet

A one-page reference for writing Debian Composer YAML recipes.

## Minimal Recipe

```yaml
name: my-recipe
kind: recipe
version: 1.0.0
description: "What this does"

packages:
  - git
  - curl
```

## All Fields

```yaml
# ─── Identity ───────────────────────────────────────────────────────────
name: my-recipe          # Unique name (required)
kind: recipe           # recipe | distro | blend | persona | component | fragment
version: 1.0.0       # Semantic version
description: "..."       # One-line summary
author: Name           # Author name
license: MIT           # License (MIT, GPL-2.0+, Apache-2.0)
homepage: https://...     # Homepage URL
tags: [tag1, tag2]    # Tags for discovery

# ─── Composition ──────────────────────────────────────────────────────
# include pulls in YAML anchors and definitions from another file
includes:
  - fragments/base-anchors.yaml       # Package list anchors
  - fragments/service-templates.yaml  # Service templates

# extend inherits all fields from a parent recipe (single)
extends: base-recipe

# layers overlay on top of the recipe (multiple, last wins)
layers:
  - layer-gpu
  - layer-dev

# ─── Variables ──────────────────────────────────────────────────────────
variables:
  port:
    type: int               # string | bool | int | choice | array | object
    default: 8080          # Default value
    description: "Port"      # Human-readable description
  env:
    type: choice
    default: prod
    choices: [dev, staging, prod]  # Constrained choices
  packages:
    type: array
    default: [nginx]         # Default packages
  enable_tls:
    type: bool
    default: false

# ─── Packages ─────────────────────────────────────────────────────────
packages:
  - git
  - curl
  - build-essential
  - "{{ .editor }}"        # Variable expansion

# ─── Categories (Conditional Packages) ─────────────────────────
categories:
  base:
    condition: ''           # Empty = always installed
    description: "Base packages"
    install:
      packages:
        - git
        - curl
    verify:
      commands:
        - git --version
        - curl --version

  web-server:
    condition: enable_tls == false
    description: "Web server (no TLS)"
    install:
      packages:
        - nginx
    configure:
      service:
        name: nginx
        enable: true
        start: true
      hooks:
        post_install:
          - type: command
            command: "update-rc.d nginx defaults"
          - type: template
            command: "cat > /etc/nginx/sites-available/{{ .domain }}.conf << 'EOF'\nserver { listen 80; }\nEOF"
    verify:
      commands:
        - nginx -v
      files_exist:
        - /etc/nginx/nginx.conf

  with-tls:
    condition: enable_tls == true
    description: "Web server with TLS"
    install:
      packages:
        - nginx
        - certbot

# ─── Stacks (Pre-configured Combinations) ────────────────────────
stacks:
  minimal:
    description: "Minimal setup"
    categories:
      - base

  full-stack:
    description: "Full web server"
    categories:
      - base
      - web-server

# ─── Hooks ──────────────────────────────────────────────────────
hooks:
  pre_install:
    - type: apt
      action: update

  post_install:
    - type: command
      command: "systemctl daemon-reload"
      ignore_errors: false
      timeout: 60

    - type: template
      command: |
        cat > /etc/myapp.conf << 'EOF'
        port = {{ .port }}
        env = {{ .env }}
        EOF

    - type: script
      script: scripts/my-setup.sh
      timeout: 300

    - type: service
      action: restart
      service: nginx

# ─── Configure ───────────────────────────────────────────────────
configure:
  user_groups:
    - developers
    - data-science
  commands:
    - update-alternatives --set editor /usr/bin/vim.basic
  hooks:
    pre_configure:
      - type: command
        command: "mkdir -p /var/lib/myapp"
    post_configure:
      - type: command
        command: "chown -R myapp:myapp /var/lib/myapp"

# ─── Verify ────────────────────────────────────────────────────
verify:
  commands:
    - "nginx -v"
    - "curl -s http://localhost:{{ .port }}/health"
  services_running:
    - nginx
  ports_listening:
    - "{{ .port }}"
  files_exist:
    - /etc/myapp.conf

# ─── Requirements ─────────────────────────────────────────────
requirements:
  min_ram: 1024       # MB
  min_disk: 10        # GB
  min_cpu: 2          # cores
  gpu: false
  debian_version: "11"  # Minimum Debian version

# ─── Dependencies & Conflicts ───────────────────────────────
dependencies:
  - base-recipe
  - common-config

conflicts:
  - old-recipe
  - incompatible-setup

# ─── Post-Install Message ─────────────────────────────────
post_install:
  - "=== Installation Complete ==="
  - "Port: {{ .port }}"
  - "Run: sudo systemctl start myapp"
```

## Hook Types

| Type | Fields | Description |
|------|--------|-----------|
| `apt` | `action`: update, install, remove | APT package operations |
| `command` | `command`: shell command | Execute shell command |
| `template` | `command`: Go template | Render template, execute output |
| `script` | `script`: relative path | Run script in recipe |
| `service` | `action`, `service` | Start/stop/restart/enable service |

## Condition Expressions

```yaml
condition: enable_tls == true
condition: port > 1024
condition: '"web" in tags'
condition: init_system == sysvinit or init_system == openrc
condition: os_id == devuan
condition: has_display == true
```

Built-in variables: `os_id`, `os_id_like`, `init_system`, `has_display`, `is_root`, `hostname`, `vars.*`

## Template: Hook Example

```yaml
variables:
  domain:
    type: string
    default: localhost
  port:
    type: int
    default: 8080

hooks:
  post_install:
    - type: template
      command: |
        cat > /etc/app.conf << 'EOF'
        domain = {{ .domain }}
        port   = {{ .port }}
        {{ if .enable_tls }}
        tls    = enabled
        {{ end }}
        generated_at = {{ now | date "2006-01-02 15:04:05" }}
        EOF
```

## Extend & Layers

```yaml
# Parent recipe
name: base-workstation
packages:
  - git
  - curl
variables:
  enable_desktop:
    type: bool
    default: false

# Child recipe
name: dev-workstation
extends: base-workstation         # Inherits all fields
packages:
  - build-essential           # Added to parent's packages
  - "{{ .editor }}"          # Variable expansion
variables:
  enable_desktop:
    default: true           # Override parent's variable
  editor:
    type: choice
    default: code
    choices: [code, vim, neovim]
layers:
  - components/python-dev    # Overlay adds/overrides
  - components/docker      # Last layer wins for conflicts
```

## Common Patterns

### Service Category

```yaml
  nginx:
    <<: *service_template     # Merge in template
    install:
      packages:
        - nginx
    configure:
      service:
        name: nginx
        enable: true
        start: true
    verify:
      commands:
        - nginx -v
      ports_listening:
        - 80
```

### Conditional Init System

```yaml
  ssh:
    install:
      packages:
        - openssh-server
    configure:
      hooks:
        post_install:
          - type: command
            command: >
              init=$(cat /proc/1/comm) &&
              case "$init" in
                systemd) systemctl enable ssh ;;
                openrc-init) rc-update add ssh default ;;
                *) update-rc.d ssh enable ;;
              esac
    verify:
      commands:
        - ssh -V
```

## YAML Anchor Quick Reference

```yaml
# Define anchor
_base: &base
  install:
    packages: [git, curl]

# Use anchor
my-category:
  <<: *base              # Merge all fields
  install:
    packages:
      - *base           # Reference in array
      - wget
```

## Validation

```bash
# Validate a recipe
debian-composer validate my-recipe

# Validate all recipes
debian-composer validate --all

# Strict mode (warnings = errors)
debian-composer validate my-recipe --strict
```

## Errors → Fixes

| Error | Cause | Fix |
|-------|-------|-----|
| `name is required` | Missing `name` field | Add `name: recipe-name` |
| `unknown property 'packges'` | Typos | Use `packages:` not `packges:` |
| `condition expression error` | Bad expr syntax | Use `==`, `>`, `<`, `and`, `or` |
| `duplicate key 'packages'` | Key repeated | Merge into one `packages:` array |
| `invalid type for port` | Type mismatch | Use integer, not string |
| `template error` | Bad template | Check `{{ }}` syntax |

---

*See `internal/recipe/schema.json` for the formal JSON Schema.*
*See `recipes/template-example.yaml` for a full working example.*