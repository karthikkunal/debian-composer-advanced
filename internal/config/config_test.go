package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig("/tmp/source", "/tmp/dest", true, false)
	if cfg.sourceDir != "/tmp/source" {
		t.Errorf("expected sourceDir /tmp/source, got %s", cfg.sourceDir)
	}
	if cfg.destDir != "/tmp/dest" {
		t.Errorf("expected destDir /tmp/dest, got %s", cfg.destDir)
	}
	if !cfg.dryRun {
		t.Error("expected dryRun to be true")
	}
	if cfg.verbose {
		t.Error("expected verbose to be false")
	}
}

func TestSetBackup(t *testing.T) {
	cfg := NewConfig("/tmp/source", "/tmp/dest", false, false)

	cfg.SetBackup(true)
	if !cfg.backup {
		t.Error("expected backup to be true")
	}

	cfg.SetBackup(false)
	if cfg.backup {
		t.Error("expected backup to be false")
	}
}

func TestApplyDryRun(t *testing.T) {
	// Create source file
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	sourceFile := filepath.Join(sourceDir, ".bashrc")
	if err := os.WriteFile(sourceFile, []byte("echo hello"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig(sourceDir, destDir, true, false)

	dotfile := Dotfile{
		Source: ".bashrc",
		Target: ".bashrc",
	}

	if err := cfg.Apply(dotfile, nil); err != nil {
		t.Errorf("Apply() error = %v", err)
	}

	// Verify target was NOT created in dry-run mode
	targetFile := filepath.Join(destDir, ".bashrc")
	if _, err := os.Stat(targetFile); err == nil {
		t.Error("target file should not exist in dry-run mode")
	}
}

func TestApplyReal(t *testing.T) {
	// Create source file
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	sourceFile := filepath.Join(sourceDir, ".bashrc")
	content := []byte("echo hello")
	if err := os.WriteFile(sourceFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig(sourceDir, destDir, false, false)
	cfg.SetBackup(false)

	dotfile := Dotfile{
		Source: ".bashrc",
		Target: ".bashrc",
	}

	if err := cfg.Apply(dotfile, nil); err != nil {
		t.Errorf("Apply() error = %v", err)
	}

	// Verify target was created
	targetFile := filepath.Join(destDir, ".bashrc")
	targetContent, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("target file should exist: %v", err)
	}

	if string(targetContent) != string(content) {
		t.Errorf("expected content %q, got %q", content, targetContent)
	}
}

func TestApplyTemplate(t *testing.T) {
	// Create source template file
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	sourceFile := filepath.Join(sourceDir, ".gitconfig")
	templateContent := `[user]
    name = {{.Name}}
    email = {{.Email}}
`
	if err := os.WriteFile(sourceFile, []byte(templateContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig(sourceDir, destDir, false, false)
	cfg.SetBackup(false)

	dotfile := Dotfile{
		Source:   ".gitconfig",
		Target:   ".gitconfig",
		Template: true,
	}

	data := map[string]string{
		"Name":  "John Doe",
		"Email": "john@example.com",
	}

	if err := cfg.Apply(dotfile, data); err != nil {
		t.Errorf("Apply() error = %v", err)
	}

	// Verify template was rendered
	targetFile := filepath.Join(destDir, ".gitconfig")
	targetContent, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("target file should exist: %v", err)
	}

	expectedContent := `[user]
    name = John Doe
    email = john@example.com
`
	if string(targetContent) != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, string(targetContent))
	}
}

func TestApplyBackup(t *testing.T) {
	// Create source and existing target files
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Source file
	sourceFile := filepath.Join(sourceDir, ".bashrc")
	if err := os.WriteFile(sourceFile, []byte("new content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Existing target file
	targetFile := filepath.Join(destDir, ".bashrc")
	if err := os.WriteFile(targetFile, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig(sourceDir, destDir, false, false)
	cfg.SetBackup(true)

	dotfile := Dotfile{
		Source: ".bashrc",
		Target: ".bashrc",
	}

	if err := cfg.Apply(dotfile, nil); err != nil {
		t.Errorf("Apply() error = %v", err)
	}

	// Verify backup was created
	backupFile := filepath.Join(destDir, ".bashrc.backup")
	backupContent, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("backup file should exist: %v", err)
	}

	if string(backupContent) != "old content" {
		t.Errorf("expected backup content 'old content', got %q", backupContent)
	}
}

func TestApplyMissingSource(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig(sourceDir, destDir, false, false)

	dotfile := Dotfile{
		Source: ".nonexistent",
		Target: ".nonexistent",
	}

	err := cfg.Apply(dotfile, nil)
	if err == nil {
		t.Error("Apply() should return error for missing source file")
	}
}

func TestScanSourceDir(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")

	// Create some test files
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		".bashrc":     "bash config",
		".vimrc":      "vim config",
		".config/app": "app config",
	}

	for path, content := range files {
		fullPath := filepath.Join(sourceDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := NewConfig(sourceDir, "/tmp/dest", false, false)

	dotfiles, err := cfg.ScanSourceDir()
	if err != nil {
		t.Errorf("ScanSourceDir() error = %v", err)
	}

	if len(dotfiles) != 3 {
		t.Errorf("expected 3 dotfiles, got %d", len(dotfiles))
	}
}

func TestList(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	// Create source file
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}

	sourceFile := filepath.Join(sourceDir, ".bashrc")
	if err := os.WriteFile(sourceFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig(sourceDir, destDir, false, false)

	statuses, err := cfg.List()
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	if len(statuses) != 1 {
		t.Errorf("expected 1 status, got %d", len(statuses))
	}

	status := statuses[0]
	if !status.SourceExists {
		t.Error("expected SourceExists to be true")
	}
	if status.TargetExists {
		t.Error("expected TargetExists to be false (target not created yet)")
	}
}

func TestDiff(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	// Create source and target files
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}

	sourceFile := filepath.Join(sourceDir, ".bashrc")
	targetFile := filepath.Join(destDir, ".bashrc")

	if err := os.WriteFile(sourceFile, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetFile, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig(sourceDir, destDir, false, false)

	dotfile := Dotfile{
		Source: ".bashrc",
		Target: ".bashrc",
	}

	diff, err := cfg.Diff(dotfile)
	if err != nil {
		t.Errorf("Diff() error = %v", err)
	}

	if diff == "Files are identical" {
		t.Error("expected files to differ")
	}
}

func TestDiffIdentical(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	// Create source and target files with same content
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := "same content"
	sourceFile := filepath.Join(sourceDir, ".bashrc")
	targetFile := filepath.Join(destDir, ".bashrc")

	if err := os.WriteFile(sourceFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := NewConfig(sourceDir, destDir, false, false)

	dotfile := Dotfile{
		Source: ".bashrc",
		Target: ".bashrc",
	}

	diff, err := cfg.Diff(dotfile)
	if err != nil {
		t.Errorf("Diff() error = %v", err)
	}

	if diff != "Files are identical" {
		t.Errorf("expected 'Files are identical', got %q", diff)
	}
}

func TestConvertToDotfile(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"home/user/.bashrc", ".bashrc"},
		{"home/user/.config/app", ".config/app"},
		{".bashrc", ".bashrc"},
		{"config/app", ".config/app"},
	}

	for _, tt := range tests {
		result := convertToDotfile(tt.input)
		if result != tt.expected {
			t.Errorf("convertToDotfile(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFormatDotfileStatus(t *testing.T) {
	statuses := []DotfileStatus{
		{
			Dotfile:      Dotfile{Source: ".bashrc", Target: ".bashrc"},
			SourceExists: true,
			TargetExists: false,
		},
		{
			Dotfile:      Dotfile{Source: ".vimrc", Target: ".vimrc"},
			SourceExists: true,
			TargetExists: true,
			NeedsUpdate:  true,
		},
	}

	result := FormatDotfileStatus(statuses)

	if !contains(result, ".bashrc") {
		t.Error("expected output to contain '.bashrc'")
	}
	if !contains(result, ".vimrc") {
		t.Error("expected output to contain '.vimrc'")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsMiddle(s, substr))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
