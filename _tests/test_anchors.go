package main

import (
	"fmt"
	"github.com/debian-composer/debian-composer-go/internal/recipe"
)

func main() {
	p := recipe.NewParser("../../debian-composer/kitchen")
	m := recipe.NewMerger(p)

	content, err := m.MergeRecipe("../../debian-composer/kitchen/recipes/omarchy.yaml")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Show first 1000 chars
	str := string(content)
	if len(str) > 1000 {
		str = str[:1000]
	}
	fmt.Println(str)
}
