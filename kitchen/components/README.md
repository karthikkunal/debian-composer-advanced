# Components

Reusable, composable component definitions that can be included in recipes.

## Structure

Each component is a standalone YAML file defining:
- `install`: packages, repositories
- `configure`: service, security settings
- `verify`: commands, services, ports
- `requirements`: min_ram, min_disk, gpu
- `post_install`: notes

## Usage

Include components in recipes:

```yaml
includes:
  - components/mysql.yaml
  - components/nginx.yaml
```

## Available Components

### Databases
- `mysql.yaml` - MySQL server
- `postgresql.yaml` - PostgreSQL server
- `redis.yaml` - Redis cache server
- `mongodb.yaml` - MongoDB document database
- `elasticsearch.yaml` - Elasticsearch search engine

### Web Servers
- `nginx.yaml` - Nginx web server
- `apache.yaml` - Apache web server
- `php.yaml` - PHP runtime

### AI/ML
- `ollama.yaml` - Local LLM runner
- `whisper.yaml` - Speech-to-text

### Infrastructure
- `base.yaml` - Base system packages

### Package Management
- `essential-tools.yaml` - Essential CLI tools including Nala (modern apt frontend) with configurable parallel downloads, timeouts, and language.
- `nala-optimize.yaml` - Mirror optimization using `nala fetch`.
- `system-snapshot.yaml` - Snapper snapshots before package operations.

## Writing New Components

Example component structure:

```yaml
name: component-name
version: 1.0.0
description: What it does
author: Your Name
license: MIT
tags: [tag1, tag2]

install:
  packages:
    - pkg1
    - pkg2

configure:
  service:
    name: servicename
    enable: true
    start: true

verify:
  commands:
    - command --version
  services_running:
    - servicename

requirements:
  min_ram: 1024
  min_disk: 5

post_install:
  - "Usage note"
```