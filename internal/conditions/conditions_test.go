package conditions

import (
	"runtime"
	"testing"
)

func TestEvalSimpleConditions(t *testing.T) {
	ev := NewEvaluator()
	ev.SetHardware(&HardwareEnv{
		Arch:      runtime.GOARCH,
		CPUCores:  4,
		RAMMB:     8192,
		RAMGB:     8,
		HasGPU:    true,
		HasNVIDIA: true,
	})

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"simple true", "true", true},
		{"simple false", "false", false},
		{"arch check", `arch == "amd64"`, runtime.GOARCH == "amd64"},
		{"ram check", "ram_gb >= 4", true},
		{"ram check fail", "ram_gb >= 32", false},
		{"has_gpu", "has_gpu == true", true},
		{"has_nvidia", "has_nvidia", true},
		{"compound and", "has_gpu and ram_gb >= 4", true},
		{"compound or", "has_amd or has_nvidia", true},
		{"not expression", "not has_amd", true},
		{"comparison", "cpu_cores > 2", true},
		{"in array", `arch in ["amd64", "arm64"]`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ev.Eval(tt.condition)
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tt.condition, err)
			}
			if result != tt.expected {
				t.Errorf("Eval(%q) = %v, want %v", tt.condition, result, tt.expected)
			}
		})
	}
}

func TestEvalWithVariables(t *testing.T) {
	ev := NewEvaluator()
	ev.SetVariables(map[string]interface{}{
		"primary_lang": "go",
		"enable_gui":   true,
		"version":      "1.0",
	})

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"var string eq", `vars["primary_lang"] == "go"`, true},
		{"var bool", `vars["enable_gui"]`, true},
		{"var comparison", `vars["version"] == "1.0"`, true},
		{"direct var", "primary_lang == \"go\"", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ev.Eval(tt.condition)
			if err != nil {
				t.Fatalf("Eval(%q) error: %v", tt.condition, err)
			}
			if result != tt.expected {
				t.Errorf("Eval(%q) = %v, want %v", tt.condition, result, tt.expected)
			}
		})
	}
}

func TestEmptyCondition(t *testing.T) {
	ev := NewEvaluator()
	result, err := ev.Eval("")
	if err != nil {
		t.Fatalf("Eval(\"\") error: %v", err)
	}
	if !result {
		t.Error("Empty condition should return true")
	}
}

func TestValidateExpression(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		valid     bool
	}{
		{"valid simple", "true", true},
		{"valid comparison", "ram_gb >= 4", true},
		{"valid compound", "has_gpu and ram_gb >= 8", true},
		{"invalid syntax", "ram_gb >>=", false},
		{"invalid type", "ram_gb + \"string\"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExpression(tt.condition)
			if tt.valid && err != nil {
				t.Errorf("ValidateExpression(%q) unexpected error: %v", tt.condition, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateExpression(%q) expected error but got none", tt.condition)
			}
		})
	}
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		expected  []string
	}{
		{"no vars", "ram_gb >= 4", []string{}},
		{"single var", "my_var > 0", []string{"my_var"}},
		{"multiple vars", "lang == \"go\" && version >= 2", []string{"lang", "version"}},
		{"skip builtins", "has_gpu && cpu_cores >= 4", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := ExtractVariables(tt.condition)
			if len(vars) != len(tt.expected) {
				t.Errorf("ExtractVariables(%q) = %v, want %v", tt.condition, vars, tt.expected)
				return
			}
			for i, v := range vars {
				if v != tt.expected[i] {
					t.Errorf("ExtractVariables(%q)[%d] = %q, want %q", tt.condition, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestDetectHardware(t *testing.T) {
	hw, err := DetectHardware()
	if err != nil {
		t.Fatalf("DetectHardware() error: %v", err)
	}

	if hw.Arch == "" {
		t.Error("Arch should not be empty")
	}
	if hw.CPUThreads < 1 {
		t.Error("CPUThreads should be >= 1")
	}

	t.Logf("Hardware: arch=%s, cpu=%d, ram=%dMB", hw.Arch, hw.CPUThreads, hw.RAMMB)
}

func TestDetectEnvironment(t *testing.T) {
	env, err := DetectEnvironment()
	if err != nil {
		t.Fatalf("DetectEnvironment() error: %v", err)
	}

	if env.User == "" {
		t.Error("User should not be empty")
	}
	if env.Hostname == "" {
		t.Error("Hostname should not be empty")
	}

	t.Logf("Environment: user=%s, hostname=%s, display=%v", env.User, env.Hostname, env.HasDisplay)
}

func TestEvalWithVars(t *testing.T) {
	ev := NewEvaluator()
	ev.SetHardware(&HardwareEnv{RAMGB: 8})

	// Test with override vars
	result, err := ev.EvalWithVars("ram_gb > 4", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("Expected true")
	}

	// Test that original vars are restored
	result, err = ev.Eval("ram_gb > 4")
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("Expected true after EvalWithVars")
	}
}

func TestFormatError(t *testing.T) {
	err := ValidateExpression("invalid syntax !!!")
	formatted := FormatError(err, "invalid syntax !!!")
	if formatted == "" {
		t.Error("FormatError should return non-empty string")
	}
	t.Logf("FormatError: %s", formatted)
}
