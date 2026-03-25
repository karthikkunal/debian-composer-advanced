package cli

import (
	"fmt"
	"time"

	"github.com/debian-composer/debian-composer-go/internal/recipe"
	"github.com/debian-composer/debian-composer-go/internal/state"
	"github.com/debian-composer/debian-composer-go/internal/tui"
	"github.com/debian-composer/debian-composer-go/internal/types"
	"github.com/spf13/cobra"
)

func init() {
	// Blend command - just installs a recipe with kind="pure blend"
	blendCmd := &cobra.Command{
		Use:   "blend [name]",
		Short: "Install a Debian Pure Blend",
		Long: `Install a Debian Pure Blend recipe.

Debian Pure Blends (kind: "pure blend") are pre-configured recipe
collections curated for specific use cases like education, science,
or GIS work.

This is equivalent to:
  debian-composer install <name>

Examples:
  debian-composer blend debian-edu
  debian-composer blend debian-science --var=education_level=university`,
		Args: cobra.ExactArgs(1),
		RunE: runBlend,
	}
	blendCmd.Flags().StringSlice("var", nil, "Set variable (key=value)")
	blendCmd.Flags().String("stack", "", "Use pre-configured stack")
	blendCmd.Flags().Bool("list-tasks", false, "List available stacks/categories")
	blendCmd.Flags().BoolP("interactive", "i", false, "Interactive task selection (TUI)")
	blendCmd.Flags().StringSlice("categories", nil, "Filter by specific categories (comma-separated)")
	blendCmd.Flags().Bool("list-categories", false, "List all available categories")
	blendCmd.Flags().Bool("no-deps", false, "Skip dependency resolution")
	rootCmd.AddCommand(blendCmd)

	// List blends command
	blendsCmd := &cobra.Command{
		Use:   "blends",
		Short: "List available Debian Pure Blends",
		RunE:  runListBlends,
	}
	rootCmd.AddCommand(blendsCmd)
}

