package validation

import (
	"testing"
)

func TestNewValidator(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}
	if v == nil {
		t.Fatal("NewValidator() returned nil")
	}
}

func TestValidateYAML_Valid(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	validYAML := []byte(`
name: test-recipe
version: 1.0.0
description: A test recipe
author: Test Author
packages:
  - vim
  - git
  - curl
`)

	result, err := v.ValidateYAML(validYAML)
	if err != nil {
		t.Fatalf("ValidateYAML() failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid, got errors: %v", result.Errors)
	}
}

func TestValidateYAML_InvalidName(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	invalidYAML := []byte(`
name: "Invalid Name With Spaces"
version: 1.0.0
`)

	result, err := v.ValidateYAML(invalidYAML)
	if err != nil {
		t.Fatalf("ValidateYAML() failed: %v", err)
	}

	// Should fail due to invalid name pattern
	if result.Valid {
		t.Error("Expected invalid result for uppercase name")
	}
}

func TestValidateYAML_MissingRequired(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	invalidYAML := []byte(`
description: Missing name and version
`)

	result, err := v.ValidateYAML(invalidYAML)
	if err != nil {
		t.Fatalf("ValidateYAML() failed: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid result for missing required fields")
	}
}

func TestValidateYAML_InvalidVersion(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	invalidYAML := []byte(`
name: test
version: not-a-version
`)

	result, err := v.ValidateYAML(invalidYAML)
	if err != nil {
		t.Fatalf("ValidateYAML() failed: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid result for invalid version")
	}
}

func TestValidateYAML_Warnings(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	validYAML := []byte(`
name: test-recipe
version: 1.0.0
`)

	result, err := v.ValidateYAML(validYAML)
	if err != nil {
		t.Fatalf("ValidateYAML() failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid, got errors: %v", result.Errors)
	}

	// Should have warnings for missing description and author
	if len(result.Warnings) < 2 {
		t.Errorf("Expected at least 2 warnings, got %d", len(result.Warnings))
	}
}

func TestValidateYAML_InvalidYAML(t *testing.T) {
	v, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	invalidYAML := []byte(`
name: test
version: 1.0.0
  invalid indentation
`)

	result, err := v.ValidateYAML(invalidYAML)
	if err != nil {
		t.Fatalf("ValidateYAML() failed: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid result for malformed YAML")
	}
}

func TestCheckDependencies(t *testing.T) {
	available := []string{"vim", "git", "curl", "python3", "gcc"}

	tests := []struct {
		name        string
		packages    []string
		expectValid bool
	}{
		{
			name:        "all available",
			packages:    []string{"vim", "git"},
			expectValid: true,
		},
		{
			name:        "with version constraint",
			packages:    []string{"vim (>= 8.0)", "git (= 1:2.30.0)"},
			expectValid: true,
		},
		{
			name:        "some missing",
			packages:    []string{"vim", "nonexistent"},
			expectValid: true, // warnings, not errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckDependencies(tt.packages, available)
			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, result.Valid)
			}
		})
	}
}

func TestValidateVariables(t *testing.T) {
	definitions := map[string]interface{}{
		"primary_lang": map[string]interface{}{
			"type":     "string",
			"required": true,
			"choices":  []interface{}{"python", "go", "rust", "javascript"},
		},
		"editor": map[string]interface{}{
			"type":    "string",
			"default": "vim",
		},
	}

	tests := []struct {
		name        string
		vars        map[string]interface{}
		expectValid bool
	}{
		{
			name: "valid choice",
			vars: map[string]interface{}{
				"primary_lang": "go",
			},
			expectValid: true,
		},
		{
			name: "invalid choice",
			vars: map[string]interface{}{
				"primary_lang": "ruby",
			},
			expectValid: false,
		},
		{
			name:        "missing required",
			vars:        map[string]interface{}{},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateVariables(tt.vars, definitions)
			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v, errors: %v",
					tt.expectValid, result.Valid, result.Errors)
			}
		})
	}
}

func TestFormatErrors(t *testing.T) {
	result := &ValidationResult{
		Valid:    false,
		Errors:   []string{"error 1", "error 2"},
		Warnings: []string{"warning 1"},
	}

	formatted := FormatErrors(result)
	if formatted == "" {
		t.Error("FormatErrors() returned empty string")
	}
	if len(formatted) < 10 {
		t.Error("FormatErrors() returned too short string")
	}
}

