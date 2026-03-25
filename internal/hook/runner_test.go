package hook

import (
	"testing"

	"github.com/debian-composer/debian-composer-go/internal/types"
)

// MockAPTRunnerForRunner for testing
type MockAPTRunnerForRunner struct {
	installed []string
}

func (m *MockAPTRunnerForRunner) Install(packages ...string) error {
	m.installed = append(m.installed, packages...)
	return nil
}

func (m *MockAPTRunnerForRunner) Remove(packages ...string) error {
	return nil
}

func (m *MockAPTRunnerForRunner) Update() error {
	return nil
}

func (m *MockAPTRunnerForRunner) IsInstalled(pkg string) bool {
	for _, installed := range m.installed {
		if installed == pkg {
			return true
		}
	}
	return false
}

func (m *MockAPTRunnerForRunner) BatchInstall(packages []string) error {
	m.installed = append(m.installed, packages...)
	return nil
}

func TestNewRecipeRunner(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	config := RunnerConfig{
		DryRun:   true,
		Verbose:  true,
		Kitchen:  "/test/kitchen",
	}

	runner := NewRecipeRunner(config, mock)

	if runner == nil {
		t.Fatal("NewRecipeRunner returned nil")
	}
	if !runner.dryRun {
		t.Error("Runner should have dryRun=true")
	}
	if !runner.verbose {
		t.Error("Runner should have verbose=true")
	}
}

func TestRecipeRunnerRun(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	config := RunnerConfig{
		DryRun:  true,
		Verbose: false,
	}

	runner := NewRecipeRunner(config, mock)

	recipe := &types.ResolvedRecipe{
		Recipe: types.Recipe{
			Name:        "test-recipe",
			Description: "Test recipe",
			Install: types.InstallHook{
				Pre:  []string{"echo pre-install"},
				Post: []string{"echo post-install"},
			},
			Configure: types.ConfigureHook{
				UserGroups: []string{"testgroup"},
				Commands:   []string{"echo configure"},
			},
			Verify: types.VerifyHook{
				Commands: []string{"echo verify"},
			},
		},
	}

	result := runner.Run(recipe, nil)

	if result == nil {
		t.Fatal("Run returned nil")
	}
	if result.RecipeName != "test-recipe" {
		t.Errorf("Expected recipe name 'test-recipe', got %s", result.RecipeName)
	}
	if !result.Success {
		t.Errorf("Run should succeed, error: %v", result.Error)
	}
	if len(result.Phases) == 0 {
		t.Error("Expected at least one phase")
	}
}

func TestRecipeRunnerRunWithFailingHook(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	config := RunnerConfig{
		DryRun:  false,
		Verbose: false,
	}

	runner := NewRecipeRunner(config, mock)

	recipe := &types.ResolvedRecipe{
		Recipe: types.Recipe{
			Name: "test-recipe-fail",
			Install: types.InstallHook{
				Pre: []string{"false"}, // This will fail
			},
		},
	}

	result := runner.Run(recipe, nil)

	if result == nil {
		t.Fatal("Run returned nil")
	}
	if result.Success {
		t.Error("Run should fail when pre-install hook fails")
	}
}

func TestRecipeRunnerRunWithVariables(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	config := RunnerConfig{
		DryRun:  true,
		Verbose: false,
	}

	runner := NewRecipeRunner(config, mock)

	recipe := &types.ResolvedRecipe{
		Recipe: types.Recipe{
			Name: "test-recipe-vars",
			Configure: types.ConfigureHook{
				Commands: []string{"echo {{.Var1}}"},
			},
		},
	}

	variables := map[string]interface{}{
		"Var1": "test-value",
	}

	result := runner.Run(recipe, variables)

	if result == nil {
		t.Fatal("Run returned nil")
	}
	if !result.Success {
		t.Errorf("Run should succeed, error: %v", result.Error)
	}
}

func TestBuildCommandHooks(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	config := RunnerConfig{
		DryRun: true,
	}

	runner := NewRecipeRunner(config, mock)

	commands := []string{"echo 1", "echo 2", "echo 3"}
	hooks := runner.buildCommandHooks(commands)

	if len(hooks) != 3 {
		t.Errorf("Expected 3 hooks, got %d", len(hooks))
	}

	for i, hook := range hooks {
		if hook.Type != "command" {
			t.Errorf("Hook %d should be command type", i)
		}
		if hook.Command != commands[i] {
			t.Errorf("Hook %d command mismatch", i)
		}
		if hook.Timeout != 300 {
			t.Errorf("Hook %d should have 300s timeout", i)
		}
	}
}

func TestBuildConfigureHooks(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	config := RunnerConfig{
		DryRun: true,
	}

	runner := NewRecipeRunner(config, mock)

	recipe := &types.ResolvedRecipe{
		Recipe: types.Recipe{
			Name: "test",
			Configure: types.ConfigureHook{
				UserGroups: []string{"group1", "group2"},
				Commands:   []string{"cmd1", "cmd2"},
			},
		},
	}

	hooks := runner.buildConfigureHooks(recipe)

	// 2 groups + 2 commands = 4 hooks
	if len(hooks) != 4 {
		t.Errorf("Expected 4 hooks, got %d", len(hooks))
	}
}

