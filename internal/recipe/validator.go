package recipe

import (
	_ "embed"
	"fmt"

	"github.com/kaptinlin/jsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
var schemaData []byte

// Validator handles JSON schema validation for recipes
type Validator struct {
	schema *jsonschema.Schema
}

// NewValidator creates a new recipe validator
func NewValidator() (*Validator, error) {
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(schemaData)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}

	return &Validator{schema: schema}, nil
}

// Validate validates YAML data against the recipe schema
func (v *Validator) Validate(data []byte) error {
	// Convert YAML to map[string]interface{} for jsonschema
	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	result := v.schema.Validate(m)
	if !result.IsValid() {
		errs := result.Errors
		return fmt.Errorf("schema validation failed: %v", errs)
	}

	return nil
}

// ValidateRecipe is a helper to validate a recipe file
func ValidateRecipe(data []byte) error {
	v, err := NewValidator()
	if err != nil {
		return err
	}
	return v.Validate(data)
}
