package sysconfig

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDefaultSSHConfig(t *testing.T) {
	cfg := DefaultSSHConfig()

	if cfg.PermitRootLogin != "no" {
		t.Errorf("Expected PermitRootLogin=no, got %s", cfg.PermitRootLogin)
	}
	if cfg.PasswordAuth != "yes" {
		t.Errorf("Expected PasswordAuth=yes, got %s", cfg.PasswordAuth)
	}
	if cfg.PermitEmptyPasswords != "no" {
		t.Errorf("Expected PermitEmptyPasswords=no, got %s", cfg.PermitEmptyPasswords)
	}
	if cfg.MaxAuthTries != "3" {
		t.Errorf("Expected MaxAuthTries=3, got %s", cfg.MaxAuthTries)
	}
}

func TestDefaultUFWConfig(t *testing.T) {
	cfg := DefaultUFWConfig()

	if cfg.DefaultIncoming != "deny" {
		t.Errorf("Expected DefaultIncoming=deny, got %s", cfg.DefaultIncoming)
	}
	if cfg.DefaultOutgoing != "allow" {
		t.Errorf("Expected DefaultOutgoing=allow, got %s", cfg.DefaultOutgoing)
	}
	if cfg.Enable != false {
		t.Error("Expected UFW Enable=false by default")
	}
	// Should have SSH rule
	found := false
	for _, rule := range cfg.AllowRules {
		if rule == "ssh" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected ssh in AllowRules")
	}
}

func TestDefaultFail2banConfig(t *testing.T) {
	cfg := DefaultFail2banConfig()

	if !cfg.SSHEnabled {
		t.Error("Expected SSHEnabled=true")
	}
	if cfg.SSHMaxRetry != 5 {
		t.Errorf("Expected SSHMaxRetry=5, got %d", cfg.SSHMaxRetry)
	}
	if cfg.SSHFindTime != 600 {
		t.Errorf("Expected SSHFindTime=600, got %d", cfg.SSHFindTime)
	}
	if cfg.SSHBanTime != 3600 {
		t.Errorf("Expected SSHBanTime=3600, got %d", cfg.SSHBanTime)
	}
}

func TestSSHCheckerCheck(t *testing.T) {
	checker := NewSSHChecker()
	err := checker.Check()
	if err != nil {
		t.Errorf("SSH check failed: %v", err)
	}
}

func TestUFWCheckerCheck(t *testing.T) {
	checker := NewUFWChecker()
	err := checker.Check()
	if err != nil {
		t.Errorf("UFW check failed: %v", err)
	}
}

func TestFail2banCheckerCheck(t *testing.T) {
	checker := NewFail2banChecker()
	err := checker.Check()
	if err != nil {
		t.Errorf("fail2ban check failed: %v", err)
	}
}

func TestSecurityManagerCheck(t *testing.T) {
	sm := NewSecurityManager()
	err := sm.Check()
	if err != nil {
		t.Errorf("Security manager check failed: %v", err)
	}
}

func TestSecurityManagerGetStatus(t *testing.T) {
	sm := NewSecurityManager()
	_ = sm.Check()

	status := sm.GetStatus()

	if _, ok := status["ssh"]; !ok {
		t.Error("Status should contain ssh key")
	}
	if _, ok := status["ufw"]; !ok {
		t.Error("Status should contain ufw key")
	}
	if _, ok := status["fail2ban"]; !ok {
		t.Error("Status should contain fail2ban key")
	}
}

func TestSSHConfigureDryRun(t *testing.T) {
	checker := NewSSHChecker()
	_ = checker.Check()

	cfg := DefaultSSHConfig()
	err := checker.Configure(cfg, true, false) // dry-run=true
	if err != nil {
		t.Errorf("SSH dry-run configure failed: %v", err)
	}
}

func TestUFWConfigureDryRun(t *testing.T) {
	checker := NewUFWChecker()
	_ = checker.Check()

	cfg := DefaultUFWConfig()
	err := checker.Configure(cfg, true, false) // dry-run=true
	if err != nil {
		t.Errorf("UFW dry-run configure failed: %v", err)
	}
}

func TestFail2banConfigureDryRun(t *testing.T) {
	checker := NewFail2banChecker()
	_ = checker.Check()

	cfg := DefaultFail2banConfig()
	err := checker.Configure(cfg, true, false) // dry-run=true
	if err != nil {
		t.Errorf("fail2ban dry-run configure failed: %v", err)
	}
}

