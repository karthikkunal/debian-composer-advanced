# Mixins - Specialization Templates
# ============================================================================
# Mixins add specialized functionality to base personas
#
# Available Mixins (10 total):
# ============================================================================

| Mixin | Description | Best Combined With |
|-------|-------------|-------------------|
| `ai-ml` | AI/ML tools (PyTorch, TensorFlow, CUDA) | developer, researcher, student |
| `web-dev` | Web development (Node.js, frameworks) | developer, student |
| `data-science` | Data analysis (R, Jupyter, pandas) | researcher, student, developer |
| `devops` | DevOps (K8s, Docker, CI/CD) | developer, sysadmin |
| `security` | Security tools (pentesting, auditing) | sysadmin, researcher |
| `video-production` | Video editing, streaming | creative, gamer |
| `audio-production` | Audio editing, DAW, plugins | creative, musician |
| `gaming` | Gaming platforms, overlays | gamer, desktop |
| `education` | Teaching, classroom tools | student, educator |
| `mobile-dev` | Android, Flutter, React Native | developer, student |

## Usage

```bash
# Persona + Mixin
sudo debian-composer --persona=developer --mixin=ai-ml

# Multiple mixins
sudo debian-composer --persona=developer --mixin=web-dev --mixin=devops

# Modular composition with mixin
sudo debian-composer --base=desktop --role=developer --mixin=ai-ml
```

## Creating Custom Mixins

```yaml
name: mixin-yourname
version: 1.0.0
description: What it adds
author: Your Name
license: MIT
tags: [mixin, category]

includes:
  - fragments/base-anchors.yaml
  - fragments/service-templates.yaml

variables:
  your_var:
    type: boolean
    default: true

packages:
  - your-package

categories:
  your-category:
    <<: *service_template
    description: "Your category"
    install:
      packages:
        - your-package
    verify:
      commands:
        - your-command --version

post_install:
  - "=== Your Mixin Applied ==="
```

## Governance: The 10-Mixin Limit

To maintain simplicity, we enforce a **10-mixin limit**:

1. **No Overlap**: New mixins must demonstrate <50% overlap with existing ones
2. **Community Review**: New mixins require community approval
3. **Deprecation**: To add a new mixin, an existing one may need consolidation

See [personas/README.md](../personas/README.md) for full governance details.
