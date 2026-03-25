package deb

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xor-gate/debpkg"
)

// PackageInfo contains metadata for a .deb package
type PackageInfo struct {
	Name            string
	Version         string
	Architecture    string
	Maintainer      string
	MaintainerEmail string
	Homepage        string
	ShortDesc       string
	LongDesc        string
	Section         string
	Priority        string
	Depends         []string
	Recommends      []string
	Suggests        []string
	Provides        []string
	Replaces        []string
	Conflicts       []string
}

// Builder creates .deb packages
type Builder struct {
	info       PackageInfo
	stagingDir string
	outputDir  string
	dryRun     bool
	verbose    bool
	files      []FileEntry
}

// FileEntry represents a file to include in the package
type FileEntry struct {
	Source      string // Source path on disk
	Destination string // Destination path in package
	Mode        os.FileMode
	Owner       string
	Group       string
}

// NewBuilder creates a new .deb package builder
func NewBuilder(info PackageInfo, outputDir string, dryRun, verbose bool) *Builder {
	return &Builder{
		info:       info,
		outputDir:  outputDir,
		stagingDir: "",
		dryRun:     dryRun,
		verbose:    verbose,
		files:      []FileEntry{},
	}
}

// AddFile adds a file to the package
func (b *Builder) AddFile(source, destination string, mode os.FileMode) {
	b.files = append(b.files, FileEntry{
		Source:      source,
		Destination: destination,
		Mode:        mode,
		Owner:       "root",
		Group:       "root",
	})
}

// AddFileWithOwner adds a file with specific owner/group
func (b *Builder) AddFileWithOwner(source, destination, owner, group string, mode os.FileMode) {
	b.files = append(b.files, FileEntry{
		Source:      source,
		Destination: destination,
		Mode:        mode,
		Owner:       owner,
		Group:       group,
	})
}

// AddDirectory adds all files from a directory to the package
func (b *Builder) AddDirectory(sourceDir, destPrefix string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destPrefix, relPath)
		b.AddFile(path, destPath, info.Mode())

		return nil
	})
}

// Build creates the .deb package
func (b *Builder) Build() (string, error) {
	if len(b.files) == 0 {
		return "", fmt.Errorf("no files added to package")
	}

	// Create staging directory
	stagingDir, err := os.MkdirTemp("", "debpkg-*")
	if err != nil {
		return "", fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)
	b.stagingDir = stagingDir

	// Create package structure
	if err := b.createPackageStructure(); err != nil {
		return "", fmt.Errorf("failed to create package structure: %w", err)
	}

	// Copy files to staging
	if err := b.copyFiles(); err != nil {
		return "", fmt.Errorf("failed to copy files: %w", err)
	}

	// Create control file
	if err := b.createControlFile(); err != nil {
		return "", fmt.Errorf("failed to create control file: %w", err)
	}

	// Create md5sums
	if err := b.createMD5Sums(); err != nil {
		return "", fmt.Errorf("failed to create md5sums: %w", err)
	}

	// Build the .deb
	outputPath, err := b.buildDeb()
	if err != nil {
		return "", fmt.Errorf("failed to build .deb: %w", err)
	}

	return outputPath, nil
}

func (b *Builder) createPackageStructure() error {
	dirs := []string{
		filepath.Join(b.stagingDir, "DEBIAN"),
		filepath.Join(b.stagingDir, "usr", "bin"),
		filepath.Join(b.stagingDir, "usr", "lib"),
		filepath.Join(b.stagingDir, "usr", "share"),
		filepath.Join(b.stagingDir, "etc"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (b *Builder) copyFiles() error {
	for _, file := range b.files {
		destPath := filepath.Join(b.stagingDir, file.Destination)
		destDir := filepath.Dir(destPath)

		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}

		if b.verbose {
			fmt.Printf("  Adding: %s -> %s\n", file.Source, file.Destination)
		}

		if b.dryRun {
			continue
		}

		// Read source file
		content, err := os.ReadFile(file.Source)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file.Source, err)
		}

		// Write to destination
		if err := os.WriteFile(destPath, content, file.Mode); err != nil {
			return fmt.Errorf("failed to write %s: %w", destPath, err)
		}
	}

	return nil
}

