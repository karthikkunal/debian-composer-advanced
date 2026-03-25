package main

import (
    "fmt"
    "os"
    "strings"
)

func extractAnchorsFromFragment(content string) []string {
    var anchors []string
    lines := strings.Split(content, "\n")
    
    for i := 0; i < len(lines); i++ {
        line := lines[i]
        trimmed := strings.TrimSpace(line)
        
        // Skip comments and empty lines
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
        
        // Add inline anchor
        if anchorName != "" && valuePart != "" {
            anchors = append(anchors, fmt.Sprintf("_%s: &%s %s", anchorName, anchorName, valuePart))
        }
    }
    
    return anchors
}

func main() {
    // Read base-anchors.yaml
    data, _ := os.ReadFile("../../debian-composer/kitchen/components/fragments/base-anchors.yaml")
    anchors := extractAnchorsFromFragment(string(data))
    fmt.Printf("Found %d anchors\n", len(anchors))
    for i, a := range anchors {
        if i < 5 || strings.Contains(a, "dev_packages") {
            fmt.Println(a)
        }
    }
}
