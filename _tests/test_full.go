package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// Test loading fragment with path resolution
	kitchen := "../../debian-composer/kitchen"
	fragPath := "components/base.yaml"

	// Full path
	fullPath := kitchen + "/" + fragPath
	data, err := os.ReadFile(fullPath)
	if err != nil {
		fmt.Println("Error loading:", err)
		return
	}

	fmt.Println("Loaded fragment, size:", len(data))

	// Check if fragment has includes
	content := string(data)
	if strings.Contains(content, "includes:") {
		fmt.Println("Fragment has includes section")
	} else {
		fmt.Println("Fragment has NO includes section")
	}

	// The fragment base.yaml doesn't have includes but uses anchors
	// Those anchors are in fragments/base-anchors.yaml
	// So we need to look for anchor references and find their definitions
}
