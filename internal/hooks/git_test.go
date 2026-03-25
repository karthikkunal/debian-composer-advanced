package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager("/tmp/test", true)
	if m.repoPath != "/tmp/test" {
		t.Errorf("expected repoPath /tmp/test, got %s", m.repoPath)
	}
	if !m.dryRun {
		t.Error("expected dryRun to be true")
	}
}

func TestIsGitRepo(t *testing.T) {
	// Create a temporary directory with .git folder
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	m := NewManager(tmpDir, false)
	if !m.IsGitRepo() {
		t.Error("expected IsGitRepo to return true for directory with .git")
	}

	// Test non-git directory
	nonGitDir := t.TempDir()
	m2 := NewManager(nonGitDir, false)
	if m2.IsGitRepo() {
		t.Error("expected IsGitRepo to return false for directory without .git")
	}
}

func TestInstall(t *testing.T) {
	// Create a temporary directory with .git folder
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "hooks"), 0755); err != nil {
		t.Fatal(err)
	}

	// Test dry-run mode
	m := NewManager(tmpDir, true)
	if err := m.Install(); err != nil {
		t.Errorf("Install() error = %v", err)
	}

	// Verify no hooks were created in dry-run
	hooksDir := filepath.Join(gitDir, "hooks")
	if _, err := os.Stat(filepath.Join(hooksDir, "pre-commit")); err == nil {
		t.Error("pre-commit hook should not exist in dry-run mode")
	}

	// Test actual install
	m2 := NewManager(tmpDir, false)
	if err := m2.Install(); err != nil {
		t.Errorf("Install() error = %v", err)
	}

	// Verify hooks were created
	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	if _, err := os.Stat(preCommitPath); err != nil {
		t.Error("pre-commit hook should exist after install")
	}

	prePushPath := filepath.Join(hooksDir, "pre-push")
	if _, err := os.Stat(prePushPath); err != nil {
		t.Error("pre-push hook should exist after install")
	}

	// Verify hooks are executable
	info, err := os.Stat(preCommitPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("pre-commit hook should be executable")
	}
}

func TestUninstall(t *testing.T) {
	// Create a temporary directory with .git folder and hooks
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create some hooks
	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	prePushPath := filepath.Join(hooksDir, "pre-push")
	if err := os.WriteFile(preCommitPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prePushPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	// Uninstall
	m := NewManager(tmpDir, false)
	if err := m.Uninstall(); err != nil {
		t.Errorf("Uninstall() error = %v", err)
	}

	// Verify hooks were removed
	if _, err := os.Stat(preCommitPath); err == nil {
		t.Error("pre-commit hook should be removed after uninstall")
	}
	if _, err := os.Stat(prePushPath); err == nil {
		t.Error("pre-push hook should be removed after uninstall")
	}
}

func TestStatus(t *testing.T) {
	// Create a temporary directory with .git folder
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create one hook
	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(preCommitPath, []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	m := NewManager(tmpDir, false)
	hooks, err := m.Status()
	if err != nil {
		t.Errorf("Status() error = %v", err)
	}

	if len(hooks) != 4 {
		t.Errorf("expected 4 hooks, got %d", len(hooks))
	}

	// Check pre-commit is enabled
	var preCommitFound bool
	for _, h := range hooks {
		if h.Type == HookPreCommit {
			preCommitFound = true
			if !h.Enabled {
				t.Error("pre-commit should be enabled")
			}
		}
	}
	if !preCommitFound {
		t.Error("pre-commit hook not found in status")
	}
}

func TestFormatHookStatus(t *testing.T) {
	hooks := []GitHook{
		{Type: HookPreCommit, Enabled: true},
		{Type: HookPrePush, Enabled: false},
	}

	result := FormatHookStatus(hooks)
	expected := "Git Hooks Status:\n─────────────────\n  pre-commit           enabled\n  pre-push             disabled\n"

	if result != expected {
		t.Errorf("FormatHookStatus() = %q, want %q", result, expected)
	}
}

func TestGenerateScripts(t *testing.T) {
	m := NewManager("/tmp", false)

	preCommitScript := m.generatePreCommitScript()
	if len(preCommitScript) == 0 {
		t.Error("pre-commit script should not be empty")
	}
	if !contains(preCommitScript, "pre-commit hook") {
		t.Error("pre-commit script should contain 'pre-commit hook'")
	}

	prePushScript := m.generatePrePushScript()
	if len(prePushScript) == 0 {
		t.Error("pre-push script should not be empty")
	}
	if !contains(prePushScript, "pre-push hook") {
		t.Error("pre-push script should contain 'pre-push hook'")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestInstallWithoutGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir, false)
	err := m.Install()
	if err == nil {
		t.Error("Install() should return error for non-git repository")
	}
}

func TestIsGitRepoWithFile(t *testing.T) {
	// Test when .git is a file (e.g., in worktrees or submodules)
	tmpDir := t.TempDir()
	gitFile := filepath.Join(tmpDir, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManager(tmpDir, false)
	if m.IsGitRepo() {
		t.Error("IsGitRepo should return false when .git is a file")
	}
}