func TestSetEnvironment(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	config := RunnerConfig{
		DryRun: true,
	}

	runner := NewRecipeRunner(config, mock)

	env := map[string]string{
		"TEST_VAR": "test_value",
		"PATH":     "/usr/bin",
	}

	runner.SetEnvironment(env)

	executor := runner.GetExecutor()
	executorEnv := executor.GetEnv()

	if executorEnv["TEST_VAR"] != "test_value" {
		t.Errorf("Expected TEST_VAR=test_value, got %s", executorEnv["TEST_VAR"])
	}
	if executorEnv["PATH"] != "/usr/bin" {
		t.Errorf("Expected PATH=/usr/bin, got %s", executorEnv["PATH"])
	}
}

func TestFormatInstallResult(t *testing.T) {
	result := &InstallResult{
		RecipeName: "test-recipe",
		Success:    true,
		Phases: []PhaseResult{
			{
				Phase: PhaseInstall,
				Summary: &ExecutionSummary{
					Phase:     PhaseInstall,
					Success:   true,
					TotalTime: 100000000, // 100ms
				},
			},
		},
		TotalTime: 200000000, // 200ms
	}

	formatted := FormatInstallResult(result)

	if formatted == "" {
		t.Error("FormatInstallResult should return non-empty string")
	}
	if !contains(formatted, "test-recipe") {
		t.Error("Formatted result should contain recipe name")
	}
	if !contains(formatted, "Success") {
		t.Error("Formatted result should contain Success status")
	}
}

func TestFormatInstallResultFailure(t *testing.T) {
	result := &InstallResult{
		RecipeName: "test-recipe",
		Success:    false,
		Error:      nil,
		Phases: []PhaseResult{
			{
				Phase: PhaseInstall,
				Summary: &ExecutionSummary{
					Phase:     PhaseInstall,
					Success:   false,
					TotalTime: 100000000,
				},
			},
		},
		TotalTime: 200000000,
	}

	formatted := FormatInstallResult(result)

	if !contains(formatted, "Failed") {
		t.Error("Formatted result should contain Failed status")
	}
}

func TestSecurityHardeningRunner(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	runner := NewSecurityHardeningRunner(true, false, mock) // dry-run=true

	if runner == nil {
		t.Fatal("NewSecurityHardeningRunner returned nil")
	}
}

func TestRunSSHHardening(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	runner := NewSecurityHardeningRunner(true, false, mock)

	summary := runner.RunSSHHardening(nil)

	if summary == nil {
		t.Fatal("RunSSHHardening returned nil")
	}
	// In dry-run, should succeed
	if !summary.Success {
		t.Errorf("RunSSHHardening should succeed in dry-run: %v", summary.Errors)
	}
}

func TestRunFirewallHardening(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	runner := NewSecurityHardeningRunner(true, false, mock)

	summary := runner.RunFirewallHardening(nil)

	if summary == nil {
		t.Fatal("RunFirewallHardening returned nil")
	}
	if !summary.Success {
		t.Errorf("RunFirewallHardening should succeed in dry-run: %v", summary.Errors)
	}
}

func TestRunFail2banHardening(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	runner := NewSecurityHardeningRunner(true, false, mock)

	summary := runner.RunFail2banHardening(nil)

	if summary == nil {
		t.Fatal("RunFail2banHardening returned nil")
	}
	if !summary.Success {
		t.Errorf("RunFail2banHardening should succeed in dry-run: %v", summary.Errors)
	}
}

func TestRunFullSecurityHardening(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	runner := NewSecurityHardeningRunner(true, false, mock)

	success, err := runner.RunFullSecurityHardening(nil)

	if !success {
		t.Errorf("RunFullSecurityHardening should succeed in dry-run: %v", err)
	}
	if err != nil {
		t.Errorf("RunFullSecurityHardening should not return error in dry-run: %v", err)
	}
}

func TestRecipeRunnerWithSkipOptions(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	config := RunnerConfig{
		DryRun:       true,
		Verbose:      false,
		SkipSSH:      true,
		SkipFirewall: true,
		SkipFail2ban: true,
	}

	runner := NewRecipeRunner(config, mock)

	if !runner.skipSSH {
		t.Error("skipSSH should be true")
	}
	if !runner.skipFirewall {
		t.Error("skipFirewall should be true")
	}
	if !runner.skipFail2ban {
		t.Error("skipFail2ban should be true")
	}
}

func TestRecipeRunnerUninstall(t *testing.T) {
	mock := &MockAPTRunnerForRunner{}
	config := RunnerConfig{
		DryRun:  true,
		Verbose: false,
	}

	runner := NewRecipeRunner(config, mock)

	recipe := &types.ResolvedRecipe{
		Recipe: types.Recipe{
			Name: "test-recipe",
		},
	}

	result := runner.Uninstall(recipe, nil)

	if result == nil {
		t.Fatal("Uninstall returned nil")
	}
	if result.RecipeName != "test-recipe" {
		t.Errorf("Expected recipe name 'test-recipe', got %s", result.RecipeName)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