func runBlend(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	name := args[0]
	vars, _ := cmd.Flags().GetStringSlice("var")
	stack, _ := cmd.Flags().GetString("stack")
	listTasks, _ := cmd.Flags().GetBool("list-tasks")
	listCats, _ := cmd.Flags().GetBool("list-categories")
	interactive, _ := cmd.Flags().GetBool("interactive")
	categories, _ := cmd.Flags().GetStringSlice("categories")
	_ = categories // TODO: use to filter categories during installation
	varMap := parseVars(vars)

	resolver := getResolver()
	apt := getAPT()

	// Parse the recipe to verify it's a blend
	parser := getParser()
	r, err := parser.Parse(name)
	if err != nil {
		return fmt.Errorf("blend not found: %w", err)
	}

	// Verify it's a pure blend
	if r.Kind != "pure blend" && r.Kind != "blend" {
		fmt.Printf("Warning: '%s' has kind=%q, not 'pure blend'\n", name, r.Kind)
		fmt.Println("Continuing anyway...")
	}

	// List tasks/categories mode
	if listTasks {
		return printBlendTasks(r)
	}

	// List categories mode
	if listCats {
		return printBlendCategories(r)
	}

	// Interactive selection mode
	if interactive && !dryRun && stack == "" {
		selected := selectTasksInteractive(r)
		if selected != "" {
			stack = selected
			fmt.Printf("Selected: %s\n", stack)
		}
	}

	fmt.Printf("Debian Pure Blend: %s\n", r.Name)
	fmt.Printf("Description: %s\n", r.Description)
	if len(r.Tags) > 0 {
		fmt.Printf("Tags: %v\n", r.Tags)
	}
	fmt.Println()

	if dryRun {
		fmt.Println("[DRY-RUN] No changes will be made")
	}

	start := time.Now()

	// Resolve recipe(s) with dependencies
	noDeps, _ := cmd.Flags().GetBool("no-deps")
	var resolvedRecipes []*types.ResolvedRecipe
	if noDeps {
		// Single recipe resolution
		resolved, err := resolver.Resolve(name, varMap)
		if err != nil {
			return fmt.Errorf("resolve blend: %w", err)
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
		// Apply stack only to the main blend (last in order)
		if stack != "" {
			mainIdx := len(resolvedRecipes) - 1
			mainResolved, err := resolver.ResolveStack(name, stack, varMap)
			if err != nil {
				return fmt.Errorf("resolve stack for main blend: %w", err)
			}
			resolvedRecipes[mainIdx] = mainResolved
		}
	}
	// The main blend is the last in the list
	resolved := resolvedRecipes[len(resolvedRecipes)-1]

	// Check for conflicts with already installed recipes
	if !dryRun {
		st, err := getState()
		if err == nil {
			defer st.Close()
			for _, conflict := range r.Conflicts {
				if st.IsRecipeInstalled(conflict) {
					return fmt.Errorf("conflict: blend %s conflicts with installed recipe %s", name, conflict)
				}
			}
		}
	}

	packages := resolved.GetAllPackages()
	fmt.Printf("Packages to install: %d\n", len(packages))

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

	elapsed := time.Since(start)
	fmt.Printf("\nBlend '%s' installed in %v\n", name, elapsed.Round(time.Millisecond))

	// Show post-install messages
	if len(r.PostInstall) > 0 {
		fmt.Println("\nPost-install notes:")
		for _, msg := range r.PostInstall {
			fmt.Printf("  - %s\n", msg)
		}
	}

	return nil
}

func runListBlends(cmd *cobra.Command, args []string) error {
	parser := getParser()

	// Get blends by kind
	names, err := parser.ListByKind("pure blend")
	if err != nil {
		return err
	}

	// Also check for "blend" kind
	blendNames, _ := parser.ListByKind("blend")
	names = append(names, blendNames...)

	if len(names) == 0 {
		fmt.Println("No Debian Pure Blends found in kitchen.")
		fmt.Println("Blend recipes should have kind: 'pure blend' in their YAML.")
		return nil
	}

	fmt.Printf("Debian Pure Blends (%d):\n\n", len(names))

	// Print each blend
	for _, name := range names {
		r, err := parser.Parse(name)
		if err != nil {
			fmt.Printf("  %-25s (parse error)\n", name)
			continue
		}
		fmt.Printf("  %-25s %s\n", r.Name, truncate(r.Description, 55))
	}

	fmt.Println("\nUse 'debian-composer blend <name>' to install a blend.")
	fmt.Println("Use 'debian-composer info <name>' to see blend details.")
	fmt.Println("Use 'debian-composer gallery --kind=blend' for detailed view.")

	return nil
}

func printBlendTasks(r *types.Recipe) error {
	if len(r.Categories) == 0 && len(r.Stacks) == 0 {
		fmt.Println("No tasks/categories defined in this blend.")
		return nil
	}

	if len(r.Stacks) > 0 {
		fmt.Printf("Available Stacks (%d):\n", len(r.Stacks))
		for name, stack := range r.Stacks {
			fmt.Printf("  %-25s %s\n", name, stack.Description)
		}
		fmt.Println()
	}

	if len(r.Categories) > 0 {
		fmt.Printf("Available Categories (%d):\n", len(r.Categories))
		for name, cat := range r.Categories {
			fmt.Printf("  %-25s %s (%d packages)\n", name, cat.Description, len(cat.Packages))
		}
	}

	return nil
}

func printBlendCategories(r *types.Recipe) error {
	if len(r.Categories) == 0 {
		fmt.Println("No categories defined in this blend.")
		return nil
	}

	fmt.Printf("Available Categories (%d):\n", len(r.Categories))
	for name, cat := range r.Categories {
		fmt.Printf("  %-25s %s (%d packages)\n", name, cat.Description, len(cat.Packages))
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// selectTasksInteractive shows an interactive task selection using TUI
func selectTasksInteractive(r *types.Recipe) string {
	// Convert stacks to list items
	var items []tui.ListItem

	for name, stack := range r.Stacks {
		items = append(items, tui.ListItem{
			ItemTitle:       fmt.Sprintf("[stack] %s", name),
			ItemDescription: stack.Description,
			Value:           name,
		})
	}

	for name, cat := range r.Categories {
		items = append(items, tui.ListItem{
			ItemTitle:       fmt.Sprintf("[category] %s", name),
			ItemDescription: fmt.Sprintf("%s (%d packages)", cat.Description, len(cat.Packages)),
			Value:           name,
		})
	}

	if len(items) == 0 {
		fmt.Println("No stacks/categories defined in this blend.")
		return ""
	}

	// Show interactive selection
	choice, err := tui.Select(fmt.Sprintf("Select task for %s", r.Name), items, 20)
	if err != nil || choice == nil || choice.Cancelled {
		return ""
	}

	return choice.Item.Value
}
