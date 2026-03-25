package hook

import (
	"fmt"
	"time"

	"github.com/debian-composer/debian-composer-go/internal/service"
	"github.com/debian-composer/debian-composer-go/internal/types"
)

// RecipeRunner orchestrates recipe installation with full hook support
type RecipeRunner struct {
	executor    *Executor
	serviceMgr  *service.Manager
	dryRun      bool
	verbose     bool
	withSnapshot bool
	skipSSH     bool
	skipFirewall bool
	skipFail2ban bool
}

// RunnerConfig holds configuration for recipe runner
type RunnerConfig struct {
	DryRun       bool
	Verbose      bool
	WithSnapshot bool
	SkipSSH      bool
	SkipFirewall bool
	SkipFail2ban bool
	Kitchen      string
}

// NewRecipeRunner creates a new recipe runner
func NewRecipeRunner(config RunnerConfig, aptRunner APTRunner) *RecipeRunner {
	executor := NewExecutor(config.DryRun, config.Verbose, aptRunner)
	if config.Kitchen != "" {
		executor.SetKitchen(config.Kitchen)
	}

	return &RecipeRunner{
		executor:     executor,
		serviceMgr:   service.New(config.DryRun),
		dryRun:       config.DryRun,
		verbose:      config.Verbose,
		withSnapshot: config.WithSnapshot,
		skipSSH:      config.SkipSSH,
		skipFirewall: config.SkipFirewall,
		skipFail2ban: config.SkipFail2ban,
	}
}

// InstallResult contains the result of a recipe installation
type InstallResult struct {
	RecipeName    string
	Success       bool
	Phases        []PhaseResult
	TotalTime     time.Duration
	Error         error
	PackagesInstalled []string
}

// PhaseResult contains the result of a phase execution
type PhaseResult struct {
	Phase   Phase
	Summary *ExecutionSummary
}

// Run executes the full recipe installation lifecycle
func (r *RecipeRunner) Run(recipe *types.ResolvedRecipe, variables map[string]interface{}) *InstallResult {
	result := &InstallResult{
		RecipeName: recipe.Name,
		Success:    true,
		Phases:     []PhaseResult{},
	}

	start := time.Now()

	if r.verbose {
		fmt.Printf("\n[runner] Starting recipe: %s\n", recipe.Name)
	}

	// Phase 1: Pre-install hooks
	if len(recipe.Install.Pre) > 0 {
		if r.verbose {
			fmt.Println("[runner] Executing pre-install hooks...")
		}

		hooks := r.buildCommandHooks(recipe.Install.Pre)
		summary := r.executor.ExecuteHooks(HookSet{PreInstall: hooks}, PhaseInstall, recipe.Name, variables)
		result.Phases = append(result.Phases, PhaseResult{Phase: PhaseInstall, Summary: summary})

		if !summary.Success {
			result.Success = false
			result.Error = fmt.Errorf("pre-install hooks failed")
			return result
		}
	}

	// Phase 2: Main installation (packages) - handled by caller
	// Packages are installed by the recipe manager, not hooks

	// Phase 3: Post-install hooks
	if len(recipe.Install.Post) > 0 {
		if r.verbose {
			fmt.Println("[runner] Executing post-install hooks...")
		}

		hooks := r.buildCommandHooks(recipe.Install.Post)
		summary := r.executor.ExecuteHooks(HookSet{PostInstall: hooks}, PhaseInstall, recipe.Name, variables)
		result.Phases = append(result.Phases, PhaseResult{Phase: PhaseInstall, Summary: summary})

		if !summary.Success {
			result.Success = false
			result.Error = fmt.Errorf("post-install hooks failed")
			return result
		}
	}

	// Phase 4: Pre-configure hooks
	// (none in current schema, but reserved for future)

	// Phase 5: Configuration (user groups, services, commands)
	if len(recipe.Configure.Commands) > 0 || len(recipe.Configure.UserGroups) > 0 {
		if r.verbose {
			fmt.Println("[runner] Executing configuration...")
		}

		hooks := r.buildConfigureHooks(recipe)
		summary := r.executor.ExecuteHooks(HookSet{PreConfigure: hooks}, PhaseConfigure, recipe.Name, variables)
		result.Phases = append(result.Phases, PhaseResult{Phase: PhaseConfigure, Summary: summary})

		if !summary.Success {
			result.Success = false
			result.Error = fmt.Errorf("configuration failed")
			return result
		}
	}

	// Phase 6: Post-configure hooks
	// (handled in Configure.Commands)

	// Phase 7: Verification
	if len(recipe.Verify.Commands) > 0 {
		if r.verbose {
			fmt.Println("[runner] Executing verification...")
		}

		hooks := r.buildCommandHooks(recipe.Verify.Commands)
		summary := r.executor.ExecuteHooks(HookSet{PreVerify: hooks}, PhaseVerify, recipe.Name, variables)
		result.Phases = append(result.Phases, PhaseResult{Phase: PhaseVerify, Summary: summary})

		if !summary.Success {
			result.Success = false
			result.Error = fmt.Errorf("verification failed")
			// Don't return immediately, let user see what failed
		}
	}

	result.TotalTime = time.Since(start)
	return result
}

// Uninstall executes the recipe uninstallation lifecycle
func (r *RecipeRunner) Uninstall(recipe *types.ResolvedRecipe, variables map[string]interface{}) *InstallResult {
	result := &InstallResult{
		RecipeName: recipe.Name,
		Success:    true,
		Phases:     []PhaseResult{},
	}

	start := time.Now()

	if r.verbose {
		fmt.Printf("\n[runner] Uninstalling recipe: %s\n", recipe.Name)
	}

	// Pre-uninstall hooks would go here if defined
	// For now, we just remove packages

	result.TotalTime = time.Since(start)
	return result
}

