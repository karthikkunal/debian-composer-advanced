package desktop

import (
	"fmt"
	"sort"
	"strings"

	"github.com/debian-composer/debian-composer-go/internal/hardware"
)

type Environment struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MinRAMMB    int    `json:"min_ram_mb"`
	MinDiskGB   int    `json:"min_disk_gb"`
	RequiresGPU bool   `json:"requires_gpu"`
	Packages    []string
}

var Environments = map[string]Environment{
	"gnome": {
		Name:        "gnome",
		Description: "Modern, polished, and user-friendly desktop environment",
		MinRAMMB:    4096,
		MinDiskGB:   15,
		RequiresGPU: false,
		Packages:    []string{"gnome", "gnome-core", "gnome-software"},
	},
	"kde": {
		Name:        "kde",
		Description: "Feature-rich, customizable KDE Plasma desktop",
		MinRAMMB:    4096,
		MinDiskGB:   15,
		RequiresGPU: false,
		Packages:    []string{"plasma-desktop", "plasma-workspace", "plasma-desktop"},
	},
	"xfce": {
		Name:        "xfce",
		Description: "Lightweight, fast, and stable desktop environment",
		MinRAMMB:    2048,
		MinDiskGB:   10,
		RequiresGPU: false,
		Packages:    []string{"xfce4", "xfce4-goodies"},
	},
	"mate": {
		Name:        "mate",
		Description: "Traditional GNOME 2 fork, classic desktop experience",
		MinRAMMB:    2048,
		MinDiskGB:   10,
		RequiresGPU: false,
		Packages:    []string{"mate-desktop-environment", "mate-desktop-environment-core"},
	},
	"cinnamon": {
		Name:        "cinnamon",
		Description: "Modern, elegant, and full-featured desktop",
		MinRAMMB:    3072,
		MinDiskGB:   12,
		RequiresGPU: false,
		Packages:    []string{"cinnamon", "cinnamon-desktop-environment"},
	},
	"lxde": {
		Name:        "lxde",
		Description: "Ultra-lightweight desktop for older hardware",
		MinRAMMB:    1024,
		MinDiskGB:   8,
		RequiresGPU: false,
		Packages:    []string{"lxde", "lxde-core"},
	},
	"lxqt": {
		Name:        "lxqt",
		Description: "Modern Qt-based lightweight desktop",
		MinRAMMB:    1536,
		MinDiskGB:   8,
		RequiresGPU: false,
		Packages:    []string{"lxqt", "lxqt-core"},
	},
	"i3": {
		Name:        "i3",
		Description: "Tiling window manager for keyboard-driven workflow",
		MinRAMMB:    1024,
		MinDiskGB:   5,
		RequiresGPU: false,
		Packages:    []string{"i3", "i3-wm", "i3status", "dmenu"},
	},
	"sway": {
		Name:        "sway",
		Description: "Wayland tiling window manager, i3-compatible",
		MinRAMMB:    2048,
		MinDiskGB:   8,
		RequiresGPU: true,
		Packages:    []string{"sway", "swayidle", "waybar"},
	},
	"budgie": {
		Name:        "budgie",
		Description: "Modern, simple, and elegant desktop",
		MinRAMMB:    2048,
		MinDiskGB:   10,
		RequiresGPU: false,
		Packages:    []string{"budgie-desktop", "budgie-desktop-view"},
	},
	"deepin": {
		Name:        "deepin",
		Description: "Beautiful, macOS-like desktop environment",
		MinRAMMB:    4096,
		MinDiskGB:   20,
		RequiresGPU: false,
		Packages:    []string{"deepin-desktop-theme", "deepin-desktop-environment"},
	},
	"cosmic": {
		Name:        "cosmic",
		Description: "COSMIC desktop (early access, available on Pop!_OS)",
		MinRAMMB:    4096,
		MinDiskGB:   15,
		RequiresGPU: true,
		Packages:    []string{"cosmic-applets", "cosmic-panel", "cosmic-workspaces"},
	},
}

func GetEnvironment(name string) (Environment, bool) {
	e, ok := Environments[strings.ToLower(name)]
	return e, ok
}

func ListEnvironments() []Environment {
	var list []Environment
	for _, e := range Environments {
		list = append(list, e)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].MinRAMMB < list[j].MinRAMMB
	})
	return list
}

func GetRecommendations(hwInfo *hardware.Info) []Environment {
	var recommended []Environment

	for _, env := range Environments {
		meetsRAM := int(hwInfo.Memory.TotalMB) >= env.MinRAMMB
		meetsGPU := !env.RequiresGPU || len(hwInfo.GPUs) > 0

		if meetsRAM && meetsGPU {
			recommended = append(recommended, env)
		}
	}

	sort.Slice(recommended, func(i, j int) bool {
		return recommended[i].MinRAMMB < recommended[j].MinRAMMB
	})

	return recommended
}

func GetCompatible(hwInfo *hardware.Info) []Environment {
	var compatible []Environment

	for _, env := range Environments {
		meetsRAM := int(hwInfo.Memory.TotalMB) >= env.MinRAMMB

		if meetsRAM {
			compatible = append(compatible, env)
		}
	}

	return compatible
}

func CheckCompatibility(env Environment, hwInfo *hardware.Info) error {
	if int(hwInfo.Memory.TotalMB) < env.MinRAMMB {
		return fmt.Errorf("insufficient RAM: have %d MB, need %d MB", hwInfo.Memory.TotalMB, env.MinRAMMB)
	}

	if env.RequiresGPU && len(hwInfo.GPUs) == 0 {
		return fmt.Errorf("environment %s requires a GPU", env.Name)
	}

	return nil
}
