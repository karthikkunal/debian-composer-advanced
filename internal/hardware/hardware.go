package hardware

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/jaypipes/ghw"
	ghwblock "github.com/jaypipes/ghw/pkg/block"
	gosutilmem "github.com/shirou/gopsutil/v3/mem"
)

var (
	cache     *Info
	cacheTime time.Time
	cacheMu   sync.RWMutex
	cacheTTL  = 5 * time.Minute
)

// Info contains hardware detection results
type Info struct {
	CPU      CPUInfo       `json:"cpu"`
	Memory   MemoryInfo    `json:"memory"`
	GPUs     []GPUInfo     `json:"gpus"`
	Disks    []DiskInfo    `json:"disks"`
	Network  []NetworkInfo `json:"network"`
	USB      []USBInfo     `json:"usb"`
	Platform PlatformInfo  `json:"platform"`
}

// CPUInfo contains CPU information
type CPUInfo struct {
	Model   string `json:"model"`
	Cores   int    `json:"cores"`
	Threads int    `json:"threads"`
	Vendor  string `json:"vendor"`
	Family  string `json:"family"`
	Flags   string `json:"flags"`
	HasVMX  bool   `json:"has_vmx"` // Intel VT-x
	HasSVM  bool   `json:"has_svm"` // AMD-V
	HasAVX  bool   `json:"has_avx"`
	HasAVX2 bool   `json:"has_avx2"`
}

// MemoryInfo contains memory information
type MemoryInfo struct {
	TotalMB     uint64 `json:"total_mb"`
	AvailableMB uint64 `json:"available_mb"`
	SwapTotalMB uint64 `json:"swap_total_mb"`
	Slots       int    `json:"slots"`
	Type        string `json:"type"`
	Speed       string `json:"speed"`
}

// GPUInfo contains GPU information
type GPUInfo struct {
	Vendor   string `json:"vendor"`
	Model    string `json:"model"`
	Driver   string `json:"driver"`
	VRAM     string `json:"vram"`
	PCIAddr  string `json:"pci_addr"`
	IsNVIDIA bool   `json:"is_nvidia"`
	IsAMD    bool   `json:"is_amd"`
	IsIntel  bool   `json:"is_intel"`
}

// DiskInfo contains disk information
type DiskInfo struct {
	Device     string `json:"device"`
	Model      string `json:"model"`
	SizeGB     uint64 `json:"size_gb"`
	Type       string `json:"type"` // ssd, hdd, nvme
	Filesystem string `json:"filesystem"`
	MountPoint string `json:"mount_point"`
}

// NetworkInfo contains network interface information
type NetworkInfo struct {
	Interface  string `json:"interface"`
	Type       string `json:"type"` // ethernet, wifi, bluetooth
	MAC        string `json:"mac"`
	IP         string `json:"ip"`
	Driver     string `json:"driver"`
	IsWireless bool   `json:"is_wireless"`
}

// USBInfo contains USB device information
type USBInfo struct {
	Vendor  string `json:"vendor"`
	Product string `json:"product"`
	ID      string `json:"id"` // vendor:product
}

// PlatformInfo contains platform information
type PlatformInfo struct {
	Architecture string `json:"architecture"`
	BIOSVendor   string `json:"bios_vendor"`
	BIOSVersion  string `json:"bios_version"`
	ProductName  string `json:"product_name"`
	ChassisType  string `json:"chassis_type"`
}

// Detect performs hardware detection using ghw and gopsutil libraries
// Results are cached for 5 minutes to improve performance
func Detect() (*Info, error) {
	cacheMu.RLock()
	if cache != nil && time.Since(cacheTime) < cacheTTL {
		cached := cache
		cacheMu.RUnlock()
		return cached, nil
	}
	cacheMu.RUnlock()

	info, err := detectUncached()
	if err != nil {
		return info, err
	}

	cacheMu.Lock()
	cache = info
	cacheTime = time.Now()
	cacheMu.Unlock()

	return info, nil
}