func (b *Builder) createControlFile() error {
	controlPath := filepath.Join(b.stagingDir, "DEBIAN", "control")

	control := fmt.Sprintf(`Package: %s
Version: %s
Architecture: %s
Maintainer: %s <%s>
Section: %s
Priority: %s
Homepage: %s
Description: %s
 %s
`,
		b.info.Name,
		b.info.Version,
		b.info.Architecture,
		b.info.Maintainer,
		b.info.MaintainerEmail,
		b.info.Section,
		b.info.Priority,
		b.info.Homepage,
		b.info.ShortDesc,
		b.info.LongDesc,
	)

	if len(b.info.Depends) > 0 {
		control += fmt.Sprintf("Depends: %s\n", joinWithCommas(b.info.Depends))
	}
	if len(b.info.Recommends) > 0 {
		control += fmt.Sprintf("Recommends: %s\n", joinWithCommas(b.info.Recommends))
	}
	if len(b.info.Suggests) > 0 {
		control += fmt.Sprintf("Suggests: %s\n", joinWithCommas(b.info.Suggests))
	}
	if len(b.info.Provides) > 0 {
		control += fmt.Sprintf("Provides: %s\n", joinWithCommas(b.info.Provides))
	}
	if len(b.info.Replaces) > 0 {
		control += fmt.Sprintf("Replaces: %s\n", joinWithCommas(b.info.Replaces))
	}
	if len(b.info.Conflicts) > 0 {
		control += fmt.Sprintf("Conflicts: %s\n", joinWithCommas(b.info.Conflicts))
	}

	if b.dryRun {
		fmt.Printf("[DRY-RUN] Would create control file:\n%s\n", control)
		return nil
	}

	return os.WriteFile(controlPath, []byte(control), 0644)
}

func (b *Builder) createMD5Sums() error {
	// md5sums file is optional but recommended
	// For simplicity, we skip it in dry-run mode
	if b.dryRun {
		return nil
	}

	// Implementation would calculate md5sums for all files
	// This is a simplified version
	return nil
}

