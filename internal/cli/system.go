package cli

import (
	"fmt"

	"github.com/debian-composer/debian-composer-go/internal/sysconfig"
	"github.com/spf13/cobra"
)

func init() {
	// System config command
	sysCmd := &cobra.Command{
		Use:   "system",
		Short: "Configure system security settings",
		Long: `Configure system security settings including SSH hardening,
firewall (UFW) setup, and fail2ban configuration.`,
		RunE: runSystem,
	}
	sysCmd.Flags().Bool("skip-ssh", false, "Skip SSH hardening")
	sysCmd.Flags().Bool("skip-firewall", false, "Skip firewall setup")
	sysCmd.Flags().Bool("skip-fail2ban", false, "Skip fail2ban configuration")
	sysCmd.Flags().Bool("apply-security", false, "Apply full security hardening (SSH, UFW, fail2ban)")
	rootCmd.AddCommand(sysCmd)
}

func runSystem(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	skipSSH, _ := cmd.Flags().GetBool("skip-ssh")
	skipFirewall, _ := cmd.Flags().GetBool("skip-firewall")
	skipFail2ban, _ := cmd.Flags().GetBool("skip-fail2ban")
	applySecurity, _ := cmd.Flags().GetBool("apply-security")

	if applySecurity {
		skipSSH = false
		skipFirewall = false
		skipFail2ban = false
	}

	fmt.Println("System Configuration")
	fmt.Println("=====================")

	// Get security manager
	sm := sysconfig.NewSecurityManager()

	// Check what is installed
	if err := sm.Check(); err != nil {
		return fmt.Errorf("security check failed: %w", err)
	}

	// Print status
	status := sm.GetStatus()
	fmt.Println("\nStatus:")
	if sshStatus, ok := status["ssh"].(map[string]interface{}); ok {
		installed := sshStatus["installed"].(bool)
		fmt.Printf("  SSH: %s\n", boolToStr(installed))
	}
	if ufwStatus, ok := status["ufw"].(map[string]interface{}); ok {
		installed := ufwStatus["installed"].(bool)
		active := ufwStatus["active"].(bool)
		fmt.Printf("  UFW: %s (active: %s)\n", boolToStr(installed), boolToStr(active))
	}
	if f2bStatus, ok := status["fail2ban"].(map[string]interface{}); ok {
		installed := f2bStatus["installed"].(bool)
		fmt.Printf("  fail2ban: %s\n", boolToStr(installed))
	}

	if dryRun {
		fmt.Println("\n[DRY-RUN] No changes will be made")
	}

	// Get default config
	cfg := sysconfig.DefaultSecurityConfig()

	// Configure SSH
	if !skipSSH && !ShouldSkip("ssh") {
		fmt.Println("\nConfiguring SSH...")
		if err := sm.SSHChecker.Configure(cfg.SSH, dryRun, verbose); err != nil {
			fmt.Printf("  Warning: %v\n", err)
		} else if !dryRun {
			fmt.Println("  ✓ SSH configured")
		}
	} else if verbose {
		fmt.Println("  [skipped] SSH hardening")
	}

	// Configure UFW
	if !skipFirewall && !ShouldSkip("firewall") {
		fmt.Println("\nConfiguring firewall...")
		if err := sm.UFWChecker.Configure(cfg.UFW, dryRun, verbose); err != nil {
			fmt.Printf("  Warning: %v\n", err)
		} else if !dryRun {
			fmt.Println("  ✓ Firewall configured")
		}
	} else if verbose {
		fmt.Println("  [skipped] Firewall setup")
	}

	// Configure fail2ban
	if !skipFail2ban && !ShouldSkip("fail2ban") {
		fmt.Println("\nConfiguring fail2ban...")
		if err := sm.Fail2banChecker.Configure(cfg.Fail2ban, dryRun, verbose); err != nil {
			fmt.Printf("  Warning: %v\n", err)
		} else if !dryRun {
			fmt.Println("  ✓ fail2ban configured")
		}
	} else if verbose {
		fmt.Println("  [skipped] fail2ban configuration")
	}

	fmt.Println("\nNote: UFW is not enabled by default. Run 'sudo ufw enable' when ready.")
	fmt.Println("After configuring SSH, set up SSH keys and disable password authentication.")

	return nil
}

func boolToStr(b bool) string {
	if b {
		return "installed"
	}
	return "not installed"
}
