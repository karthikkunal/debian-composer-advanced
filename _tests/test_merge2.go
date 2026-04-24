package main

import (
	"fmt"
	"github.com/debian-composer/debian-composer-go/internal/recipe"
	"os"
)

func main() {
	p := recipe.NewParser("../../debian-composer/kitchen")

	// Test path resolution
	path, err := p.ResolvePath("components/base.yaml")
	if err != nil {
		fmt.Println("Error resolving path:", err)
	} else {
		fmt.Println("Resolved path:", path)
		data, _ := os.ReadFile(path)
		fmt.Println("File size:", len(data))
	}
}
