package recipe

import (
	"fmt"
	"os"
	"strings"

	"dario.cat/mergo"
	"github.com/debian-composer/debian-composer-go/internal/types"
	"gopkg.in/yaml.v3"
)

// Merger handles deep YAML merging for recipe composition
type Merger struct {
	parser *Parser
}

// NewMerger creates a new recipe merger
func NewMerger(p *Parser) *Merger {
	return &Merger{parser: p}
}

// MergeRecipe resolves all includes and merges into a single document
func (m *Merger) MergeRecipe(recipePath string) ([]byte, error) {
	data, err := os.ReadFile(recipePath)
	if err != nil {
		return nil, err
	}

	// Parse the includes from the main file
	includes := extractIncludes(string(data))

	// Build merged content by loading and merging all fragments
	mergedContent, err := m.mergeIncludes(data, includes)
	if err != nil {
		return nil, err
	}

	return mergedContent, nil
}

// mergeIncludes loads all includes and merges them
func (m *Merger) mergeIncludes(mainData []byte, includes []string) ([]byte, error) {
	// Collect all anchor definitions from fragments and their transitive includes
	anchorDefs := m.collectAllAnchors(includes)

	// Remove includes section from main file
	mainContent := removeIncludesSection(string(mainData))

	// Combine: anchors + main content
	var result strings.Builder

	if len(anchorDefs) > 0 {
		result.WriteString("# Resolved anchor definitions\n")
		for _, def := range anchorDefs {
			result.WriteString(def)
			result.WriteString("\n")
		}
		result.WriteString("\n")
	}

	result.WriteString(mainContent)

	return []byte(result.String()), nil
}

// collectAllAnchors recursively collects anchors from fragments
func (m *Merger) collectAllAnchors(includes []string) []string {
	var allAnchors []string
	visited := make(map[string]bool)

	for _, inc := range includes {
		m.collectAnchorsRecursive(inc, &allAnchors, visited)
	}

	return allAnchors
}

// collectAnchorsRecursive collects anchors from a fragment and its includes
func (m *Merger) collectAnchorsRecursive(includePath string, anchors *[]string, visited map[string]bool) {
	// Avoid infinite recursion
	if visited[includePath] {
		return
	}
	visited[includePath] = true

	fragData, err := m.loadFragment(includePath)
	if err != nil {
		return
	}

	content := string(fragData)

	// First, recursively process explicit includes in this fragment
	fragIncludes := extractIncludes(content)
	for _, fi := range fragIncludes {
		m.collectAnchorsRecursive(fi, anchors, visited)
	}

	// Extract anchor references (like *dev_packages) and find their definitions
	anchorRefs := extractAnchorReferences(content)
	if len(anchorRefs) > 0 {
		// Try common locations for anchor definitions
		locations := []string{
			"fragments/base-anchors.yaml",
			"fragments/service-templates.yaml",
			"fragments/blend-anchors.yaml",
		}
		for _, loc := range locations {
			if !visited[loc] {
				m.collectAnchorsRecursive(loc, anchors, visited)
			}
		}
	}

	// Extract anchors from this fragment
	fragAnchors := extractAnchorsFromFragment(content)
	*anchors = append(*anchors, fragAnchors...)
}

// loadFragment loads a fragment file
func (m *Merger) loadFragment(includePath string) ([]byte, error) {
	path, err := m.parser.resolvePath(includePath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// extractIncludes extracts include paths from YAML content
func extractIncludes(content string) []string {
	var includes []string
	lines := strings.Split(content, "\n")

	inIncludes := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "includes:") {
			inIncludes = true
			continue
		}

		if inIncludes {
			if strings.HasPrefix(trimmed, "- ") {
				path := strings.TrimPrefix(trimmed, "- ")
				path = strings.Trim(path, "\"'")
				includes = append(includes, path)
			} else if trimmed == "" {
				continue
			} else {
				break
			}
		}
	}

	return includes
}

// extractAnchorReferences finds anchor references (*name) in content
func extractAnchorReferences(content string) []string {
	refs := make(map[string]bool)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip comments
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Find *name patterns (anchor references)
		for i := 0; i < len(line); i++ {
			if line[i] == '*' && i+1 < len(line) {
				// Extract anchor name
				var name string
				for j := i + 1; j < len(line); j++ {
					ch := line[j]
					if ch == ' ' || ch == '\t' || ch == ',' || ch == ']' || ch == '#' {
						break
					}
					name += string(ch)
				}
				if name != "" {
					refs[name] = true
				}
			}
		}
	}

	var result []string
	for name := range refs {
		result = append(result, name)
	}
	return result
}

// removeIncludesSection removes the includes section from YAML
func removeIncludesSection(content string) string {
	lines := strings.Split(content, "\n")
	var output []string
	inIncludes := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "includes:") {
			inIncludes = true
			continue
		}

		if inIncludes {
			if strings.HasPrefix(trimmed, "- ") {
				continue
			} else if trimmed == "" {
				continue
			} else {
				inIncludes = false
			}
		}

		if !inIncludes {
			output = append(output, line)
		}
	}

	return strings.Join(output, "\n")
}

