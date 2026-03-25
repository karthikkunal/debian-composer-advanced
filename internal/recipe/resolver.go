package recipe

import (
	"fmt"
	"strings"

	"github.com/debian-composer/debian-composer-go/internal/types"
)

// Resolver handles recipe composition (includes, extends, layers, variables)
type Resolver struct {
	parser *Parser
}

// NewResolver creates a new recipe resolver
func NewResolver(parser *Parser) *Resolver {
	return &Resolver{parser: parser}
}

// Resolve fully resolves a recipe with all includes, extends, and variables
func (r *Resolver) Resolve(name string, vars map[string]string) (*types.ResolvedRecipe, error) {
	recipe, err := r.parser.Parse(name)
	if err != nil {
		return nil, err
	}

	// Work on a deep copy of the main recipe to avoid modifying cache
	base := *recipe
	base.Variables = make(map[string]types.Variable)
	for k, v := range recipe.Variables {
		base.Variables[k] = v
	}

	// 1. Resolve Extends (Base inheritance)
	if recipe.Extends != "" {
		parent, err := r.parser.Parse(recipe.Extends)
		if err != nil {
			return nil, fmt.Errorf("extends %s: %w", recipe.Extends, err)
		}
		// Merge parent into base (base takes precedence for simple fields)
		if err := r.parser.merger.MergeRecipes(parent, &base); err != nil {
			return nil, fmt.Errorf("merge extends: %w", err)
		}
		base = *parent
	}

	// 2. Resolve Includes (Components/Fragments)
	for _, inc := range recipe.Includes {
		included, err := r.parser.Parse(inc)
		if err != nil {
			return nil, fmt.Errorf("include %s: %w", inc, err)
		}
		if err := r.parser.merger.MergeRecipes(&base, included); err != nil {
			return nil, fmt.Errorf("merge include %s: %w", inc, err)
		}
	}

	// 3. Resolve Layers (Overlays)
	for _, layer := range recipe.Layers {
		layered, err := r.parser.Parse(layer)
		if err != nil {
			return nil, fmt.Errorf("layer %s: %w", layer, err)
		}
		// Layer takes precedence over base
		if err := r.parser.merger.MergeRecipes(&base, layered); err != nil {
			return nil, fmt.Errorf("merge layer %s: %w", layer, err)
		}
	}

	resolved := &types.ResolvedRecipe{
		Recipe:       base,
		ResolvedVars: make(map[string]string),
	}

	// Apply variables
	for k, v := range base.Variables {
		if val, ok := vars[k]; ok {
			resolved.ResolvedVars[k] = val
		} else if v.Default != "" {
			resolved.ResolvedVars[k] = v.Default
		}
	}

	// Force variables passed via CLI to be present even if not in recipe
	for k, v := range vars {
		resolved.ResolvedVars[k] = v
	}

	// Collect all packages from resolved categories and main list
	pkgSet := make(map[string]bool)
	for _, pkg := range base.Packages {
		pkgSet[pkg] = true
	}

	// Resolve categories based on variables
	for catName, cat := range base.Categories {
		if cat.Condition == "" || r.evaluateCondition(cat.Condition, resolved.ResolvedVars) {
			resolved.ActiveCats = append(resolved.ActiveCats, catName)
			for _, pkg := range cat.Packages {
				pkgSet[pkg] = true
			}
		}
	}

	// Convert to slice
	for pkg := range pkgSet {
		resolved.AllPackages = append(resolved.AllPackages, pkg)
	}

	return resolved, nil
}

// ResolveStack resolves a recipe with a specific stack
func (r *Resolver) ResolveStack(name, stackName string, vars map[string]string) (*types.ResolvedRecipe, error) {
	resolved, err := r.Resolve(name, vars)
	if err != nil {
		return nil, err
	}

	stack, ok := resolved.Stacks[stackName]
	if !ok {
		return nil, fmt.Errorf("stack %s not found in recipe %s", stackName, name)
	}

	// Add stack categories
	pkgSet := make(map[string]bool)
	for _, pkg := range resolved.AllPackages {
		pkgSet[pkg] = true
	}

	for _, catName := range stack.Categories {
		if cat, ok := resolved.Categories[catName]; ok {
			resolved.ActiveCats = append(resolved.ActiveCats, catName)
			for _, pkg := range cat.Packages {
				pkgSet[pkg] = true
			}
		}
	}

	// Rebuild package list
	resolved.AllPackages = nil
	for pkg := range pkgSet {
		resolved.AllPackages = append(resolved.AllPackages, pkg)
	}

	return resolved, nil
}

// evaluateCondition evaluates a simple condition expression
func (r *Resolver) evaluateCondition(cond string, vars map[string]string) bool {
	// Handle simple conditions like "primary_lang == python"
	parts := strings.SplitN(cond, "==", 2)
	if len(parts) == 2 {
		varName := strings.TrimSpace(parts[0])
		expected := strings.TrimSpace(parts[1])
		actual := vars[varName]
		return actual == expected
	}

	// Handle != conditions
	parts = strings.SplitN(cond, "!=", 2)
	if len(parts) == 2 {
		varName := strings.TrimSpace(parts[0])
		expected := strings.TrimSpace(parts[1])
		actual := vars[varName]
		return actual != expected
	}

	// Default: treat as truthy check
	return vars[cond] != "" && vars[cond] != "false"
}
