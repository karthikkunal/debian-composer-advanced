# Fragments

Reusable YAML fragments that provide **anchors, aliases, and templates** for DRY (Don't Repeat Yourself) recipe configurations.

## Overview

Fragments eliminate repetition across recipes by providing:
- **Package list anchors** - Common package combinations defined once
- **Service templates** - Pre-configured category structures with merge keys
- **Requirements templates** - Standard system requirement presets
- **Configuration templates** - Reusable service configurations
- **Verification templates** - Common verification patterns

## Available Fragments

### `base-anchors.yaml`

Common anchors for package lists, requirements, and configurations.

**Package List Anchors:**

| Anchor | Packages | Use Case |
|--------|----------|----------|
| `*base_packages` | git, curl, wget, ca-certificates, gnupg | Universal prerequisites |
| `*dev_packages` | base + build-essential, make | Development tools |
| `*python_base` | python3, python3-pip, python3-venv | Python runtime |
| `*python_dev` | python_base + python3-dev | Python development |
| `*web_prerequisites` | curl, wget, git, ufw, ca-certificates, gnupg | Web server setup |
| `*db_client` | curl, wget, ca-certificates, gnupg | Database clients |
| `*container_prerequisites` | apt-transport-https, ca-certificates, curl, gnupg, lsb-release, git | Docker/K8s |
| `*ai_prerequisites` | git, curl, wget, python3, python3-pip, python3-venv | AI/ML tools |

**Requirements Templates:**

| Anchor | RAM | Disk | GPU | Use Case |
|--------|-----|------|-----|----------|
| `*req_minimal` | 512MB | 2GB | No | Lightweight tools |
| `*req_standard` | 2GB | 10GB | No | Typical recipes |
| `*req_heavy` | 8GB | 20GB | Yes | AI/ML, heavy tools |
| `*req_container` | 4GB | 20GB | No | Container runtime |

**Configuration Templates:**

| Anchor | Purpose |
|--------|---------|
| `*cfg_nginx` | Nginx service configuration |
| `*cfg_apache` | Apache service configuration |
| `*cfg_mysql` | MySQL/MariaDB configuration |
| `*cfg_postgresql` | PostgreSQL configuration |
| `*cfg_redis` | Redis configuration |
| `*cfg_docker` | Docker daemon configuration |
| `*cfg_ollama` | Ollama API service configuration |

**Verification Templates:**

| Anchor | Purpose |
|--------|---------|
| `*verify_nginx` | Nginx verification (commands, service, ports) |
| `*verify_apache` | Apache verification |
| `*verify_mysql` | MySQL verification |
| `*verify_postgresql` | PostgreSQL verification |
| `*verify_redis` | Redis verification |
| `*verify_docker` | Docker verification |
| `*verify_ollama` | Ollama verification |

**Hook Templates:**

| Anchor | Purpose |
|--------|---------|
| `*hook_pre_install` | Standard pre-install hook (apt update) |
| `*hook_post_install` | Standard post-install hook |
| `*hook_pre_configure` | Standard pre-configure hook |
| `*hook_post_configure` | Standard post-configure hook |

### `service-templates.yaml`

Template structures for service categories using merge keys.

**Base Templates:**

| Template | Purpose |
|----------|---------|
| `*service_template` | Base template for all services |
| `*web_server_template` | Web servers (nginx, apache, caddy) |
| `*database_server_template` | Database servers (mysql, postgresql) |
| `*cache_server_template` | Cache servers (redis, memcached) |
| `*app_server_template` | Application servers (php-fpm, gunicorn) |
| `*container_service_template` | Container services (docker, containerd) |
| `*api_service_template` | API services (ollama, lmstudio) |
| `*manual_install_template` | Manual download installations |
| `*script_install_template` | Script-based installations |
| `*python_package_template` | Python pip packages |
| `*docker_container_template` | Docker container deployments |

**Pre-configured Services (Ready to Use):**

| Service | Anchor | Description |
|---------|--------|-------------|
| Nginx | `*nginx_service` | Full nginx configuration |
| Apache | `*apache_service` | Full apache configuration |
| MySQL | `*mysql_service` | MySQL with security settings |
| PostgreSQL | `*postgresql_service` | PostgreSQL with peer auth |
| Redis | `*redis_service` | Redis with persistence |
| Docker | `*docker_service` | Docker with logging config |
| Ollama | `*ollama_service` | Ollama LLM service |

## Usage

### Including Fragments in Recipes

```yaml
# In your recipe YAML file
includes:
  - fragments/base-anchors.yaml
  - fragments/service-templates.yaml
```

### Using Package Anchors

```yaml
# Instead of listing packages repeatedly
packages:
  - git
  - curl
  - wget
  - ca-certificates
  - gnupg

# Use an anchor
packages: *base_packages
```

### Using Merge Keys for Categories

```yaml
# Instead of full category definition
categories:
  nginx:
    install:
      packages:
        - nginx
        - nginx-extras
    configure:
      service:
        name: nginx
        enable: true
        start: true
      directories:
        config: /etc/nginx
        logs: /var/log/nginx
        www: /var/www/html
    verify:
      commands:
        - nginx -v
      services_running:
        - nginx
      ports_listening:
        - 80
        - 443

# Use a pre-configured service template
categories:
  nginx:
    <<: *nginx_service  # Inherits everything
```

### Using Merge Keys with Overrides

```yaml
# Inherit from template, override specific values
categories:
  nginx:
    <<: *nginx_service
    install:
      packages:
        - nginx
        - nginx-extras
        - nginx-module-headers-more  # Add extra package

# Inherit requirements with override
requirements:
  <<: *req_standard
  min_ram: 4096  # Override default 2GB
```

### Creating Local Anchors

```yaml
# Define recipe-specific anchors
.local_packages: &local_packages [package1, package2, package3]

# Use them
packages: *local_packages

# Extend existing anchors
.extended: &extended [*base_packages, *python_base, extra-package]
```

## Validation

Validate recipes with anchors using `yq`:

```bash
# Single file validation (anchors won't resolve)
yq eval '.' recipes/my-recipe.yaml

# With anchor files (anchors resolve correctly)
cat fragments/base-anchors.yaml recipes/my-recipe.yaml | yq eval '.' -

# Explode merge keys to see final structure
cat fragments/base-anchors.yaml fragments/service-templates.yaml recipes/my-recipe.yaml | yq eval 'explode(.)' -
```

## Best Practices

1. **Always include fragments** at the top of your `includes` list
2. **Prefer anchors** over repeating package lists
3. **Use merge keys** for service categories
4. **Override, don't duplicate** - inherit from templates and override only what's different
5. **Define local anchors** for recipe-specific repeated values
6. **Document anchor usage** in complex recipes with comments

## Contributing

When adding new anchors:

1. **Group related anchors** together (packages, requirements, configs)
2. **Use descriptive names** with clear prefixes (`_base_`, `_req_`, `_cfg_`)
3. **Add comments** explaining what each anchor contains
4. **Keep anchors focused** - one purpose per anchor
5. **Update this README** with new anchor documentation

## Examples

See these recipes for anchor usage examples:
- `recipes/minimal-setup.yaml` - Basic anchor references
- `recipes/web-server.yaml` - Merge keys for service categories
- `recipes/docker-dev.yaml` - Complex anchor combinations
- `recipes/ai-tools.yaml` - Mixed anchor patterns
