package pkgmgr

import (
	"fmt"
	"os/exec"
	"strings"
)

// Manager handles multi-package-manager operations
type Manager struct {
	dryRun  bool
	useNala bool
}

// New creates a new package manager
func New(dryRun bool) *Manager {
	useNala := commandExists("nala")
	return &Manager{dryRun: dryRun, useNala: useNala}
}

// SetUseNala overrides the default nala detection
func (m *Manager) SetUseNala(use bool) {
	m.useNala = use
}

// PackageType represents the package manager type
type PackageType string

const (
	PackageApt      PackageType = "apt"
	PackageFlatpak  PackageType = "flatpak"
	PackageSnap     PackageType = "snap"
	PackageAppImage PackageType = "appimage"
	PackageBinary   PackageType = "binary"
	PackagePip      PackageType = "pip"
	PackageNpm      PackageType = "npm"
	PackageGo       PackageType = "go"
)

// Package represents a package to install
type Package struct {
	Name    string      `yaml:"name" json:"name"`
	Type    PackageType `yaml:"type" json:"type"` // apt, flatpak, snap, etc.
	Version string      `yaml:"version" json:"version"`
	Source  string      `yaml:"source" json:"source"` // flatpak remote, npm scope, etc.
}

// aptCommand returns the apt frontend command (nala if available, else apt-get)
func (m *Manager) aptCommand() string {
	if m.useNala {
		return "nala"
	}
	return "apt-get"
}

// ParsePackage parses a package string into a Package
// Format: "pkg", "type:pkg", "type:pkg@version", "type:source:pkg"
func ParsePackage(pkgStr string) Package {
	pkg := Package{
		Type: PackageApt, // Default to apt
	}

	// Check for type prefix (e.g., "flatpak:org.gimp.GIMP")
	if idx := strings.Index(pkgStr, ":"); idx > 0 {
		prefix := pkgStr[:idx]
		rest := pkgStr[idx+1:]

		// Check if it's a known type
		switch strings.ToLower(prefix) {
		case "flatpak", "fp":
			pkg.Type = PackageFlatpak
			pkg.Name = rest
			return pkg
		case "snap", "sn":
			pkg.Type = PackageSnap
			pkg.Name = rest
			return pkg
		case "appimage", "ai":
			pkg.Type = PackageAppImage
			pkg.Name = rest
			return pkg
		case "binary", "bin":
			pkg.Type = PackageBinary
			pkg.Name = rest
			return pkg
		case "pip", "python":
			pkg.Type = PackagePip
			pkg.Name = rest
			return pkg
		case "npm", "node":
			pkg.Type = PackageNpm
			pkg.Name = rest
			return pkg
		case "go", "golang":
			pkg.Type = PackageGo
			pkg.Name = rest
			return pkg
		}
	}

	// Not a known prefix, treat as apt package with possible version
	pkg.Name = pkgStr
	return pkg
}

// ParsePackages parses multiple package strings
func ParsePackages(pkgStrs []string) []Package {
	var packages []Package
	for _, s := range pkgStrs {
		packages = append(packages, ParsePackage(s))
	}
	return packages
}

// Install installs a package using the appropriate package manager
func (m *Manager) Install(pkg Package) error {
	if m.dryRun {
		fmt.Printf("[DRY-RUN] Would install %s: %s\n", pkg.Type, pkg.Name)
		return nil
	}

	switch pkg.Type {
	case PackageApt:
		return m.installApt(pkg)
	case PackageFlatpak:
		return m.installFlatpak(pkg)
	case PackageSnap:
		return m.installSnap(pkg)
	case PackagePip:
		return m.installPip(pkg)
	case PackageNpm:
		return m.installNpm(pkg)
	case PackageGo:
		return m.installGo(pkg)
	case PackageAppImage:
		return m.installAppImage(pkg)
	case PackageBinary:
		return m.installBinary(pkg)
	default:
		return fmt.Errorf("unsupported package type: %s", pkg.Type)
	}
}

