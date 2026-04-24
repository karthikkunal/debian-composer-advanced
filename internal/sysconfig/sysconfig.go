package sysconfig

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/debian-composer/debian-composer-go/internal/initsys"
)

// SSHConfig holds SSH server configuration options
type SSHConfig struct {
	PermitRootLogin      string
	PasswordAuth         string
	PermitEmptyPasswords string
	X11Forwarding        string
	MaxAuthTries         string
	ClientAliveInterval  string
	ClientAliveCountMax  string
	AllowUsers           []string
	AllowGroups          []string
	DenyUsers            []string
	DenyGroups           []string
	LoginGraceTime       string
	MaxSessions          string
}

// DefaultSSHConfig returns the default hardened SSH configuration
func DefaultSSHConfig() *SSHConfig {
	return &SSHConfig{
		PermitRootLogin:      "no",
		PasswordAuth:         "yes", // Keep enabled for initial access
		PermitEmptyPasswords: "no",
		X11Forwarding:        "yes",
		MaxAuthTries:         "3",
		ClientAliveInterval:  "300",
		ClientAliveCountMax:  "2",
		AllowUsers:           []string{},
		AllowGroups:          []string{},
		DenyUsers:            []string{},
		DenyGroups:           []string{},
		LoginGraceTime:       "120",
		MaxSessions:          "10",
	}
}

// SSHChecker provides SSH server detection and configuration
type SSHChecker struct {
	configPath   string
	isInstalled  bool
	configExists bool
}

// NewSSHChecker creates a new SSH checker
func NewSSHChecker() *SSHChecker {
	return &SSHChecker{
		configPath:   "/etc/ssh/sshd_config",
		isInstalled:  false,
		configExists: false,
	}
}

// Check verifies SSH server installation and configuration
func (s *SSHChecker) Check() error {
	// Check if SSH server is installed
	if _, err := os.Stat(s.configPath); err == nil {
		s.configExists = true
		s.isInstalled = true
	}

	// Also check if sshd binary exists
	if _, err := exec.LookPath("sshd"); err == nil {
		s.isInstalled = true
	}

	return nil
}

// IsInstalled returns true if SSH server is installed
func (s *SSHChecker) IsInstalled() bool {
	return s.isInstalled
}

// Configure applies SSH hardening configuration
func (s *SSHChecker) Configure(cfg *SSHConfig, dryRun, verbose bool) error {
	if !s.isInstalled {
		if verbose {
			fmt.Println("  [ssh] SSH server not installed, skipping configuration")
		}
		return nil
	}

	if dryRun {
		fmt.Println("  [dry-run] Would configure SSH server")
		fmt.Printf("  [dry-run]   PermitRootLogin: %s\n", cfg.PermitRootLogin)
		fmt.Printf("  [dry-run]   PasswordAuthentication: %s\n", cfg.PasswordAuth)
		fmt.Printf("  [dry-run]   PermitEmptyPasswords: %s\n", cfg.PermitEmptyPasswords)
		fmt.Printf("  [dry-run]   X11Forwarding: %s\n", cfg.X11Forwarding)
		fmt.Printf("  [dry-run]   MaxAuthTries: %s\n", cfg.MaxAuthTries)
		fmt.Printf("  [dry-run]   ClientAliveInterval: %s\n", cfg.ClientAliveInterval)
		fmt.Printf("  [dry-run]   ClientAliveCountMax: %s\n", cfg.ClientAliveCountMax)
		if len(cfg.AllowUsers) > 0 {
			fmt.Printf("  [dry-run]   AllowUsers: %s\n", strings.Join(cfg.AllowUsers, " "))
		}
		if len(cfg.AllowGroups) > 0 {
			fmt.Printf("  [dry-run]   AllowGroups: %s\n", strings.Join(cfg.AllowGroups, " "))
		}
		if len(cfg.DenyUsers) > 0 {
			fmt.Printf("  [dry-run]   DenyUsers: %s\n", strings.Join(cfg.DenyUsers, " "))
		}
		if len(cfg.DenyGroups) > 0 {
			fmt.Printf("  [dry-run]   DenyGroups: %s\n", strings.Join(cfg.DenyGroups, " "))
		}
		if cfg.LoginGraceTime != "" {
			fmt.Printf("  [dry-run]   LoginGraceTime: %s\n", cfg.LoginGraceTime)
		}
		if cfg.MaxSessions != "" {
			fmt.Printf("  [dry-run]   MaxSessions: %s\n", cfg.MaxSessions)
		}
		return nil
	}

	if !s.configExists {
		return fmt.Errorf("SSH config file not found: %s", s.configPath)
	}

	// Create backup
	backupPath := s.configPath + ".debian-composer.bak"
	if err := copyFile(s.configPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup SSH config: %w", err)
	}

	// Apply configuration
	if err := s.applySSHConfig(cfg); err != nil {
		// Restore backup on failure
		_ = copyFile(backupPath, s.configPath)
		return fmt.Errorf("failed to apply SSH config: %w", err)
	}

	// Restart SSH service
	if err := restartService("ssh"); err != nil {
		// Try alternative service name
		if err := restartService("sshd"); err != nil {
			fmt.Printf("  [warning] Could not restart SSH service: %v\n", err)
		}
	}

	return nil
}