func (b *Builder) buildDeb() (string, error) {
	if b.outputDir == "" {
		b.outputDir = "."
	}

	// Ensure output directory exists
	if err := os.MkdirAll(b.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	debName := fmt.Sprintf("%s_%s_%s.deb", b.info.Name, b.info.Version, b.info.Architecture)
	outputPath := filepath.Join(b.outputDir, debName)

	if b.dryRun {
		fmt.Printf("[DRY-RUN] Would create %s\n", outputPath)
		return outputPath, nil
	}

	// Create the package using debpkg library
	pkg := debpkg.New()
	defer pkg.Close()

	pkg.SetName(b.info.Name)
	pkg.SetVersion(b.info.Version)
	pkg.SetArchitecture(b.info.Architecture)
	pkg.SetMaintainer(b.info.Maintainer)
	pkg.SetMaintainerEmail(b.info.MaintainerEmail)
	pkg.SetHomepage(b.info.Homepage)
	pkg.SetShortDescription(b.info.ShortDesc)
	pkg.SetDescription(b.info.LongDesc)

	// Set optional fields
	if b.info.Section != "" {
		pkg.SetSection(b.info.Section)
	}
	if b.info.Priority != "" {
		pkg.SetPriority(debpkg.Priority(b.info.Priority))
	}

	// Set dependencies as comma-separated strings
	if len(b.info.Depends) > 0 {
		pkg.SetDepends(joinWithCommas(b.info.Depends))
	}
	if len(b.info.Recommends) > 0 {
		pkg.SetRecommends(joinWithCommas(b.info.Recommends))
	}
	if len(b.info.Suggests) > 0 {
		pkg.SetSuggests(joinWithCommas(b.info.Suggests))
	}
	if len(b.info.Provides) > 0 {
		pkg.SetProvides(joinWithCommas(b.info.Provides))
	}
	if len(b.info.Replaces) > 0 {
		pkg.SetReplaces(joinWithCommas(b.info.Replaces))
	}
	if len(b.info.Conflicts) > 0 {
		pkg.SetConflicts(joinWithCommas(b.info.Conflicts))
	}

	// Add all files
	for _, file := range b.files {
		if err := pkg.AddFile(file.Source, file.Destination); err != nil {
			return "", fmt.Errorf("failed to add file %s: %w", file.Source, err)
		}
	}

	// Write the package
	if err := pkg.Write(outputPath); err != nil {
		return "", fmt.Errorf("failed to write package: %w", err)
	}

	if b.verbose {
		fmt.Printf("Created: %s\n", outputPath)
	}

	return outputPath, nil
}

func joinWithCommas(items []string) string {
	if len(items) == 0 {
		return ""
	}
	result := items[0]
	for _, item := range items[1:] {
		result += ", " + item
	}
	return result
}

// GetPackageFilename returns the expected filename for the package
func (b *Builder) GetPackageFilename() string {
	return fmt.Sprintf("%s_%s_%s.deb", b.info.Name, b.info.Version, b.info.Architecture)
}

// PackageFromYAML creates a builder from a YAML spec file (debpkg.yml format)
func PackageFromYAML(yamlPath string, outputDir string, dryRun, verbose bool) (*Builder, error) {
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	// Parse basic info from YAML
	// This is a simplified parser - a full implementation would use YAML unmarshaling
	info := PackageInfo{
		Architecture: "all",
		Section:      "utils",
		Priority:     "optional",
	}

	// Simple YAML parsing (in production, use gopkg.in/yaml.v3)
	// For now, return error prompting user to use the API directly
	_ = content
	_ = info

	return NewBuilder(info, outputDir, dryRun, verbose), fmt.Errorf("YAML parsing not yet implemented - use the Go API directly")
}

// ValidatePackage validates a .deb package file
func ValidatePackage(debPath string) error {
	if _, err := os.Stat(debPath); err != nil {
		return fmt.Errorf("package not found: %w", err)
	}

	// Basic validation - check file extension
	if filepath.Ext(debPath) != ".deb" {
		return fmt.Errorf("file does not have .deb extension")
	}

	return nil
}

// GetDefaultPackageInfo returns default package info for debian-composer
func GetDefaultPackageInfo() PackageInfo {
	return PackageInfo{
		Name:            "debian-composer",
		Version:         "1.0.0",
		Architecture:    "all",
		Maintainer:      "Debian Composer Team",
		MaintainerEmail: "debian-composer@example.com",
		Homepage:        "https://github.com/debian-composer/debian-composer-go",
		ShortDesc:       "Post-install configuration and recipe manager for Debian",
		LongDesc:        "Debian Composer helps you compose your perfect Debian setup with sensible defaults and modular recipes. It supports hardware detection, recipe management, and system configuration.",
		Section:         "utils",
		Priority:        "optional",
		Depends:         []string{"bash", "sudo"},
	}
}

// FormatBuildSummary formats a build summary for display
func FormatBuildSummary(info PackageInfo, outputPath string, fileCount int) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	return fmt.Sprintf(`Package Build Summary
════════════════════
Name:         %s
Version:      %s
Architecture: %s
Maintainer:   %s <%s>
Section:      %s
Priority:     %s
Files:        %d
Output:       %s
Built:        %s
`,
		info.Name,
		info.Version,
		info.Architecture,
		info.Maintainer,
		info.MaintainerEmail,
		info.Section,
		info.Priority,
		fileCount,
		outputPath,
		timestamp,
	)
}
