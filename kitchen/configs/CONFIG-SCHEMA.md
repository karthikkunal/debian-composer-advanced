# Debian Composer Configuration Schema
# ============================================================================
# Version: 1.0.0
#
# This document defines the schema for user-defined configuration files.
# Config files can be YAML or JSON format (YAML recommended).
# ============================================================================

# ----------------------------------------------------------------------------
# Required Fields
# ----------------------------------------------------------------------------
name: string        # Configuration identifier (e.g., "dev", "personal")
version: string     # Config version (semver format: X.Y.Z or X.Y.Z-tag)

# ----------------------------------------------------------------------------
# Optional Fields
# ----------------------------------------------------------------------------
description: string             # Human-readable description
environment: string             # Target environment (dev/prod/personal/test/staging)
author: string                  # Config author
license: string                 # License for the config
tags: array[string]             # Search tags

# ----------------------------------------------------------------------------
# Inheritance
# ----------------------------------------------------------------------------
extends: array[string]          # Parent configs to inherit from (processed in order)

# Example:
# extends:
#   - base
#   - dev

# ----------------------------------------------------------------------------
# Variables (Template variables for the config)
# ----------------------------------------------------------------------------
variables:
  variable_name:
    type: string                # Variable type (string, integer, boolean, array)
    default: any                # Default value
    description: string         # Variable description

# Example:
# variables:
#   enable_docker:
#     type: boolean
#     default: true
#     description: "Enable Docker support"

# ----------------------------------------------------------------------------
# Packages (Software to install)
# ----------------------------------------------------------------------------
packages:
  # Simple format (APT package)
  - package_name

  # Object format (multi-installer support)
  - name: package_name
    type: string                # apt, flatpak, appimage, binary
    condition: string           # Conditional installation
    apt_fallback: string        # Fallback if preferred type unavailable

# Example:
# packages:
#   - git
#   - curl
#   - name: code
#     type: apt
#     apt_fallback: vim
#   - name: org.gimp.GIMP
#     type: flatpak
#     apt_fallback: gimp

# ----------------------------------------------------------------------------
# Recipes (Debian Composer recipes to include)
# ----------------------------------------------------------------------------
recipes: array[string]          # Recipe names to run

# Example:
# recipes:
#   - debian-dev-tools
#   - docker-dev

# ----------------------------------------------------------------------------
# Blends (Debian Pure Blends to apply)
# ----------------------------------------------------------------------------
blends: array[string]           # Blend names to apply

# Example:
# blends:
#   - debian-edu
#   - debian-science

# ----------------------------------------------------------------------------
# Settings (Configuration settings)
# ----------------------------------------------------------------------------
settings:
  # Nested settings structure
  category:
    key: value

# Example:
# settings:
#   shell:
#     default: zsh
#     prompt: powerlevel10k
#   git:
#     editor: code --wait
#   security:
#     ssh_root_login: false

# ----------------------------------------------------------------------------
# Services (System services to manage)
# ----------------------------------------------------------------------------
services:
  - name: string                # Service name
    enabled: boolean            # Enable on boot
    state: string               # started, stopped, restarted
    condition: string           # Conditional activation

# Example:
# services:
#   - name: ssh
#     enabled: true
#     state: started
#   - name: docker
#     enabled: true
#     state: started
#     condition: enable_docker == true

# ----------------------------------------------------------------------------
# Files (Files and directories to create)
# ----------------------------------------------------------------------------
files:
  - path: string                # File or directory path
    type: string                # file, directory, symlink
    content: string             # File content (for type: file)
    append: boolean             # Append to existing file
    permissions: string         # File permissions (e.g., "644")
    owner: string               # File owner
    group: string               # File group

# Example:
# files:
#   - path: ~/projects
#     type: directory
#   - path: ~/.gitconfig.local
#     type: file
#     content: |
#       [user]
#         name = Developer
#     permissions: "600"

# ----------------------------------------------------------------------------
# Hooks (Lifecycle hooks)
# ----------------------------------------------------------------------------
hooks:
  pre_apply: array[string]      # Commands to run before applying config
  post_apply: array[string]     # Commands to run after applying config

# Example:
# hooks:
#   pre_apply:
#     - echo "Starting configuration..."
#   post_apply:
#     - echo "Configuration complete!"
#     - systemctl restart ssh

# ----------------------------------------------------------------------------
# Security (Security configuration)
# ----------------------------------------------------------------------------
security:
  firewall:                     # Firewall rules
    - name: string
      port: integer
      protocol: string
      source: string

  sshd_config:                  # SSH daemon configuration
    key: value

# Example:
# security:
#   firewall:
#     - name: Allow SSH
#       port: 22
#       protocol: tcp
#   sshd_config:
#     PermitRootLogin: "no"
#     PasswordAuthentication: "no"

# ----------------------------------------------------------------------------
# Desktop (Desktop environment configuration)
# ----------------------------------------------------------------------------
desktop:
  extensions: array[string]     # Desktop extensions to install
  wallpaper:
    source: string
    slideshow: boolean
  panel:
    position: string
    autohide: boolean

# Example:
# desktop:
#   extensions:
#     - dash-to-dock
#     - user-themes
#   wallpaper:
#     source: default
#     slideshow: false

# ----------------------------------------------------------------------------
# Applications (Application preferences)
# ----------------------------------------------------------------------------
applications:
  defaults:                     # Default applications
    web_browser: string
    email: string
    terminal: string
    files: string

  startup: array[string]        # Startup applications

# Example:
# applications:
#   defaults:
#     web_browser: firefox-esr.desktop
#     email: thunderbird.desktop
#   startup:
#     - nm-applet
#     - dropbox

# ============================================================================
# Complete Example
# ============================================================================

name: my-dev-config
version: 1.0.0
description: My personal development configuration
environment: dev
author: John Doe
tags: [development, python, docker]

extends:
  - base
  - dev

variables:
  enable_docker: true
  enable_python: true
  preferred_ide: vscode

packages:
  - git
  - curl
  - name: code
    type: apt
  - name: docker-ce
    type: apt
    condition: enable_docker == true

recipes:
  - debian-dev-tools
  - docker-dev

settings:
  shell:
    default: zsh
  git:
    editor: code --wait

services:
  - name: docker
    enabled: true
    state: started

hooks:
  post_apply:
    - echo "Development environment ready!"
