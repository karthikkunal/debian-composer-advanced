package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	sprig "github.com/Masterminds/sprig/v3"
)

// Config manages dotfile configuration
type Config struct {
	sourceDir string
	destDir   string
	backup    bool
	dryRun    bool
	verbose   bool
	templates map[string]*template.Template
}

// Dotfile represents a single dotfile entry
type Dotfile struct {
	Source     string // Path relative to source directory
	Target     string // Target path (absolute or relative to home)
	Mode       os.FileMode
	Template   bool // Whether to process as template
	Executable bool // Whether to make executable
}

// NewConfig creates a new config manager
func NewConfig(sourceDir, destDir string, dryRun, verbose bool) *Config {
	return &Config{
		sourceDir: sourceDir,
		destDir:   destDir,
		backup:    true,
		dryRun:    dryRun,
		verbose:   verbose,
		templates: make(map[string]*template.Template),
	}
}

// SetBackup enables or disables backup of existing files
func (c *Config) SetBackup(backup bool) {
	c.backup = backup
}

// AddTemplate adds a template with a name for use in dotfiles
func (c *Config) AddTemplate(name, content string) error {
	tmpl, err := template.New(name).Funcs(sprig.TxtFuncMap()).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}
	c.templates[name] = tmpl
	return nil
}

// Apply applies a dotfile to its target location
func (c *Config) Apply(dotfile Dotfile, data interface{}) error {
	sourcePath := filepath.Join(c.sourceDir, dotfile.Source)
	targetPath := c.resolveTargetPath(dotfile.Target)

	// Check source exists
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("source file not found: %s", sourcePath)
	}

	if c.dryRun {
		fmt.Printf("[DRY-RUN] Would apply %s -> %s\n", sourcePath, targetPath)
		return nil
	}

	// Backup existing file if requested
	if c.backup {
		if err := c.backupFile(targetPath); err != nil {
			return fmt.Errorf("failed to backup %s: %w", targetPath, err)
		}
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	// Apply the file
	if dotfile.Template {
		if err := c.applyTemplate(sourcePath, targetPath, data); err != nil {
			return err
		}
	} else {
		if err := c.copyFile(sourcePath, targetPath); err != nil {
			return err
		}
	}

	// Set permissions
	mode := dotfile.Mode
	if mode == 0 {
		if dotfile.Executable {
			mode = 0755
		} else {
			mode = 0644
		}
	}
	if err := os.Chmod(targetPath, mode); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", targetPath, err)
	}

	if c.verbose {
		fmt.Printf("Applied %s -> %s\n", sourcePath, targetPath)
	}

	return nil
}

// ApplyAll applies all dotfiles from a directory
func (c *Config) ApplyAll(data interface{}) error {
	dotfiles, err := c.ScanSourceDir()
	if err != nil {
		return err
	}

	for _, dotfile := range dotfiles {
		if err := c.Apply(dotfile, data); err != nil {
			return err
		}
	}

	return nil
}

// ScanSourceDir scans the source directory for dotfiles
func (c *Config) ScanSourceDir() ([]Dotfile, error) {
	var dotfiles []Dotfile

	err := filepath.Walk(c.sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path from source dir
		relPath, err := filepath.Rel(c.sourceDir, path)
		if err != nil {
			return err
		}

		// Convert to dotfile target name
		targetName := convertToDotfile(relPath)
		dotfile := Dotfile{
			Source:   relPath,
			Target:   targetName,
			Template: strings.HasSuffix(relPath, ".tmpl"),
		}
		dotfiles = append(dotfiles, dotfile)

		return nil
	})

	return dotfiles, err
}

