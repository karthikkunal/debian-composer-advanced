package apt

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/bluet/syspkg"
	"github.com/bluet/syspkg/manager"
	"github.com/debian-composer/debian-composer-go/internal/pkgmgr"
)

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// APT handles Debian package operations via syspkg and multi-package-manager support.
// It acts as a unified interface for apt, nala, flatpak, snap, pip, npm, go, etc.
type APT struct {
	dryRun  bool
	pm      syspkg.PackageManager
	pkgMgr  *pkgmgr.Manager
	useNala bool
}

// New creates a new APT handler backed by syspkg and pkgmgr
func New(dryRun bool) *APT {
	// Determine preferred apt frontend based on environment variable
	useNala := commandExists("nala")
	if env := os.Getenv("DEBIAN_COMPOSER_PREFERRED_APT"); env != "" {
		switch strings.ToLower(env) {
		case "nala":
			useNala = true
		case "apt", "apt-get":
			useNala = false
		}
	}

	sp, err := syspkg.New(syspkg.IncludeOptions{Apt: true})
	if err != nil {
		// Fall back to nil pm; all operations will no-op or error gracefully
		pkgMgr := pkgmgr.New(dryRun)
		pkgMgr.SetUseNala(useNala)
		return &APT{dryRun: dryRun, pkgMgr: pkgMgr, useNala: useNala}
	}
	pm, err := sp.GetPackageManager("apt")
	if err != nil {
		pkgMgr := pkgmgr.New(dryRun)
		pkgMgr.SetUseNala(useNala)
		return &APT{dryRun: dryRun, pkgMgr: pkgMgr, useNala: useNala}
	}
	pkgMgr := pkgmgr.New(dryRun)
	pkgMgr.SetUseNala(useNala)
	return &APT{dryRun: dryRun, pm: pm, pkgMgr: pkgMgr, useNala: useNala}
}

func (a *APT) opts() *manager.Options {
	return &manager.Options{
		AssumeYes: true,
		DryRun:    a.dryRun,
	}
}

func (a *APT) available() bool {
	return a.pm != nil
}

func (a *APT) nalaAvailable() bool {
	return a.useNala && commandExists("nala")
}

// runNala runs nala command with sudo and returns output
func (a *APT) runNala(args ...string) ([]byte, error) {
	if !a.nalaAvailable() {
		return nil, fmt.Errorf("nala not available")
	}
	cmd := exec.Command("sudo", append([]string{"nala"}, args...)...)
	return cmd.CombinedOutput()
}

// Install installs packages (supports apt, flatpak, snap, etc.)
func (a *APT) Install(packages ...string) error {
	if len(packages) == 0 {
		return nil
	}
	// Parse each package and delegate to appropriate manager
	for _, pkgStr := range packages {
		pkg := pkgmgr.ParsePackage(pkgStr)
		if pkg.Type == pkgmgr.PackageApt && a.useNala {
			// Use pkgmgr (nala) for apt packages when nala is available
			if err := a.pkgMgr.Install(pkg); err != nil {
				return err
			}
		} else if pkg.Type == pkgmgr.PackageApt && a.available() {
			// Use syspkg for apt packages
			if _, err := a.pm.Install([]string{pkg.Name}, a.opts()); err != nil {
				return err
			}
		} else {
			// Use pkgmgr for non-apt packages
			if err := a.pkgMgr.Install(pkg); err != nil {
				return err
			}
		}
	}
	return nil
}

// BatchInstall installs multiple packages, grouping apt packages for batch install
func (a *APT) BatchInstall(packages []string) error {
	if len(packages) == 0 {
		return nil
	}
	// Group packages by type
	var aptPkgs []string
	var otherPkgs []pkgmgr.Package
	for _, pkgStr := range packages {
		pkg := pkgmgr.ParsePackage(pkgStr)
		if pkg.Type == pkgmgr.PackageApt {
			aptPkgs = append(aptPkgs, pkg.Name)
		} else {
			otherPkgs = append(otherPkgs, pkg)
		}
	}
	// Install apt packages in batch
	if len(aptPkgs) > 0 {
		if a.useNala {
			// Use pkgmgr (nala) for apt packages when nala is available
			// Convert aptPkgs to Package slice
			pkgs := make([]pkgmgr.Package, len(aptPkgs))
			for i, name := range aptPkgs {
				pkgs[i] = pkgmgr.Package{Name: name, Type: pkgmgr.PackageApt}
			}
			if err := a.pkgMgr.InstallAll(pkgs); err != nil {
				return err
			}
		} else if a.available() {
			if _, err := a.pm.Install(aptPkgs, a.opts()); err != nil {
				return err
			}
		}
	}
	// Install other packages one by one
	for _, pkg := range otherPkgs {
		if err := a.pkgMgr.Install(pkg); err != nil {
			return err
		}
	}
	return nil
}

