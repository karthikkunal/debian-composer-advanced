package recipe

import (
	"fmt"

	"github.com/debian-composer/debian-composer-go/internal/types"
)

// DependencyResolver handles recipe dependency chains and conflict detection
type DependencyResolver struct {
	parser *Parser
	state  StateGetter
}

// StateGetter provides method to check installed recipes
type StateGetter interface {
	IsRecipeInstalled(name string) bool
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(parser *Parser, state StateGetter) *DependencyResolver {
	return &DependencyResolver{
		parser: parser,
		state:  state,
	}
}

// ResolveAll resolves a recipe and all its dependencies (transitive)
// Returns a slice of resolved recipes in installation order (dependencies first)
func (d *DependencyResolver) ResolveAll(name string, vars map[string]string) ([]*types.ResolvedRecipe, error) {
	visited := make(map[string]bool)
	inStack := make(map[string]bool) // for cycle detection
	var order []*types.ResolvedRecipe

	var resolve func(string) error
	resolve = func(current string) error {
		if visited[current] {
			return nil
		}
		if inStack[current] {
			return fmt.Errorf("circular dependency detected: %s -> %s", current, current)
		}
		inStack[current] = true

		recipe, err := d.parser.Parse(current)
		if err != nil {
			return fmt.Errorf("parse dependency %s: %w", current, err)
		}

		// Resolve dependencies first
		for _, dep := range recipe.Dependencies {
			if err := resolve(dep); err != nil {
				return err
			}
		}

		// Now resolve this recipe
		resolver := NewResolver(d.parser)
		resolved, err := resolver.Resolve(current, vars)
		if err != nil {
			return fmt.Errorf("resolve %s: %w", current, err)
		}

		// Check conflicts with already installed recipes
		if d.state != nil {
			for _, conflict := range recipe.Conflicts {
				if d.state.IsRecipeInstalled(conflict) {
					return fmt.Errorf("conflict: recipe %s conflicts with installed recipe %s", current, conflict)
				}
			}
		}

		// Check conflicts with recipes already in the order (being installed)
		for _, installed := range order {
			for _, conflict := range recipe.Conflicts {
				if installed.Name == conflict {
					return fmt.Errorf("conflict: recipe %s conflicts with recipe %s in same batch", current, conflict)
				}
			}
		}

		visited[current] = true
		inStack[current] = false
		order = append(order, resolved)
		return nil
	}

	if err := resolve(name); err != nil {
		return nil, err
	}

	return order, nil
}

// CheckConflicts checks if a recipe conflicts with any already installed recipes
func (d *DependencyResolver) CheckConflicts(name string) error {
	recipe, err := d.parser.Parse(name)
	if err != nil {
		return err
	}
	if d.state == nil {
		return nil
	}
	for _, conflict := range recipe.Conflicts {
		if d.state.IsRecipeInstalled(conflict) {
			return fmt.Errorf("conflict: recipe %s conflicts with installed recipe %s", name, conflict)
		}
	}
	return nil
}
