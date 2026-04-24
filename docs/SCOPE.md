# Scope: Debian Composer vs Ansible

Debian Composer and Ansible occupy different positions on the automation spectrum.

## Comparison

| Dimension | Debian Composer | Ansible |
|-----------|----------------|---------|
| **Focus** | Post-install host configuration | Infrastructure provisioning + config management |
| **Scope** | Single machine | Fleet / remote nodes |
| **YAML** | Declarative recipe | Declarative playbook |
| **Execution** | Recipe install command | `ansible-playbook` + inventory |
| **State** | SQLite state DB | Dynamic inventory / facts |
| **Convergence** | `debian-composer install` | `ansible-pull` or push |
| **Host-level only** | Yes | Partial (with pull mode) |
| **Remote nodes** | No | Yes (push mode) |
| **Package manager** | apt/nala + flatpak/snap/pip | apt module |
| **Init system** | systemd, sysvinit, OpenRC, runit | systemd module (primary) |

## When to Use Debian Composer

Debian Composer is purpose-built for **host-level post-installation configuration**:

- **Single-host declarative setup** — You want to define what a machine *is* (not orchestrate fleet changes)
- **Recipe sharing** — You want to publish/share a system configuration as a portable YAML file
- **No inventory needed** — You run directly on the host, not remotely
- **Init system abstraction** — You need to support systemd-free systems (Devuan, antiX)
- **YAML composability** — You want `include/extend/layer` for modular configs
- **No SSH required** — No passwordless SSH, no inventory, no control node

**Examples:**
- Set up a developer workstation from a shared recipe
- Configure a home lab server with specific services
- Apply a team's standard configuration to a new machine
- Create portable system configs for community sharing

## When to Use Ansible

Ansible is purpose-built for **fleet-level infrastructure automation**:

- **Multi-host orchestration** — You need to configure 10+ machines from one control node
- **Infrastructure provisioning** — Creating VMs, networks, cloud resources
- **Remote execution** — You cannot log into machines directly
- **Idempotent config management** — Continuous config drift correction across a fleet
- **Cloud integrations** — AWS/GCP/Azure modules built in
- **CI/CD pipelines** — Ansible Tower, AAP, community collections

**Examples:**
- Provision and configure a 50-node web server fleet
- Manage network devices and firewalls
- Cloud resource orchestration (EC2, S3, RDS)
- Continuous configuration enforcement

## Cross-over: Using Both

Debian Composer and Ansible can complement each other:

```
┌─────────────────────────────────────────────────────────────┐
│ Ansible (Fleet Provisioning)                          │
│  - Provision VMs / cloud resources                 │
│  - Install base Debian system                       │
│  - Deploy debian-composer binary                  │
│  - Invoke debian-composer via ansible.raw          │
│    (or ansible.builtin.command)                  │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│ Debian Composer (Host Configuration)              │
│  - Apply recipe with debian-composer install     │
│  - Recipe handles packages + services          │
│  - Recipe handles init system abstraction    │
│  - Recipe handles validation               │
└─────────────────────────────────────────────────────────────┘
```

### Example: Ansible calling Debian Composer

```yaml
# ansible-playbook.yml
- hosts: workstations
  become: true
  tasks:
    - name: Ensure Debian Composer is installed
      ansible.builtin.apt:
        url: https://.../debian-composer.deb
        state: present

    - name: Apply standard developer workstation recipe
      ansible.builtin.command:
        cmd: debian-composer install debian-developer --dry-run
      register: result

    - name: Show preview
      ansible.builtin.debug:
        msg: "{{ result.stdout }}"

    - name: Actually install
      when: "'--dry-run' not in '{{ lookup('env', 'DRY_RUN') }}'|default('')"
      ansible.builtin.command:
        cmd: debian-composer install debian-developer
```

### Example: Debian Composer recipe calling Ansible (rare)

```yaml
# recipe in debian-composer
hooks:
  post_install:
    - type: command
      command: >
        ansible-pull -i localhost -U https://git/my-fleet.git
        # Pull fleet config AFTER host is set up
```

## Key Differences in Detail

### Package Management

Debian Composer uses a **unified package manager interface**:

```yaml
# Debian Composer — package in recipe
packages:
  - nginx
  - postgresql
  - flatpak: org.gimp.GIMP
  - snap: code --classic
  - pip: numpy
```

Ansible uses **per-type modules**:

```yaml
# Ansible — different modules per type
tasks:
  - ansible.builtin.apt:
      name: nginx
  - community.general.flatpak:
      name: org.gimp.GIMP
  - ansible.builtin.pip:
      name: numpy
```

### Service Management

Debian Composer uses **init system abstraction** (`initsys`):

```yaml
# Works on systemd, sysvinit, OpenRC, runit
configure:
  service:
    name: nginx
    enable: true
    start: true
```

Ansible defaults to **systemd** (with `become: true`):

```yaml
# Ansible — defaults to systemd
tasks:
  - ansible.builtin.service:
      name: nginx
      enabled: true
      state: started
```

### State Tracking

Debian Composer uses **SQLite** for state tracking:

```bash
debian-composer list                # Show installed recipes
debian-composer remove my-recipe   # Remove recipe
```

Ansible uses **dynamic facts**:

```yaml
# Ansible — check changed
tasks:
  - ansible.builtin.apt:
      name: nginx
  register: result
- ansible.builtin.debug:
    msg: "{{ result.changed }}"
```

## Conclusion

Debian Composer is **not trying to replace Ansible**. They serve different purposes:

- **Ansible** → Fleet orchestration, infrastructure provisioning, cloud management
- **Debian Composer** → Host-level configuration, recipe sharing, init system abstraction

Use Debian Composer when you want portable, shareable YAML recipes that work across all Debian derivatives including Devuan. Use Ansible when you need to manage a fleet from a central control node.

The ideal workflow: **Ansible provisions → Debian Composer configures**. This is the `S1 + T1` strategy — focus Debian Composer on what it does best (host config) rather than competing on Ansible's terrain (fleet orchestration).