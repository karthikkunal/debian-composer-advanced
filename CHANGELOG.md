# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Changed
- **Dependencies**: Consolidated YAML libraries — replaced `goccy/go-yaml` (direct) with
  `gopkg.in/yaml.v3` as the single YAML parser across all recipe packages
- **Dependencies**: Replaced `briandowns/spinner` and `schollz/progressbar/v3` with
  `charmbracelet/bubbles` spinner and progress components (already a direct dependency);
  removes two direct deps without changing public API in `internal/ux`
- **Dependencies**: Replaced raw `database/sql` in state manager with `jmoiron/sqlx`;
  positional `row.Scan` sequences replaced by `db.Get`/`db.Select` with tagged structs
- **Recipe parser**: YAML anchor expansion now routes complex cases through the pure-Go
  `YQResolver` (yaml.v3 `yaml.Node`-based); removed dependency on the external `yq` binary
  and removed `runYQExpand` / `os/exec` from `parser.go`
- **BREAKING**: Git hooks rewritten in Go (replaces shell-based hooks)
  - Pre-commit and pre-push hooks now use Go implementations
  - Removed shell check modules (checks/, lib/)
  - Added support for: NO_COLOR, DEBUG, VERBOSE, feature flags
  - Changed-only mode automatically detects staged files
  - Configurable via environment variables

### Added
- Project structure and architecture
- YAML recipe parsing with support for includes, extends, and layers
- Recipe resolver for variable substitution
- Recipe merger for combining multiple recipe layers
- CLI with commands for recipe management
- SQLite-based state management for tracking installed recipes
- APT integration for package operations
- Categories and stacks support for conditional installation
- Verification hooks for post-install validation

### Features
- Recipe composition (include, extend, layer)
- Conditional package installation based on system state
- Pre-configured stacks combining multiple categories
- User and group configuration hooks
- Post-install messages and notifications

### Dependencies
- github.com/spf13/cobra (CLI)
- gopkg.in/yaml.v3 (YAML parsing and anchor expansion)
- github.com/mattn/go-sqlite3 (SQLite driver)
- github.com/jmoiron/sqlx (ergonomic SQL)
- github.com/charmbracelet/bubbles (spinner and progress UI)
- github.com/charmbracelet/bubbletea (TUI framework)
- github.com/charmbracelet/lipgloss (terminal styling)

## [0.1.0] - 2025-01-XX

### Added
- Initial release
- Basic recipe parsing
- CLI entry point
- Package installation foundation