// detectUncached performs the actual hardware detection with parallel goroutines
func detectUncached() (*Info, error) {
	info := &Info{}

	host, err := ghw.Host()
	if err != nil {
		// ghw may return partial results with an error; continue with what we have
		info.Memory = detectMemory()
		return info, fmt.Errorf("hardware detection: %w", err)
	}

	var wg sync.WaitGroup
	var cpu CPUInfo
	var mem MemoryInfo
	var gpus []GPUInfo
	var disks []DiskInfo
	var nets []NetworkInfo
	var usbs []USBInfo
	var platform PlatformInfo

	wg.Add(7)
	go func() {
		defer wg.Done()
		cpu = detectCPU(host)
	}()
	go func() {
		defer wg.Done()
		mem = detectMemory()
	}()
	go func() {
		defer wg.Done()
		gpus = detectGPU(host)
	}()
	go func() {
		defer wg.Done()
		disks = detectDisk(host)
	}()
	go func() {
		defer wg.Done()
		nets = detectNetwork(host)
	}()
	go func() {
		defer wg.Done()
		usbs = detectUSB(host)
	}()
	go func() {
		defer wg.Done()
		platform = detectPlatform(host)
	}()
	wg.Wait()

	info.CPU = cpu
	info.Memory = mem
	info.GPUs = gpus
	info.Disks = disks
	info.Network = nets
	info.USB = usbs
	info.Platform = platform

	return info, nil
}

func detectCPU(host *ghw.HostInfo) CPUInfo {
	cpu := CPUInfo{}
	if host.CPU == nil || len(host.CPU.Processors) == 0 {
		return cpu
	}

	p := host.CPU.Processors[0]
	cpu.Model = p.Model
	cpu.Vendor = p.Vendor
	cpu.Cores = int(p.NumCores)
	cpu.Threads = int(p.NumThreads)

	// Collect capability flags across all processors
	flagSet := make(map[string]bool)
	for _, proc := range host.CPU.Processors {
		for _, cap := range proc.Capabilities {
			flagSet[cap] = true
		}
	}

	flags := make([]string, 0, len(flagSet))
	for f := range flagSet {
		flags = append(flags, f)
	}
	cpu.Flags = strings.Join(flags, " ")
	cpu.HasVMX = flagSet["vmx"]
	cpu.HasSVM = flagSet["svm"]
	cpu.HasAVX = flagSet["avx"]
	cpu.HasAVX2 = flagSet["avx2"]

	return cpu
}

func detectMemory() MemoryInfo {
	mem := MemoryInfo{}

	if vmem, err := gosutilmem.VirtualMemory(); err == nil {
		mem.TotalMB = vmem.Total / 1024 / 1024
		mem.AvailableMB = vmem.Available / 1024 / 1024
	}

	if swap, err := gosutilmem.SwapMemory(); err == nil {
		mem.SwapTotalMB = swap.Total / 1024 / 1024
	}

	return mem
}

func detectGPU(host *ghw.HostInfo) []GPUInfo {
	var gpus []GPUInfo
	if host.GPU == nil {
		return gpus
	}

	for _, card := range host.GPU.GraphicsCards {
		gpu := GPUInfo{PCIAddr: card.Address}

		if card.DeviceInfo != nil {
			if card.DeviceInfo.Product != nil {
				gpu.Model = card.DeviceInfo.Product.Name
			}
			if card.DeviceInfo.Vendor != nil {
				gpu.Vendor = card.DeviceInfo.Vendor.Name
				lower := strings.ToLower(gpu.Vendor)
				gpu.IsNVIDIA = strings.Contains(lower, "nvidia")
				gpu.IsAMD = strings.Contains(lower, "amd") || strings.Contains(lower, "advanced micro")
				gpu.IsIntel = strings.Contains(lower, "intel")
			}
		}

		gpus = append(gpus, gpu)
	}

	return gpus
}

func detectDisk(host *ghw.HostInfo) []DiskInfo {
	var disks []DiskInfo
	if host.Block == nil {
		return disks
	}

	for _, d := range host.Block.Disks {
		if d.DriveType == ghwblock.DriveTypeFDD || d.IsRemovable {
			continue
		}

		disk := DiskInfo{
			Device: "/dev/" + d.Name,
			Model:  d.Model,
			SizeGB: d.SizeBytes / 1024 / 1024 / 1024,
		}

		switch d.DriveType {
		case ghwblock.DriveTypeSSD:
			disk.Type = "ssd"
		case ghwblock.DriveTypeHDD:
			disk.Type = "hdd"
		default:
			if strings.Contains(strings.ToLower(d.StorageController.String()), "nvme") {
				disk.Type = "nvme"
			} else {
				disk.Type = "unknown"
			}
		}

		for _, part := range d.Partitions {
			if part.MountPoint != "" {
				disk.MountPoint = part.MountPoint
				disk.Filesystem = part.Type
				break
			}
		}

		disks = append(disks, disk)
	}

	return disks
}