// InstallAll installs multiple packages
func (m *Manager) InstallAll(packages []Package) error {
	// Group by type for batch operations
	byType := make(map[PackageType][]Package)
	for _, pkg := range packages {
		byType[pkg.Type] = append(byType[pkg.Type], pkg)
	}

	// Install apt packages in batch (fastest)
	if aptPkgs, ok := byType[PackageApt]; ok {
		if err := m.installAptBatch(aptPkgs); err != nil {
			return fmt.Errorf("apt install: %w", err)
		}
		delete(byType, PackageApt)
	}

	// Install others one by one
	for pkgType, pkgs := range byType {
		for _, pkg := range pkgs {
			if err := m.Install(pkg); err != nil {
				return fmt.Errorf("%s install %s: %w", pkgType, pkg.Name, err)
			}
		}
	}

	return nil
}

// Remove removes a package
func (m *Manager) Remove(pkg Package) error {
	if m.dryRun {
		fmt.Printf("[DRY-RUN] Would remove %s: %s\n", pkg.Type, pkg.Name)
		return nil
	}

	switch pkg.Type {
	case PackageApt:
		return m.removeApt(pkg)
	case PackageFlatpak:
		return m.removeFlatpak(pkg)
	case PackageSnap:
		return m.removeSnap(pkg)
	default:
		return fmt.Errorf("remove not supported for type: %s", pkg.Type)
	}
}

// IsInstalled checks if a package is installed
func (m *Manager) IsInstalled(pkg Package) bool {
	switch pkg.Type {
	case PackageApt:
		return m.isAptInstalled(pkg.Name)
	case PackageFlatpak:
		return m.isFlatpakInstalled(pkg.Name)
	case PackageSnap:
		return m.isSnapInstalled(pkg.Name)
	default:
		return false
	}
}

// Available checks if the package manager is available
func (m *Manager) Available(pkgType PackageType) bool {
	switch pkgType {
	case PackageApt:
		return commandExists("apt-get") || commandExists("nala")
	case PackageFlatpak:
		return commandExists("flatpak")
	case PackageSnap:
		return commandExists("snap")
	case PackagePip:
		return commandExists("pip3") || commandExists("pip")
	case PackageNpm:
		return commandExists("npm")
	case PackageGo:
		return commandExists("go")
	default:
		return false
	}
}

// GetAvailableManagers returns list of available package managers
func (m *Manager) GetAvailableManagers() []PackageType {
	var available []PackageType
	for _, t := range []PackageType{PackageApt, PackageFlatpak, PackageSnap, PackagePip, PackageNpm, PackageGo} {
		if m.Available(t) {
			available = append(available, t)
		}
	}
	return available
}

// === APT implementations ===

func (m *Manager) installApt(pkg Package) error {
	args := []string{"install", "-y"}
	if pkg.Version != "" {
		pkg.Name = fmt.Sprintf("%s=%s", pkg.Name, pkg.Version)
	}
	args = append(args, pkg.Name)

	cmd := exec.Command("sudo", append([]string{m.aptCommand()}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apt install failed: %w\nOutput: %s", err, out)
	}
	return nil
}

func (m *Manager) installAptBatch(packages []Package) error {
	args := []string{"install", "-y"}
	for _, pkg := range packages {
		name := pkg.Name
		if pkg.Version != "" {
			name = fmt.Sprintf("%s=%s", pkg.Name, pkg.Version)
		}
		args = append(args, name)
	}

	cmd := exec.Command("sudo", append([]string{m.aptCommand()}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apt batch install failed: %w\nOutput: %s", err, out)
	}
	return nil
}

func (m *Manager) removeApt(pkg Package) error {
	cmd := exec.Command("sudo", m.aptCommand(), "remove", "-y", pkg.Name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apt remove failed: %w\nOutput: %s", err, out)
	}
	return nil
}

func (m *Manager) isAptInstalled(name string) bool {
	cmd := exec.Command("dpkg", "-s", name)
	return cmd.Run() == nil
}

// === Flatpak implementations ===

func (m *Manager) installFlatpak(pkg Package) error {
	// Ensure flatpak is installed
	if !m.Available(PackageFlatpak) {
		return fmt.Errorf("flatpak not installed")
	}

	// Add flathub remote if not present
	exec.Command("flatpak", "remote-add", "--if-not-exists", "flathub", "https://flathub.org/repo/flathub.flatpakrepo").Run()

	remote := "flathub"
	if pkg.Source != "" {
		remote = pkg.Source
	}

	cmd := exec.Command("flatpak", "install", "-y", remote, pkg.Name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("flatpak install failed: %w\nOutput: %s", err, out)
	}
	return nil
}

func (m *Manager) removeFlatpak(pkg Package) error {
	cmd := exec.Command("flatpak", "uninstall", "-y", pkg.Name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("flatpak uninstall failed: %w\nOutput: %s", err, out)
	}
	return nil
}

func (m *Manager) isFlatpakInstalled(name string) bool {
	cmd := exec.Command("flatpak", "list", "--app")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), name)
}