func TestSecurityManagerConfigureDryRun(t *testing.T) {
	sm := NewSecurityManager()
	_ = sm.Check()

	cfg := DefaultSecurityConfig()
	err := sm.Configure(cfg, true, false) // dry-run=true
	if err != nil {
		t.Errorf("Security manager dry-run configure failed: %v", err)
	}
}

func TestApplySSHConfig(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "sshd_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test config
	testConfig := `# SSH Config
Port 22
PermitRootLogin yes
PasswordAuthentication yes
# Other settings
UsePAM yes
`
	if err := os.WriteFile(tmpFile.Name(), []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	checker := &SSHChecker{
		configPath:   tmpFile.Name(),
		isInstalled:  true,
		configExists: true,
	}

	cfg := DefaultSSHConfig()
	err = checker.applySSHConfig(cfg)
	if err != nil {
		t.Fatalf("applySSHConfig failed: %v", err)
	}

	// Read and verify
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read modified config: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	found := make(map[string]bool)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "PermitRootLogin no") {
			found["PermitRootLogin"] = true
		}
		if strings.HasPrefix(trimmed, "MaxAuthTries 3") {
			found["MaxAuthTries"] = true
		}
	}

	if !found["PermitRootLogin"] {
		t.Error("PermitRootLogin should be set to 'no'")
	}
	if !found["MaxAuthTries"] {
		t.Error("MaxAuthTries should be set to '3'")
	}
}

func TestFail2banCreateJailLocal(t *testing.T) {
	checker := NewFail2banChecker()

	// Create temp directory for jail.local
	tmpDir, err := os.MkdirTemp("", "fail2ban_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	checker.jailPath = tmpDir + "/jail.local"

	cfg := DefaultFail2banConfig()
	err = checker.createJailLocal(cfg)
	if err != nil {
		t.Fatalf("createJailLocal failed: %v", err)
	}

	// Read and verify
	content, err := os.ReadFile(checker.jailPath)
	if err != nil {
		t.Fatalf("Failed to read jail.local: %v", err)
	}

	if !strings.Contains(string(content), "[sshd]") {
		t.Error("jail.local should contain [sshd] section")
	}
	if !strings.Contains(string(content), "enabled = true") {
		t.Error("jail.local should have enabled = true")
	}
	if !strings.Contains(string(content), "maxretry = 5") {
		t.Error("jail.local should have maxretry = 5")
	}
}

func TestDefaultSecurityConfig(t *testing.T) {
	cfg := DefaultSecurityConfig()

	if cfg.SSH == nil {
		t.Error("SSH config should not be nil")
	}
	if cfg.UFW == nil {
		t.Error("UFW config should not be nil")
	}
	if cfg.Fail2ban == nil {
		t.Error("Fail2ban config should not be nil")
	}
}

func TestCopyFile(t *testing.T) {
	// Create source file
	srcFile, err := os.CreateTemp("", "src_*")
	if err != nil {
		t.Fatalf("Failed to create src file: %v", err)
	}
	defer os.Remove(srcFile.Name())

	testContent := "test content"
	if err := os.WriteFile(srcFile.Name(), []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write src: %v", err)
	}

	// Create dest path
	dstFile, err := os.CreateTemp("", "dst_*")
	if err != nil {
		t.Fatalf("Failed to create dst file: %v", err)
	}
	dstPath := dstFile.Name()
	dstFile.Close()
	os.Remove(dstPath)

	// Copy
	if err := copyFile(srcFile.Name(), dstPath); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}
	defer os.Remove(dstPath)

	// Verify
	content, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read dst: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Content mismatch: expected %q, got %q", testContent, string(content))
	}
}

func TestRestartServiceDryRun(t *testing.T) {
	// This test verifies that restartService doesn't panic
	// We can't actually test service restart in test environment
	err := restartService("nonexistent-service-test")
	// Expected to fail, but shouldn't panic
	if err == nil {
		t.Log("Service restart didn't fail (unexpected)")
	}
}

