package cli

import (
	"fmt"
	"time"

	"github.com/debian-composer/debian-composer-go/internal/recipe"
	"github.com/debian-composer/debian-composer-go/internal/state"
	"github.com/debian-composer/debian-composer-go/internal/types"
	"github.com/debian-composer/debian-composer-go/internal/ux"
	"github.com/spf13/cobra"
)

func init() {
	// Persona command - installs a recipe with kind="persona"
	personaCmd := &cobra.Command{
		Use:   "persona [name]",
		Short: "Install packages for a user persona",
		Long: `Install packages tailored for a specific user type (kind: "persona").

Personas are recipes that provide a quick way to set up a system for
a particular role. They can be combined with mixins for additional
specialization.

Examples:
  debian-composer persona developer
  debian-composer persona student --mixin=ai-ml
  debian-composer persona creative --mixin=video-editing`,
		Args: cobra.ExactArgs(1),
		RunE: runPersona,
	}
	personaCmd.Flags().StringSlice("var", nil, "Set variable (key=value)")
	personaCmd.Flags().StringSlice("mixin", nil, "Add mixin for specialization")
	personaCmd.Flags().String("stack", "", "Use pre-configured stack")
	personaCmd.Flags().Bool("no-deps", false, "Skip dependency resolution")
	rootCmd.AddCommand(personaCmd)

	// Mixin command - installs a recipe with kind="mixin"
	mixinCmd := &cobra.Command{
		Use:   "mixin [name]",
		Short: "Add a mixin to current setup",
		Long: `Add a mixin for additional specialization (kind: "mixin").

Mixins are overlay recipes that add specialized packages to an existing setup.

Examples:
  debian-composer mixin ai-ml
  debian-composer mixin devops
  debian-composer mixin security`,
		Args: cobra.ExactArgs(1),
		RunE: runMixin,
	}
	mixinCmd.Flags().StringSlice("var", nil, "Set variable (key=value)")
	mixinCmd.Flags().Bool("no-deps", false, "Skip dependency resolution")
	rootCmd.AddCommand(mixinCmd)

	// List personas command
	personasCmd := &cobra.Command{
		Use:   "personas",
		Short: "List available personas",
		RunE:  runListPersonas,
	}
	rootCmd.AddCommand(personasCmd)

	// List mixins command
	mixinsCmd := &cobra.Command{
		Use:   "mixins",
		Short: "List available mixins",
		RunE:  runListMixins,
	}
	rootCmd.AddCommand(mixinsCmd)
}

