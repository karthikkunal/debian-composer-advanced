package validation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/kaptinlin/jsonschema"
	"gopkg.in/yaml.v3"
)

// VarType represents variable types
type VarType string

const (
	VarTypeString  VarType = "string"
	VarTypeInteger VarType = "integer"
	VarTypeBoolean VarType = "boolean"
	VarTypeList    VarType = "list"
	VarTypePath    VarType = "path"
	VarTypePort    VarType = "port"
	VarTypeURL     VarType = "url"
	VarTypeEmail   VarType = "email"
)

// VarDefinition defines a variable with type constraints
type VarDefinition struct {
	Type        VarType     `json:"type" yaml:"type"`
	Default     interface{} `json:"default,omitempty" yaml:"default,omitempty"`
	Required    bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Choices     []string    `json:"choices,omitempty" yaml:"choices,omitempty"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Min         *float64    `json:"min,omitempty" yaml:"min,omitempty"`
	Max         *float64    `json:"max,omitempty" yaml:"max,omitempty"`
	Pattern     string      `json:"pattern,omitempty" yaml:"pattern,omitempty"`
}

// ValidateVars validates variables against typed definitions
func ValidateVars(vars map[string]interface{}, definitions map[string]VarDefinition) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	validate := validator.New()

	// Register custom validators
	registerCustomValidators(validate)

	for name, def := range definitions {
		val, hasVal := vars[name]

		// Check required
		if def.Required && !hasVal {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Required variable '%s' is missing", name))
			continue
		}

		if !hasVal {
			// Use default if available
			if def.Default != nil {
				vars[name] = def.Default
			}
			continue
		}

		// Type validation
		if err := validateType(name, val, def, validate); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
			continue
		}

		// Range validation (for numbers)
		if err := validateRange(name, val, def); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, err.Error())
			continue
		}

		// Choices validation
		if len(def.Choices) > 0 {
			if err := validateChoices(name, val, def.Choices); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, err.Error())
				continue
			}
		}

		// Pattern validation (for strings)
		if def.Pattern != "" && def.Type == VarTypeString {
			if strVal, ok := val.(string); ok {
				if matched, _ := regexp.MatchString(def.Pattern, strVal); !matched {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("Variable '%s' value '%s' does not match pattern '%s'", name, strVal, def.Pattern))
				}
			}
		}

		// Additional validation using go-playground/validator tags
		if err := validateWithTag(validate, name, val, def); err != nil {
			result.Warnings = append(result.Warnings, err.Error())
		}
	}

	return result
}

// validateType validates the type of a variable value
func validateType(name string, val interface{}, def VarDefinition, validate *validator.Validate) error {
	switch def.Type {
	case VarTypeString:
		if _, ok := val.(string); !ok {
			return fmt.Errorf("variable '%s' must be a string, got %T", name, val)
		}

	case VarTypeInteger:
		switch v := val.(type) {
		case int:
		case int64:
		case float64:
			// Check if it's actually an integer
			if v != float64(int64(v)) {
				return fmt.Errorf("variable '%s' must be an integer, got float %v", name, v)
			}
		case string:
			if _, err := strconv.ParseInt(v, 10, 64); err != nil {
				return fmt.Errorf("variable '%s' must be an integer, got string '%s'", name, v)
			}
		default:
			return fmt.Errorf("variable '%s' must be an integer, got %T", name, val)
		}

	case VarTypeBoolean:
		switch val.(type) {
		case bool:
			// OK
		case string:
			s := strings.ToLower(val.(string))
			if s != "true" && s != "false" && s != "yes" && s != "no" && s != "1" && s != "0" {
				return fmt.Errorf("variable '%s' must be a boolean, got '%s'", name, val)
			}
		default:
			return fmt.Errorf("variable '%s' must be a boolean, got %T", name, val)
		}

	case VarTypeList:
		switch val.(type) {
		case []interface{}:
		case []string:
		case string:
			// Comma-separated string is acceptable
		default:
			return fmt.Errorf("variable '%s' must be a list, got %T", name, val)
		}

	case VarTypePath:
		str, ok := val.(string)
		if !ok {
			return fmt.Errorf("variable '%s' (path) must be a string, got %T", name, val)
		}
		if !strings.HasPrefix(str, "/") && !strings.HasPrefix(str, "./") && !strings.HasPrefix(str, "~/") {
			// Warning added by caller
		}

	case VarTypePort:
		var port int
		switch v := val.(type) {
		case int:
			port = v
		case int64:
			port = int(v)
		case float64:
			port = int(v)
		case string:
			p, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("variable '%s' must be a valid port number, got '%s'", name, v)
			}
			port = p
		default:
			return fmt.Errorf("variable '%s' must be a port number, got %T", name, val)
		}
		if port < 1 || port > 65535 {
			return fmt.Errorf("variable '%s' port %d must be between 1-65535", name, port)
		}

	case VarTypeURL:
		str, ok := val.(string)
		if !ok {
			return fmt.Errorf("variable '%s' (url) must be a string, got %T", name, val)
		}
		if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
			// Warning: URL should start with http(s)
		}

	case VarTypeEmail:
		str, ok := val.(string)
		if !ok {
			return fmt.Errorf("variable '%s' (email) must be a string, got %T", name, val)
		}
		if err := validate.Var(str, "email"); err != nil {
			return fmt.Errorf("variable '%s' must be a valid email, got '%s'", name, str)
		}
	}

	return nil
}

// validateRange validates numeric range constraints
func validateRange(name string, val interface{}, def VarDefinition) error {
	if def.Min == nil && def.Max == nil {
		return nil
	}

	var num float64
	switch v := val.(type) {
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case float64:
		num = v
	case string:
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil // Type validation will catch this
		}
		num = n
	default:
		return nil // Type validation will catch this
	}

	if def.Min != nil && num < *def.Min {
		return fmt.Errorf("variable '%s' value %v must be >= %v", name, num, *def.Min)
	}
	if def.Max != nil && num > *def.Max {
		return fmt.Errorf("variable '%s' value %v must be <= %v", name, num, *def.Max)
	}

	return nil
}

// validateChoices validates that value is one of the allowed choices
func validateChoices(name string, val interface{}, choices []string) error {
	strVal := fmt.Sprintf("%v", val)
	for _, choice := range choices {
		if strVal == choice {
			return nil
		}
	}
	return fmt.Errorf("variable '%s' value '%v' must be one of: %v", name, val, choices)
}

// validateWithTag validates using go-playground/validator tags
func validateWithTag(validate *validator.Validate, name string, val interface{}, def VarDefinition) error {
	// Add custom validation logic based on VarDefinition
	// This is a placeholder for future custom tag implementations
	return nil
}

// registerCustomValidators registers custom validation tags
func registerCustomValidators(validate *validator.Validate) {
	// Register custom validation tags if needed
	// Example: validate.RegisterValidation("port", validatePort)
}

// RecipeSchema is the JSON Schema for recipe validation
var RecipeSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["name", "version"],
  "properties": {
    "name": {
      "type": "string",
      "minLength": 1,
      "pattern": "^[a-z0-9][a-z0-9-]*$",
      "description": "Recipe name (lowercase, hyphens allowed)"
    },
    "version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+\\.\\d+$",
      "description": "Semantic version"
    },
    "description": {
      "type": "string"
    },
    "author": {
      "type": "string"
    },
    "license": {
      "type": "string"
    },
    "kind": {
      "type": "string",
      "enum": ["", "distro", "pure blend", "persona", "mixin"]
    },
    "tags": {
      "type": "array",
      "items": {"type": "string"}
    },
    "includes": {
      "type": "array",
      "items": {"type": "string"}
    },
    "extends": {
      "type": "string"
    },
    "variables": {
      "type": "object"
    },
    "packages": {
      "type": "array",
      "items": {"type": "string"}
    },
    "categories": {
      "type": "object"
    },
    "stacks": {
      "type": "object"
    },
    "requirements": {
      "type": "object",
      "properties": {
        "ram_mb": {"type": "integer", "minimum": 0},
        "disk_gb": {"type": "integer", "minimum": 0},
        "cpu_cores": {"type": "integer", "minimum": 1},
        "gpu": {"type": "string"},
        "debian_version": {"type": "string"}
      }
    },
    "services": {
      "type": "array"
    },
    "hooks": {
      "type": "object"
    },
    "post_install_messages": {
      "type": "array",
      "items": {"type": "string"}
    }
  },
  "additionalProperties": true
}`

// Validator validates recipes against the schema
type Validator struct {
	schema *jsonschema.Schema
}

// ValidationResult contains validation outcome
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// NewValidator creates a new recipe validator
func NewValidator() (*Validator, error) {
	compiler := jsonschema.NewCompiler()

	schema, err := compiler.Compile([]byte(RecipeSchema))
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	return &Validator{
		schema: schema,
	}, nil
}

// ValidateYAML validates YAML content against the recipe schema
func (v *Validator) ValidateYAML(yamlContent []byte) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// First, check if it's valid YAML
	var raw map[string]interface{}
	if err := yaml.Unmarshal(yamlContent, &raw); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid YAML: %v", err))
		return result, nil
	}

	// Convert to JSON for schema validation
	jsonBytes, err := yamlToJSON(yamlContent)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to convert YAML to JSON: %v", err))
		return result, nil
	}

	// Validate against schema
	validation := v.schema.Validate(jsonBytes)
	if !validation.IsValid() {
		result.Valid = false
		for _, err := range validation.Errors {
			result.Errors = append(result.Errors, err.Error())
		}
	}

	// Add warnings for missing recommended fields
	if _, ok := raw["description"]; !ok {
		result.Warnings = append(result.Warnings, "Missing 'description' field (recommended)")
	}
	if _, ok := raw["author"]; !ok {
		result.Warnings = append(result.Warnings, "Missing 'author' field (recommended)")
	}

	// Check for deprecated fields
	if _, ok := raw["meta"]; ok {
		result.Warnings = append(result.Warnings, "Using deprecated 'meta' field, use top-level fields instead")
	}

	return result, nil
}

