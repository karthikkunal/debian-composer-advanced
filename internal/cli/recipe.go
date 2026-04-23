package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/debian-composer/debian-composer-go/internal/hook"
	"github.com/debian-composer/debian-composer-go/internal/recipe"
	"github.com/debian-composer/debian-composer-go/internal/state"
	"github.com/debian-composer/debian-composer-go/internal/types"
	"github.com/spf13/cobra"
)

func init() {
	// Install command
	installCmd := &cobra.Command{
		Use:   "install [recipe]",
		Short: "Install a recipe",
		Args:  cobra.ExactArgs(1),
		RunE:  runInstall,
	}
	installCmd.Flags().StringSlice("category", nil, "Install specific categories only")
	installCmd.Flags().StringSlice("var", nil, "Set variable (key=value)")
	installCmd.Flags().String("stack", "", "Use pre-configured stack")
	installCmd.Flags().String("phase", "all", "Execution phase: install, configure, verify, or all")
	installCmd.Flags().Bool("with-snapshot", false, "Create Timeshift snapshot before installation")
	installCmd.Flags().Bool("skip-ssh", false, "Skip SSH hardening")
	installCmd.Flags().Bool("skip-firewall", false, "Skip firewall setup")
	installCmd.Flags().Bool("skip-fail2ban", false, "Skip fail2ban configuration")
	installCmd.Flags().Bool("apply-security", false, "Apply security hardening (SSH, UFW, fail2ban)")
	installCmd.Flags().Bool("no-deps", false, "Skip dependency resolution")
	rootCmd.AddCommand(installCmd)

	// Remove command
	removeCmd := &cobra.Command{
		Use:   "remove [recipe]",
		Short: "Remove an installed recipe",
		Args:  cobra.ExactArgs(1),
		RunE:  runRemove,
	}
	removeCmd.Flags().Bool("keep-config", false, "Keep configuration files")
	rootCmd.AddCommand(removeCmd)

	// Switch command (remove + install)
	switchCmd := &cobra.Command{
		Use:   "switch [recipe]",
		Short: "Switch to a different recipe",
		Args:  cobra.ExactArgs(1),
		RunE:  runSwitch,
	}
	switchCmd.Flags().StringSlice("var", nil, "Set variable (key=value)")
	switchCmd.Flags().String("stack", "", "Use pre-configured stack")
	rootCmd.AddCommand(switchCmd)

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed recipes",
		RunE:  runList,
	}
	rootCmd.AddCommand(listCmd)

	// Available command
	availableCmd := &cobra.Command{
		Use:   "available",
		Short: "List available recipes",
		RunE:  runAvailable,
	}
	rootCmd.AddCommand(availableCmd)

	// Info command
	infoCmd := &cobra.Command{
		Use:   "info [recipe]",
		Short: "Show recipe details",
		Args:  cobra.ExactArgs(1),
		RunE:  runInfo,
	}
	rootCmd.AddCommand(infoCmd)

	// Import command
	importCmd := &cobra.Command{
		Use:   "import [path]",
		Short: "Import a recipe from a YAML file",
		Args:  cobra.ExactArgs(1),
		RunE:  runImport,
	}
	importCmd.Flags().String("name", "", "Name for the imported recipe (defaults to filename)")
	rootCmd.AddCommand(importCmd)

	// Export command
	exportCmd := &cobra.Command{
		Use:   "export [recipe] [output-path]",
		Short: "Export a recipe to a YAML file",
		Args:  cobra.ExactArgs(2),
		RunE:  runExport,
	}
	exportCmd.Flags().Bool("standalone", false, "Bundle all includes and anchors into a single file")
	rootCmd.AddCommand(exportCmd)

	// Validate command
	validateCmd := &cobra.Command{
		Use:   "validate [recipe/path]",
		Short: "Validate recipe against JSON schema",
		Args:  cobra.RangeArgs(0, 1),
		RunE:  runValidate,
	}
	validateCmd.Flags().Bool("all", false, "Validate all recipes in kitchen")
	rootCmd.AddCommand(validateCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	name := args[0]
	stack, _ := cmd.Flags().GetString("stack")
	vars, _ := cmd.Flags().GetStringSlice("var")
	varMap := parseVars(vars)
	phase, _ := cmd.Flags().GetString("phase")
	withSnapshot, _ := cmd.Flags().GetBool("with-snapshot")
	skipSSH, _ := cmd.Flags().GetBool("skip-ssh")
	skipFirewall, _ := cmd.Flags().GetBool("skip-firewall")
	skipFail2ban, _ := cmd.Flags().GetBool("skip-fail2ban")
	applySecurity, _ := cmd.Flags().GetBool("apply-security")
	noDeps, _ := cmd.Flags().GetBool("no-deps")

	fmt.Printf("Installing recipe: %s\n", name)
	if dryRun {
		fmt.Println("[DRY-RUN] No changes will be made")
	}
	start := time.Now()

	resolver := getResolver()
	apt := getAPT()

	// Skip state in dry-run
	var st *state.Manager
	var err error
	if !dryRun {
		st, err = getState()
		if err != nil {
			return err
		}
		defer st.Close()
	}

	// Check for conflicts with already installed recipes
	if !dryRun && st != nil {
		parser := getParser()
		recipe, err := parser.Parse(name)
		if err == nil {
			for _, conflict := range recipe.Conflicts {
				if st.IsRecipeInstalled(conflict) {
					return fmt.Errorf("conflict: recipe %s conflicts with installed recipe %s", name, conflict)
				}
			}
		}
	}

	// Resolve recipe(s)
	var resolvedRecipes []*types.ResolvedRecipe
	if noDeps {
		// Single recipe resolution
		resolved, err := resolver.Resolve(name, varMap)
		if err != nil {
			return fmt.Errorf("resolve recipe: %w", err)
		}
		if stack != "" {
			resolved, err = resolver.ResolveStack(name, stack, varMap)
			if err != nil {
				return fmt.Errorf("resolve stack: %w", err)
			}
		}
		resolvedRecipes = []*types.ResolvedRecipe{resolved}
	} else {
		// Dependency-aware resolution
		var stateGetter recipe.StateGetter
		if st != nil {
			stateGetter = st
		}
		depResolver := recipe.NewDependencyResolver(getParser(), stateGetter)
		resolvedRecipes, err = depResolver.ResolveAll(name, varMap)
		if err != nil {
			return fmt.Errorf("resolve dependencies: %w", err)
		}
		// Apply stack only to the main recipe (last in order)
		if stack != "" {
			mainIdx := len(resolvedRecipes) - 1
			mainResolved, err := resolver.ResolveStack(name, stack, varMap)
			if err != nil {
				return fmt.Errorf("resolve stack for main recipe: %w", err)
			}
			resolvedRecipes[mainIdx] = mainResolved
		}
	}

	// Create snapshot before installation if requested (only for main recipe)
	if withSnapshot {
		if err := createSnapshotIfRequested(name); err != nil {
			fmt.Printf("Warning: snapshot failed: %v\n", err)
			fmt.Println("Continuing with installation...")
		}
	}

	// Apply security hardening if requested (once)
	if applySecurity && !dryRun {
		fmt.Println("\nApplying security hardening...")
		securityRunner := hook.NewSecurityHardeningRunner(dryRun, verbose, apt)

		success, err := securityRunner.RunFullSecurityHardening(nil)
		if err != nil {
			fmt.Printf("Warning: security hardening encountered issues: %v\n", err)
		}
		if !success {
			fmt.Println("Security hardening completed with some issues")
		} else {
			fmt.Println("Security hardening completed successfully")
		}
	}

// Validate phase
	validPhases := map[string]bool{
		"install":   true,
		"configure": true,
		"verify":    true,
		"all":       true,
	}
	if !validPhases[phase] {
		return fmt.Errorf("invalid phase: %s (must be: install, configure, verify, or all)", phase)
	}

	// Execute phases based on flag
	installPhase := (phase == "install" || phase == "all")
	configurePhase := (phase == "configure" || phase == "all")
	verifyPhase := (phase == "verify" || phase == "all")

	// Install each recipe in order
	for idx, rec := range resolvedRecipes {
		isMain := (idx == len(resolvedRecipes)-1)
		// Skip already installed recipes (if not dry-run and state)
		if !dryRun && st != nil && st.IsRecipeInstalled(rec.Name) {
			fmt.Printf("Skipping already installed recipe: %s\n", rec.Name)
			continue
		}

		packages := rec.GetAllPackages()
		fmt.Printf("\nInstalling recipe '%s' (%d packages)...\n", rec.Name, len(packages))

		if dryRun {
			// Show packages for dry-run
			for i, pkg := range packages {
				if i < 10 {
					fmt.Printf("  - %s\n", pkg)
				}
			}
			if len(packages) > 10 {
				fmt.Printf("  ... and %d more\n", len(packages)-10)
			}
			if !(installPhase || configurePhase || verifyPhase) {
				continue
			}
		}

		// Create hook runner for this recipe
		runnerConfig := hook.RunnerConfig{
			DryRun:       dryRun,
			Verbose:      verbose,
			WithSnapshot: isMain && withSnapshot, // snapshot already taken for main recipe
			SkipSSH:      skipSSH,
			SkipFirewall: skipFirewall,
			SkipFail2ban: skipFail2ban,
			Kitchen:      kitchenPath,
		}
		runner := hook.NewRecipeRunner(runnerConfig, apt)

		// Execute pre-install hooks (only in install phase)
		if installPhase && len(rec.Install.Pre) > 0 {
			fmt.Println("\nExecuting pre-install hooks...")
			hooks := runner.GetExecutor()
			hookSet := hook.HookSet{
				PreInstall: make([]hook.Hook, 0, len(rec.Install.Pre)),
			}
			for _, cmdStr := range rec.Install.Pre {
				hookSet.PreInstall = append(hookSet.PreInstall, hook.Hook{
					Type:    "command",
					Command: cmdStr,
					Timeout: 300,
				})
			}
			summary := hooks.ExecuteHooks(hookSet, hook.PhaseInstall, rec.Name, interfaceMapToInterface(varMap))
			if !summary.Success {
				return fmt.Errorf("pre-install hooks failed for %s: %v", rec.Name, summary.Errors)
			}
		}

		// Batch install packages for this recipe (only in install phase)
		if installPhase {
			fmt.Printf("\nInstalling packages...\n")
			if err := apt.BatchInstall(packages); err != nil {
				return fmt.Errorf("package installation failed for %s: %w", rec.Name, err)
			}
		}

		// Execute post-install hooks (only in install phase)
		if installPhase && len(rec.Install.Post) > 0 {
			fmt.Println("\nExecuting post-install hooks...")
			hooks := runner.GetExecutor()
			hookSet := hook.HookSet{
				PostInstall: make([]hook.Hook, 0, len(rec.Install.Post)),
			}
			for _, cmdStr := range rec.Install.Post {
				hookSet.PostInstall = append(hookSet.PostInstall, hook.Hook{
					Type:    "command",
					Command: cmdStr,
					Timeout: 300,
				})
			}
			summary := hooks.ExecuteHooks(hookSet, hook.PhaseInstall, rec.Name, interfaceMapToInterface(varMap))
			if !summary.Success {
				fmt.Printf("Warning: post-install hooks had issues: %v\n", summary.Errors)
			}
		}

		// Execute configuration hooks (only in configure phase)
		if configurePhase && (len(rec.Configure.Commands) > 0 || len(rec.Configure.UserGroups) > 0) {
			fmt.Println("\nExecuting configuration...")
			hooks := runner.GetExecutor()
			hookSet := hook.HookSet{}
			// User group creation
			for _, group := range rec.Configure.UserGroups {
				hookSet.PreConfigure = append(hookSet.PreConfigure, hook.Hook{
					Type:    "command",
					Command: fmt.Sprintf("getent group %s >/dev/null || groupadd %s", group, group),
					Timeout: 60,
				})
			}
			// Configuration commands
			for _, cmdStr := range rec.Configure.Commands {
				hookSet.PreConfigure = append(hookSet.PreConfigure, hook.Hook{
					Type:    "command",
					Command: cmdStr,
					Timeout: 300,
				})
			}
			summary := hooks.ExecuteHooks(hookSet, hook.PhaseConfigure, rec.Name, interfaceMapToInterface(varMap))
			if !summary.Success {
				fmt.Printf("Warning: configuration had issues: %v\n", summary.Errors)
			}
		}

		// Execute verification hooks (only in verify phase)
		if verifyPhase && len(rec.Verify.Commands) > 0 {
			fmt.Println("\nExecuting verification...")
			hooks := runner.GetExecutor()
			hookSet := hook.HookSet{
				PreVerify: make([]hook.Hook, 0, len(rec.Verify.Commands)),
			}
			for _, cmdStr := range rec.Verify.Commands {
				hookSet.PreVerify = append(hookSet.PreVerify, hook.Hook{
					Type:    "command",
					Command: cmdStr,
					Timeout: 60,
				})
			}
			summary := hooks.ExecuteHooks(hookSet, hook.PhaseVerify, rec.Name, interfaceMapToInterface(varMap))
			if !summary.Success {
				fmt.Printf("Warning: verification failed: %v\n", summary.Errors)
			}
		}

// Save state for this recipe
		if st != nil {
			st.SaveRecipe(rec)
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("\nRecipe '%s' installed in %v\n", name, elapsed.Round(time.Millisecond))

	return nil
}

func runRemove(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	name := args[0]
	keepConfig, _ := cmd.Flags().GetBool("keep-config")

	fmt.Printf("Removing recipe: %s\n", name)
	if dryRun {
		fmt.Println("[DRY-RUN] No changes will be made")
	}
	start := time.Now()

	st, err := getState()
	if err != nil {
		return err
	}
	defer st.Close()

	// Get installed recipe
	installed, err := st.GetRecipe(name)
	if err != nil {
		return fmt.Errorf("recipe not installed: %s", name)
	}

	fmt.Printf("Packages to remove: %d\n", len(installed.Packages))
	for i, pkg := range installed.Packages {
		if i < 10 {
			fmt.Printf("  - %s\n", pkg)
		}
	}
	if len(installed.Packages) > 10 {
		fmt.Printf("  ... and %d more\n", len(installed.Packages)-10)
	}

	if !dryRun {
		// Remove packages
		apt := getAPT()
		if len(installed.Packages) > 0 {
			if keepConfig {
				fmt.Printf("Removing packages (keeping configs)...\n")
				apt.PurgeKeepConfig(installed.Packages...)
			} else {
				fmt.Printf("Removing packages...\n")
				apt.Remove(installed.Packages...)
			}
		}
		// Remove state
		st.DeleteRecipe(name)
	}

	elapsed := time.Since(start)
	fmt.Printf("Recipe '%s' removed in %v\n", name, elapsed.Round(time.Millisecond))

	return nil
}

func runSwitch(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	name := args[0]
	vars, _ := cmd.Flags().GetStringSlice("var")
	stack, _ := cmd.Flags().GetString("stack")

	fmt.Printf("Switching to recipe: %s\n", name)
	if dryRun {
		fmt.Println("[DRY-RUN] No changes will be made")
	}
	start := time.Now()

	// Skip state in dry-run
	var st *state.Manager
	var err error
	if !dryRun {
		st, err = getState()
		if err != nil {
			return err
		}
		defer st.Close()
	}

	resolver := getResolver()
	apt := getAPT()
	varMap := parseVars(vars)

	// Get current recipe for comparison (skip in dry-run)
	if !dryRun && st != nil {
		recipes, _ := st.ListRecipes()
		if len(recipes) > 0 {
			current := recipes[0]
			fmt.Printf("Will remove current recipe: %s (%d packages)\n", current.Name, len(current.Packages))
			apt.Remove(current.Packages...)
			st.DeleteRecipe(current.Name)
		}
	}

	// Resolve new recipe
	resolved, err := resolver.Resolve(name, varMap)
	if err != nil {
		return err
	}

	if stack != "" {
		resolved, err = resolver.ResolveStack(name, stack, varMap)
		if err != nil {
			return err
		}
	}

	packages := resolved.GetAllPackages()
	fmt.Printf("Will install new recipe: %s (%d packages)\n", name, len(packages))

	if !dryRun {
		apt.BatchInstall(packages)
		st.SaveRecipe(resolved)
	}

	elapsed := time.Since(start)
	fmt.Printf("Switched to '%s' in %v\n", name, elapsed.Round(time.Millisecond))

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	st, err := getState()
	if err != nil {
		return err
	}
	defer st.Close()

	recipes, err := st.ListRecipes()
	if err != nil {
		return err
	}

	if len(recipes) == 0 {
		fmt.Println("No recipes installed")
		return nil
	}

	fmt.Println("Installed recipes:")
	for _, r := range recipes {
		fmt.Printf("  %s (%s) - installed %s\n", r.Name, r.Version, r.Installed.Format("2006-01-02"))
	}

	return nil
}

func runAvailable(cmd *cobra.Command, args []string) error {
	parser := getParser()
	names, err := parser.List()
	if err != nil {
		return err
	}

	fmt.Println("Available recipes:")
	for _, name := range names {
		fmt.Printf("  - %s\n", name)
	}

	return nil
}

func runInfo(cmd *cobra.Command, args []string) error {
	parser := getParser()
	recipe, err := parser.Parse(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Name: %s\n", recipe.Name)
	fmt.Printf("Version: %s\n", recipe.Version)
	fmt.Printf("Description: %s\n", recipe.Description)
	fmt.Printf("Kind: %s\n", recipe.Kind)
	fmt.Printf("Packages: %d\n", len(recipe.Packages))
	fmt.Printf("Categories: %d\n", len(recipe.Categories))
	fmt.Printf("Stacks: %d\n", len(recipe.Stacks))

	if len(recipe.Variables) > 0 {
		fmt.Println("\nVariables:")
		for name, v := range recipe.Variables {
			fmt.Printf("  %s (default: %s) - %s\n", name, v.Default, v.Description)
		}
	}

	return nil
}

func runImport(cmd *cobra.Command, args []string) error {
	path := args[0]
	name, _ := cmd.Flags().GetString("name")

	mgr := getManager()
	if err := mgr.Import(path, name); err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), ".yaml")
	}
	fmt.Printf("✓ Recipe '%s' imported successfully\n", name)
	return nil
}

func runExport(cmd *cobra.Command, args []string) error {
	name := args[0]
	target := args[1]
	standalone, _ := cmd.Flags().GetBool("standalone")

	mgr := getManager()
	if err := mgr.Export(name, target, standalone); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	fmt.Printf("✓ Recipe '%s' exported to %s\n", name, target)
	if standalone {
		fmt.Println("  (Exported as standalone file with bundled dependencies)")
	}
	return nil
}

func parseVars(vars []string) map[string]string {
	result := make(map[string]string)
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// interfaceMapToInterface converts map[string]string to map[string]interface{}
func interfaceMapToInterface(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

// kitchenPath returns the kitchen path from the global variable
var kitchenPath string

func init() {
	// This will be set by the root command initialization
	kitchenPath = kitchen
}
