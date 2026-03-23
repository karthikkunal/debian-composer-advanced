package recipe

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// YQResolver resolves YAML anchors in-process using yaml.v3
type YQResolver struct{}

// NewYQResolver creates a new resolver
func NewYQResolver() *YQResolver {
	return &YQResolver{}
}

// IsAvailable always returns true — no external binary required
func (y *YQResolver) IsAvailable() bool {
	return true
}

// HasComplexAnchors detects if content has patterns that benefit from full anchor expansion
func HasComplexAnchors(content []byte) bool {
	str := string(content)

	// YAML merge keys (<<: *anchor)
	if strings.Contains(str, "<<:") && strings.Contains(str, "*") {
		return true
	}

	// Objects nested in arrays
	lines := strings.Split(str, "\n")
	for i := 0; i < len(lines)-1; i++ {
		line := strings.TrimSpace(lines[i])
		nextLine := strings.TrimSpace(lines[i+1])

		if strings.HasPrefix(line, "- ") && strings.Contains(line, ":") {
			if strings.HasPrefix(lines[i+1], "    ") && strings.Contains(nextLine, ":") {
				if !strings.HasPrefix(nextLine, "- ") {
					return true
				}
			}
		}
	}

	// Many includes
	if len(extractIncludes(str)) > 5 {
		return true
	}

	return false
}

// Resolve resolves YAML anchors by parsing and re-marshalling with yaml.v3,
// which expands all alias nodes inline — no subprocess required.
func (y *YQResolver) Resolve(content []byte, recipePath string) ([]byte, error) {
	includes := extractIncludes(string(content))

	recipeDir := ""
	if idx := strings.LastIndex(recipePath, "/"); idx >= 0 {
		recipeDir = recipePath[:idx+1]
	}

	// Collect anchor definitions from include files
	var anchorDefs []string
	visited := make(map[string]bool)
	for _, inc := range includes {
		incPath := resolveIncludePath(inc, recipeDir)
		if incPath != "" && !visited[incPath] {
			visited[incPath] = true
			if data, err := os.ReadFile(incPath); err == nil {
				anchorDefs = append(anchorDefs, extractAnchorsFromFragment(string(data))...)
			}
		}
	}

	// Remove includes section from main content
	mainContent := removeIncludesSection(string(content))

	// Replace anchor references with null so yaml.v3 can parse cleanly;
	// the real anchors are prepended below.
	anchorRefPattern := regexp.MustCompile(`\*[a-zA-Z_][a-zA-Z0-9_-]*`)
	cleanedMain := anchorRefPattern.ReplaceAllString(mainContent, "null")

	var combined strings.Builder
	if len(anchorDefs) > 0 {
		combined.WriteString("# Anchor definitions\n")
		for _, def := range anchorDefs {
			combined.WriteString(def)
			combined.WriteString("\n")
		}
		combined.WriteString("\n")
	}
	combined.WriteString(cleanedMain)

	// Parse with yaml.v3 — this expands all &anchor / *alias references inline.
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(combined.String()), &node); err != nil {
		return nil, fmt.Errorf("yaml expand anchors: %w", err)
	}

	expanded, err := yaml.Marshal(&node)
	if err != nil {
		return nil, fmt.Errorf("yaml marshal expanded: %w", err)
	}

	return expanded, nil
}

// resolveIncludePath resolves an include path relative to the recipe directory
func resolveIncludePath(include, recipeDir string) string {
	if recipeDir != "" {
		relPath := recipeDir + include
		if !strings.HasSuffix(relPath, ".yaml") {
			relPath += ".yaml"
		}
		if _, err := os.Stat(relPath); err == nil {
			return relPath
		}
	}

	if strings.HasPrefix(include, "fragments/") && recipeDir != "" {
		kitchenDir := strings.TrimSuffix(recipeDir, "recipes/")
		if kitchenDir != recipeDir {
			altPath := kitchenDir + "components/" + include
			if !strings.HasSuffix(altPath, ".yaml") {
				altPath += ".yaml"
			}
			if _, err := os.Stat(altPath); err == nil {
				return altPath
			}
		}
	}

	return ""
}

// HybridResolver tries Go parser first, falls back to full anchor expansion for complex cases
type HybridResolver struct {
	goParser   *Parser
	goMerger   *Merger
	yqResolver *YQResolver
}

// NewHybridResolver creates a hybrid resolver
func NewHybridResolver(p *Parser) *HybridResolver {
	return &HybridResolver{
		goParser:   p,
		goMerger:   NewMerger(p),
		yqResolver: NewYQResolver(),
	}
}

// Resolve resolves a recipe using Go parser or full anchor expansion for complex cases
func (h *HybridResolver) Resolve(name string) ([]byte, error) {
	path, err := h.goParser.resolvePath(name)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if HasComplexAnchors(data) {
		return h.yqResolver.Resolve(data, path)
	}

	includes := extractIncludes(string(data))
	if len(includes) > 0 {
		return h.goMerger.mergeIncludes(data, includes)
	}

	return data, nil
}
