package hardware

import (
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	// This test may fail in CI environments without hardware
	info, err := Detect()

	// Even with errors, we should get partial results
	if info == nil {
		t.Fatal("Detect() returned nil info")
	}

	// CPU should always be populated (even if unknown)
	if info.CPU.Model == "" && err == nil {
		t.Log("CPU model not detected (may be expected in container)")
	}

	// Test String() method
	str := info.String()
	if str == "" {
		t.Fatal("String() returned empty")
	}
}

func TestCPUInfo(t *testing.T) {
	cpu := CPUInfo{
		Model:   "Test CPU",
		Cores:   4,
		Threads: 8,
		Vendor:  "Test Vendor",
		HasVMX:  true,
		HasAVX:  true,
	}

	if cpu.Model != "Test CPU" {
		t.Errorf("Expected model 'Test CPU', got '%s'", cpu.Model)
	}
	if cpu.Cores != 4 {
		t.Errorf("Expected 4 cores, got %d", cpu.Cores)
	}
	if !cpu.HasVMX {
		t.Error("Expected HasVMX to be true")
	}
}

func TestRecommendPackages(t *testing.T) {
	tests := []struct {
		name     string
		info     Info
		wantLen  int
		contains []string
	}{
		{
			name: "NVIDIA GPU",
			info: Info{
				CPU: CPUInfo{HasVMX: true},
				GPUs: []GPUInfo{
					{IsNVIDIA: true},
				},
			},
			wantLen:  3,
			contains: []string{"nvidia-driver", "qemu-kvm"},
		},
		{
			name: "AMD GPU",
			info: Info{
				GPUs: []GPUInfo{
					{IsAMD: true},
				},
			},
			wantLen:  2,
			contains: []string{"firmware-amd-graphics"},
		},
		{
			name: "Intel GPU",
			info: Info{
				GPUs: []GPUInfo{
					{IsIntel: true},
				},
			},
			wantLen:  2,
			contains: []string{"firmware-misc-nonfree"},
		},
		{
			name:    "No GPU",
			info:    Info{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgs := tt.info.RecommendPackages()
			if len(pkgs) < tt.wantLen {
				t.Errorf("Expected at least %d packages, got %d", tt.wantLen, len(pkgs))
			}
			for _, want := range tt.contains {
				found := false
				for _, pkg := range pkgs {
					if strings.Contains(pkg, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected package containing '%s' in recommendations", want)
				}
			}
		})
	}
}

func TestString(t *testing.T) {
	info := Info{
		CPU: CPUInfo{
			Model:   "Intel Core i7",
			Cores:   8,
			Threads: 16,
		},
		Memory: MemoryInfo{
			TotalMB:     32768,
			AvailableMB: 16384,
		},
		GPUs: []GPUInfo{
			{Vendor: "NVIDIA", Model: "RTX 3080"},
		},
	}

	str := info.String()

	checks := []string{"CPU:", "Intel Core i7", "8 cores", "16 threads", "Memory:", "32768 MB", "GPUs:", "NVIDIA"}
	for _, check := range checks {
		if !strings.Contains(str, check) {
			t.Errorf("String() missing '%s'", check)
		}
	}
}
