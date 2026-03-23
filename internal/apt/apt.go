package apt

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/bluet/syspkg"
	"github.com/bluet/syspkg/manager"
)

// APT handles Debian package operations via syspkg
type APT struct {
	dryRun bool
	pm     syspkg.PackageManager
}

// New creates a new APT handler backed by syspkg
func New(dryRun bool) *APT {
	sp, err := syspkg.New(syspkg.IncludeOptions{Apt: true})
	if err != nil {
		// Fall back to nil pm; all operations will no-op or error gracefully
		return &APT{dryRun: dryRun}
	}
	pm, err := sp.GetPackageManager("apt")
	if err != nil {
		return &APT{dryRun: dryRun}
	}
	return &APT{dryRun: dryRun, pm: pm}
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

// Install installs packages
func (a *APT) Install(packages ...string) error {
	if len(packages) == 0 {
		return nil
	}
	if !a.available() {
		return fmt.Errorf("apt package manager not available")
	}
	_, err := a.pm.Install(packages, a.opts())
	return err
}

// BatchInstall installs multiple packages in a single call
func (a *APT) BatchInstall(packages []string) error {
	return a.Install(packages...)
}

// Remove removes packages
func (a *APT) Remove(packages ...string) error {
	if len(packages) == 0 {
		return nil
	}
	if !a.available() {
		return fmt.Errorf("apt package manager not available")
	}
	_, err := a.pm.Delete(packages, a.opts())
	return err
}

// Update runs apt update (refresh package index)
func (a *APT) Update() error {
	if !a.available() {
		return fmt.Errorf("apt package manager not available")
	}
	return a.pm.Refresh(a.opts())
}

// IsInstalled checks if a package is installed
func (a *APT) IsInstalled(pkg string) bool {
	if !a.available() {
		return false
	}
	results, err := a.pm.Find([]string{pkg}, &manager.Options{})
	if err != nil {
		return false
	}
	for _, r := range results {
		if r.Name == pkg && r.Status == "installed" {
			return true
		}
	}
	return false
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
	args := append([]string{"purge", "-y"}, packages...)
	return exec.Command("apt-get", args...).Run()
}

// PurgeKeepConfig removes packages but keeps their configuration files
func (a *APT) PurgeKeepConfig(packages ...string) error {
	return a.Remove(packages...)
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
