package recipe

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/debian-composer/debian-composer-go/internal/types"
	"gopkg.in/yaml.v3"
)

// Parser handles fast YAML recipe parsing
type Parser struct {
	kitchenPath    string
	recipesPath    string
	componentsPath string
	merger         *Merger
	hybridResolver *HybridResolver
	cache          map[string]*types.Recipe
	mu             sync.RWMutex
}

// NewParser creates a new recipe parser
func NewParser(kitchenPath string) *Parser {
	p := &Parser{
		kitchenPath:    kitchenPath,
		recipesPath:    filepath.Join(kitchenPath, "recipes"),
		componentsPath: filepath.Join(kitchenPath, "components"),
		cache:          make(map[string]*types.Recipe),
	}
	p.merger = NewMerger(p)
	p.hybridResolver = NewHybridResolver(p)
	return p
}

// Parse loads and parses a recipe file (with caching)
func (p *Parser) Parse(name string) (*types.Recipe, error) {
	// Check cache first
	p.mu.RLock()
	if cached, ok := p.cache[name]; ok {
		p.mu.RUnlock()
		return cached, nil
	}
	p.mu.RUnlock()

	// Try recipes first, then components
	path, err := p.resolvePath(name)
	if err != nil {
		return nil, fmt.Errorf("recipe not found: %s", name)
	}

	recipe, err := p.parseFile(path)
	if err != nil {
		return nil, err
	}

	// Cache the result
	p.mu.Lock()
	p.cache[name] = recipe
	p.mu.Unlock()

	return recipe, nil
}

// parseFile reads and parses a YAML recipe file with include resolution
func (p *Parser) parseFile(path string) (*types.Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	// Resolve includes and anchors using the hybrid resolver:
	// - complex anchors (merge keys, many includes) → YQResolver (pure Go, yaml.v3)
	// - simple includes → Go merger
	includes := extractIncludes(string(data))

	var content []byte
	if len(includes) > 0 || HasComplexAnchors(data) {
		if HasComplexAnchors(data) {
			content, err = p.hybridResolver.yqResolver.Resolve(data, path)
		} else {
			content, err = p.merger.mergeIncludes(data, includes)
		}
		if err != nil {
			content = data // fallback: parse the file as-is
		}
	} else {
		content = data
	}

	// Parse the resolved content
	var recipe types.Recipe
	if err := yaml.Unmarshal(content, &recipe); err != nil {
		// If direct parse failed, try YQResolver as last resort
		if expanded, yqErr := p.hybridResolver.yqResolver.Resolve(data, path); yqErr == nil {
			if err := yaml.Unmarshal(expanded, &recipe); err == nil {
				content = expanded
			}
		}
		// If still failing, extract packages from raw YAML
		if len(recipe.Packages) == 0 {
			extractPackagesFromRaw(data, &recipe)
		}
	}

	recipe.FilePath = path
	recipe.LoadedAt = time.Now()

	// Normalize package names (handle YAML anchors)
	recipe.Packages = normalizePackages(recipe.Packages)

	return &recipe, nil
}

// extractPackagesFromRaw extracts packages from raw YAML when parsing fails
func extractPackagesFromRaw(data []byte, recipe *types.Recipe) {
	lines := strings.Split(string(data), "\n")
	inPackages := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "packages:") {
			inPackages = true
			continue
		}
		if inPackages {
			if strings.HasPrefix(trimmed, "- ") {
				pkg := strings.TrimPrefix(trimmed, "- ")
				recipe.Packages = append(recipe.Packages, pkg)
			} else if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				break
			}
		}
	}
}

// resolveAnchorReferences replaces *anchor references with their values
func resolveAnchorReferences(content []byte) ([]byte, error) {
	// Collect all anchor definitions
	anchorMap := make(map[string]string)
	collectAnchorsFromContent(string(content), anchorMap)

	if len(anchorMap) == 0 {
		return content, nil
	}

	result := string(content)

	// Replace anchor references
	for name, value := range anchorMap {
		ref := " * " + name
		result = strings.ReplaceAll(result, ref, " "+value)

		// Also try without spaces
		ref2 := "*" + name
		if value != "" {
			result = strings.ReplaceAll(result, ref2, value)
		}
	}

	return []byte(result), nil
}

