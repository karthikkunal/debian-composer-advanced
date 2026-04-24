package main

import (
	"fmt"
	"os"
	"strings"
)

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

func extractAnchors(content string) []string {
	var anchors []string
	lines := strings.Split(content, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}
		if !strings.Contains(line, ":") || !strings.Contains(line, "&") {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		afterColon := strings.TrimSpace(line[colonIdx+1:])
		if !strings.HasPrefix(afterColon, "&") {
			continue
		}
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
		valuePart := anchorPart[len(anchorName):]
		valuePart = strings.TrimSpace(valuePart)
		if anchorName != "" && valuePart != "" {
			anchors = append(anchors, fmt.Sprintf("_%s: &%s %s", anchorName, anchorName, valuePart))
		}
	}
	return anchors
}

func main() {
	// Read base-anchors.yaml
	data, _ := os.ReadFile("../../debian-composer/kitchen/components/fragments/base-anchors.yaml")
	anchors := extractAnchors(string(data))
	fmt.Printf("Found %d anchors\n", len(anchors))
	for i, a := range anchors {
		if i < 5 {
			fmt.Println(a)
		}
	}
}
