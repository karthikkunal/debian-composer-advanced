package main

import (
    "fmt"
    "os"
    "strings"
)

func HasComplexAnchors(content []byte) bool {
    str := string(content)
    
    // Check for YAML merge keys
    if strings.Contains(str, "<<:") && strings.Contains(str, "*") {
        fmt.Println("Found merge keys")
        return true
    }
    
    // Check for objects in arrays
    lines := strings.Split(str, "\n")
    for i := 0; i < len(lines)-1; i++ {
        line := strings.TrimSpace(lines[i])
        nextLine := strings.TrimSpace(lines[i+1])
        
        if strings.HasPrefix(line, "- ") && strings.Contains(line, ":") {
            if strings.HasPrefix(lines[i+1], "    ") && strings.Contains(nextLine, ":") {
                if !strings.HasPrefix(nextLine, "- ") {
                    fmt.Printf("Found object in array at line %d: %s\n", i+1, line)
                    return true
                }
            }
        }
    }
    
    return false
}

func main() {
    data, _ := os.ReadFile("../../debian-composer/kitchen/recipes/omarchy.yaml")
    fmt.Println("HasComplexAnchors:", HasComplexAnchors(data))
}
