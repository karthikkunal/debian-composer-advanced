package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	// Read omarchy.yaml
	data, _ := os.ReadFile("../debian-composer/kitchen/recipes/omarchy.yaml")
	content := string(data)

	// Replace all *anchor references with null using regex
	anchorRefPattern := regexp.MustCompile(`\*[a-zA-Z_][a-zA-Z0-9_-]*`)
	cleaned := anchorRefPattern.ReplaceAllString(content, "null")

	// Write cleaned file
	os.WriteFile("/tmp/omarchy_clean.yaml", []byte(cleaned), 0644)

	// Now merge
	cmd := exec.Command("yq", "eval-all", ".",
		"/tmp/omarchy_clean.yaml",
		"../debian-composer/kitchen/components/fragments/base-anchors.yaml")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("yq error:", err)
		if len(out) > 0 {
			fmt.Println(string(out[:500]))
		}
	} else {
		fmt.Println("yq succeeded, length:", len(out))
		str := string(out)
		// Show packages section
		if idx := strings.Index(str, "packages:"); idx > 0 {
			fmt.Println("\nPackages section:")
			fmt.Println(str[idx : idx+300])
		}
	}
}
