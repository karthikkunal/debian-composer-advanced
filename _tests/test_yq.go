package main

import (
	"fmt"
	"os/exec"
)

func main() {
	// Check yq
	path, err := exec.LookPath("yq")
	if err != nil {
		fmt.Println("yq not found:", err)
		return
	}
	fmt.Println("yq found at:", path)

	// Try to use yq on omarchy.yaml
	cmd := exec.Command("yq", "eval", ".", "../../debian-composer/kitchen/recipes/omarchy.yaml")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("yq error:", err)
		fmt.Println(string(out[:500]))
	} else {
		fmt.Println("yq succeeded, output length:", len(out))
	}
}