func (s *SSHChecker) applySSHConfig(cfg *SSHConfig) error {
	// Read original config
	content, err := os.ReadFile(s.configPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	modified := make([]string, 0, len(lines))

	// Track which settings we've applied
	applied := make(map[string]bool)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comments and empty lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			modified = append(modified, line)
			continue
		}

		// Check if this is a line we want to modify
		newLine := line
		if strings.HasPrefix(trimmed, "PermitRootLogin") {
			newLine = fmt.Sprintf("PermitRootLogin %s", cfg.PermitRootLogin)
			applied["PermitRootLogin"] = true
		} else if strings.HasPrefix(trimmed, "PasswordAuthentication") {
			newLine = fmt.Sprintf("PasswordAuthentication %s", cfg.PasswordAuth)
			applied["PasswordAuthentication"] = true
		} else if strings.HasPrefix(trimmed, "PermitEmptyPasswords") {
			newLine = fmt.Sprintf("PermitEmptyPasswords %s", cfg.PermitEmptyPasswords)
			applied["PermitEmptyPasswords"] = true
		} else if strings.HasPrefix(trimmed, "X11Forwarding") {
			newLine = fmt.Sprintf("X11Forwarding %s", cfg.X11Forwarding)
			applied["X11Forwarding"] = true
		} else if strings.HasPrefix(trimmed, "MaxAuthTries") {
			newLine = fmt.Sprintf("MaxAuthTries %s", cfg.MaxAuthTries)
			applied["MaxAuthTries"] = true
		} else if strings.HasPrefix(trimmed, "ClientAliveInterval") {
			newLine = fmt.Sprintf("ClientAliveInterval %s", cfg.ClientAliveInterval)
			applied["ClientAliveInterval"] = true
		} else if strings.HasPrefix(trimmed, "ClientAliveCountMax") {
			newLine = fmt.Sprintf("ClientAliveCountMax %s", cfg.ClientAliveCountMax)
			applied["ClientAliveCountMax"] = true
		} else if strings.HasPrefix(trimmed, "LoginGraceTime") {
			newLine = fmt.Sprintf("LoginGraceTime %s", cfg.LoginGraceTime)
			applied["LoginGraceTime"] = true
		} else if strings.HasPrefix(trimmed, "MaxSessions") {
			newLine = fmt.Sprintf("MaxSessions %s", cfg.MaxSessions)
			applied["MaxSessions"] = true
		} else if strings.HasPrefix(trimmed, "AllowUsers") {
			if len(cfg.AllowUsers) > 0 {
				newLine = fmt.Sprintf("AllowUsers %s", strings.Join(cfg.AllowUsers, " "))
			} else {
				newLine = "# AllowUsers"
			}
			applied["AllowUsers"] = true
		} else if strings.HasPrefix(trimmed, "AllowGroups") {
			if len(cfg.AllowGroups) > 0 {
				newLine = fmt.Sprintf("AllowGroups %s", strings.Join(cfg.AllowGroups, " "))
			} else {
				newLine = "# AllowGroups"
			}
			applied["AllowGroups"] = true
		} else if strings.HasPrefix(trimmed, "DenyUsers") {
			if len(cfg.DenyUsers) > 0 {
				newLine = fmt.Sprintf("DenyUsers %s", strings.Join(cfg.DenyUsers, " "))
			} else {
				newLine = "# DenyUsers"
			}
			applied["DenyUsers"] = true
		} else if strings.HasPrefix(trimmed, "DenyGroups") {
			if len(cfg.DenyGroups) > 0 {
				newLine = fmt.Sprintf("DenyGroups %s", strings.Join(cfg.DenyGroups, " "))
			} else {
				newLine = "# DenyGroups"
			}
			applied["DenyGroups"] = true
		}

		modified = append(modified, newLine)
	}

	// Add any settings that weren't in the original file
	if !applied["PermitRootLogin"] {
		modified = append(modified, fmt.Sprintf("PermitRootLogin %s", cfg.PermitRootLogin))
	}
	if !applied["PasswordAuthentication"] {
		modified = append(modified, fmt.Sprintf("PasswordAuthentication %s", cfg.PasswordAuth))
	}
	if !applied["PermitEmptyPasswords"] {
		modified = append(modified, fmt.Sprintf("PermitEmptyPasswords %s", cfg.PermitEmptyPasswords))
	}
	if !applied["X11Forwarding"] {
		modified = append(modified, fmt.Sprintf("X11Forwarding %s", cfg.X11Forwarding))
	}
	if !applied["MaxAuthTries"] {
		modified = append(modified, fmt.Sprintf("MaxAuthTries %s", cfg.MaxAuthTries))
	}
	if !applied["ClientAliveInterval"] {
		modified = append(modified, fmt.Sprintf("ClientAliveInterval %s", cfg.ClientAliveInterval))
	}
	if !applied["ClientAliveCountMax"] {
		modified = append(modified, fmt.Sprintf("ClientAliveCountMax %s", cfg.ClientAliveCountMax))
	}
	if !applied["LoginGraceTime"] && cfg.LoginGraceTime != "" {
		modified = append(modified, fmt.Sprintf("LoginGraceTime %s", cfg.LoginGraceTime))
	}
	if !applied["MaxSessions"] && cfg.MaxSessions != "" {
		modified = append(modified, fmt.Sprintf("MaxSessions %s", cfg.MaxSessions))
	}
	if !applied["AllowUsers"] && len(cfg.AllowUsers) > 0 {
		modified = append(modified, fmt.Sprintf("AllowUsers %s", strings.Join(cfg.AllowUsers, " ")))
	}
	if !applied["AllowGroups"] && len(cfg.AllowGroups) > 0 {
		modified = append(modified, fmt.Sprintf("AllowGroups %s", strings.Join(cfg.AllowGroups, " ")))
	}
	if !applied["DenyUsers"] && len(cfg.DenyUsers) > 0 {
		modified = append(modified, fmt.Sprintf("DenyUsers %s", strings.Join(cfg.DenyUsers, " ")))
	}
	if !applied["DenyGroups"] && len(cfg.DenyGroups) > 0 {
		modified = append(modified, fmt.Sprintf("DenyGroups %s", strings.Join(cfg.DenyGroups, " ")))
	}

	// Write modified config
	return os.WriteFile(s.configPath, []byte(strings.Join(modified, "\n")), 0644)
}

