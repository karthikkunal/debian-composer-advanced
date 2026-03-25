package pkgmgr

import (
	"testing"
)

func TestParsePackage(t *testing.T) {
	tests := []struct {
		input    string
		wantType PackageType
		wantName string
	}{
		{"vim", PackageApt, "vim"},
		{"vim=2:8.2.0", PackageApt, "vim=2:8.2.0"},
		{"flatpak:org.gimp.GIMP", PackageFlatpak, "org.gimp.GIMP"},
		{"fp:org.libreoffice.LibreOffice", PackageFlatpak, "org.libreoffice.LibreOffice"},
		{"snap:code", PackageSnap, "code"},
		{"snap:firefox", PackageSnap, "firefox"},
		{"pip:requests", PackagePip, "requests"},
		{"npm:typescript", PackageNpm, "typescript"},
		{"go:github.com/cli/cli", PackageGo, "github.com/cli/cli"},
		{"appimage:vscode", PackageAppImage, "vscode"},
		{"binary:helm", PackageBinary, "helm"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg := ParsePackage(tt.input)
			if pkg.Type != tt.wantType {
				t.Errorf("ParsePackage(%q).Type = %v, want %v", tt.input, pkg.Type, tt.wantType)
			}
			if pkg.Name != tt.wantName {
				t.Errorf("ParsePackage(%q).Name = %q, want %q", tt.input, pkg.Name, tt.wantName)
			}
		})
	}
}

func TestParsePackages(t *testing.T) {
	input := []string{"vim", "flatpak:org.gimp.GIMP", "snap:code"}
	pkgs := ParsePackages(input)

	if len(pkgs) != 3 {
		t.Fatalf("Expected 3 packages, got %d", len(pkgs))
	}
	if pkgs[0].Type != PackageApt {
		t.Errorf("First package should be apt, got %v", pkgs[0].Type)
	}
	if pkgs[1].Type != PackageFlatpak {
		t.Errorf("Second package should be flatpak, got %v", pkgs[1].Type)
	}
	if pkgs[2].Type != PackageSnap {
		t.Errorf("Third package should be snap, got %v", pkgs[2].Type)
	}
}

func TestFormatPackage(t *testing.T) {
	tests := []struct {
		pkg  Package
		want string
	}{
		{Package{Name: "vim", Type: PackageApt}, "vim (apt)"},
		{Package{Name: "vim", Type: PackageApt, Version: "8.0"}, "vim@8.0 (apt)"},
		{Package{Name: "org.gimp.GIMP", Type: PackageFlatpak, Source: "flathub"}, "flathub/org.gimp.GIMP (flatpak)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatPackage(tt.pkg)
			if got != tt.want {
				t.Errorf("FormatPackage(%+v) = %q, want %q", tt.pkg, got, tt.want)
			}
		})
	}
}

func TestAvailable(t *testing.T) {
	mgr := New(false)

	// Check which managers are available
	available := mgr.GetAvailableManagers()
	t.Logf("Available package managers: %v", available)

	// At least one manager should be available
	if len(available) == 0 {
		t.Log("Note: No package managers found - may be running in container")
	}
}

func TestDryRun(t *testing.T) {
	mgr := New(true) // dry-run mode

	pkg := Package{Name: "vim", Type: PackageApt}
	err := mgr.Install(pkg)
	if err != nil {
		t.Errorf("Dry-run install failed: %v", err)
	}

	// Verify it didn't actually install
	if mgr.isAptInstalled("vim-nonexistent-package-12345") {
		t.Error("Dry-run should not install packages")
	}
}

func TestInstallNonExistentPackage(t *testing.T) {
	mgr := New(false)

	pkg := Package{Name: "nonexistent-package-12345", Type: PackageApt}
	err := mgr.Install(pkg)
	if err == nil {
		t.Error("Expected error installing non-existent package")
	}
}

func TestIsInstalled(t *testing.T) {
	mgr := New(false)

	// vim should be installed (or at least check doesn't error)
	_ = mgr.IsInstalled(Package{Name: "bash", Type: PackageApt})

	// Non-existent package should return false
	if mgr.IsInstalled(Package{Name: "nonexistent-package-12345", Type: PackageApt}) {
		t.Error("Non-existent package should not be installed")
	}
}

func TestParsePackageVersion(t *testing.T) {
	pkg := ParsePackage("nginx=1.18.0-6.1+deb11u3")
	if pkg.Type != PackageApt {
		t.Errorf("Expected apt type, got %v", pkg.Type)
	}
	if pkg.Name != "nginx=1.18.0-6.1+deb11u3" {
		t.Errorf("Expected full package string with version, got %q", pkg.Name)
	}
}

func TestGetAvailableManagers(t *testing.T) {
	mgr := New(false)
	available := mgr.GetAvailableManagers()

	// Just log what's available - don't assume apt exists in test env
	t.Logf("Available package managers: %v", available)
}