func TestValidateVars(t *testing.T) {
	tests := []struct {
		name        string
		vars        map[string]interface{}
		defs        map[string]VarDefinition
		expectValid bool
		expectErrs  int
	}{
		{
			name: "valid string",
			vars: map[string]interface{}{"name": "test"},
			defs: map[string]VarDefinition{
				"name": {Type: VarTypeString, Required: true},
			},
			expectValid: true,
		},
		{
			name: "missing required",
			vars: map[string]interface{}{},
			defs: map[string]VarDefinition{
				"name": {Type: VarTypeString, Required: true},
			},
			expectValid: false,
			expectErrs:  1,
		},
		{
			name: "wrong type - string for int",
			vars: map[string]interface{}{"port": "not-a-number"},
			defs: map[string]VarDefinition{
				"port": {Type: VarTypeInteger},
			},
			expectValid: false,
		},
		{
			name: "valid integer",
			vars: map[string]interface{}{"port": 8080},
			defs: map[string]VarDefinition{
				"port": {Type: VarTypeInteger},
			},
			expectValid: true,
		},
		{
			name: "integer out of range",
			vars: map[string]interface{}{"port": 99999},
			defs: map[string]VarDefinition{
				"port": {
					Type: VarTypePort,
					Min:  float64Ptr(1),
					Max:  float64Ptr(65535),
				},
			},
			expectValid: false,
		},
		{
			name: "valid port",
			vars: map[string]interface{}{"port": 8080},
			defs: map[string]VarDefinition{
				"port": {Type: VarTypePort},
			},
			expectValid: true,
		},
		{
			name: "invalid choice",
			vars: map[string]interface{}{"lang": "ruby"},
			defs: map[string]VarDefinition{
				"lang": {
					Type:    VarTypeString,
					Choices: []string{"go", "python", "rust"},
				},
			},
			expectValid: false,
		},
		{
			name: "valid choice",
			vars: map[string]interface{}{"lang": "go"},
			defs: map[string]VarDefinition{
				"lang": {
					Type:    VarTypeString,
					Choices: []string{"go", "python", "rust"},
				},
			},
			expectValid: true,
		},
		{
			name: "boolean from string",
			vars: map[string]interface{}{"debug": "true"},
			defs: map[string]VarDefinition{
				"debug": {Type: VarTypeBoolean},
			},
			expectValid: true,
		},
		{
			name: "boolean native",
			vars: map[string]interface{}{"debug": true},
			defs: map[string]VarDefinition{
				"debug": {Type: VarTypeBoolean},
			},
			expectValid: true,
		},
		{
			name: "valid email",
			vars: map[string]interface{}{"email": "user@example.com"},
			defs: map[string]VarDefinition{
				"email": {Type: VarTypeEmail},
			},
			expectValid: true,
		},
		{
			name: "default value applied",
			vars: map[string]interface{}{},
			defs: map[string]VarDefinition{
				"editor": {Type: VarTypeString, Default: "vim"},
			},
			expectValid: true,
		},
		{
			name: "integer from string",
			vars: map[string]interface{}{"port": "9090"},
			defs: map[string]VarDefinition{
				"port": {Type: VarTypeInteger},
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateVars(tt.vars, tt.defs)
			if result.Valid != tt.expectValid {
				t.Errorf("ValidateVars() valid = %v, want %v\nErrors: %v",
					result.Valid, tt.expectValid, result.Errors)
			}
			if tt.expectErrs > 0 && len(result.Errors) < tt.expectErrs {
				t.Errorf("Expected at least %d errors, got %d", tt.expectErrs, len(result.Errors))
			}
		})
	}
}

func TestVarDefinition(t *testing.T) {
	// Test that VarDefinition can be used in YAML-style definitions
	defs := map[string]VarDefinition{
		"primary_lang": {
			Type:     VarTypeString,
			Required: true,
			Choices:  []string{"go", "python", "rust", "javascript"},
		},
		"port": {
			Type: VarTypePort,
			Min:  float64Ptr(1024),
			Max:  float64Ptr(65535),
		},
		"enable_ssl": {
			Type:    VarTypeBoolean,
			Default: false,
		},
		"install_path": {
			Type:    VarTypePath,
			Default: "/opt/myapp",
		},
		"workers": {
			Type:    VarTypeInteger,
			Default: 4,
			Min:     float64Ptr(1),
			Max:     float64Ptr(32),
		},
	}

	vars := map[string]interface{}{
		"primary_lang": "go",
		"port":         8080,
		"enable_ssl":   true,
		"workers":      8,
	}

	result := ValidateVars(vars, defs)
	if !result.Valid {
		t.Errorf("Expected valid, got errors: %v", result.Errors)
	}
}

func TestValidateVarsWithPattern(t *testing.T) {
	defs := map[string]VarDefinition{
		"version": {
			Type:    VarTypeString,
			Pattern: `^\d+\.\d+\.\d+$`,
		},
	}

	tests := []struct {
		value string
		valid bool
	}{
		{"1.0.0", true},
		{"2.3.4", true},
		{"invalid", false},
		{"1.0", false},
	}

	for _, tt := range tests {
		vars := map[string]interface{}{"version": tt.value}
		result := ValidateVars(vars, defs)
		if result.Valid != tt.valid {
			t.Errorf("ValidateVars(version=%s) valid=%v, want %v", tt.value, result.Valid, tt.valid)
		}
	}
}

func float64Ptr(f float64) *float64 {
	return &f
}