// UFWConfig holds UFW firewall configuration
type UFWConfig struct {
	DefaultIncoming string
	DefaultOutgoing string
	AllowRules      []string
	Enable          bool
}

// DefaultUFWConfig returns the default UFW configuration
func DefaultUFWConfig() *UFWConfig {
	return &UFWConfig{
		DefaultIncoming: "deny",
		DefaultOutgoing: "allow",
		AllowRules:      []string{"ssh"},
		Enable:          false, // Don't enable by default, let user decide
	}
}

// UFWChecker provides UFW firewall detection and configuration
type UFWChecker struct {
	isInstalled bool
	isActive    bool
}

// NewUFWChecker creates a new UFW checker
func NewUFWChecker() *UFWChecker {
	return &UFWChecker{
		isInstalled: false,
		isActive:    false,
	}
}

// Check verifies UFW installation and status
func (u *UFWChecker) Check() error {
	// Check if ufw binary exists
	if _, err := exec.LookPath("ufw"); err == nil {
		u.isInstalled = true

		// Check if active
		cmd := exec.Command("ufw", "status")
		output, err := cmd.Output()
		if err == nil && strings.Contains(string(output), "Status: active") {
			u.isActive = true
		}
	}

	return nil
}

// IsInstalled returns true if UFW is installed
func (u *UFWChecker) IsInstalled() bool {
	return u.isInstalled
}