// Remove removes packages (supports apt, flatpak, snap, etc.)
func (a *APT) Remove(packages ...string) error {
	if len(packages) == 0 {
		return nil
	}
	for _, pkgStr := range packages {
		pkg := pkgmgr.ParsePackage(pkgStr)
		if pkg.Type == pkgmgr.PackageApt && a.useNala {
			// Use pkgmgr (nala) for apt packages when nala is available
			if err := a.pkgMgr.Remove(pkg); err != nil {
				return err
			}
		} else if pkg.Type == pkgmgr.PackageApt && a.available() {
			if _, err := a.pm.Delete([]string{pkg.Name}, a.opts()); err != nil {
				return err
			}
		} else {
			// pkgmgr.Remove currently only supports apt, flatpak, snap
			if err := a.pkgMgr.Remove(pkg); err != nil {
				return err
			}
		}
	}
	return nil
}

// PurgeKeepConfig removes packages but keeps configuration files
func (a *APT) PurgeKeepConfig(packages ...string) error {
	// For apt, this is equivalent to remove (not purge)
	return a.Remove(packages...)
}

// Update runs apt update (refresh package index)
func (a *APT) Update() error {
	if a.useNala {
		cmd := exec.Command("sudo", "nala", "update")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("nala update failed: %w\nOutput: %s", err, out)
		}
		return nil
	}
	if !a.available() {
		return fmt.Errorf("apt package manager not available")
	}
	return a.pm.Refresh(a.opts())
}

// IsInstalled checks if a package is installed (supports multiple package types)
func (a *APT) IsInstalled(pkg string) bool {
	parsed := pkgmgr.ParsePackage(pkg)
	if parsed.Type == pkgmgr.PackageApt && a.available() {
		results, err := a.pm.Find([]string{parsed.Name}, &manager.Options{})
		if err != nil {
			return false
		}
		for _, r := range results {
			if r.Name == parsed.Name && r.Status == "installed" {
				return true
			}
		}
		return false
	}
	// For non-apt packages, use pkgmgr
	return a.pkgMgr.IsInstalled(parsed)
}

// Search searches for packages using the package name/keyword
func (a *APT) Search(query string) ([]PackageInfo, error) {
	if !a.available() {
		return nil, fmt.Errorf("apt package manager not available")
	}
	results, err := a.pm.Find([]string{query}, &manager.Options{})
	if err != nil {
		return nil, err
	}
	pkgs := make([]PackageInfo, 0, len(results))
	for _, r := range results {
		pkgs = append(pkgs, PackageInfo{
			Name:    r.Name,
			Version: r.Version,
		})
	}
	return pkgs, nil
}

// ListInstalled returns all installed packages
func (a *APT) ListInstalled() ([]string, error) {
	if !a.available() {
		return nil, fmt.Errorf("apt package manager not available")
	}
	results, err := a.pm.ListInstalled(&manager.Options{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(results))
	for _, r := range results {
		names = append(names, r.Name)
	}
	return names, nil
}

// SearchFormatted returns formatted search results using nala if available
func (a *APT) SearchFormatted(query string) (string, error) {
	if !a.nalaAvailable() {
		return "", fmt.Errorf("nala not available")
	}
	out, err := a.runNala("search", query)
	if err != nil {
		return "", fmt.Errorf("nala search failed: %w\nOutput: %s", err, out)
	}
	return string(out), nil
}

// ListInstalledFormatted returns formatted list of installed packages using nala if available
func (a *APT) ListInstalledFormatted() (string, error) {
	if !a.nalaAvailable() {
		return "", fmt.Errorf("nala not available")
	}
	out, err := a.runNala("list", "--installed")
	if err != nil {
		return "", fmt.Errorf("nala list failed: %w\nOutput: %s", err, out)
	}
	return string(out), nil
}

// Purge removes packages and their configuration files via apt-get purge.
// syspkg's Delete maps to `apt-get remove`; purge requires a direct call.
func (a *APT) Purge(packages ...string) error {
	if len(packages) == 0 {
		return nil
	}
	if a.dryRun {
		fmt.Printf("[dry-run] Would purge: %s\n", strings.Join(packages, ", "))
		return nil
	}
	frontend := "apt-get"
	if a.useNala {
		frontend = "nala"
	}
	args := append([]string{"purge", "-y"}, packages...)
	return exec.Command(frontend, args...).Run()
}

// GetDependencies returns package dependencies via apt-cache.
// Not covered by syspkg interface; kept as a thin subprocess call.
func (a *APT) GetDependencies(pkg string) ([]string, error) {
	cmd := exec.Command("apt-cache", "depends", "--installed", pkg)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var deps []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Depends:") {
			dep := strings.TrimSpace(strings.TrimPrefix(line, "Depends:"))
			deps = append(deps, dep)
		}
	}
	return deps, nil
}

