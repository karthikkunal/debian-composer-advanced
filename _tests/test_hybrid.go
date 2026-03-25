package main

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
)

func HasComplexAnchors(content []byte) bool {
    str := string(content)
    
    if strings.Contains(str, "<<:") && strings.Contains(str, "*") {
        return true
    }
    
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
    
    return false
}

func main() {
    data, _ := os.ReadFile("../../debian-composer/kitchen/recipes/omarchy.yaml")
    
    fmt.Println("HasComplexAnchors:", HasComplexAnchors(data))
    
    // Check yq availability
    path, err := exec.LookPath("yq")
    fmt.Println("yq path:", path, "err:", err)
    
    // Try direct yq merge
    cmd := exec.Command("yq", "eval-all", ".",
        "../../debian-composer/kitchen/recipes/omarchy.yaml",
        "../../debian-composer/kitchen/components/base.yaml",
        "../../debian-composer/kitchen/components/fragments/base-anchors.yaml")
    out, err := cmd.CombinedOutput()
    if err != nil {
        fmt.Println("yq error:", err)
        if len(out) > 0 {
            fmt.Println(string(out[:500]))
        }
    } else {
        fmt.Println("yq succeeded, output length:", len(out))
    }
}