// yamlToJSON converts YAML bytes to JSON bytes
func yamlToJSON(yamlBytes []byte) ([]byte, error) {
	var data interface{}
	if err := yaml.Unmarshal(yamlBytes, &data); err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

// ValidateRecipe validates a parsed recipe structure
func (v *Validator) ValidateRecipe(recipe interface{}) (*ValidationResult, error) {
	// Marshal to YAML then validate
	yamlBytes, err := yaml.Marshal(recipe)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal recipe: %w", err)
	}

	return v.ValidateYAML(yamlBytes)
}

// CheckDependencies checks if required packages exist
func CheckDependencies(packages []string, available []string) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	availMap := make(map[string]bool)
	for _, pkg := range available {
		availMap[pkg] = true
	}

	for _, pkg := range packages {
		// Strip version constraints
		name := pkg
		if idx := strings.Index(pkg, " "); idx > 0 {
			name = pkg[:idx]
		}
		if idx := strings.Index(pkg, "="); idx > 0 {
			name = pkg[:idx]
		}
		if idx := strings.Index(pkg, ">"); idx > 0 {
			name = pkg[:idx]
		}
		if idx := strings.Index(pkg, "<"); idx > 0 {
			name = pkg[:idx]
		}

		if !availMap[name] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Package '%s' not found in repository", name))
		}
	}

	return result
}