// collectAnchorsFromContent extracts anchor definitions from content
func collectAnchorsFromContent(content string, anchors map[string]string) {
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Look for patterns like: key: &name value
		if !strings.Contains(line, "&") {
			continue
		}

		// Split on & to find anchor name
		parts := strings.SplitN(line, "&", 2)
		if len(parts) != 2 {
			continue
		}

		// Extract anchor name
		afterAnchor := strings.TrimSpace(parts[1])
		var anchorName string

		for _, ch := range afterAnchor {
			if ch == ' ' || ch == ':' || ch == '\t' || ch == '[' || ch == ',' {
				break
			}
			anchorName += string(ch)
		}

		if anchorName == "" || strings.HasPrefix(anchorName, "*") {
			continue
		}

		// Get the value after &name
		valuePart := afterAnchor[len(anchorName):]
		valuePart = strings.TrimSpace(valuePart)

		// If value is empty, it might be on the next line (array or nested)
		if valuePart == "" && i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if strings.HasPrefix(nextLine, "-") || strings.HasPrefix(nextLine, "[") {
				// Collect array value
				var arrayLines []string
				for j := i + 1; j < len(lines); j++ {
					nl := strings.TrimSpace(lines[j])
					if nl == "" && j > i+2 {
						break
					}
					if strings.HasPrefix(nl, "-") || strings.HasPrefix(nl, "[") || strings.HasPrefix(nl, "]") {
						arrayLines = append(arrayLines, nl)
					} else if nl == "" {
						continue
					} else {
						break
					}
				}
				if len(arrayLines) > 0 {
					// Convert array to inline format
					var items []string
					for _, al := range arrayLines {
						al = strings.TrimPrefix(al, "- ")
						al = strings.Trim(al, "[]")
						al = strings.TrimSpace(al)
						if al != "" {
							items = append(items, al)
						}
					}
					if len(items) > 0 {
						valuePart = "[" + strings.Join(items, ", ") + "]"
					}
				}
			}
		}

		if anchorName != "" && valuePart != "" {
			anchors[anchorName] = valuePart
		}
	}
}

// resolvePath finds the actual file for a recipe name
func (p *Parser) resolvePath(name string) (string, error) {
	// If name contains slash, it's a relative path from kitchen
	if strings.Contains(name, "/") {
		// Try as relative path from kitchen directory (components/... or recipes/...)
		path := filepath.Join(p.kitchenPath, name)
		if !strings.HasSuffix(path, ".yaml") {
			path += ".yaml"
		}
		if fileExists(path) {
			return path, nil
		}

		// Special case: fragments/ might be under components/fragments/
		if strings.HasPrefix(name, "fragments/") {
			altPath := filepath.Join(p.kitchenPath, "components", name)
			if !strings.HasSuffix(altPath, ".yaml") {
				altPath += ".yaml"
			}
			if fileExists(altPath) {
				return altPath, nil
			}
		}

		// Try as-is (absolute or relative to cwd)
		if fileExists(name) {
			return name, nil
		}
		if fileExists(name + ".yaml") {
			return name + ".yaml", nil
		}
		return "", fs.ErrNotExist
	}

	// Try recipes directory
	recipePath := filepath.Join(p.recipesPath, name+".yaml")
	if fileExists(recipePath) {
		return recipePath, nil
	}

	// Try components directory
	compPath := filepath.Join(p.componentsPath, name+".yaml")
	if fileExists(compPath) {
		return compPath, nil
	}

	return "", fs.ErrNotExist
}

// List returns all available recipe names
func (p *Parser) List() ([]string, error) {
	var names []string

	// List recipes
	entries, err := os.ReadDir(p.recipesPath)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}

	return names, nil
}

// ListByKind returns recipes filtered by kind
func (p *Parser) ListByKind(kind string) ([]string, error) {
	all, err := p.List()
	if err != nil {
		return nil, err
	}

	var filtered []string
	for _, name := range all {
		recipe, err := p.Parse(name)
		if err != nil {
			continue
		}
		if recipe.Kind == kind {
			filtered = append(filtered, name)
		}
	}

	return filtered, nil
}

// ClearCache clears the parsing cache
func (p *Parser) ClearCache() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache = make(map[string]*types.Recipe)
}

// normalizePackages handles YAML anchor references
func normalizePackages(packages []string) []string {
	var result []string
	for _, pkg := range packages {
		// Skip empty or anchor-only entries
		if pkg == "" || strings.HasPrefix(pkg, "*") {
			continue
		}
		result = append(result, pkg)
	}
	return result
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ParseAll loads all recipes (for listing/info commands)
func (p *Parser) ParseAll() (map[string]*types.Recipe, error) {
	names, err := p.List()
	if err != nil {
		return nil, err
	}

	result := make(map[string]*types.Recipe)
	for _, name := range names {
		recipe, err := p.Parse(name)
		if err != nil {
			continue // Skip invalid recipes
		}
		result[name] = recipe
	}

	return result, nil
}

// ParseBatch parses multiple recipes in parallel (for composition)
func (p *Parser) ParseBatch(names []string) (map[string]*types.Recipe, error) {
	type result struct {
		name   string
		recipe *types.Recipe
		err    error
	}

	ch := make(chan result, len(names))
	var wg sync.WaitGroup

	for _, name := range names {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			r, err := p.Parse(n)
			ch <- result{name: n, recipe: r, err: err}
		}(name)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	recipes := make(map[string]*types.Recipe)
	var firstErr error

	for res := range ch {
		if res.err != nil && firstErr == nil {
			firstErr = res.err
		}
		if res.recipe != nil {
			recipes[res.name] = res.recipe
		}
	}

	return recipes, firstErr
}

