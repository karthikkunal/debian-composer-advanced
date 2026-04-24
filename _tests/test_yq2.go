package main

import (
	"fmt"
	"os/exec"
)

func main() {
	// Try yq with slurpfile to merge includes
	// yq -n 'load("omarchy.yaml") * load("base.yaml") * load("base-anchors.yaml")'

	// First, let's try just loading the base-anchors which has the anchors defined
	cmd := exec.Command("yq", "eval", ".",
		"../../debian-composer/kitchen/components/fragments/base-anchors.yaml")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("yq base-anchors error:", err)
	} else {
		fmt.Println("base-anchors loaded, length:", len(out))
	}

	// Now try to evaluate with anchor expansion
	// yq eval-all 'select(fileIndex==0) * select(fileIndex==1)' omarchy.yaml base-anchors.yaml
	cmd2 := exec.Command("yq", "eval-all",
		"select(fileIndex==0) * select(fileIndex==1)",
		"../../debian-composer/kitchen/recipes/omarchy.yaml",
		"../../debian-composer/kitchen/components/fragments/base-anchors.yaml")
	out2, err2 := cmd2.CombinedOutput()
	if err2 != nil {
		fmt.Println("yq merge error:", err2)
		if len(out2) > 0 {
			fmt.Println(string(out2[:500]))
		}
	} else {
		fmt.Println("yq merge succeeded, length:", len(out2))
	}
}
