# Mixins

Reusable configuration overlays that can be applied to recipes.

## Usage

Include mixins in recipes:

```yaml
includes:
  - mixins/security-hardening.yaml
  - mixins/monitoring.yaml
```

## Available Mixins

- `security-hardening.yaml` - Security headers, firewall defaults, SSL config

## Writing New Mixins

Mixins can override or add to existing recipe configuration:

```yaml
name: mixin-name
version: 1.0.0
description: What it adds

configure:
  security:
    # security settings...

firewall:
  enabled: true

post_install:
  - "Mixins note"
```