// PackageInfo contains package search result info
type PackageInfo struct {
	Name        string
	Description string
	Version     string
}

// HistoryEntry represents a nala transaction history entry
type HistoryEntry struct {
	ID       string
	Action   string
	Packages []string
	Date     string
	User     string
}

// History lists recent nala transactions
func (a *APT) History(limit int) ([]HistoryEntry, error) {
	if !a.nalaAvailable() {
		return nil, fmt.Errorf("nala not available")
	}
	args := []string{"history", "list", "--no-pager"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("--limit=%d", limit))
	}
	out, err := a.runNala(args...)
	if err != nil {
		return nil, fmt.Errorf("nala history list failed: %w\nOutput: %s", err, out)
	}
	// Parse output lines
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return []HistoryEntry{}, nil
	}
	// Skip header line
	entries := make([]HistoryEntry, 0, len(lines)-1)
	// Use regex to split by two or more spaces
	re := regexp.MustCompile(`\s{2,}`)
	for _, line := range lines[1:] {
		fields := re.Split(strings.TrimSpace(line), -1)
		if len(fields) < 5 {
			continue
		}
		pkgs := strings.Split(fields[2], ",")
		for i := range pkgs {
			pkgs[i] = strings.TrimSpace(pkgs[i])
		}
		entries = append(entries, HistoryEntry{
			ID:       fields[0],
			Action:   fields[1],
			Packages: pkgs,
			Date:     fields[3] + " " + fields[4],
			User:     fields[5],
		})
	}
	return entries, nil
}

// Rollback rolls back a nala transaction by ID
func (a *APT) Rollback(id string) error {
	if !a.nalaAvailable() {
		return fmt.Errorf("nala not available")
	}
	if a.dryRun {
		fmt.Printf("[dry-run] Would rollback transaction %s\n", id)
		return nil
	}
	out, err := a.runNala("history", "rollback", id)
	if err != nil {
		return fmt.Errorf("nala rollback failed: %w\nOutput: %s", err, out)
	}
	return nil
}

// Fetch runs nala fetch to select fastest mirrors
func (a *APT) Fetch() error {
	if !a.nalaAvailable() {
		return fmt.Errorf("nala not available")
	}
	if a.dryRun {
		fmt.Println("[dry-run] Would fetch best mirrors")
		return nil
	}
	out, err := a.runNala("fetch")
	if err != nil {
		return fmt.Errorf("nala fetch failed: %w\nOutput: %s", err, out)
	}
	return nil
}

// Upgrade runs nala upgrade (update + upgrade in one command)
func (a *APT) Upgrade() error {
	if !a.nalaAvailable() {
		return fmt.Errorf("nala not available")
	}
	if a.dryRun {
		fmt.Println("[dry-run] Would upgrade system")
		return nil
	}
	out, err := a.runNala("upgrade")
	if err != nil {
		return fmt.Errorf("nala upgrade failed: %w\nOutput: %s", err, out)
	}
	return nil
}

// SetConfig sets a nala configuration option
func (a *APT) SetConfig(key, value string) error {
	if !a.nalaAvailable() {
		return fmt.Errorf("nala not available")
	}
	if a.dryRun {
		fmt.Printf("[dry-run] Would set nala config %s=%s\n", key, value)
		return nil
	}
	// nala config set key value
	out, err := a.runNala("config", "set", key, value)
	if err != nil {
		return fmt.Errorf("nala config set failed: %w\nOutput: %s", err, out)
	}
	return nil
}
