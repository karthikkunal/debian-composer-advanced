# Hardware Detection

Debian Composer includes comprehensive hardware detection to recommend appropriate packages and drivers.

## Detection Capabilities

- **CPU**: Model, cores, threads, vendor, virtualization support (VT-x, AMD-V), AVX/AVX2
- **Memory**: Total RAM, available RAM, swap, memory type/speed
- **GPUs**: Vendor (NVIDIA, AMD, Intel), model, driver, VRAM
- **Disks**: Device, model, size, type (SSD/HDD/NVMe), filesystem, mount point
- **Network**: Interface, type (ethernet/wifi), MAC, IP, driver
- **USB Devices**: Vendor, product, ID
- **Platform**: Architecture, BIOS vendor/version, product name, chassis type

## Using Hardware Detection

```bash
# Detect hardware and display information
debian-composer probe-hardware

# Show recommended packages based on hardware
debian-composer probe-hardware --recommend
```

## Automatic Package Recommendations

Based on detected hardware, Debian Composer can recommend:

- **GPU drivers**: NVIDIA proprietary/AMD open-source/Intel
- **WiFi firmware**: Broadcom, Realtek, etc.
- **Virtualization packages**: KVM, VirtualBox
- **Performance tools**: CPU microcode, power management

## Integration with Recipes

Recipes can specify hardware requirements:

```yaml
requirements:
  min_ram: 4096    # MB
  min_disk: 20     # GB
  min_cpu: 2       # cores
  gpu: true        # requires GPU
  debian_version: "12"
```

If hardware doesn't meet requirements, installation will show a warning.

## Hardware Caching

Hardware detection results are cached for 5 minutes to improve performance on repeated operations.

## Architecture Support

Debian Composer supports multiple architectures:
- amd64 (x86_64)
- arm64 (aarch64)
- i386 (x86)
- armhf
- ppc64el
- s390x

## Future Enhancements

- Integration with `fwupdmgr` for firmware updates
- High-refresh-rate monitor detection
- Specialized Wi-Fi chip detection
