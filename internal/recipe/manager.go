package recipe

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manager handles recipe file operations (import, export, copy)
type Manager struct {
	parser *Parser
	merger *Merger
}

// NewManager creates a new recipe manager
func NewManager(parser *Parser) *Manager {
	return &Manager{
		parser: parser,
		merger: parser.merger,
	}
}

// Import copies a recipe from an external path into the kitchen
func (m *Manager) Import(srcPath string, name string) error {
	// Parse to validate it's a valid recipe
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}

	// Validate against JSON schema
	if err := ValidateRecipe(data); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}

	// Try to unmarshal to validate structure (additional check)
	var recipe map[string]interface{}
	if err := yaml.Unmarshal(data, &recipe); err != nil {
		return fmt.Errorf("invalid recipe YAML: %w", err)
	}

	// Use provided name or filename from path
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(srcPath), ".yaml")
	}

	// Target path in kitchen
	targetPath := filepath.Join(m.parser.recipesPath, name+".yaml")

	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Check if already exists
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("recipe '%s' already exists in kitchen", name)
	}

	// Copy the file
	return os.WriteFile(targetPath, data, 0644)
}

// Export writes a recipe to a file
func (m *Manager) Export(name string, targetPath string, standalone bool) error {
	var data []byte
	var err error

	// Find the actual file path for the recipe name
	path, err := m.parser.resolvePath(name)
	if err != nil {
		return fmt.Errorf("recipe not found: %s", name)
	}

	if standalone {
		// Use merger to resolve all includes and anchors
		data, err = m.merger.MergeRecipe(path)
		if err != nil {
			return fmt.Errorf("merge recipe: %w", err)
		}
	} else {
		// Just read the raw recipe file
		data, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read recipe: %w", err)
		}
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return fmt.Errorf("write export: %w", err)
	}

	return nil
}

// CopyRecipe duplicates a recipe in the kitchen
func (m *Manager) CopyRecipe(srcName, dstName string) error {
	srcPath, err := m.parser.resolvePath(srcName)
	if err != nil {
		return fmt.Errorf("source recipe not found: %s", srcName)
	}

	dstPath := filepath.Join(m.parser.recipesPath, dstName+".yaml")
	if _, err := os.Stat(dstPath); err == nil {
		return fmt.Errorf("destination recipe already exists: %s", dstName)
	}

	// Copy the file
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
