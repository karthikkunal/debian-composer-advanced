package main

import (
	"fmt"
	"github.com/debian-composer/debian-composer-go/internal/recipe"
	"os"
)

func main() {
	p := recipe.NewParser("../../debian-composer/kitchen")
	m := recipe.NewMerger(p)

	content, err := m.MergeRecipe("../../debian-composer/kitchen/recipes/omarchy.yaml")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(string(content[:2000]))
	os.WriteFile("/tmp/merged.yaml", content, 0644)
}
