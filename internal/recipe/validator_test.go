package recipe

import (
	"testing"
)

func TestValidateRecipe(t *testing.T) {
	// 1. Valid recipe
	validRecipe := `
name: test-recipe
kind: recipe
version: 1.0.0
packages:
  - curl
  - wget
`
	if err := ValidateRecipe([]byte(validRecipe)); err != nil {
		t.Errorf("Validation failed for valid recipe: %v", err)
	}

	// 2. Invalid recipe (missing kind)
	invalidRecipe1 := `
name: test-recipe
version: 1.0.0
`
	if err := ValidateRecipe([]byte(invalidRecipe1)); err == nil {
		t.Error("Validation should have failed for missing kind")
	}

	// 3. Invalid recipe (wrong type for packages)
	invalidRecipe2 := `
name: test-recipe
kind: recipe
packages: curl
`
	if err := ValidateRecipe([]byte(invalidRecipe2)); err == nil {
		t.Error("Validation should have failed for wrong packages type")
	}
}
