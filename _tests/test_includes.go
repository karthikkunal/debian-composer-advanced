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
    data, _ := os.ReadFile("../../debian-composer/kitchen/recipes/omarchy.yaml")
    includes := extractIncludes(string(data))
    fmt.Println("Found includes:", includes)
}
