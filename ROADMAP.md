# Roadmap

## Vision

Make Debian Composer the de-facto tool for composing and sharing Debian-based system configurations across all derivatives.

## v0.0.x - Feature Parity with Debian Pure Blends

The primary goal for the initial releases is to match and exceed the capabilities of Debian Pure Blends while adding composability.

### Core Blend Features
- [ ] Full Debian Pure Blend support (`task-*` packages)
- [ ] Task meta-package generation
- [ ] Blend profile parsing and installation
- [ ] Standard task categories (Debian Science, Debian Edu, Debian GIS, etc.)

### Desktop Environment Support
- [ ] GNOME desktop selection
- [ ] KDE Plasma desktop selection
- [ ] Xfce desktop selection
- [ ] LXQt desktop selection
- [ ] MATE desktop selection
- [ ] Cinnamon desktop selection
- [ ] Custom DE组合 (mixed desktop components)

### Package Categories
- [ ] Standard package classification (Priority: required, important, standard, optional)
- [ ] Section-based filtering (main, contrib, non-free, non-free-firmware)
- [ ] Architecture-aware package filtering (amd64, arm64, etc.)
- [ ] Virtual package resolution

## Short-term (v0.2.x)

### Recipe Discovery
- [ ] Recipe registry/search command
- [ ] Remote recipe URL support
- [ ] Recipe versioning

### Enhanced Installation
- [ ] Dependency resolution
- [ ] Conflict detection
- [ ] Rollback support

### CLI Improvements
- [ ] Interactive recipe builder
- [ ] Recipe validation command
- [ ] Dry-run mode

## Medium-term (v0.3.x)

### Recipe Ecosystem
- [ ] Recipe marketplace/index
- [ ] Community recipe curation
- [ ] Recipe ranking/ratings

### Integration
- [ ] distro-info integration
- [ ] Debconf integration
- [ ] Snap/Flatpak support

### Advanced Features
- [ ] Recipe templates
- [ ] Import/export bundles
- [ ] Recipe diff/compare

## Long-term (v1.0.x)

### Core Platform
- [ ] Stable recipe format (v1.0)
- [ ] Plugin system
- [ ] Recipe signing/verification
- [ ] Distributed recipe network

### Distribution Features
- [ ] Custom ISO generation
- [ ] Layered ISO building
- [ ] Container/OCI image output

### Community
- [ ] Official recipe collection
- [ ] Contributor recognition
- [ ] Documentation portal

## Ideas from Community

- Recipe format extensions for specific use cases
- Integration with popular configuration management tools
- Web UI for recipe browsing and management
- Cloud-based recipe sync

---

*Help shape the roadmap! Open an issue to suggest features or vote on priorities.*