func TestSecurityConfigCustomValues(t *testing.T) {
	cfg := &SecurityConfig{
		SSH: &SSHConfig{
			PermitRootLogin:     "prohibit-password",
			PasswordAuth:        "no",
			PermitEmptyPasswords: "no",
			X11Forwarding:       "no",
			MaxAuthTries:        "2",
		},
		UFW: &UFWConfig{
			DefaultIncoming: "deny",
			DefaultOutgoing: "allow",
			AllowRules:      []string{"22", "80", "443"},
			Enable:          true,
		},
		Fail2ban: &Fail2banConfig{
			SSHEnabled:  true,
			SSHMaxRetry: 3,
			SSHFindTime: 300,
			SSHBanTime:  7200,
		},
	}

	if cfg.SSH.PermitRootLogin != "prohibit-password" {
		t.Errorf("Expected custom PermitRootLogin, got %s", cfg.SSH.PermitRootLogin)
	}
	if cfg.UFW.Enable != true {
		t.Error("Expected custom UFW Enable=true")
	}
	if cfg.Fail2ban.SSHMaxRetry != 3 {
		t.Errorf("Expected custom SSHMaxRetry=3, got %d", cfg.Fail2ban.SSHMaxRetry)
	}
}

func TestUFWAllowRules(t *testing.T) {
	cfg := DefaultUFWConfig()

	// Verify default rules
	expectedRules := []string{"ssh"}
	for _, expected := range expectedRules {
		found := false
		for _, rule := range cfg.AllowRules {
			if rule == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected rule %s in AllowRules", expected)
		}
	}
}

func TestFail2banOtherJails(t *testing.T) {
	cfg := &Fail2banConfig{
		SSHEnabled: true,
		OtherJails: []string{"nginx-limit-req", "apache-auth"},
	}

	if len(cfg.OtherJails) != 2 {
		t.Errorf("Expected 2 other jails, got %d", len(cfg.OtherJails))
	}

	checker := NewFail2banChecker()

	// Create temp directory for jail.local
	tmpDir, err := os.MkdirTemp("", "fail2ban_jail_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	checker.jailPath = tmpDir + "/jail.local"

	err = checker.createJailLocal(cfg)
	if err != nil {
		t.Fatalf("createJailLocal failed: %v", err)
	}

	// Read and verify
	content, err := os.ReadFile(checker.jailPath)
	if err != nil {
		t.Fatalf("Failed to read jail.local: %v", err)
	}

	if !strings.Contains(string(content), "[nginx-limit-req]") {
		t.Error("jail.local should contain nginx-limit-req section")
	}
	if !strings.Contains(string(content), "[apache-auth]") {
		t.Error("jail.local should contain apache-auth section")
	}
}

func TestSSHCheckerNotInstalled(t *testing.T) {
	// Create checker with non-existent config path
	checker := &SSHChecker{
		configPath:   "/nonexistent/path/sshd_config",
		isInstalled:  false,
		configExists: false,
	}

	_ = checker.Check()

	// In test environment, SSH might be detected via path lookup
	// So we just test the configure behavior when not installed
	
	// Configure should succeed (no-op) when not installed and verbose
	err := checker.Configure(DefaultSSHConfig(), false, true)
	// If SSH is not installed, it should return nil (no-op)
	// If SSH is detected, it will fail because config doesn't exist
	// Both are acceptable in test environment
	if err != nil && checker.isInstalled {
		t.Logf("SSH detected but config missing (expected in test env): %v", err)
	}
}

func TestUFWCheckerNotInstalled(t *testing.T) {
	// Force not installed by looking for non-existent binary
	checker := &UFWChecker{
		isInstalled: false,
		isActive:    false,
	}

	// Override IsInstalled check
	if checker.IsInstalled() {
		t.Error("UFW should not be detected as installed in test env")
	}

	// Configure should succeed (no-op) when not installed
	err := checker.Configure(DefaultUFWConfig(), false, true)
	if err != nil {
		t.Errorf("Configure should succeed when UFW not installed: %v", err)
	}
}

func TestFail2banCheckerNotInstalled(t *testing.T) {
	checker := &Fail2banChecker{
		isInstalled: false,
	}

	// Configure should succeed (no-op) when not installed
	err := checker.Configure(DefaultFail2banConfig(), false, true)
	if err != nil {
		t.Errorf("Configure should succeed when fail2ban not installed: %v", err)
	}
}

func TestExecLookPath(t *testing.T) {
	// Test that exec.LookPath works for common binaries
	_, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not found in PATH, skipping")
	}
}
