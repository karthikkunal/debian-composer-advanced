package recipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagerImport(t *testing.T) {
	// Setup mock kitchen
	tmpDir, err := os.MkdirTemp("", "kitchen-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	recipesPath := filepath.Join(tmpDir, "recipes")
	if err := os.MkdirAll(recipesPath, 0755); err != nil {
		t.Fatal(err)
	}

	p := NewParser(tmpDir)
	mgr := NewManager(p)

	// Create a sample recipe file outside
	samplePath := filepath.Join(tmpDir, "sample.yaml")
	sampleContent := `
name: test-recipe
kind: recipe
version: 1.0.0
description: A test recipe
packages:
  - curl
`
	if err := os.WriteFile(samplePath, []byte(sampleContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test Import
	if err := mgr.Import(samplePath, "imported-recipe"); err != nil {
		t.Errorf("Import failed: %v", err)
	}

	// Verify file exists in kitchen
	importedPath := filepath.Join(recipesPath, "imported-recipe.yaml")
	if _, err := os.Stat(importedPath); os.IsNotExist(err) {
		t.Errorf("Imported file does not exist at %s", importedPath)
	}
}

func TestManagerExport(t *testing.T) {
	// Setup mock kitchen
	tmpDir, err := os.MkdirTemp("", "kitchen-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	recipesPath := filepath.Join(tmpDir, "recipes")
	if err := os.MkdirAll(recipesPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a recipe in the kitchen
	recipeContent := `
name: test-recipe
kind: recipe
version: 1.0.0
packages:
  - wget
`
	recipePath := filepath.Join(recipesPath, "test-recipe.yaml")
	if err := os.WriteFile(recipePath, []byte(recipeContent), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewParser(tmpDir)
	mgr := NewManager(p)

	// Test Export
	exportPath := filepath.Join(tmpDir, "exported.yaml")
	if err := mgr.Export("test-recipe", exportPath, false); err != nil {
		t.Errorf("Export failed: %v", err)
	}

	// Verify exported file
	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "wget") {
		t.Errorf("Exported content mismatch: %s", string(data))
	}
}
