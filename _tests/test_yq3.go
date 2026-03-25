package main

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
)

func main() {
    // Read omarchy.yaml and remove anchor references temporarily
    data, _ := os.ReadFile("../../debian-composer/kitchen/recipes/omarchy.yaml")
    content := string(data)
    
    // Replace *anchor references with a placeholder
    // This is a hack to make yq not fail on anchor references
    lines := strings.Split(content, "\n")
    var cleaned []string
    for _, line := range lines {
        // Replace *anchor with null (will be filled by merge)
        cleanedLine := line
        if strings.Contains(line, "*") && !strings.Contains(line, "&") {
            // This is likely an anchor reference
            cleanedLine = strings.ReplaceAll(cleanedLine, "*dev_packages", "null")
            cleanedLine = strings.ReplaceAll(cleanedLine, "*cli_modern_bundle", "null")
            cleanedLine = strings.ReplaceAll(cleanedLine, "*cli_git_enhanced", "null")
        }
        cleaned = append(cleaned, cleanedLine)
    }
    
    // Write cleaned file
    cleanedContent := strings.Join(cleaned, "\n")
    os.WriteFile("/tmp/omarchy_cleaned.yaml", []byte(cleanedContent), 0644)
    
    // Now merge with includes
    cmd := exec.Command("yq", "eval-all", ".",
        "/tmp/omarchy_cleaned.yaml",
        "../../debian-composer/kitchen/components/fragments/base-anchors.yaml")
    out, err := cmd.CombinedOutput()
    if err != nil {
        fmt.Println("yq error:", err)
        if len(out) > 0 {
            fmt.Println(string(out[:500]))
        }
    } else {
        fmt.Println("yq merge succeeded, length:", len(out))
        // Show a snippet
        str := string(out)
        if len(str) > 1000 {
            str = str[:1000]
        }
        fmt.Println(str)
    }
}
