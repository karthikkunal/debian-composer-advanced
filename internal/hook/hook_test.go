package hook

import (
	"fmt"
	"strings"
	"testing"
)

// MockAPTRunner for testing
type MockAPTRunner struct {
	installed map[string]bool
	updated   bool
}

func (m *MockAPTRunner) Install(packages ...string) error {
	for _, pkg := range packages {
		m.installed[pkg] = true
	}
	return nil
}

func (m *MockAPTRunner) Remove(packages ...string) error {
	for _, pkg := range packages {
		delete(m.installed, pkg)
	}
	return nil
}

func (m *MockAPTRunner) Update() error {
	m.updated = true
	return nil
}

func (m *MockAPTRunner) IsInstalled(pkg string) bool {
	return m.installed[pkg]
}

func (m *MockAPTRunner) BatchInstall(packages []string) error {
	for _, pkg := range packages {
		m.installed[pkg] = true
	}
	return nil
}

func TestExecuteAPTHook(t *testing.T) {
	mock := &MockAPTRunner{
		installed: make(map[string]bool),
	}

	executor := NewExecutor(false, false, mock)

	// Test install hook
	hook := Hook{
		Type:    "apt",
		Action:  "install",
		Package: "vim",
	}

	result := executor.executeHook(hook, PhaseInstall, TimingPost, 0, nil)
	if !result.Success {
		t.Errorf("Install hook failed: %v", result.Error)
	}
	if !mock.installed["vim"] {
		t.Error("vim should be installed")
	}

	// Test update hook
	updateHook := Hook{
		Type:   "apt",
		Action: "update",
	}

	result = executor.executeHook(updateHook, PhaseInstall, TimingPre, 0, nil)
	if !result.Success {
		t.Errorf("Update hook failed: %v", result.Error)
	}
	if !mock.updated {
		t.Error("update should be called")
	}
}

func TestExecuteDryRun(t *testing.T) {
	mock := &MockAPTRunner{
		installed: make(map[string]bool),
	}

	executor := NewExecutor(true, false, mock) // dry-run=true

	hook := Hook{
		Type:    "apt",
		Action:  "install",
		Package: "vim",
	}

	result := executor.executeHook(hook, PhaseInstall, TimingPost, 0, nil)
	if !result.Success {
		t.Errorf("Dry-run hook failed: %v", result.Error)
	}
	if mock.installed["vim"] {
		t.Error("vim should NOT be installed in dry-run")
	}
}

func TestExecuteCommandHook(t *testing.T) {
	executor := NewExecutor(false, false, nil)

	hook := Hook{
		Type:    "command",
		Command: "echo hello",
	}

	result := executor.executeHook(hook, PhaseInstall, TimingPost, 0, nil)
	if !result.Success {
		t.Errorf("Command hook failed: %v", result.Error)
	}
}

func TestExecuteCommandHookTimeout(t *testing.T) {
	executor := NewExecutor(false, false, nil)

	hook := Hook{
		Type:    "command",
		Command: "sleep 60",
		Timeout: 1, // 1 second
	}

	result := executor.executeHook(hook, PhaseInstall, TimingPost, 0, nil)
	if result.Success {
		t.Error("Command hook should have timed out")
	}
}

func TestExecuteHooks(t *testing.T) {
	mock := &MockAPTRunner{
		installed: make(map[string]bool),
	}

	executor := NewExecutor(false, false, mock)

	hooks := HookSet{
		PreInstall: []Hook{
			{Type: "apt", Action: "update"},
		},
		PostInstall: []Hook{
			{Type: "apt", Action: "install", Package: "vim"},
			{Type: "apt", Action: "install", Package: "git"},
		},
	}

	summary := executor.ExecuteHooks(hooks, PhaseInstall, "test", nil)
	if !summary.Success {
		t.Errorf("ExecuteHooks failed: %v", summary.Errors)
	}
	// Pre-install (update) + Post-install (vim, git) = 3 hooks
	if len(summary.Results) != 3 {
		t.Errorf("Expected 3 hook results, got %d", len(summary.Results))
	}
	if !mock.installed["vim"] || !mock.installed["git"] {
		t.Error("Both vim and git should be installed")
	}
}

func TestExecuteHooksWithError(t *testing.T) {
	executor := NewExecutor(false, false, nil)

	hooks := HookSet{
		PreInstall: []Hook{
			{Type: "command", Command: "false"}, // Will fail
		},
		PostInstall: []Hook{
			{Type: "command", Command: "echo should not run"},
		},
	}

	summary := executor.ExecuteHooks(hooks, PhaseInstall, "test", nil)
	if summary.Success {
		t.Error("ExecuteHooks should have failed")
	}
	// Post-install should not run after pre-install failure
	if len(summary.Results) != 1 {
		t.Errorf("Expected 1 hook result (pre-install only), got %d", len(summary.Results))
	}
}

func TestExecuteHooksIgnoreError(t *testing.T) {
	executor := NewExecutor(false, false, nil)

	hooks := HookSet{
		PreInstall: []Hook{
			{Type: "command", Command: "false", Ignore: true}, // Will fail but ignored
		},
		PostInstall: []Hook{
			{Type: "command", Command: "echo should run"},
		},
	}

	summary := executor.ExecuteHooks(hooks, PhaseInstall, "test", nil)
	if !summary.Success {
		t.Error("ExecuteHooks should succeed when error is ignored")
	}
	if len(summary.Results) != 2 {
		t.Errorf("Expected 2 hook results, got %d", len(summary.Results))
	}
}

func TestExecuteFailureHooks(t *testing.T) {
	executor := NewExecutor(false, false, nil)

	hooks := HookSet{
		OnFailure: []Hook{
			{Type: "command", Command: "echo cleanup"},
		},
	}

	summary := executor.ExecuteFailureHooks(hooks, fmt.Errorf("test error"), nil)
	if !summary.Success {
		t.Errorf("Failure hooks failed: %v", summary.Errors)
	}
}

func TestExecuteRollbackHooks(t *testing.T) {
	executor := NewExecutor(false, false, nil)

	hooks := HookSet{
		Rollback: []Hook{
			{Type: "command", Command: "echo rollback"},
		},
	}

	summary := executor.ExecuteRollbackHooks(hooks, nil)
	if !summary.Success {
		t.Errorf("Rollback hooks failed: %v", summary.Errors)
	}
}

func TestParseHookSet(t *testing.T) {
	hooksMap := map[string][]map[string]interface{}{
		"pre_install": {
			{"type": "apt", "action": "update"},
			{"type": "apt", "action": "install", "package": "curl"},
		},
		"post_install": {
			{"type": "command", "command": "echo done", "ignore_errors": true},
		},
	}

	set := ParseHookSet(hooksMap)

	if len(set.PreInstall) != 2 {
		t.Errorf("Expected 2 pre_install hooks, got %d", len(set.PreInstall))
	}
	if len(set.PostInstall) != 1 {
		t.Errorf("Expected 1 post_install hook, got %d", len(set.PostInstall))
	}
	if !set.PostInstall[0].Ignore {
		t.Error("ignore_errors should be true")
	}
}

func TestFormatSummary(t *testing.T) {
	summary := &ExecutionSummary{
		Phase: PhaseInstall,
		Results: []Result{
			{Phase: PhaseInstall, Timing: TimingPre, Success: true},
			{Phase: PhaseInstall, Timing: TimingPost, Success: true},
		},
		Success: true,
	}

	formatted := FormatSummary(summary)
	if !strings.Contains(formatted, "install") {
		t.Error("Summary should contain phase name")
	}
}
