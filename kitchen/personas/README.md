# Persona System - Mixin Architecture

## Overview

The persona system uses a **mixin-based architecture** to provide flexible, composable user profiles while maintaining a limit of **20 total personas**. This approach reduces duplication and allows users to customize their setup without creating dozens of separate persona files.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    PERSONA SYSTEM                            │
├─────────────────────────────────────────────────────────────┤
│  Base Personas (10)  +  Specialization Mixins (10) = 20     │
│                                                                │
│  ┌──────────────┐    ┌──────────────────────────────────┐   │
│  │ student      │    │ ai-ml                            │   │
│  │ developer    │    │ web-dev                          │   │
│  │ creative     │    │ data-science                     │   │
│  │ gamer        │    │ devops                           │   │
│  │ researcher   │    │ mobile-dev                       │   │
│  │ sysadmin     │    │ security                         │   │
│  │ homelab      │    │ video-production                 │   │
│  │ desktop      │    │ audio-production                 │   │
│  │ privacy      │    │ gaming                           │   │
│  │ accessibility│    │ education                        │   │
│  └──────────────┘    └──────────────────────────────────┘   │
│                                                                │
│  Components (Reusable Building Blocks)                        │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ base, editors, dev-toolchain, desktop-apps, etc.       │  │
│  └────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Usage

### Basic Usage

```bash
# Install a base persona
sudo debian-composer --persona=developer

# Install persona with mixin
sudo debian-composer --persona=developer --mixin=ai-ml

# Install persona with multiple mixins
sudo debian-composer --persona=student --mixin=programming --mixin=design

# Set variables for customization
sudo debian-composer --persona=developer --var=primary_lang=rust
```

### Available Base Personas

| Persona | Description | Best For |
|---------|-------------|----------|
| `student` | Note-taking, research, productivity | Students (high school to PhD) |
| `developer` | IDEs, toolchains, containers | Software developers |
| `creative` | Design tools, media editors | Designers, artists |
| `gamer` | Gaming platforms, drivers, overlays | PC gamers |
| `researcher` | Data analysis, academic tools | Researchers, academics |
| `sysadmin` | Server tools, monitoring, security | System administrators |
| `homelab` | Self-hosting, containers, automation | Home server enthusiasts |
| `desktop` | General desktop, productivity | Everyday users |
| `privacy` | Privacy tools, encryption | Privacy-conscious users |
| `accessibility` | Screen readers, accessibility tools | Users with accessibility needs |

### Available Mixins

| Mixin | Description | Adds |
|-------|-------------|------|
| `ai-ml` | AI/ML tooling | PyTorch, TensorFlow, CUDA |
| `web-dev` | Web development | Node.js, web frameworks |
| `data-science` | Data analysis | Jupyter, R, pandas |
| `devops` | DevOps tooling | Docker, K8s, CI/CD |
| `mobile-dev` | Mobile development | Android SDK, Flutter |
| `security` | Security tools | Penetration testing, auditing |
| `video-production` | Video editing | Kdenlive, Blender, OBS |
| `audio-production` | Audio production | Ardour, Audacity, DAWs |
| `gaming` | Gaming enhancements | Steam, Lutris, MangoHud |
| `education` | Teaching tools | Classroom management, quizzes |

## Creating Custom Combinations

### Example Combinations

```bash
# Data Scientist
sudo debian-composer --persona=researcher --mixin=data-science --mixin=ai-ml

# Web Developer
sudo debian-composer --persona=developer --mixin=web-dev

# Game Developer
sudo debian-composer --persona=developer --mixin=gaming

# Security Researcher
sudo debian-composer --persona=researcher --mixin=security

# Content Creator
sudo debian-composer --persona=creative --mixin=video-production --mixin=audio-production

# DevOps Engineer
sudo debian-composer --persona=sysadmin --mixin=devops

# Privacy-Focused Developer
sudo debian-composer --persona=developer --mixin=privacy

# Student Programmer
sudo debian-composer --persona=student --mixin=programming
```

## File Structure

```
recipes/
├── personas/
│   ├── README.md              # This file
│   ├── base/
│   │   ├── student.yaml       # Base persona
│   │   ├── developer.yaml     # Base persona
│   │   ├── creative.yaml      # Base persona
│   │   ├── gamer.yaml         # Base persona
│   │   ├── researcher.yaml    # Base persona
│   │   ├── sysadmin.yaml      # Base persona
│   │   ├── homelab.yaml       # Base persona
│   │   ├── desktop.yaml       # Base persona
│   │   ├── privacy.yaml       # Base persona
│   │   └── accessibility.yaml # Base persona
│   └── mixins/
│       ├── ai-ml.yaml         # Specialization mixin
│       ├── web-dev.yaml       # Specialization mixin
│       ├── data-science.yaml  # Specialization mixin
│       ├── devops.yaml        # Specialization mixin
│       ├── mobile-dev.yaml    # Specialization mixin
│       ├── security.yaml      # Specialization mixin
│       ├── video-production.yaml
│       ├── audio-production.yaml
│       ├── gaming.yaml
│       └── education.yaml
├── components/                 # Reusable building blocks
│   ├── base.yaml
│   ├── editors.yaml
│   ├── dev-toolchain.yaml
│   └── ...
└── fragments/                  # YAML anchors and templates
    ├── base-anchors.yaml
    └── service-templates.yaml
```

