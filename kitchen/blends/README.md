# Debian Pure Blends

This directory contains **Debian Pure Blend** configurations for Debian Composer.

## What are Debian Pure Blends?

Debian Pure Blends are official Debian subsets tailored for specific purposes. Each blend provides a curated collection of packages and configurations for a particular domain or user group.

## Available Blends

| Blend | Description | Documentation |
|-------|-------------|---------------|
| [debian-edu.yaml](debian-edu.yaml) | Education (preschool through university) | [Docs](../docs/BLENDS.md#debian-edu) |
| [debian-med.yaml](debian-med.yaml) | Medical practice and research | [Docs](../docs/BLENDS.md#debian-med) |
| [debian-astro.yaml](debian-astro.yaml) | Astronomy and astrophysics | [Docs](../docs/BLENDS.md#debian-astro) |
| [debian-gis.yaml](debian-gis.yaml) | Geographic Information Systems | [Docs](../docs/BLENDS.md#debian-gis) |
| [debian-science.yaml](debian-science.yaml) | Scientific computing | [Docs](../docs/BLENDS.md#debian-science) |

## Quick Start

### Install a Blend

```bash
# Install Debian Edu
sudo debian-composer --blend=debian-edu

# Install Debian Med
sudo debian-composer --blend=debian-med

# Install Debian Astro
sudo debian-composer --blend=debian-astro

# Install Debian GIS
sudo debian-composer --blend=debian-gis

# Install Debian Science
sudo debian-composer --blend=debian-science
```

### With Variables

```bash
# Install Debian Edu for primary school
sudo debian-composer --blend=debian-edu --var=education_level=primary

# Install Debian Med for medical practice
sudo debian-composer --blend=debian-med --var=med_focus=practice
```

### With Stacks

```bash
# Install school server with LTSP
sudo debian-composer --blend=debian-edu --stack=school-server

# Install medical practice
sudo debian-composer --blend=debian-med --stack=practice-complete

# Install amateur astronomy setup
sudo debian-composer --blend=debian-astro --stack=amateur-complete
```

## Blend Structure

Each blend YAML file contains:

```yaml
name: debian-edu
version: 1.0.0
description: Debian Edu Pure Blend

metadata:
  category: education
  target_audience: [students, teachers]
  min_ram: 4096
  min_disk: 30

variables:
  education_level:
    type: string
    default: general
    choices: [preschool, primary, high-school, university, general]

packages:
  - *blend_essentials
  - *edu_workstation

categories:
  grade/primary:
    condition: education_level == primary
    install:
      packages: [tuxmath, tuxtype]

stacks:
  primary-school:
    description: Primary school setup
    categories: [grade/preschool, grade/primary]
```

## Creating a New Blend

1. Create a new YAML file in this directory
2. Follow the [blend schema](../blends/blend-schema.yaml)
3. Use anchors from `fragments/base-anchors.yaml`
4. Define variables, categories, and stacks
5. Test with `--dry-run` and `--check` modes

## References

- [Official Debian Blends](https://www.debian.org/blends/)
- [Debian Edu](https://blends.debian.org/edu/)
- [Debian Med](https://blends.debian.org/med/)
- [Debian Science](https://blends.debian.org/science/)
- [Debian Astro](https://blends.debian.org/astro/)
- [Debian GIS](https://blends.debian.org/gis/)

## Documentation

- [Full Blends Documentation](../docs/BLENDS.md)
- [Recipe Documentation](../README.md)
- [Usage Guide](../docs/USAGE.md)