// ValidateVariables checks variable values against constraints
func ValidateVariables(vars map[string]interface{}, definitions map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	for name, defRaw := range definitions {
		def, ok := defRaw.(map[string]interface{})
		if !ok {
			continue
		}

		val, hasVal := vars[name]

		// Check required
		if required, ok := def["required"].(bool); ok && required && !hasVal {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Required variable '%s' is missing", name))
			continue
		}

		if !hasVal {
			continue
		}

		// Check choices
		if choices, ok := def["choices"].([]interface{}); ok && len(choices) > 0 {
			valid := false
			for _, choice := range choices {
				if fmt.Sprintf("%v", choice) == fmt.Sprintf("%v", val) {
					valid = true
					break
				}
			}
			if !valid {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Variable '%s' has invalid value '%v', must be one of: %v", name, val, choices))
			}
		}
	}

	return result
}

// FormatErrors formats validation errors for display
func FormatErrors(result *ValidationResult) string {
	var sb strings.Builder

	if result.Valid {
		sb.WriteString("Validation passed")
		if len(result.Warnings) > 0 {
			sb.WriteString(fmt.Sprintf(" (with %d warnings)", len(result.Warnings)))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString(fmt.Sprintf("Validation failed with %d errors:\n", len(result.Errors)))
		for i, err := range result.Errors {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err))
		}
	}

	if len(result.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("\nWarnings (%d):\n", len(result.Warnings)))
		for i, warn := range result.Warnings {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, warn))
		}
	}

	return sb.String()
}
