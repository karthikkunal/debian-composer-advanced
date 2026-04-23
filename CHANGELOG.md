# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Changed
- **Init System Abstraction**: Service management now works across systemd, sysvinit, runit, and OpenRC instead of only systemd
- **Conditions**: Added `os_id`, `os_id_like`, and `init_system` variables for conditional recipe evaluation
- **Documentation**: Added Devuan support documentation for systemd-free systems

### Added
- `internal/initsys`: Init system detection and cross-init service management (start/stop/enable/restart/disable)
- `internal/osinfo`: OS identity parsing from `/etc/os-release` (ID, IDLike, Version, etc.)

## [0.1.0] - 2025-01-XX

### Added
- Initial release
- Basic recipe parsing
- CLI entry point
- Package installation foundation