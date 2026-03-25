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
    inObject := false
    var arrayKey string
    var objectKey string
    var arrayLines []string
    var objectLines []string

    for i := 0; i < len(lines); i++ {
        line := lines[i]
        trimmed := strings.TrimSpace(line)
        isIndented := strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")

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
            if isIndented || trimmed == "" {
                if trimmed != "" {
                    arrayLines = append(arrayLines, trimmed)
                }
                continue
            }
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
                if trimmed != "" {
                    objectLines = append(objectLines, trimmed)
                }
                continue
            }
            if !isIndented && trimmed != "" {
                if len(objectLines) > 0 {
                    anchors = append(anchors, formatAnchorObject(objectKey, objectLines)...)
                }
                inObject = false
                objectLines = nil
            }
        }

        // Skip comments and empty lines
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

        if valuePart == "" {
            if i+1 < len(lines) {
                nextLine := strings.TrimSpace(lines[i+1])
                if strings.HasPrefix(nextLine, "-") {
                    inArray = true
                    arrayKey = anchorName
                    arrayLines = nil
                    continue
                } else if strings.HasPrefix(lines[i+1], "  ") || strings.HasPrefix(lines[i+1], "\t") {
                    inObject = true
                    objectKey = anchorName
                    objectLines = nil
                    continue
                }
            }
            continue
        }

        if anchorName != "" && valuePart != "" {
            anchors = append(anchors, fmt.Sprintf("_%s: &%s %s", anchorName, anchorName, valuePart))
        }
    }

    if inArray && len(arrayLines) > 0 {
        anchors = append(anchors, formatAnchorArray(arrayKey, arrayLines)...)
    }
    if inObject && len(objectLines) > 0 {
        anchors = append(anchors, formatAnchorObject(objectKey, objectLines)...)
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

func formatAnchorObject(anchorName string, lines []string) []string {
    var result []string
    result = append(result, fmt.Sprintf("_%s: &%s", anchorName, anchorName))
    for _, line := range lines {
        result = append(result, line)
    }
    return result
}

func main() {
    data, _ := os.ReadFile("../../debian-composer/kitchen/components/fragments/base-anchors.yaml")
    anchors := extractAnchorsFromFragment(string(data))
    
    // Find service-related anchors
    for _, a := range anchors {
        if strings.Contains(a, "service") {
            fmt.Println(a)
        }
    }
}