// buildCommandHooks converts a list of command strings to Hook objects
func (r *RecipeRunner) buildCommandHooks(commands []string) []Hook {
	hooks := make([]Hook, 0, len(commands))
	for _, cmd := range commands {
		hooks = append(hooks, Hook{
			Type:    "command",
			Command: cmd,
			Timeout: 300, // 5 minutes default
		})
	}
	return hooks
}

// buildConfigureHooks builds hooks from recipe configuration
func (r *RecipeRunner) buildConfigureHooks(recipe *types.ResolvedRecipe) []Hook {
	var hooks []Hook

	// User group creation hooks
	for _, group := range recipe.Configure.UserGroups {
		hooks = append(hooks, Hook{
			Type:    "command",
			Command: fmt.Sprintf("getent group %s >/dev/null || groupadd %s", group, group),
			Timeout: 60,
		})
	}

	// Configuration command hooks
	for _, cmd := range recipe.Configure.Commands {
		hooks = append(hooks, Hook{
			Type:    "command",
			Command: cmd,
			Timeout: 300,
		})
	}

	return hooks
}

// SetEnvironment sets environment variables for hook execution
func (r *RecipeRunner) SetEnvironment(env map[string]string) {
	for k, v := range env {
		r.executor.SetEnv(k, v)
	}
}

// GetExecutor returns the underlying executor for advanced usage
func (r *RecipeRunner) GetExecutor() *Executor {
	return r.executor
}

// FormatInstallResult formats an install result for display
func FormatInstallResult(result *InstallResult) string {
	status := "✓ Success"
	if !result.Success {
		status = "✗ Failed"
	}

	output := fmt.Sprintf("\nRecipe: %s\nStatus: %s\nDuration: %v\n",
		result.RecipeName, status, result.TotalTime.Round(time.Millisecond))

	if len(result.Phases) > 0 {
		output += "\nPhases:\n"
		for _, phase := range result.Phases {
			phaseStatus := "✓"
			if !phase.Summary.Success {
				phaseStatus = "✗"
			}
			output += fmt.Sprintf("  %s %s (%v)\n",
				phaseStatus, phase.Phase, phase.Summary.TotalTime.Round(time.Millisecond))
		}
	}

	if result.Error != nil {
		output += fmt.Sprintf("\nError: %v\n", result.Error)
	}

	return output
}

// SecurityHardeningRunner provides security hardening execution
type SecurityHardeningRunner struct {
	executor *Executor
}

// NewSecurityHardeningRunner creates a new security hardening runner
func NewSecurityHardeningRunner(dryRun, verbose bool, aptRunner APTRunner) *SecurityHardeningRunner {
	return &SecurityHardeningRunner{
		executor: NewExecutor(dryRun, verbose, aptRunner),
	}
}

// RunSSHHardening executes SSH hardening hooks
func (r *SecurityHardeningRunner) RunSSHHardening(variables map[string]interface{}) *ExecutionSummary {
	hooks := HookSet{
		PreInstall: []Hook{
			{
				Type:    "apt",
				Action:  "install",
				Package: "openssh-server",
			},
		},
		PostInstall: []Hook{
			{
				Type:    "command",
				Command: "systemctl enable ssh",
				Timeout: 30,
			},
			{
				Type:    "command",
				Command: "systemctl restart ssh",
				Timeout: 30,
			},
		},
	}

	return r.executor.ExecuteHooks(hooks, PhaseInstall, "ssh-hardening", variables)
}

// RunFirewallHardening executes firewall hardening hooks
func (r *SecurityHardeningRunner) RunFirewallHardening(variables map[string]interface{}) *ExecutionSummary {
	hooks := HookSet{
		PreInstall: []Hook{
			{
				Type:    "apt",
				Action:  "install",
				Package: "ufw",
			},
		},
		PostInstall: []Hook{
			{
				Type:    "command",
				Command: "ufw default deny incoming",
				Timeout: 30,
			},
			{
				Type:    "command",
				Command: "ufw default allow outgoing",
				Timeout: 30,
			},
			{
				Type:    "command",
				Command: "ufw allow ssh",
				Timeout: 30,
			},
		},
	}

	return r.executor.ExecuteHooks(hooks, PhaseInstall, "firewall-hardening", variables)
}

// RunFail2banHardening executes fail2ban hardening hooks
func (r *SecurityHardeningRunner) RunFail2banHardening(variables map[string]interface{}) *ExecutionSummary {
	hooks := HookSet{
		PreInstall: []Hook{
			{
				Type:    "apt",
				Action:  "install",
				Package: "fail2ban",
			},
		},
		PostInstall: []Hook{
			{
				Type:    "command",
				Command: "systemctl enable fail2ban",
				Timeout: 30,
			},
			{
				Type:    "command",
				Command: "systemctl restart fail2ban",
				Timeout: 30,
			},
		},
	}

	return r.executor.ExecuteHooks(hooks, PhaseInstall, "fail2ban-hardening", variables)
}

// RunFullSecurityHardening executes all security hardening
func (r *SecurityHardeningRunner) RunFullSecurityHardening(variables map[string]interface{}) (bool, error) {
	allSuccess := true

	// SSH Hardening
	summary := r.RunSSHHardening(variables)
	if !summary.Success {
		allSuccess = false
	}

	// Firewall Hardening
	summary = r.RunFirewallHardening(variables)
	if !summary.Success {
		allSuccess = false
	}

	// Fail2ban Hardening
	summary = r.RunFail2banHardening(variables)
	if !summary.Success {
		allSuccess = false
	}

	if !allSuccess {
		return false, fmt.Errorf("one or more security hardening steps failed")
	}

	return true, nil
}
