package recipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveWithLayers(t *testing.T) {
	// Setup mock kitchen
	tmpDir, err := os.MkdirTemp("", "layers-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	recipesPath := filepath.Join(tmpDir, "recipes")
	if err := os.MkdirAll(recipesPath, 0755); err != nil {
		t.Fatal(err)
	}

	// 1. Create Base Recipe
	baseContent := `
name: base-recipe
kind: recipe
version: 1.0.0
packages:
  - curl
variables:
  theme:
    default: light
`
	os.WriteFile(filepath.Join(recipesPath, "base.yaml"), []byte(baseContent), 0644)

	// 2. Create Layer 1 (adds packages, overrides variable)
	layer1Content := `
name: layer-1
kind: recipe
packages:
  - wget
variables:
  theme:
    default: dark
`
	os.WriteFile(filepath.Join(recipesPath, "layer1.yaml"), []byte(layer1Content), 0644)

	// 3. Create Main Recipe with Layer
	mainContent := `
name: main-recipe
kind: recipe
layers:
  - base
  - layer1
packages:
  - git
`
	os.WriteFile(filepath.Join(recipesPath, "main.yaml"), []byte(mainContent), 0644)

	p := NewParser(tmpDir)
	r := NewResolver(p)

	resolved, err := r.Resolve("main", nil)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Verify packages (should be curl + wget + git)
	pkgs := strings.Join(resolved.AllPackages, " ")
	for _, expected := range []string{"curl", "wget", "git"} {
		if !strings.Contains(pkgs, expected) {
			t.Errorf("Missing expected package %s in %v", expected, resolved.AllPackages)
		}
	}

	// Verify variable override (should be dark from layer1)
	if resolved.ResolvedVars["theme"] != "dark" {
		t.Errorf("Variable override failed: expected dark, got %s", resolved.ResolvedVars["theme"])
	}
}

func TestResolveWithExtendsAndLayers(t *testing.T) {
	// Setup mock kitchen
	tmpDir, err := os.MkdirTemp("", "extends-layers-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	recipesPath := filepath.Join(tmpDir, "recipes")
	os.MkdirAll(recipesPath, 0755)

	// Parent
	os.WriteFile(filepath.Join(recipesPath, "parent.yaml"), []byte(`
name: parent
kind: recipe
packages: [parent-pkg]
`), 0644)

	// Layer
	os.WriteFile(filepath.Join(recipesPath, "overlay.yaml"), []byte(`
name: overlay
kind: recipe
packages: [overlay-pkg]
`), 0644)

	// Child
	os.WriteFile(filepath.Join(recipesPath, "child.yaml"), []byte(`
name: child
kind: recipe
extends: parent
layers: [overlay]
packages: [child-pkg]
`), 0644)

	p := NewParser(tmpDir)
	r := NewResolver(p)

	resolved, err := r.Resolve("child", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Should have all packages
	pkgs := strings.Join(resolved.AllPackages, " ")
	expected := []string{"parent-pkg", "overlay-pkg", "child-pkg"}
	for _, e := range expected {
		if !strings.Contains(pkgs, e) {
			t.Errorf("Missing %s", e)
		}
	}
}
