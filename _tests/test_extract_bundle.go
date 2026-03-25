package main

import (
    "fmt"
    "os"
    "strings"
)

func extractAnchorsFromFragment(content string) []string {
    var anchors []string
    lines := strings.Split(content, "\n")
    
    inArray := false
    var arrayKey string
    var arrayLines []string
    
    for i := 0; i < len(lines); i++ {
        line := lines[i]
        trimmed := strings.TrimSpace(line)
        
        // Skip comments and empty lines
        if strings.HasPrefix(trimmed, "#") || trimmed == "" {
            if inArray && trimmed == "" {
                // End of array
                if len(arrayLines) > 0 {
                    anchors = append(anchors, formatAnchorArray(arrayKey, arrayLines)...)
                }
                inArray = false
                arrayLines = nil
            }
            continue
        }
        
        // Look for anchor definitions: key: &name or key: &name value
        if !strings.Contains(line, ":") || !strings.Contains(line, "&") {
            // If we're in an array, this might be an array line
            if inArray && (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "-")) {
                arrayLines = append(arrayLines, trimmed)
                continue
            }
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
        
        // If we were in an array, finalize it first
        if inArray && len(arrayLines) > 0 {
            anchors = append(anchors, formatAnchorArray(arrayKey, arrayLines)...)
            inArray = false
            arrayLines = nil
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
        
        // Check if value continues on next lines (multi-line array)
        if valuePart == "" && i+1 < len(lines) {
            nextLine := strings.TrimSpace(lines[i+1])
            if strings.HasPrefix(nextLine, "-") || strings.HasPrefix(nextLine, "[") {
                inArray = true
                arrayKey = anchorName
                arrayLines = nil
                continue
            }
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
    
    return anchors
}

func formatAnchorArray(anchorName string, lines []string) []string {
    var result []string
    result = append(result, fmt.Sprintf("_%s: &%s", anchorName, anchorName))
    for _, line := range lines {
        result = append(result, "  "+strings.TrimSpace(line))
    }
    return result
}

func main() {
    data, _ := os.ReadFile("../../debian-composer/kitchen/components/fragments/base-anchors.yaml")
    anchors := extractAnchorsFromFragment(string(data))
    fmt.Printf("Found %d anchors\n", len(anchors))
    
    // Find cli_modern_bundle
    for _, a := range anchors {
        if strings.Contains(a, "cli_modern_bundle") {
            fmt.Println("Found cli_modern_bundle:")
            fmt.Println(a)
            break
        }
    }
}
