package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// Simulate what merger does
	kitchen := "../../debian-composer/kitchen"

	// For each include, load and check for anchors
	includes := []string{
		"components/base.yaml",
		"fragments/base-anchors.yaml",
	}

	visited := make(map[string]bool)

	var collectAnchors func(string)
	collectAnchors = func(path string) {
		if visited[path] {
			return
		}
		visited[path] = true

		fullPath := kitchen + "/" + path
		data, err := os.ReadFile(fullPath)
		if err != nil {
			fmt.Println("Error loading", path, ":", err)
			return
		}

		fragContent := string(data)
		fmt.Printf("Loaded %s (%d bytes)\n", path, len(fragContent))

		// Check for anchor references
		hasRefs := strings.Contains(fragContent, "*python_base") ||
			strings.Contains(fragContent, "*dev_packages")
		fmt.Printf("  Has anchor refs: %v\n", hasRefs)

		// Check if it has anchor definitions
		hasDefs := strings.Contains(fragContent, "&dev_packages")
		fmt.Printf("  Has &dev_packages: %v\n", hasDefs)

		// Recurse if has refs
		if hasRefs {
			fmt.Println("  Checking for fragment includes...")
			// Try common fragment paths
			collectAnchors("fragments/base-anchors.yaml")
		}
	}

	for _, inc := range includes {
		collectAnchors(inc)
	}

	fmt.Println("\nVisited paths:", visited)
}
