package recipe

import (
	"bufio"
	"os"
	"strings"
)

// preprocessor handles YAML anchor resolution across files
type preprocessor struct {
	parser *Parser
}

// newPreprocessor creates a new preprocessor
func newPreprocessor(p *Parser) *preprocessor {
	return &preprocessor{parser: p}
}

// resolveIncludes preprocesses a recipe file to resolve cross-file anchors
func (pp *preprocessor) resolveIncludes(content []byte, recipePath string) ([]byte, error) {
	lines := strings.Split(string(content), "\n")
	var includes []string
	var output []string

	// Find includes section and collect include paths
	inIncludes := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Start of includes section
		if strings.HasPrefix(trimmed, "includes:") {
			inIncludes = true
			continue
		}

		// In includes section - collect paths
		if inIncludes {
			if strings.HasPrefix(trimmed, "- ") {
				includePath := strings.TrimPrefix(trimmed, "- ")
				includePath = strings.Trim(includePath, "\"'")
				includes = append(includes, includePath)
				continue
			} else if trimmed == "" || strings.HasPrefix(line, " ") {
				// Still in includes (blank line or indented)
				continue
			} else {
				// End of includes section
				inIncludes = false
			}
		}

		// Add non-includes lines to output
		output = append(output, line)
	}

	// Load anchor definitions from fragments
	var anchors []string
	for _, inc := range includes {
		if defs, err := pp.extractAnchorsFromInclude(inc); err == nil {
			anchors = append(anchors, defs...)
		}
	}

	// Combine: anchors first, then recipe (without includes section)
	if len(anchors) > 0 {
		final := []string{"# Auto-extracted anchor definitions"}
		final = append(final, anchors...)
		final = append(final, "")
		final = append(final, output...)
		return []byte(strings.Join(final, "\n")), nil
	}

	return []byte(strings.Join(output, "\n")), nil
}

// extractAnchorsFromInclude extracts just anchor definitions from an include file
func (pp *preprocessor) extractAnchorsFromInclude(includePath string) ([]string, error) {
	path, err := pp.parser.resolvePath(includePath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return extractAnchorDefinitions(string(data)), nil
}

// extractAnchorDefinitions extracts lines containing anchor definitions
// These are lines with & that define anchors, plus their associated values
func extractAnchorDefinitions(content string) []string {
	var anchors []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()

		// Lines containing & are anchor definitions
		if strings.Contains(line, "&") {
			// Include the anchor line and collect the value
			anchors = append(anchors, line)

			// Check if value continues on next lines (for arrays)
			// Simple heuristic: if line ends with [ but no ], collect until ]
			if strings.Contains(line, "[") && !strings.Contains(line, "]") {
				for scanner.Scan() {
					nextLine := scanner.Text()
					anchors = append(anchors, nextLine)
					if strings.Contains(nextLine, "]") {
						break
					}
				}
			}
		}
	}

	return anchors
}