// IsActive returns true if UFW is active
func (u *UFWChecker) IsActive() bool {
	return u.isActive
}

// Configure applies UFW firewall configuration
func (u *UFWChecker) Configure(cfg *UFWConfig, dryRun, verbose bool) error {
	if !u.isInstalled {
		if verbose {
			fmt.Println("  [ufw] UFW not installed, skipping firewall configuration")
		}
		return nil
	}

	if dryRun {
		fmt.Println("  [dry-run] Would configure UFW firewall")
		fmt.Printf("  [dry-run]   Default incoming: %s\n", cfg.DefaultIncoming)
		fmt.Printf("  [dry-run]   Default outgoing: %s\n", cfg.DefaultOutgoing)
		for _, rule := range cfg.AllowRules {
			fmt.Printf("  [dry-run]   Allow: %s\n", rule)
		}
		if cfg.Enable {
			fmt.Println("  [dry-run]   Would enable UFW")
		}
		return nil
	}

	// Set default policies
	if err := u.setDefaultPolicies(cfg.DefaultIncoming, cfg.DefaultOutgoing); err != nil {
		return fmt.Errorf("failed to set UFW default policies: %w", err)
	}

	// Add allow rules
	for _, rule := range cfg.AllowRules {
		if err := u.allowRule(rule); err != nil {
			return fmt.Errorf("failed to add UFW rule %s: %w", rule, err)
		}
	}

	// Enable if requested
	if cfg.Enable {
		if err := u.enable(); err != nil {
			return fmt.Errorf("failed to enable UFW: %w", err)
		}
	}

	return nil
}

func (u *UFWChecker) setDefaultPolicies(incoming, outgoing string) error {
	cmd := exec.Command("ufw", "default", incoming, "incoming")
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("ufw", "default", outgoing, "outgoing")
	return cmd.Run()
}

func (u *UFWChecker) allowRule(rule string) error {
	cmd := exec.Command("ufw", "allow", rule)
	return cmd.Run()
}

func (u *UFWChecker) enable() error {
	// Use yes to automatically confirm
	cmd := exec.Command("sh", "-c", "yes | ufw enable")
	return cmd.Run()
}

// Fail2banConfig holds fail2ban configuration
type Fail2banConfig struct {
	SSHEnabled  bool
	SSHMaxRetry int
	SSHFindTime int
	SSHBanTime  int
	OtherJails  []string
}

// DefaultFail2banConfig returns the default fail2ban configuration
func DefaultFail2banConfig() *Fail2banConfig {
	return &Fail2banConfig{
		SSHEnabled:  true,
		SSHMaxRetry: 5,
		SSHFindTime: 600,  // 10 minutes
		SSHBanTime:  3600, // 1 hour
		OtherJails:  []string{},
	}
}