// List lists all dotfiles and their status
func (c *Config) List() ([]DotfileStatus, error) {
	dotfiles, err := c.ScanSourceDir()
	if err != nil {
		return nil, err
	}

	var statuses []DotfileStatus
	for _, dotfile := range dotfiles {
		status := DotfileStatus{
			Dotfile: dotfile,
		}

		sourcePath := filepath.Join(c.sourceDir, dotfile.Source)
		targetPath := c.resolveTargetPath(dotfile.Target)

		// Check if source exists
		if info, err := os.Stat(sourcePath); err == nil {
			status.SourceExists = true
			status.SourceSize = info.Size()
			status.SourceModTime = info.ModTime()
		}

		// Check if target exists
		if info, err := os.Stat(targetPath); err == nil {
			status.TargetExists = true
			status.TargetSize = info.Size()
			status.TargetModTime = info.ModTime()
			status.NeedsUpdate = status.SourceModTime.After(status.TargetModTime)
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// Diff shows differences between source and target
func (c *Config) Diff(dotfile Dotfile) (string, error) {
	sourcePath := filepath.Join(c.sourceDir, dotfile.Source)
	targetPath := c.resolveTargetPath(dotfile.Target)

	sourceContent, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read source: %w", err)
	}

	targetContent, err := os.ReadFile(targetPath)
	if err != nil {
		return "", fmt.Errorf("target not found: %w", err)
	}

	// Simple diff - just show if files are different
	if string(sourceContent) == string(targetContent) {
		return "Files are identical", nil
	}

	return fmt.Sprintf("Files differ:\nSource (%d bytes):\n%s\n\nTarget (%d bytes):\n%s",
		len(sourceContent), sourceContent,
		len(targetContent), targetContent), nil
}

// Backup backs up a file
func (c *Config) Backup(path string) error {
	return c.backupFile(path)
}

func (c *Config) resolveTargetPath(target string) string {
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Join(c.destDir, target)
}

func (c *Config) backupFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		// File doesn't exist, nothing to backup
		return nil
	}

	backupPath := path + ".backup"
	if c.verbose {
		fmt.Printf("Backing up %s -> %s\n", path, backupPath)
	}

	return c.copyFile(path, backupPath)
}

func (c *Config) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

func (c *Config) applyTemplate(src, dst string, data interface{}) error {
	tmpl, err := template.New(filepath.Base(src)).Funcs(sprig.TxtFuncMap()).ParseFiles(src)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", src, err)
	}

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	if err := tmpl.Execute(dest, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", src, err)
	}

	return nil
}

// convertToDotfile converts a path like "home/user/.config/foo" to ".config/foo"
func convertToDotfile(path string) string {
	parts := strings.Split(path, string(filepath.Separator))

	// Skip "home" or "user" prefixes
	startIdx := 0
	for i, part := range parts {
		if strings.HasPrefix(part, ".") || part == "config" {
			startIdx = i
			break
		}
		if part == "home" || part == "user" {
			startIdx = i + 1
			continue
		}
		startIdx = i
		break
	}

	result := strings.Join(parts[startIdx:], string(filepath.Separator))
	if !strings.HasPrefix(result, ".") {
		result = "." + result
	}
	return result
}

// DotfileStatus represents the status of a dotfile
type DotfileStatus struct {
	Dotfile       Dotfile
	SourceExists  bool
	TargetExists  bool
	SourceSize    int64
	TargetSize    int64
	SourceModTime time.Time
	TargetModTime time.Time
	NeedsUpdate   bool
}

// FormatDotfileStatus formats dotfile status for display
func FormatDotfileStatus(statuses []DotfileStatus) string {
	var sb strings.Builder

	sb.WriteString("Dotfiles Status:\n")
	sb.WriteString("─────────────────────────────────────────\n")
	fmt.Fprintf(&sb, "%-30s %-10s %-10s %s\n", "Source", "Source", "Target", "Status")
	fmt.Fprintf(&sb, "%-30s %-10s %-10s %s\n", "──────", "──────", "──────", "──────")

	for _, status := range statuses {
		sourceStatus := "missing"
		if status.SourceExists {
			sourceStatus = "ok"
		}

		targetStatus := "missing"
		if status.TargetExists {
			targetStatus = "ok"
		}

		updateStatus := ""
		if status.NeedsUpdate {
			updateStatus = "needs update"
		}

		fmt.Fprintf(&sb, "%-30s %-10s %-10s %s\n",
			status.Dotfile.Source, sourceStatus, targetStatus, updateStatus)
	}

	return sb.String()
}
