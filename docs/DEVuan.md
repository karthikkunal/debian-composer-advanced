# Devuan Support

Debian Composer provides first-class support for Devuan and other systemd-free Debian derivatives.

## Supported Systems

| Distribution | Init System | Status |
|-------------|-----------|----------|-------|
| **Devuan** (Ascii/Beowulf/Daedalus) | sysvinit | Fully supported |
| **Devuan** (Chimaera/testing) | sysvinit | Fully supported |
| **antiX** | sysvinit | Fully supported |
| **MX Linux** | sysvinit | Fully supported |
| **Artix Linux** | OpenRC | Fully supported |
| **Alpine Linux** | OpenRC | Fully supported |
| **Void Linux** | runit | Fully supported |
| **GoboLinux** | runit | Fully supported |

## Quick Start

```bash
# Install Devuan workstation
sudo debian-composer --distro=debian-devuan

# With nala APT frontend (Devuan 5+)
sudo debian-composer --distro=debian-devuan --stack=devuan-nala

# Server (no desktop)
sudo debian-composer --distro=debian-devuan --stack=minimal-server
```

## Init System Detection

Debian Composer automatically detects the init system:

```bash
# Check detection
debian-composer info | grep -i init

# Or via internal package
cat /proc/1/comm
```

Output is one of: `systemd`, `openrc-init`, `runit`, `sysvinit`, `unknown`

## Key Differences from Systemd

| Feature | Systemd | SysVInit/OpenRC/Runit |
|---------|--------|---------------------|
| Service enable | `systemctl enable` | `update-rc.d defaults` / `rc-update add` / `ln -s` |
| Service start | `systemctl start` | `service NAME start` / `rc-service start` / `sv start` |
| Service status | `systemctl is-active` | `service NAME status` / `rc-service status` / `sv status` |
| Service restart | `systemctl restart` | `service NAME restart` / `rc-service restart` / `sv restart` |
| Log daemon | journald | rsyslog / syslog-ng / socklog |
| Cron alternative | systemd-cron | cron / anacron / fcron |
| Network | networkd / systemd-networkd | ifupdown / dhcpcd / NetworkManager |
| Login manager | logind | consolekit / without |

## Using Conditionals

Recipes can use `init_system` for conditional logic:

```yaml
variables:
  init_system:
    type: string
    default: sysvinit
    choices: [sysvinit, openrc, runit, systemd]

categories:
  nginx:
    condition: init_system == sysvinit or init_system == openrc or init_system == runit
    description: "Nginx (sysvinit compatible)"
    install:
      packages:
        - nginx
    configure:
      hooks:
        post_install:
          - type: command
            command: "update-rc.d nginx defaults 2>/dev/null || true"
```

## Service Equivalents

| Systemd Unit | SysVInit | OpenRC | Runit |
|-------------|----------|--------|-------|
| `nginx.service` | `update-rc.d nginx defaults` | `rc-update add nginx default` | `ln -s /etc/sv/nginx /etc/runit/runsvdir/default/` |
| `cron.service` | `update-rc.d cron defaults` | `rc-update add cron default` | `ln -s /etc/sv/cron /etc/runit/runsvdir/default/` |
| `ssh.service` | `update-rc.d ssh enable` | `rc-update add ssh default` | `ln -s /etc/sv/ssh /etc/runit/runsvdir/default/` |
| `rsyslog.service` | `update-rc.d rsyslog defaults` | `rc-update add rsyslog default` | `ln -s /etc/sv/rsyslog /etc/runit/runsvdir/default/` |

## APT Configuration

### Devuan Sources

```sourceslist
# /etc/apt/sources.list.d/devuan.list
deb http://pkgmaster.devuan.org/merged daedalus main
deb http://pkgmaster.devuan.org/merged daedalus-updates main
deb http://pkgmaster.devuan.org/merged daedalus-security main
```

### APT Preferences

```aptconf
# /etc/apt/apt.conf.d/99devuan-composer
APT::Get-Options::Allow-Insecure-Releases "false";
Acquire::Languages "en";
```

### APT Frontend Selection

```bash
# Use apt-get (default)
sudo debian-composer install my-recipe

# Use nala (Devuan 5+)
DEBIAN_COMPOSER_PREFERRED_APT=nala sudo debian-composer install my-recipe

# Or via environment variable in recipe
variables:
  preferred_apt:
    type: string
    default: apt-get
    choices: [apt-get, nala]
```

## Recipe Examples

### Devuan-Only Service

```yaml
name: my-service-devuan
version: 1.0.0

includes:
  - fragments/devuan-init.yaml

categories:
  service/my-service:
    <<: *sysvinit_service
    description: "My service (sysvinit compatible)"
    install:
      packages:
        - my-service
    configure:
      service:
        name: my-service
        enable: true
        start: true
```

### Multi-Init Service

```yaml
categories:
  service/my-service:
    description: "My service (all init systems)"
    install:
      packages:
        - my-service
    configure:
      hooks:
        post_install:
          - type: command
            command: |
              init_system=$(cat /proc/1/comm)
              case "$init_system" in
                systemd)
                  systemctl enable my-service
                  systemctl start my-service
                  ;;
                openrc-init)
                  rc-update add my-service default
                  rc-service my-service start
                  ;;
                runit|runit-init)
                  ln -sf /etc/sv/my-service /etc/runit/runsvdir/default/
                  sv my-service start
                  ;;
                *)
                  update-rc.d my-service defaults
                  service my-service start
                  ;;
              esac
```

## Verification

```bash
# Check init system
cat /proc/1/comm

# List enabled services (sysvinit)
ls /etc/rc*.d/

# List enabled services (openrc)
rc-status

# List active services (runit)
ls /etc/runit/runsvdir/current/

# Check package installation
dpkg -l | grep -E "nginx|cron|rsyslog"
```

## Known Issues

| Issue | Workaround |
|-------|----------|
| Nala not available on Devuan < 5 | Use `apt-get` instead |
| `systemctl` in hooks | Replace with `service` command |
| Systemd unit files | Use sysvinit/openrc equivalents |
| `journalctl` | Use `/var/log/syslog` or `rsyslog` |
| `systemd-networkd` | Use `ifupdown` or `NetworkManager` |

## Community Resources

- [Devuan Wiki](https://gitlab.com/devuan/djig)
- [antiX Forum](https://antixfreeforum.com/)
- [MX Linux Forum](https://forum.mxlinux.org/)
- [Artix Wiki](https://wiki.artixlinux.org/)
- [r Void Linux](https://voidlinux.org/)

---

*Debian Composer — compose across all init systems.*