// Fail2banChecker provides fail2ban detection and configuration
type Fail2banChecker struct {
	isInstalled bool
	configPath  string
	jailPath    string
}

// NewFail2banChecker creates a new fail2ban checker
func NewFail2banChecker() *Fail2banChecker {
	return &Fail2banChecker{
		isInstalled: false,
		configPath:  "/etc/fail2ban/jail.conf",
		jailPath:    "/etc/fail2ban/jail.local",
	}
}

// Check verifies fail2ban installation
func (f *Fail2banChecker) Check() error {
	// Check if fail2ban-server binary exists
	if _, err := exec.LookPath("fail2ban-server"); err == nil {
		f.isInstalled = true
	}

	// Check if config exists
	if _, err := os.Stat(f.configPath); err == nil {
		f.isInstalled = true
	}

	return nil
}

// IsInstalled returns true if fail2ban is installed
func (f *Fail2banChecker) IsInstalled() bool {
	return f.isInstalled
}

// Configure applies fail2ban configuration
func (f *Fail2banChecker) Configure(cfg *Fail2banConfig, dryRun, verbose bool) error {
	if !f.isInstalled {
		if verbose {
			fmt.Println("  [fail2ban] fail2ban not installed, skipping configuration")
		}
		return nil
	}

	if dryRun {
		fmt.Println("  [dry-run] Would configure fail2ban")
		if cfg.SSHEnabled {
			fmt.Printf("  [dry-run]   SSH jail: enabled (maxretry=%d, bantime=%d)\n", cfg.SSHMaxRetry, cfg.SSHBanTime)
		}
		return nil
	}

	// Create jail.local configuration
	if err := f.createJailLocal(cfg); err != nil {
		return fmt.Errorf("failed to create fail2ban jail.local: %w", err)
	}

	// Enable and restart fail2ban
	if err := enableService("fail2ban"); err != nil {
		fmt.Printf("  [warning] Could not enable fail2ban service: %v\n", err)
	}

	if err := restartService("fail2ban"); err != nil {
		fmt.Printf("  [warning] Could not restart fail2ban service: %v\n", err)
	}

	return nil
}

func (f *Fail2banChecker) createJailLocal(cfg *Fail2banConfig) error {
	var sb strings.Builder

	sb.WriteString("# Fail2ban jail.local configuration\n")
	sb.WriteString("# Generated by Debian Composer\n\n")

	if cfg.SSHEnabled {
		sb.WriteString("[sshd]\nenabled = true\n")
		fmt.Fprintf(&sb, "maxretry = %d\n", cfg.SSHMaxRetry)
		fmt.Fprintf(&sb, "findtime = %d\n", cfg.SSHFindTime)
		fmt.Fprintf(&sb, "bantime = %d\n", cfg.SSHBanTime)
		sb.WriteString("port = ssh\n")
		sb.WriteString("logpath = %(sshd_log)s\n")
		sb.WriteString("backend = %(sshd_backend)s\n\n")
	}

	// Add other jails
	for _, jail := range cfg.OtherJails {
		fmt.Fprintf(&sb, "[%s]\nenabled = true\n\n", jail)
	}

	// Ensure directory exists
	dir := filepath.Dir(f.jailPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(f.jailPath, []byte(sb.String()), 0644)
}

// SecurityConfig holds complete security configuration
type SecurityConfig struct {
	SSH      *SSHConfig
	UFW      *UFWConfig
	Fail2ban *Fail2banConfig
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		SSH:      DefaultSSHConfig(),
		UFW:      DefaultUFWConfig(),
		Fail2ban: DefaultFail2banConfig(),
	}
}

// SecurityManager manages system security configuration
type SecurityManager struct {
	SSHChecker      *SSHChecker
	UFWChecker      *UFWChecker
	Fail2banChecker *Fail2banChecker
}