// extractAnchorsFromFragment extracts anchor definitions from fragment content
func extractAnchorsFromFragment(content string) []string {
	var anchors []string
	lines := strings.Split(content, "\n")

	inArray := false
	inObject := false
	var arrayKey string
	var objectKey string
	var arrayLines []string
	var objectLines []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		isIndented := strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")

		// Skip _hook_* anchors - they're template definitions that can't be
		// properly represented as top-level anchors and cause parsing errors
		if strings.HasPrefix(trimmed, "_hook_") && strings.Contains(trimmed, ":") {
			continue
		}

		// Check if this is a new anchor definition - finalize any in-progress collection
		isAnchorDef := false
		if !isIndented && strings.Contains(line, ":") && strings.Contains(line, "&") {
			if colonIdx := strings.Index(line, ":"); colonIdx >= 0 {
				afterColon := strings.TrimSpace(line[colonIdx+1:])
				if strings.HasPrefix(afterColon, "&") {
					isAnchorDef = true
				}
			}
		}

		if isAnchorDef {
			// Finalize any in-progress collection
			if inArray && len(arrayLines) > 0 {
				anchors = append(anchors, formatAnchorArray(arrayKey, arrayLines)...)
				inArray = false
				arrayLines = nil
			}
			if inObject && len(objectLines) > 0 {
				anchors = append(anchors, formatAnchorObject(objectKey, objectLines)...)
				inObject = false
				objectLines = nil
			}
		}

		// If we're collecting an array, check if this line continues it
		if inArray && !isAnchorDef {
			if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "-") {
				arrayLines = append(arrayLines, trimmed)
				continue
			}
			// Indented content or empty line continues array
			if isIndented || trimmed == "" {
				if trimmed != "" {
					arrayLines = append(arrayLines, trimmed)
				}
				continue
			}
			// New key at top level - end of array
			if !isIndented && trimmed != "" {
				if len(arrayLines) > 0 {
					anchors = append(anchors, formatAnchorArray(arrayKey, arrayLines)...)
				}
				inArray = false
				arrayLines = nil
			}
		}

		// If we're collecting an object, check if this line continues it
		if inObject && !isAnchorDef {
			if isIndented || trimmed == "" {
				if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
					// Store the trimmed line - formatAnchorObject will add indentation
					objectLines = append(objectLines, trimmed)
				}
				continue
			}
			// New key at top level - end of object
			if !isIndented && trimmed != "" {
				if len(objectLines) > 0 {
					anchors = append(anchors, formatAnchorObject(objectKey, objectLines)...)
				}
				inObject = false
				objectLines = nil
			}
		}

		// Skip comments and empty lines (but only after checking collections)
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}

		// Look for anchor definitions: key: &name or key: &name value
		if !strings.Contains(line, ":") || !strings.Contains(line, "&") {
			continue
		}

		// Find pattern: word: &word
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}

		afterColon := strings.TrimSpace(line[colonIdx+1:])

		// Must start with & (anchor definition)
		if !strings.HasPrefix(afterColon, "&") {
			continue
		}
		// Extract anchor name after &
		anchorPart := strings.TrimSpace(afterColon[1:])
		var anchorName string

		for _, ch := range anchorPart {
			if ch == ' ' || ch == '\t' || ch == ',' {
				break
			}
			anchorName += string(ch)
		}

		if anchorName == "" || strings.HasPrefix(anchorName, "*") {
			continue
		}

		// Get the value after &name
		valuePart := anchorPart[len(anchorName):]
		valuePart = strings.TrimSpace(valuePart)

		// If value is empty, next lines might be array or object items
		if valuePart == "" {
			// Peek at next line to determine if array or object
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(nextLine, "-") {
					inArray = true
					arrayKey = anchorName
					arrayLines = nil
					continue
				} else if strings.HasPrefix(lines[i+1], "  ") || strings.HasPrefix(lines[i+1], "\t") {
					// Indented content - object
					inObject = true
					objectKey = anchorName
					objectLines = nil
					continue
				}
			}
			continue
		}

		// Add inline anchor
		if anchorName != "" && valuePart != "" {
			anchors = append(anchors, fmt.Sprintf("_%s: &%s %s", anchorName, anchorName, valuePart))
		}
	}

	// Handle any remaining array
	if inArray && len(arrayLines) > 0 {
		anchors = append(anchors, formatAnchorArray(arrayKey, arrayLines)...)
	}

	// Handle any remaining object
	if inObject && len(objectLines) > 0 {
		anchors = append(anchors, formatAnchorObject(objectKey, objectLines)...)
	}

	return anchors
}

// formatAnchorArray formats an array anchor definition
func formatAnchorArray(anchorName string, lines []string) []string {
	var result []string
	result = append(result, fmt.Sprintf("_%s: &%s", anchorName, anchorName))
	for _, line := range lines {
		result = append(result, "  "+strings.TrimSpace(line))
	}
	return result
}

// formatAnchorObject formats an object anchor definition
func formatAnchorObject(anchorName string, lines []string) []string {
	var result []string
	result = append(result, fmt.Sprintf("_%s: &%s", anchorName, anchorName))
	for _, line := range lines {
		result = append(result, "  "+line)
	}
	return result
}

// DeepMerge performs a deep merge of two YAML documents.
// Overlay values take precedence; slices are appended rather than replaced.
func DeepMerge(base, overlay []byte) ([]byte, error) {
	var baseMap, overlayMap map[string]interface{}

	if err := yaml.Unmarshal(base, &baseMap); err != nil {
		return nil, fmt.Errorf("parse base: %w", err)
	}
	if err := yaml.Unmarshal(overlay, &overlayMap); err != nil {
		return nil, fmt.Errorf("parse overlay: %w", err)
	}

	if err := mergo.Merge(&baseMap, overlayMap, mergo.WithOverride, mergo.WithAppendSlice); err != nil {
		return nil, fmt.Errorf("merge: %w", err)
	}

	return yaml.Marshal(baseMap)
}

// MergeRecipes deep merges two recipe structs.
// Overlay takes precedence. Slices are appended.
func (m *Merger) MergeRecipes(base, overlay *types.Recipe) error {
	return mergo.Merge(base, overlay, mergo.WithOverride, mergo.WithAppendSlice)
}