func detectNetwork(host *ghw.HostInfo) []NetworkInfo {
	var nets []NetworkInfo
	if host.Network == nil {
		return nets
	}

	for _, nic := range host.Network.NICs {
		if nic.Name == "lo" || nic.IsVirtual {
			continue
		}

		net := NetworkInfo{
			Interface: nic.Name,
			MAC:       nic.MacAddress,
		}

		lower := strings.ToLower(nic.Name)
		switch {
		case strings.HasPrefix(lower, "wl") || strings.Contains(lower, "wifi") || strings.Contains(lower, "wlan"):
			net.IsWireless = true
			net.Type = "wifi"
		case strings.HasPrefix(lower, "bt") || strings.Contains(lower, "bluetooth"):
			net.Type = "bluetooth"
		default:
			net.Type = "ethernet"
		}

		// Cross-check with sysfs wireless directory
		if !net.IsWireless {
			if _, err := os.Stat(fmt.Sprintf("/sys/class/net/%s/wireless", nic.Name)); err == nil {
				net.IsWireless = true
				net.Type = "wifi"
			}
		}

		nets = append(nets, net)
	}

	return nets
}

func detectUSB(host *ghw.HostInfo) []USBInfo {
	var usbs []USBInfo
	if host.USB == nil {
		return usbs
	}

	for _, dev := range host.USB.Devices {
		usbs = append(usbs, USBInfo{
			ID:      dev.VendorID + ":" + dev.ProductID,
			Product: dev.Product,
		})
	}

	return usbs
}

func detectPlatform(host *ghw.HostInfo) PlatformInfo {
	plat := PlatformInfo{
		Architecture: runtime.GOARCH,
	}

	if host.Product != nil {
		plat.ProductName = host.Product.Name
	}
	if host.BIOS != nil {
		plat.BIOSVendor = host.BIOS.Vendor
		plat.BIOSVersion = host.BIOS.Version
	}
	if host.Chassis != nil {
		plat.ChassisType = host.Chassis.Type
	}

	return plat
}

// RecommendPackages returns package recommendations based on hardware
func (i *Info) RecommendPackages() []string {
	var pkgs []string

	for _, gpu := range i.GPUs {
		if gpu.IsNVIDIA {
			pkgs = append(pkgs, "nvidia-driver", "nvidia-settings")
		} else if gpu.IsAMD {
			pkgs = append(pkgs, "firmware-amd-graphics", "xserver-xorg-video-amdgpu")
		} else if gpu.IsIntel {
			pkgs = append(pkgs, "firmware-misc-nonfree", "xserver-xorg-video-intel")
		}
	}

	if i.CPU.HasVMX || i.CPU.HasSVM {
		pkgs = append(pkgs, "qemu-kvm", "libvirt-clients", "libvirt-daemon-system", "virt-manager")
	}

	return pkgs
}

// String returns a human-readable summary
func (i *Info) String() string {
	var sb strings.Builder

	sb.WriteString("CPU: ")
	if i.CPU.Model != "" {
		sb.WriteString(fmt.Sprintf("%s (%d cores, %d threads)\n", i.CPU.Model, i.CPU.Cores, i.CPU.Threads))
	} else {
		sb.WriteString("unknown\n")
	}

	sb.WriteString("Memory: ")
	if i.Memory.TotalMB > 0 {
		sb.WriteString(fmt.Sprintf("%d MB total, %d MB available\n", i.Memory.TotalMB, i.Memory.AvailableMB))
	} else {
		sb.WriteString("unknown\n")
	}

	sb.WriteString(fmt.Sprintf("GPUs: %d detected\n", len(i.GPUs)))
	for _, gpu := range i.GPUs {
		sb.WriteString(fmt.Sprintf("  - %s %s\n", gpu.Vendor, gpu.Model))
	}

	sb.WriteString(fmt.Sprintf("Disks: %d detected\n", len(i.Disks)))
	for _, disk := range i.Disks {
		sb.WriteString(fmt.Sprintf("  - %s (%d GB)\n", disk.Device, disk.SizeGB))
	}

	sb.WriteString(fmt.Sprintf("Network: %d interfaces\n", len(i.Network)))
	for _, net := range i.Network {
		sb.WriteString(fmt.Sprintf("  - %s (%s)\n", net.Interface, net.Type))
	}

	return sb.String()
}
