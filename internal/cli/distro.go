package cli

import (
	"fmt"
	"time"

	"github.com/debian-composer/debian-composer-go/internal/ux"
	"github.com/spf13/cobra"
)

func init() {
	// Distro command - installs a recipe with kind="distro"
	distroCmd := &cobra.Command{
		Use:   "distro [name]",
		Short: "Install an opinionated distro configuration",
		Long: `Install an opinionated pre-configured Debian setup (kind: "distro").

Distros are recipes that provide sensible defaults for specific use cases.

Available distros:
  debian-developer  - Developer workstation with common tools
  debian-scientist  - Scientific computing environment
  debian-creator    - Creative professional setup
  debian-homelab    - Self-hosted server configuration

Examples:
  debian-composer distro debian-developer
  debian-composer distro debian-developer --var=primary_lang=rust`,
		Args: cobra.ExactArgs(1),
		RunE: runDistro,
	}
	distroCmd.Flags().StringSlice("var", nil, "Set variable (key=value)")
	distroCmd.Flags().String("stack", "", "Use pre-configured stack")
	rootCmd.AddCommand(distroCmd)

	// List distros command
	distrosCmd := &cobra.Command{
		Use:   "distros",
		Short: "List available distro configurations",
		RunE:  runListDistros,
	}
	rootCmd.AddCommand(distrosCmd)
}

func runDistro(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	name := args[0]
	vars, _ := cmd.Flags().GetStringSlice("var")
	stack, _ := cmd.Flags().GetString("stack")
	varMap := parseVars(vars)

	fmt.Printf("Installing distro: %s\n", name)
	if dryRun {
		fmt.Println("[DRY-RUN] No changes will be made")
	}

	start := time.Now()
	resolver := getResolver()
	apt := getAPT()

	// Resolve the distro recipe
	resolved, err := resolver.Resolve(name, varMap)
	if err != nil {
		return fmt.Errorf("resolve distro: %w", err)
	}

	if stack != "" {
		resolved, err = resolver.ResolveStack(name, stack, varMap)
		if err != nil {
			return fmt.Errorf("resolve stack: %w", err)
		}
	}

	packages := resolved.GetAllPackages()
	fmt.Printf("Packages to install: %d\n", len(packages))

	// Create snapshot before installation if requested
	if err := createSnapshotIfRequested(name); err != nil {
		fmt.Printf("Warning: snapshot failed: %v\n", err)
		fmt.Println("Continuing with installation...")
	}

	if dryRun {
		for i, pkg := range packages {
			if i < 15 {
				fmt.Printf("  - %s\n", pkg)
			}
		}
		if len(packages) > 15 {
			fmt.Printf("  ... and %d more\n", len(packages)-15)
		}
	} else {
		fmt.Printf("Installing distro packages...\n")
		apt.BatchInstall(packages)

		st, err := getState()
		if err == nil {
			defer st.Close()
			st.SaveRecipe(resolved)
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("Distro '%s' installed in %v\n", name, elapsed.Round(time.Millisecond))

	// Show post-install message
	if len(resolved.PostInstall) > 0 {
		fmt.Println("\nPost-install notes:")
		for _, msg := range resolved.PostInstall {
			fmt.Printf("  - %s\n", msg)
		}
	}

	return nil
}

func runListDistros(cmd *cobra.Command, args []string) error {
	parser := getParser()
	names, err := parser.List()
	if err != nil {
		return err
	}

	var distros []struct {
		Name        string
		Description string
	}

	for _, name := range names {
		r, err := parser.Parse(name)
		if err != nil {
			continue
		}
		if r.Kind == "distro" {
			distros = append(distros, struct {
				Name        string
				Description string
			}{
				Name:        r.Name,
				Description: r.Description,
			})
		}
	}

	if len(distros) == 0 {
		fmt.Println("No distro configurations found.")
		return nil
	}

	fmt.Printf("Distro Configurations (%d):\n\n", len(distros))
	headers := []string{"Name", "Description"}
	rows := make([][]string, len(distros))
	for i, d := range distros {
		rows[i] = []string{d.Name, truncate(d.Description, 60)}
	}
	ux.PrintTable(headers, rows)

	return nil
}
