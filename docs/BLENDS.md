# Debian Pure Blends

Debian Composer supports all 11 official Debian Pure Blends, allowing you to install curated software collections for specific use cases.

## Available Blends

| Blend | Description | Task Package |
|-------|-------------|--------------|
| debian-accessibility | Assistive technology tools | task-accessibility |
| debian-astro | Astronomy and astrophysics | task-astro |
| debian-edu | Educational tools (preschool → university) | task-edu |
| debian-games | Gaming collection (arcade, strategy, RPG) | task-games |
| debian-gis | Geographic Information Systems | task-gis |
| debian-hamradio | Amateur radio tools | task-hamradio |
| debian-junior | Applications for children | task-junior |
| debian-med | Medical practice and research | task-med |
| debian-multimedia | Audio/video production | task-multimedia |
| debian-science | Scientific computing | task-science |
| freedombox | Personal server for privacy | task-freedombox |

## Installing Blends

```bash
# Install a full blend
debian-composer blend debian-edu
debian-composer blend debian-science

# Install with variables (conditional installation)
debian-composer blend debian-edu --var=education_level=university
debian-composer blend debian-astro --var=astro_focus=amateur

# Interactive selection
debian-composer blend debian-edu --interactive
```

## Blend Variables

Each blend can define variables to customize installation:

- `education_level`: preschool, primary, secondary, university
- `astro_focus`: amateur, professional, research
- `media_focus`: audio, video, both

Variables are defined in the blend's YAML recipe under `variables`.

## Conditional Categories

Blends can have conditional categories that install only relevant packages based on variables.

Example:
```yaml
categories:
  astronomy_basic:
    condition: "astro_focus == amateur"
    packages:
      - stellarium
      - celestia
  astronomy_research:
    condition: "astro_focus == research"
    packages:
      - astropy
      - matplotlib
```

## Stack Selection

Pre-configured stacks provide curated subsets of a blend:

```bash
debian-composer blend debian-edu --stack=primary-school
```

## Conflict Detection

Debian Composer automatically detects conflicts between blends. If two blends conflict, installation will fail with an error message.

## Rollback

If Snapper is installed, you can create snapshots before blend installation:

```bash
debian-composer blend debian-edu --with-snapshot
```

## State Management

Installed blends are tracked in SQLite database (`/var/lib/debian-composer/state.db`). You can list installed blends:

```bash
debian-composer list
```
