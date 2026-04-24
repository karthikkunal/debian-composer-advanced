# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- **`debian-devuan` distro recipe**: Full Devuan/systemd-free workstation with sysvinit, OpenRC, and runit support
- **`devuan-backend` component**: Devuan APT sources, apt.conf, and sysvinit service equivalents
- **`devuan-init` fragment**: SysVInit/OpenRC/Runit service templates and runlevel mappings
- **Production readiness checklist** (`docs/PRODUCTION-CHECKLIST.md`): pre-flight checks, backup, dry-run, snapper, schema validation, init system checks, rollback procedures, and periodic health checks
- **Devuan support documentation** (`docs/DEVuan.md`): init system detection, service equivalents, APT configuration, known issues
- **`template-example` recipe**: Config file templating with `template:` hook type and Sprig functions
- **Cross-blend composability** for Pure Blend recipes using `include/extend/layer` composition
- **Go text/template hooks**: `template:` hook type uses `text/template` with Sprig functions for config file generation at installation time
- **`debian-edu` and `debian-science` blends**: Refactored to use `include/extend/layer` composition with shared anchors from `fragments/`
- **Condition-based template categories**: Support for `condition:` on categories with `template:` hooks for conditional config generation

### Changed
- **`debian-devuan` as first-class target**: Supported alongside systemd-based Debian derivatives
- **`internal/initsys.Detect()`**: Exposed `os_id`, `os_id_like`, and `init_system` variables for recipe conditionals
- **`internal/initsys`**: Canonical single package for all init system operations — detection, service management (start/stop/restart/enable/disable/status), daemon reload, and service listing across systemd, sysvinit, OpenRC, and runit
- **Service management**: Gracefully falls back to sysvinit/openrc/runit equivalents when systemd is unavailable
- **`internal/initsys.DetectSystemdFeatures()`**: Probes systemd features (journald, networkd, resolved, logind, timedated) for graceful degradation
- **kitchen/README.md**: Added Devuan documentation section, config file templating examples, cross-blend composition guide

### Docs
- **`docs/SCHEMA-CHEATSHEET.md`**: One-page YAML recipe reference — all fields, hook types, condition expressions, template examples, common errors → fixes
- **`docs/SCOPE.md`**: Scope positioning — Debian Composer vs Ansible comparison, when to use each, cross-over workflow (Ansible provisions → Debian Composer configures)
- **`docs/PRODUCTION-CHECKLIST.md`**: Pre-flight checks, backup, dry-run, snapper, schema validation, init system checks, rollback procedures, and periodic health checks

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