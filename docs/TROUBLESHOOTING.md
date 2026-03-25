# Troubleshooting Guide

## Common Issues

### Package Installation Fails

**Symptom:** Package installation fails with error messages.

**Solutions:**
1. Update apt cache:
   ```bash
   sudo apt update
   # or, if using Nala:
   sudo nala update
   ```

2. Check if package exists:
   ```bash
   apt-cache search <package-name>
   # or, if using Nala:
   nala search <package-name>
   ```

3. Run with verbose output:
   ```bash
   debian-composer install <recipe> --verbose
   ```

4. Try dry-run to see what would happen:
   ```bash
   debian-composer install <recipe> --dry-run
   ```

### Recipe Not Found

**Symptom:** `Error: recipe not found: <name>`

**Solutions:**
1. List available recipes:
   ```bash
   debian-composer gallery
   ```

2. Check kitchen path is correct:
   ```bash
   debian-composer info <recipe>
   ```

3. Validate recipe syntax:
   ```bash
   debian-composer validate <recipe>
   ```

### YAML Parsing Errors

**Symptom:** `Error: invalid YAML: ...`

**Solutions:**
1. Validate recipe:
   ```bash
   debian-composer validate <recipe>
   ```

2. Check YAML syntax with linter:
   ```bash
   yamllint <recipe>.yaml
   ```

3. Common issues:
   - Inconsistent indentation
   - Missing colons after keys
   - Arrays not properly prefixed with `-`

### Dependency Conflicts

**Symptom:** `Error: conflict: recipe X conflicts with installed recipe Y`

**Solutions:**
1. List installed recipes:
   ```bash
   debian-composer list
   ```

2. Remove conflicting recipe first:
   ```bash
   debian-composer remove <conflicting-recipe>
   ```

3. Check recipe conflicts field:
   ```bash
   debian-composer info <recipe>
   ```

### Circular Dependencies

**Symptom:** `Error: circular dependency detected`

**Solutions:**
1. Check recipe dependencies in:
   ```bash
   debian-composer info <recipe>
   ```

2. Use `--no-deps` to skip dependency resolution:
   ```bash
   debian-composer install <recipe> --no-deps
   ```

### Hardware Detection Fails

**Symptom:** Hardware detection returns incomplete information.

**Solutions:**
1. Run hardware probe:
   ```bash
   debian-composer probe-hardware
   ```

2. Clear hardware cache (5-minute TTL):
   ```bash
   # Wait 5 minutes or restart the application
   ```

3. Check permissions for hardware queries:
   ```bash
   sudo debian-composer probe-hardware
   ```

### Snapper Snapshot Fails

**Symptom:** `Warning: snapshot failed: ...`

**Solutions:**
1. Check Snapper is installed:
   ```bash
   snapper --version
   ```

2. Check Snapper configuration:
   ```bash
   sudo snapper list-configs
   ```

3. Create snapshot manually:
   ```bash
   sudo snapper create -c timeline --description "debian-composer"
   ```

4. Skip snapshot with `--no-snapshot`:
   ```bash
   debian-composer install <recipe>
   ```

### SSH Hardening Fails

**Symptom:** SSH configuration fails.

**Solutions:**
1. Check SSH is installed:
   ```bash
   which sshd
   ```

2. Dry-run to see changes:
   ```bash
   debian-composer system --dry-run
   ```

3. Backup SSH config manually:
   ```bash
   sudo cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup
   ```

### Permission Denied Errors

**Symptom:** `Error: permission denied`

**Solutions:**
1. Use sudo for system-wide installation:
   ```bash
   sudo debian-composer install <recipe>
   ```

2. Check file permissions:
   ```bash
   ls -la /var/lib/debian-composer/
   ```

### Database Locked

**Symptom:** `Error: database is locked`

**Solutions:**
1. Close other instances:
   ```bash
   pkill debian-composer
   ```

2. Remove stale lock:
   ```bash
   sudo rm /var/lib/debian-composer/state.db
   ```

### Test Failures

**Symptom:** Tests fail during development.

**Solutions:**
1. Run specific test:
   ```bash
   go test ./internal/<module> -v
   ```

2. Run all tests:
   ```bash
   go test ./...
   ```

3. Check test coverage:
   ```bash
   go test -cover ./...
   ```

## Debug Mode

Enable verbose logging:
```bash
debian-composer install <recipe> --verbose
```

## Getting Help

- Check logs: `/var/log/debian-composer/`
- Report issues: https://github.com/debian-composer/debian-composer-go/issues