## Writing a Base Persona

```yaml
name: persona-developer
version: 1.0.0
description: Pre-configured development environment
author: Your Name
license: MIT
tags: [persona, developer, programming]

# Include base components
includes:
  - components/base.yaml
  - components/editors.yaml
  - components/dev-toolchain.yaml
  - fragments/base-anchors.yaml
  - fragments/service-templates.yaml

# Define customizable variables
variables:
  primary_lang:
    type: string
    default: python
    choices: [python, rust, javascript, go]
  editor_choice:
    type: string
    default: vscode
    choices: [vscode, neovim, emacs]

# Define categories (optional - can be provided by mixins)
categories:
  vcs:
    <<: *service_template
    description: "Version control"
    install:
      packages:
        - git
        - gh

# Define stacks (pre-configured combinations)
stacks:
  basic-dev:
    description: "Basic development setup"
    categories:
      - vcs
      - runtime-python
      - editor-vscode

# System requirements
requirements:
  <<: *req_heavy
  min_ram: 4096
  min_disk: 20

# Post-installation notes
post_install:
  - "=== Developer Persona Installed ==="
  - "Primary Language: ${primary_lang}"
  - "Editor: ${editor_choice}"
```

## Writing a Mixin

```yaml
name: mixin-ai-ml
version: 1.0.0
description: AI/ML development tooling
author: Your Name
license: MIT
tags: [mixin, ai, ml, deep-learning]

# Mixins can include additional components
includes:
  - components/ollama.yaml
  - components/whisper.yaml

# Mixins add or override categories
categories:
  ai-frameworks:
    <<: *service_template
    description: "Deep learning frameworks"
    install:
      packages:
        - python3-pytorch
        - python3-tensorflow
        - python3-keras
    verify:
      commands:
        - python3 -c "import torch; print(torch.__version__)"

  cuda-support:
    condition: has_nvidia_gpu == true
    <<: *service_template
    description: "NVIDIA CUDA support"
    install:
      packages:
        - nvidia-cuda-toolkit
        - nvidia-driver

# Mixins can add new variables
variables:
  enable_cuda:
    type: boolean
    default: auto
    description: "Enable CUDA support (auto detects hardware)"

# Mixins can add post-install notes
post_install:
  - "=== AI/ML Mixin Applied ==="
  - "CUDA Enabled: ${enable_cuda}"
  - "Test PyTorch: python3 -c 'import torch'"
```

## Governance: The 20-Persona Limit

To maintain simplicity and avoid decision fatigue, the persona system enforces a **20-persona limit**:

### Rules

1. **Maximum 10 Base Personas**: Core user types that represent distinct use cases
2. **Maximum 10 Mixins**: Specializations that can be combined with base personas
3. **No Overlap**: New personas must demonstrate <70% overlap with existing ones
4. **Deprecation Process**: To add a new persona, an existing one may need to be deprecated
5. **Community Review**: New personas require community approval via PR review

### Deprecation Criteria

A persona may be deprecated if:
- <100 installations in 3 months
- >70% overlap with another persona
- Community votes for consolidation
- Maintainer abandons the persona

### Proposal Process

To propose a new persona:

1. **Check Overlap**: Demonstrate <70% overlap with existing personas
2. **Justify**: Explain why it can't be achieved with existing mixins
3. **Community Vote**: Open GitHub issue for community feedback
4. **Replace or Deprecate**: If at 20 limit, propose which persona to deprecate

## Migration Guide

### From Old Persona System

If you have existing persona configurations:

```bash
# Old way (no longer supported)
sudo debian-composer --persona=data-scientist

# New way (with mixins)
sudo debian-composer --persona=researcher --mixin=data-science

# Old way
sudo debian-composer --persona=web-developer

# New way
sudo debian-composer --persona=developer --mixin=web-dev
```

## Troubleshooting

### Mixin Conflicts

If two mixins conflict (e.g., both try to configure the same service):

```bash
# Check for conflicts before installing
debian-composer --persona=developer --mixin=web-dev --mixin=data-science --check

# Use --var to resolve conflicts
sudo debian-composer --persona=developer --mixin=web-dev --var=node_version=18
```

### Missing Dependencies

```bash
# Check what components a persona/mixin requires
debian-composer --persona=developer --show-deps
```

## Contributing

We welcome community contributions! See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

### Needed Personas/Mixins

- [ ] `robotics` mixin (ROS, Gazebo)
- [ ] `bioinformatics` mixin (BioPython, genomics tools)
- [ ] `blockchain` mixin (Ethereum, Solidity)
- [ ] `iot` mixin (MQTT, Home Assistant)

---

**Version:** 1.0.0
**Last Updated:** 2026-03-19
**Maintainer:** Debian Composer Team