// === Snap implementations ===

func (m *Manager) installSnap(pkg Package) error {
	if !m.Available(PackageSnap) {
		return fmt.Errorf("snap not installed")
	}

	cmd := exec.Command("sudo", "snap", "install", pkg.Name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("snap install failed: %w\nOutput: %s", err, out)
	}
	return nil
}

func (m *Manager) removeSnap(pkg Package) error {
	cmd := exec.Command("sudo", "snap", "remove", pkg.Name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("snap remove failed: %w\nOutput: %s", err, out)
	}
	return nil
}

func (m *Manager) isSnapInstalled(name string) bool {
	cmd := exec.Command("snap", "list", name)
	return cmd.Run() == nil
}

// === Pip implementations ===

func (m *Manager) installPip(pkg Package) error {
	name := pkg.Name
	if pkg.Version != "" {
		name = fmt.Sprintf("%s==%s", pkg.Name, pkg.Version)
	}

	cmd := exec.Command("pip3", "install", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pip install failed: %w\nOutput: %s", err, out)
	}
	return nil
}

// === NPM implementations ===

func (m *Manager) installNpm(pkg Package) error {
	name := pkg.Name
	if pkg.Version != "" {
		name = fmt.Sprintf("%s@%s", pkg.Name, pkg.Version)
	}

	cmd := exec.Command("npm", "install", "-g", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install failed: %w\nOutput: %s", err, out)
	}
	return nil
}

// === Go implementations ===

func (m *Manager) installGo(pkg Package) error {
	cmd := exec.Command("go", "install", pkg.Name+"@latest")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go install failed: %w\nOutput: %s", err, out)
	}
	return nil
}

// === AppImage implementations ===

func (m *Manager) installAppImage(pkg Package) error {
	// AppImage typically requires downloading
	// pkg.Name should be a URL or identifier
	fmt.Printf("AppImage installation requires manual download: %s\n", pkg.Name)
	fmt.Println("Download from the official source and place in ~/Applications/")
	return nil
}

// === Binary implementations ===

func (m *Manager) installBinary(pkg Package) error {
	// Binary typically requires downloading
	// pkg.Name should be a URL or identifier
	fmt.Printf("Binary installation: %s\n", pkg.Name)
	fmt.Println("Download from the official source and place in /usr/local/bin/")
	return nil
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// FormatPackage formats a package for display
func FormatPackage(pkg Package) string {
	if pkg.Version != "" {
		return fmt.Sprintf("%s@%s (%s)", pkg.Name, pkg.Version, pkg.Type)
	}
	if pkg.Source != "" {
		return fmt.Sprintf("%s/%s (%s)", pkg.Source, pkg.Name, pkg.Type)
	}
	return fmt.Sprintf("%s (%s)", pkg.Name, pkg.Type)
}