// NewSecurityManager creates a new security manager
func NewSecurityManager() *SecurityManager {
	return &SecurityManager{
		SSHChecker:      NewSSHChecker(),
		UFWChecker:      NewUFWChecker(),
		Fail2banChecker: NewFail2banChecker(),
	}
}

// SSHChecker returns the SSH checker
func (sm *SecurityManager) SSHChecker_() *SSHChecker {
	return sm.SSHChecker
}

// UFWChecker returns the UFW checker
func (sm *SecurityManager) UFWChecker_() *UFWChecker {
	return sm.UFWChecker
}

// Fail2banChecker returns the fail2ban checker
func (sm *SecurityManager) Fail2banChecker_() *Fail2banChecker {
	return sm.Fail2banChecker
}

// Check verifies all security components
func (sm *SecurityManager) Check() error {
	if err := sm.SSHChecker.Check(); err != nil {
		return fmt.Errorf("SSH check failed: %w", err)
	}
	if err := sm.UFWChecker.Check(); err != nil {
		return fmt.Errorf("UFW check failed: %w", err)
	}
	if err := sm.Fail2banChecker.Check(); err != nil {
		return fmt.Errorf("fail2ban check failed: %w", err)
	}
	return nil
}

// Configure applies all security configurations
func (sm *SecurityManager) Configure(cfg *SecurityConfig, dryRun, verbose bool) error {
	// Configure SSH
	if err := sm.SSHChecker.Configure(cfg.SSH, dryRun, verbose); err != nil {
		return fmt.Errorf("SSH configuration failed: %w", err)
	}

	// Configure UFW
	if err := sm.UFWChecker.Configure(cfg.UFW, dryRun, verbose); err != nil {
		return fmt.Errorf("UFW configuration failed: %w", err)
	}

	// Configure fail2ban
	if err := sm.Fail2banChecker.Configure(cfg.Fail2ban, dryRun, verbose); err != nil {
		return fmt.Errorf("fail2ban configuration failed: %w", err)
	}

	return nil
}

// GetStatus returns status of all security components
func (sm *SecurityManager) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"ssh": map[string]interface{}{
			"installed": sm.SSHChecker.IsInstalled(),
		},
		"ufw": map[string]interface{}{
			"installed": sm.UFWChecker.IsInstalled(),
			"active":    sm.UFWChecker.IsActive(),
		},
		"fail2ban": map[string]interface{}{
			"installed": sm.Fail2banChecker.IsInstalled(),
		},
	}
}

// Helper functions

func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, 0644)
}

func restartService(name string) error {
	return initsys.RestartService(name)
}

func enableService(name string) error {
	return initsys.EnableService(name)
}

// IsSSHDaemonRunning checks if SSH daemon is running
func IsSSHDaemonRunning() (bool, error) {
	if initsys.IsActive("ssh") {
		return true, nil
	}
	return initsys.IsActive("sshd"), nil
}

// ConfigureSSHWithPrompt configures SSH with user confirmation
func ConfigureSSHWithPrompt(cfg *SSHConfig, dryRun, verbose, skipConfirm bool) error {
	if !skipConfirm && !dryRun {
		fmt.Println("\nSSH Hardening Configuration:")
		fmt.Printf("  - PermitRootLogin: %s\n", cfg.PermitRootLogin)
		fmt.Printf("  - MaxAuthTries: %s\n", cfg.MaxAuthTries)
		fmt.Printf("  - ClientAliveInterval: %s\n", cfg.ClientAliveInterval)
		fmt.Println("\nThis will modify /etc/ssh/sshd_config and restart SSH service.")
		fmt.Print("Continue? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("SSH configuration skipped")
			return nil
		}
	}

	checker := NewSSHChecker()
	if err := checker.Check(); err != nil {
		return err
	}

	return checker.Configure(cfg, dryRun, verbose)
}