func runPersona(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	name := args[0]
	vars, _ := cmd.Flags().GetStringSlice("var")
	mixins, _ := cmd.Flags().GetStringSlice("mixin")
	stack, _ := cmd.Flags().GetString("stack")
	noDeps, _ := cmd.Flags().GetBool("no-deps")
	varMap := parseVars(vars)

	fmt.Printf("Installing persona: %s\n", name)
	if len(mixins) > 0 {
		fmt.Printf("With mixins: %v\n", mixins)
	}
	if dryRun {
		fmt.Println("[DRY-RUN] No changes will be made")
	}

	start := time.Now()
	resolver := getResolver()
	apt := getAPT()

	// Check for conflicts with already installed recipes
	if !dryRun {
		st, err := getState()
		if err == nil {
			defer st.Close()
			parser := getParser()
			recipe, err := parser.Parse(name)
			if err == nil {
				for _, conflict := range recipe.Conflicts {
					if st.IsRecipeInstalled(conflict) {
						return fmt.Errorf("conflict: persona %s conflicts with installed recipe %s", name, conflict)
					}
				}
			}
		}
	}

	// Resolve persona(s) with dependencies
	var resolvedRecipes []*types.ResolvedRecipe
	if noDeps {
		// Single recipe resolution
		resolved, err := resolver.Resolve(name, varMap)
		if err != nil {
			return fmt.Errorf("resolve persona: %w", err)
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
		var err error
		var st *state.Manager
		if !dryRun {
			st, err = getState()
			if err != nil {
				return err
			}
			defer st.Close()
		}
		var stateGetter recipe.StateGetter
		if st != nil {
			stateGetter = st
		}
		depResolver := recipe.NewDependencyResolver(getParser(), stateGetter)
		resolvedRecipes, err = depResolver.ResolveAll(name, varMap)
		if err != nil {
			return fmt.Errorf("resolve dependencies: %w", err)
		}
		// Apply stack only to the main persona (last in order)
		if stack != "" {
			mainIdx := len(resolvedRecipes) - 1
			mainResolved, err := resolver.ResolveStack(name, stack, varMap)
			if err != nil {
				return fmt.Errorf("resolve stack for main persona: %w", err)
			}
			resolvedRecipes[mainIdx] = mainResolved
		}
	}

	// Apply mixins (additional packages) with dependency resolution
	mixinPackages := []string{}
	for _, mixin := range mixins {
		// Resolve mixin with dependencies
		var mixinResolvedRecipes []*types.ResolvedRecipe
		var err error
		if noDeps {
			// Single recipe resolution
			resolved, err := resolver.Resolve(mixin, varMap)
			if err != nil {
				fmt.Printf("Warning: could not resolve mixin '%s': %v\n", mixin, err)
				continue
			}
			mixinResolvedRecipes = []*types.ResolvedRecipe{resolved}
		} else {
			// Dependency-aware resolution
			var st *state.Manager
			if !dryRun {
				st, err = getState()
				if err != nil {
					fmt.Printf("Warning: could not get state for mixin '%s': %v\n", mixin, err)
					continue
				}
				defer st.Close()
			}
			var stateGetter recipe.StateGetter
			if st != nil {
				stateGetter = st
			}
			depResolver := recipe.NewDependencyResolver(getParser(), stateGetter)
			mixinResolvedRecipes, err = depResolver.ResolveAll(mixin, varMap)
			if err != nil {
				fmt.Printf("Warning: could not resolve mixin dependencies '%s': %v\n", mixin, err)
				continue
			}
		}
		// Collect packages from all resolved recipes (including dependencies)
		for _, rec := range mixinResolvedRecipes {
			// Skip if already installed (if not dry-run and state)
			if !dryRun {
				st, err := getState()
				if err == nil {
					defer st.Close()
					if st.IsRecipeInstalled(rec.Name) {
						fmt.Printf("Skipping already installed mixin dependency: %s\n", rec.Name)
						continue
					}
				}
			}
			mixinPackages = append(mixinPackages, rec.GetAllPackages()...)
		}
	}

	// Create snapshot before installation if requested
	if err := createSnapshotIfRequested(name); err != nil {
		fmt.Printf("Warning: snapshot failed: %v\n", err)
		fmt.Println("Continuing with installation...")
	}

	// Install each recipe in order
	for _, rec := range resolvedRecipes {
		// Skip already installed recipes (if not dry-run and state)
		if !dryRun {
			st, err := getState()
			if err == nil {
				defer st.Close()
				if st.IsRecipeInstalled(rec.Name) {
					fmt.Printf("Skipping already installed recipe: %s\n", rec.Name)
					continue
				}
			}
		}

		recPackages := rec.GetAllPackages()
		fmt.Printf("\nInstalling recipe '%s' (%d packages)...\n", rec.Name, len(recPackages))

		if dryRun {
			for i, pkg := range recPackages {
				if i < 10 {
					fmt.Printf("  - %s\n", pkg)
				}
			}
			if len(recPackages) > 10 {
				fmt.Printf("  ... and %d more\n", len(recPackages)-10)
			}
			continue
		}

		// Batch install packages for this recipe
		if err := apt.BatchInstall(recPackages); err != nil {
			return fmt.Errorf("package installation failed for %s: %w", rec.Name, err)
		}

		// Save state for this recipe
		st, err := getState()
		if err == nil {
			defer st.Close()
			st.SaveRecipe(rec)
		}
	}

	// Install mixin packages (without hooks)
	if len(mixinPackages) > 0 {
		fmt.Printf("\nInstalling mixin packages (%d packages)...\n", len(mixinPackages))
		if !dryRun {
			if err := apt.BatchInstall(mixinPackages); err != nil {
				return fmt.Errorf("mixin package installation failed: %w", err)
			}
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("Persona '%s' installed in %v\n", name, elapsed.Round(time.Millisecond))

	return nil
}

func runMixin(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	name := args[0]
	vars, _ := cmd.Flags().GetStringSlice("var")
	varMap := parseVars(vars)

	fmt.Printf("Installing mixin: %s\n", name)
	if dryRun {
		fmt.Println("[DRY-RUN] No changes will be made")
	}

	resolver := getResolver()
	apt := getAPT()
	noDeps, _ := cmd.Flags().GetBool("no-deps")

	// Resolve mixin(s) with dependencies
	var resolvedRecipes []*types.ResolvedRecipe
	if noDeps {
		// Single recipe resolution
		resolved, err := resolver.Resolve(name, varMap)
		if err != nil {
			return fmt.Errorf("resolve mixin: %w", err)
		}
		resolvedRecipes = []*types.ResolvedRecipe{resolved}
	} else {
		// Dependency-aware resolution
		var st *state.Manager
		var err error
		if !dryRun {
			st, err = getState()
			if err != nil {
				return err
			}
			defer st.Close()
		}
		var stateGetter recipe.StateGetter
		if st != nil {
			stateGetter = st
		}
		depResolver := recipe.NewDependencyResolver(getParser(), stateGetter)
		resolvedRecipes, err = depResolver.ResolveAll(name, varMap)
		if err != nil {
			return fmt.Errorf("resolve dependencies: %w", err)
		}
	}

	// Collect packages from all resolved recipes (including dependencies)
	var allPackages []string
	for _, rec := range resolvedRecipes {
		// Skip already installed recipes (if not dry-run and state)
		if !dryRun {
			st, err := getState()
			if err == nil {
				defer st.Close()
				if st.IsRecipeInstalled(rec.Name) {
					fmt.Printf("Skipping already installed recipe: %s\n", rec.Name)
					continue
				}
			}
		}
		allPackages = append(allPackages, rec.GetAllPackages()...)
	}

	fmt.Printf("Packages to install: %d\n", len(allPackages))

	if dryRun {
		for i, pkg := range allPackages {
			if i < 15 {
				fmt.Printf("  - %s\n", pkg)
			}
		}
		if len(allPackages) > 15 {
			fmt.Printf("  ... and %d more\n", len(allPackages)-15)
		}
	} else {
		fmt.Printf("Installing mixin packages...\n")
		if err := apt.BatchInstall(allPackages); err != nil {
			return fmt.Errorf("mixin package installation failed: %w", err)
		}
		// Save state for each recipe
		st, err := getState()
		if err == nil {
			defer st.Close()
			for _, rec := range resolvedRecipes {
				st.SaveRecipe(rec)
			}
		}
	}

	return nil
}

func runListPersonas(cmd *cobra.Command, args []string) error {
	return listByKind("persona", "Personas")
}

func runListMixins(cmd *cobra.Command, args []string) error {
	return listByKind("mixin", "Mixins")
}

func listByKind(kind, title string) error {
	parser := getParser()
	names, err := parser.List()
	if err != nil {
		return err
	}

	var items []struct {
		Name        string
		Description string
	}

	for _, name := range names {
		r, err := parser.Parse(name)
		if err != nil {
			continue
		}
		if r.Kind == kind {
			items = append(items, struct {
				Name        string
				Description string
			}{
				Name:        r.Name,
				Description: r.Description,
			})
		}
	}

	if len(items) == 0 {
		fmt.Printf("No %s found.\n", title)
		return nil
	}

	fmt.Printf("%s (%d):\n\n", title, len(items))
	headers := []string{"Name", "Description"}
	rows := make([][]string, len(items))
	for i, item := range items {
		rows[i] = []string{item.Name, truncate(item.Description, 60)}
	}
	ux.PrintTable(headers, rows)

	return nil
}
