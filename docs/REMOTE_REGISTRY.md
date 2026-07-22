# Remote Recipe Registry

This feature adds remote recipe discovery and verified downloads to Debian Composer.

## Commands

```bash
debian-composer registry add community https://example.org/index.yaml
debian-composer registry list
debian-composer registry update
debian-composer registry search mlops
debian-composer registry install community/mlops-workstation
```

## Registry index format

```yaml
apiVersion: composer.debian.org/v1alpha1
kind: RegistryIndex
metadata:
  name: community
  updatedAt: 2026-07-22T10:00:00Z
recipes:
  - name: mlops-workstation
    version: 1.0.0
    description: MLOps workstation
    download_url: recipes/mlops-workstation.yaml
    sha256: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
    tags: [mlops, kubernetes]
    architectures: [amd64, arm64]
    distributions: [debian, ubuntu]
```

## Security controls

- HTTPS is required unless `--allow-http` is explicitly used.
- Cross-host redirects are rejected.
- Index and recipe response sizes are limited.
- Recipe downloads use temporary files and atomic renames.
- SHA-256 is verified before the recipe enters the kitchen.
- Existing recipes are not overwritten without `--force`.
