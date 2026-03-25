package main

import (
    "fmt"
    "os/exec"
)

func main() {
    // Try yq merge command
    fmt.Println("Trying merge command:")
    cmd := exec.Command("yq", "merge",
        "../../debian-composer/kitchen/recipes/omarchy.yaml",
        "../../debian-composer/kitchen/components/fragments/base-anchors.yaml")
    out, err := cmd.CombinedOutput()
    if err != nil {
        fmt.Println("yq merge error:", err)
        if len(out) > 0 {
            fmt.Println(string(out[:500]))
        }
    } else {
        fmt.Println("yq merge succeeded, length:", len(out))
        // Show snippet
        str := string(out)
        if len(str) > 500 {
            str = str[:500]
        }
        fmt.Println(str)
    }
}
