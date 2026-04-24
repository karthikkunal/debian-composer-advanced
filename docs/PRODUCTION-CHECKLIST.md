# Production Readiness Checklist

Use this checklist before deploying Debian Composer in a production environment.

## Pre-Flight Checks

### System Requirements

- [ ] **Minimum RAM:** 512 MB (1 GB recommended)
- [ ] **Minimum Disk:** 10 GB free (20 GB recommended)
- [ ] **CPU:** 1 core (2+ recommended)
- [ ] **Init System:** Verify with `cat /proc/1/comm`
- [ ] **Root Access:** Confirm sudo/root access for package installation

### Package Manager Verification

- [ ] **APT Index Updated:** `sudo apt update` succeeds
- [ ] **Network Access:** `ping -c 1 debian.org` succeeds
- [ ] **HTTPS Support:** `apt-get -o Acquire::https::Timeout=10 update` succeeds
- [ ] **Keyring Installed:** `apt-key list | grep -q Debian || apt-key list | grep -q Devuan`

### Backup & Rollback

- [ ] **Snapper Available:** `snapper --version` (optional but recommended)
- [ ] **Snapshot Created:** `sudo snapper create --description "pre-debian-composer"`
- [ ] **Filesystem Snapshots:** BTRFS/ZFS snapshots configured
- [ ] **Backup Verified:** `/var/backups/` or external backup current
- [ ] **Home Backup:** User configs backed up to external media

### Recipe Validation

- [ ] **Schema Validated:** `debian-composer validate <recipe> --strict`
- [ ] **Dry-Run Passed:** `debian-composer install <recipe> --dry-run`
- [ ] **No Conflicts Detected:** `debian-composer install <recipe> --dry-run | grep -i conflict`
- [ ] **Dependencies Resolved:** All `includes:` and `extends:` files exist
- [ ] **Variables Documented:** All required variables have defaults

### Safety Features

- [ ] **Dry-Run First:** Always run `--dry-run` before actual installation
- [ ] **Snapshot Before:** Run with `--with-snapshot` on production systems
- [ ] **Conflict Detection:** Check existing recipes with `debian-composer list`
- [ ] **Verification Enabled:** All `verify:` commands return expected results
- [ ] **Rollback Tested:** `debian-composer rollback --recipe <name>` works

### Init System Compatibility

- [ ] **Init Detected:** Check output of `debian-composer info | grep -i init`
- [ ] **Service Commands:** `update-rc.d` / `rc-service` / `sv` available
- [ ] **No Systemd Dependencies:** `apt-cache rdepends systemd | grep -v "^systemd$" | grep -v "^Reverse$"` shows acceptable deps
- [ ] **Service Equivalents:** All required services have sysvinit/openrc/runit equivalents

### Network & Security

- [ ] **HTTPS Sources Only:** All repository URLs use `https://`
- [ ] **GPG Keys Verified:** `apt-key list` shows all required keys
- [ ] **Firewall Configured:** `ufw status` (if applicable)
- [ ] **No Untrusted Repos:** `/etc/apt/sources.list.d/` reviewed
- [ ] **Audit Log:** `journalctl -u debian-composer` accessible (sysvinit: `/var/log/`)

### Monitoring

- [ ] **Logging Configured:** `/var/log/debian-composer/` exists
- [ ] **State DB Backup:** `/var/lib/debian-composer/state.db` included in backups
- [ ] **Cron Job:** Periodic `debian-composer list` scheduled
- [ ] **Alert Threshold:** Disk usage < 90% after installation

---

## Deployment Checklist

### Before Deployment

```bash
# 1. Full system backup
sudo tar -czvf /var/backups/pre-debian-composer-$(date +%Y%m%d).tar.gz /

# 2. Update package index
sudo apt update

# 3. Create snapshot (if using snapper)
sudo snapper create --description "pre-debian-composer"

# 4. Validate recipe
sudo debian-composer validate my-recipe --strict

# 5. Dry-run
sudo debian-composer install my-recipe --dry-run

# 6. Check for conflicts
sudo debian-composer list
```

### During Deployment

- [ ] **Keep Terminal Open:** Do not close during installation
- [ ] **Watch Output:** Note any warnings or non-fatal errors
- [ ] **Verify Commands:** Each `verify:` command passes
- [ ] **Note Post-Install Steps:** Save displayed post-install messages

### After Deployment

```bash
# 1. Verify installation
sudo debian-composer list

# 2. Test functionality
debian-composer --version

# 3. Check services
sudo service nginx status   # sysvinit
# or
sudo rc-service nginx status   # openrc

# 4. Create post-snapshot
sudo snapper create --description "post-debian-composer"

# 5. Document changes
sudo debian-composer list > /var/backups/debian-composer-installed-$(date +%Y%m%d).txt
```

---

## Rollback Procedures

### Via Snapper

```bash
# List snapshots
sudo snapper list

# Rollback to pre-installation snapshot
sudo snapper rollback <snapshot-number>

# Reboot
sudo reboot
```

### Via APT

```bash
# List installed packages
sudo dpkg -l | grep debian-composer

# Remove
sudo apt-get remove --purge debian-composer

# Restore
sudo apt-get install <previous-packages>
```

### Manual Rollback

```bash
# Restore recipe state
sudo debian-composer remove my-recipe --keep-config

# Restore config files
sudo cp /var/backups/configs/* /etc/

# Restart services
sudo service nginx restart
```

---

## Health Checks (Periodic)

```bash
#!/bin/bash
# debian-composer-health-check.sh

echo "=== Debian Composer Health Check ==="
echo ""

echo "1. Service Status:"
debian-composer list

echo "2. Recipe Validation:"
debian-composer validate --all

echo "3. Disk Usage:"
df -h /

echo "4. Memory Usage:"
free -h

echo "5. Recent Logs:"
tail -20 /var/log/debian-composer/*.log 2>/dev/null || echo "No logs found"

echo "6. Init System:"
cat /proc/1/comm

echo "7. Last Update:"
stat /var/lib/apt/periodic/update-stamp 2>/dev/null || echo "Unknown"
```

---

## Init System-Specific Checks

### SysVInit (Devuan, MX Linux, antiX)

```bash
# Check init
cat /proc/1/comm

# Check services
ls /etc/init.d/

# Service management
sudo update-rc.d nginx defaults
sudo service nginx start
```

### OpenRC (Artix, Alpine)

```bash
# Check init
cat /proc/1/comm

# Check services
rc-status

# Service management
sudo rc-update add nginx default
sudo rc-service nginx start
```

### Runit

```bash
# Check init
cat /proc/1/comm

# Check services
ls /etc/sv/

# Service management
sudo ln -s /etc/sv/nginx /etc/runit/runsvdir/default/
sudo sv nginx start
```

### Systemd (Debian default)

```bash
# Check init
cat /proc/1/comm

# Service management
sudo systemctl enable nginx
sudo systemctl start nginx
```

---

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|---------|
| Dry-run fails | Check recipe syntax with `--validate` |
| Service won't start | Check init system with `cat /proc/1/comm` |
| Package conflicts | Run `debian-composer list` to check existing |
| Schema errors | Install JSON Schema validator |
| Init system mismatch | Use `--var=init_system=<sysvinit|openrc|runit|systemd>` |

### Debug Mode

```bash
# Verbose output
DEBIAN_COMPOSER_VERBOSE=1 debian-composer install my-recipe

# Debug hooks
DEBIAN_COMPOSER_DEBUG=1 debian-composer install my-recipe --verbose

# Log to file
debian-composer install my-recipe 2>&1 | tee install.log
```

---

*Use this checklist before every production deployment.*