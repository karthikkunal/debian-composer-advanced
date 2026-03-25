package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Component represents a registry component entry
type Component struct {
	Name         string   `yaml:"name" json:"name"`
	File         string   `yaml:"file" json:"file"`
	Description  string   `yaml:"description" json:"description"`
	Tags         []string `yaml:"tags" json:"tags"`
	Category     string   `yaml:"category" json:"category"`
	HasVariables bool     `yaml:"has_variables" json:"has_variables"`
	Variables    []string `yaml:"variables,omitempty" json:"variables,omitempty"`
}

// Mixin represents cross-cutting concerns
type Mixin struct {
	Name        string   `yaml:"name" json:"name"`
	File        string   `yaml:"file" json:"file"`
	Description string   `yaml:"description" json:"description"`
	Tags        []string `yaml:"tags" json:"tags"`
	AppliesTo   []string `yaml:"applies_to" json:"applies_to"`
}

// Layer represents ordered component stacks
type Layer struct {
	Name        string `yaml:"name" json:"name"`
	File        string `yaml:"file" json:"file"`
	Description string `yaml:"description" json:"description"`
	Order       int    `yaml:"order" json:"order"`
}

// Fragment represents template libraries
type Fragment struct {
	Name        string `yaml:"name" json:"name"`
	File        string `yaml:"file" json:"file"`
	Description string `yaml:"description" json:"description"`
}

// RegistryIndex is the main registry structure
type RegistryIndex struct {
	Components []Component `yaml:"components" json:"components"`
	Mixins     []Mixin     `yaml:"mixins,omitempty" json:"mixins,omitempty"`
	Layers     []Layer     `yaml:"layers,omitempty" json:"layers,omitempty"`
	Fragments  []Fragment  `yaml:"fragments,omitempty" json:"fragments,omitempty"`
}

// Registry manages component registry operations
type Registry struct {
	basePath string
	index    *RegistryIndex
}

// NewRegistry creates a new registry client
func NewRegistry(basePath string) (*Registry, error) {
	r := &Registry{
		basePath: basePath,
	}
	if err := r.loadIndex(); err != nil {
		return nil, err
	}
	return r, nil
}

// loadIndex loads the registry index from index.yaml
func (r *Registry) loadIndex() error {
	indexPath := filepath.Join(r.basePath, "index.yaml")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("failed to read index.yaml: %w", err)
	}

	r.index = &RegistryIndex{}
	if err := yaml.Unmarshal(data, r.index); err != nil {
		return fmt.Errorf("failed to parse index.yaml: %w", err)
	}

	return nil
}

// ListComponents returns all components, optionally filtered by category
func (r *Registry) ListComponents(categoryFilter string) []Component {
	if categoryFilter == "" {
		return r.index.Components
	}

	var filtered []Component
	for _, c := range r.index.Components {
		if strings.EqualFold(c.Category, categoryFilter) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// ListCategories returns unique categories with component counts
func (r *Registry) ListCategories() map[string]int {
	categories := make(map[string]int)
	for _, c := range r.index.Components {
		categories[c.Category]++
	}
	return categories
}

// SearchComponents searches components by name, description, tags, or category
func (r *Registry) SearchComponents(term string) []Component {
	term = strings.ToLower(term)
	var results []Component

	for _, c := range r.index.Components {
		// Search in name
		if strings.Contains(strings.ToLower(c.Name), term) {
			results = append(results, c)
			continue
		}
		// Search in description
		if strings.Contains(strings.ToLower(c.Description), term) {
			results = append(results, c)
			continue
		}
		// Search in category
		if strings.Contains(strings.ToLower(c.Category), term) {
			results = append(results, c)
			continue
		}
		// Search in tags
		for _, tag := range c.Tags {
			if strings.Contains(strings.ToLower(tag), term) {
				results = append(results, c)
				break
			}
		}
	}

	return results
}

// GetComponent returns detailed info for a specific component
func (r *Registry) GetComponent(name string) (*Component, bool) {
	for _, c := range r.index.Components {
		if c.Name == name {
			return &c, true
		}
	}
	return nil, false
}

// GetComponentFile returns the absolute path to a component's YAML file
func (r *Registry) GetComponentFile(name string) (string, bool) {
	for _, c := range r.index.Components {
		if c.Name == name {
			return filepath.Join(r.basePath, c.File), true
		}
	}
	return "", false
}

// ListMixins returns all mixins
func (r *Registry) ListMixins() []Mixin {
	return r.index.Mixins
}

// ListLayers returns all layers sorted by order
func (r *Registry) ListLayers() []Layer {
	layers := make([]Layer, len(r.index.Layers))
	copy(layers, r.index.Layers)
	// Sort by order
	for i := 0; i < len(layers); i++ {
		for j := i + 1; j < len(layers); j++ {
			if layers[j].Order < layers[i].Order {
				layers[i], layers[j] = layers[j], layers[i]
			}
		}
	}
	return layers
}

// ListFragments returns all fragments
func (r *Registry) ListFragments() []Fragment {
	return r.index.Fragments
}

// GetStats returns registry statistics
func (r *Registry) GetStats() map[string]int {
	return map[string]int{
		"components": len(r.index.Components),
		"mixins":     len(r.index.Mixins),
		"layers":     len(r.index.Layers),
		"fragments":  len(r.index.Fragments),
	}
}
