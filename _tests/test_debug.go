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

func main() {
    // Read base.yaml
    data, _ := os.ReadFile("../../debian-composer/kitchen/components/base.yaml")
    content := string(data)
    
    // Check for includes
    includes := extractIncludes(content)
    fmt.Println("Includes in base.yaml:", includes)
    
    // Check for anchor references
    hasRef := strings.Contains(content, "*python_base")
    fmt.Println("Has *python_base reference:", hasRef)
    
    // Read base-anchors.yaml directly
    anchorData, err := os.ReadFile("../../debian-composer/kitchen/components/fragments/base-anchors.yaml")
    if err != nil {
        fmt.Println("Error reading base-anchors:", err)
    } else {
        fmt.Println("base-anchors.yaml size:", len(anchorData))
        // Check if it has dev_packages
        hasDevPackages := strings.Contains(string(anchorData), "&dev_packages")
        fmt.Println("Has &dev_packages:", hasDevPackages)
    }
}
