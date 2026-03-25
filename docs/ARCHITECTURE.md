# Architecture: Kitchen, Recipes, and Layers

Debian Composer uses a modular "Kitchen" system to manage system configurations. This document explains the core architectural concepts: how recipes are structured, how they are composed, and the role of the layering system.

## The Kitchen System

The **Kitchen** is the workspace where recipes and components live. It has a standard directory structure:

```text
KITCHEN_ROOT/
├── recipes/           # Top-level system definitions
│   ├── workstation.yaml
│   └── server-base.yaml
├── components/        # Reusable building blocks
│   ├── desktop-gnome.yaml
│   └── stack-docker.yaml
└── fragments/         # Low-level snippets & anchors
    ├── common-anchors.yaml
    └── security-bits.yaml
```

- `recipes/`: Top-level system definitions (Distros, Personas, Blends).
- `components/`: Reusable building blocks (Desktop environments, Development stacks).
- `fragments/`: Low-level configuration snippets and YAML anchor definitions.

## Recipe Composition

Debian Composer provides three primary mechanisms for composing system configurations: **Includes**, **Extends**, and **Layers**.

```text
      INHERITANCE                  COMPOSITION                    OVERLAYS
     (Vertical)                   (Horizontal)                   (Stackable)
          |                            |                              |
  [ Parent Recipe ]            [ Main Recipe ]                [ Layer N (Top) ]
          ^                            +                              |
          | extends                    | includes                     |
          |                            v                              v
  [ Child Recipe ]             [ Reusable Component ]         [ Layer 1 (Base) ]
```

### 1. Includes (Horizontal Composition)
`includes` are used to pull in reusable components. When a recipe includes another, it effectively merges the contents of the included file into the current one.
- **Use Case**: Adding a "web-server" component to a "debian-server" recipe.
- **Behavior**: Packages, hooks, and categories are accumulated from all included files.

### 2. Extends (Vertical Inheritance)
`extends` allows a recipe to inherit from a base recipe.
- **Use Case**: Creating a `debian-developer-rust` recipe that extends the standard `debian-developer` recipe.
- **Behavior**: The child recipe inherits all properties of the parent. The child can then add or override specific fields.

### 3. Layers (Stackable Overlays)
`layers` are the most powerful composition tool. They act as stackable overlays that can override or augment any part of the base configuration.

```text
    +---------------------------+
    |      Layer 2 (Overlay)    |  <-- Highest Precedence (Overrides)
    +---------------------------+
    |      Layer 1 (Overlay)    |
    +---------------------------+
    |      Base (Recipe)        |  <-- Lowest Precedence
    +---------------------------+
```
- **Use Case**: Applying a "security-hardening" layer or a "corporate-branding" layer on top of any distro recipe.
- **Behavior**: Layers are processed in the order they are listed. Later layers can:
    - **Override** simple fields (Version, Description, Variable defaults).
    - **Append** to list fields (Packages, Hook commands).
    - **Merge** map fields (Variables, Categories).

## Resolution Order

When you run an install command, the **Resolver** processes the recipe in this specific order:

```text
 START
   |
 [ Parse ] ------> Validates YAML against Schema
   |
 [ Extends ] ----> Loads Parent (Base Inheritance)
   |
 [ Includes ] ---> Merges Components (Horizontal)
   |
 [ Layers ] -----> Stacks Overlays (Vertical Overrides)
   |
 [ Variables ] --> Applies Defaults & CLI values
   |
 [ Categories ] -> Evaluates Conditions & Filters
   |
 [ Flatten ] ----> Generates final Package/Hook lists
   |
  END
```

1.  **Parse**: The main recipe is loaded and validated against the [JSON Schema](schema.json).
2.  **Inherit (Extends)**: If an `extends` field is present, the parent recipe is loaded as the base.
3.  **Include**: All files in the `includes` list are merged into the base.
4.  **Overlay (Layers)**: All recipes in the `layers` list are merged on top, with the final layer having the highest precedence for overrides.
5.  **Resolve Variables**: Default values and CLI-provided variables are applied.
6.  **Filter Categories**: Categories are activated or deactivated based on variable conditions.
7.  **Flatten**: The final list of packages and hooks is generated.

## Package Manager Abstraction Layer

Debian Composer uses a unified package‑manager abstraction layer that supports multiple backends (APT, Flatpak, Snap, Pip, NPM, Go) and allows switching between APT frontends (apt‑get vs. Nala).

### APT Struct (`internal/apt/handler.go`)
- Handles Debian package operations via `syspkg` and `pkgmgr`.
- Prefers Nala over apt‑get when available, respects the `DEBIAN_COMPOSER_PREFERRED_APT` environment variable.
- Routes install, remove, update, upgrade, purge, and search operations through Nala by default; falls back to apt‑get if Nala is not installed.
- Provides Nala‑specific features:
  - `History()` – list transaction history.
  - `Rollback()` – roll back a transaction.
  - `Fetch()` – select fastest mirrors.
  - `Upgrade()` – unified update+upgrade.
  - `SetConfig()` – configure Nala options (parallel downloads, timeouts, language, etc.).
  - `SearchFormatted()` / `ListInstalledFormatted()` – colored, table‑styled output.

### Kitchen Integration
- `essential-tools.yaml` installs Nala and configures it via variables:
  - `nala_parallel_downloads` (default 3)
  - `nala_fetch_timeout` (default 30 seconds)
  - `nala_fetch_retries` (default 3)
  - `nala_language` (empty = system locale)
  - `nala_scrolling_text` (default true)
  - `nala_color` (default true)
- `nala-optimize.yaml` runs `nala fetch` to select fastest mirrors.
- `system-snapshot.yaml` creates Snapper snapshots before APT/Nala operations.

### Environment Variables
- `DEBIAN_COMPOSER_PREFERRED_APT` – set to `nala` or `apt` to override detection.

## Schema Validation

Every recipe and component must conform to the standard schema. This ensures:
- Required fields like `name` and `kind` are present.
- Data types are correct (e.g., `packages` must be an array of strings).
- Enums for `kind` and variable `type` are respected.

You can validate any recipe manually using:
```bash
debian-composer validate path/to/recipe.yaml
```

## Standalone Export

To share a complex recipe without its entire kitchen context, use the `export --standalone` command. This will:
1.  Resolve all `includes` and `extends`.
2.  Bundle all referenced YAML anchors into the file.
3.  Produce a single, self-contained YAML file that can be imported into any other Debian Composer kitchen.
