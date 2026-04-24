# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [0.1.0-alpha.1] - 2026-04-24

### Added
- Init system abstraction (`internal/initsys`): service management across systemd, sysvinit, runit, and OpenRC
- OS identity module (`internal/osinfo`): parses `/etc/os-release` (ID, IDLike, Version, etc.)
- Condition variables `os_id`, `os_id_like`, and `init_system` for conditional recipe evaluation
- CLI commands and core internal packages for recipe and component management
- Kitchen component system with YAML definitions for desktops, services, and profiles
- Test suite, scripts, and hooks; optimized hardware detection with caching and parallelism
- GitLab CI/CD pipeline, Makefile, and nFPM Debian packaging support
- Nala as default package manager with apt fallback

### Changed
- Replaced custom implementations with FOSS libraries (mergo, ghw, gopsutil, expr, jsonschema)
- Replaced timeshift with snapper for system snapshots
- Switched to GPL-3.0 license

### Docs
- Added Devuan support documentation for systemd-free systems
- Updated README with adoption-focused features, usage examples, and project description