package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/debian-composer/debian-composer-go/internal/registry"
	"github.com/spf13/cobra"
)

func init() {
	// Components command - shows reusable building blocks
	componentsCmd := &cobra.Command{
		Use:     "components",
		Aliases: []string{"parts", "blocks", "registry"},
		Short:   "List and search recipe components",
		Long: `List, search, and show details of components from the registry.

Components are modular building blocks that recipes can include from kitchen/components/.

Examples:
  debian-composer components                    # List all components
  debian-composer components --category=dev     # Filter by category
  debian-composer components --search=python    # Search by term
  debian-composer component-info apache         # Show component details
  debian-composer components --mixins           # List mixins
  debian-composer components --layers           # List layers (sorted by order)`,
		RunE: runComponents,
	}
	componentsCmd.Flags().String("category", "", "Filter by category (desktop, development, database, web, etc.)")
	componentsCmd.Flags().String("search", "", "Search components by name, description, or tags")
	componentsCmd.Flags().Bool("tree", false, "Show directory tree structure")
	componentsCmd.Flags().Bool("mixins", false, "List mixins instead of components")
	componentsCmd.Flags().Bool("layers", false, "List layers instead of components")
	componentsCmd.Flags().Bool("fragments", false, "List fragments instead of components")
	componentsCmd.Flags().Bool("stats", false, "Show registry statistics")
	componentsCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	rootCmd.AddCommand(componentsCmd)

	// Component info command
	componentInfoCmd := &cobra.Command{
		Use:     "component-info [name]",
		Aliases: []string{"info", "show"},
		Short:   "Show detailed information about a component",
		Args:    cobra.ExactArgs(1),
		RunE:    runComponentInfo,
	}
	rootCmd.AddCommand(componentInfoCmd)

	// Search command (shortcut)
	searchCmd := &cobra.Command{
		Use:   "search [term]",
		Short: "Search components by name, description, or tags",
		Args:  cobra.ExactArgs(1),
		RunE:  runSearch,
	}
	rootCmd.AddCommand(searchCmd)
}

func getRegistry() (*registry.Registry, error) {
	componentsPath := filepath.Join(kitchen, "components")
	return registry.NewRegistry(componentsPath)
}

func runComponents(cmd *cobra.Command, args []string) error {
	r, err := getRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	category, _ := cmd.Flags().GetString("category")
	search, _ := cmd.Flags().GetString("search")
	showTree, _ := cmd.Flags().GetBool("tree")
	showMixins, _ := cmd.Flags().GetBool("mixins")
	showLayers, _ := cmd.Flags().GetBool("layers")
	showFragments, _ := cmd.Flags().GetBool("fragments")
	showStats, _ := cmd.Flags().GetBool("stats")

	// Show stats
	if showStats {
		stats := r.GetStats()
		fmt.Println("╔══════════════════════════════════════════════╗")
		fmt.Println("║              Registry Statistics             ║")
		fmt.Println("╚══════════════════════════════════════════════╝")
		fmt.Printf("  Components: %d\n", stats["components"])
		fmt.Printf("  Mixins:     %d\n", stats["mixins"])
		fmt.Printf("  Layers:     %d\n", stats["layers"])
		fmt.Printf("  Fragments:  %d\n", stats["fragments"])
		return nil
	}

	// Show mixins
	if showMixins {
		mixins := r.ListMixins()
		fmt.Println("╔══════════════════════════════════════════════╗")
		fmt.Println("║                    Mixins                    ║")
		fmt.Println("╚══════════════════════════════════════════════╝")
		fmt.Printf("Found %d mixins\n\n", len(mixins))
		for _, m := range mixins {
			appliesTo := strings.Join(m.AppliesTo, ", ")
			fmt.Printf("  %-25s %s\n", m.Name, m.Description)
			if len(m.Tags) > 0 {
				fmt.Printf("  %sTags: %s%s\n", "  ", strings.Join(m.Tags, ", "), "")
			}
			fmt.Printf("  %sApplies to: %s\n", "  ", appliesTo)
			fmt.Println()
		}
		return nil
	}

	// Show layers
	if showLayers {
		layers := r.ListLayers()
		fmt.Println("╔══════════════════════════════════════════════╗")
		fmt.Println("║                    Layers                    ║")
		fmt.Println("╚══════════════════════════════════════════════╝")
		fmt.Printf("Found %d layers (sorted by order)\n\n", len(layers))
		for i, l := range layers {
			fmt.Printf("  [%d] %-20s %s\n", i+1, l.Name, l.Description)
		}
		return nil
	}

	// Show fragments
	if showFragments {
		fragments := r.ListFragments()
		fmt.Println("╔══════════════════════════════════════════════╗")
		fmt.Println("║                  Fragments                   ║")
		fmt.Println("╚══════════════════════════════════════════════╝")
		fmt.Printf("Found %d fragments\n\n", len(fragments))
		for _, f := range fragments {
			fmt.Printf("  %-25s %s\n", f.Name, f.Description)
		}
		return nil
	}

	// Show tree structure
	if showTree {
		return printComponentTree(filepath.Join(kitchen, "components"))
	}

	// Get components
	var components []registry.Component
	if search != "" {
		components = r.SearchComponents(search)
		fmt.Printf("Search results for: %s\n\n", search)
	} else {
		components = r.ListComponents(category)
	}

	if len(components) == 0 {
		fmt.Println("No components found matching criteria.")
		return nil
	}

	// Display header
	if search != "" {
		fmt.Println("╔══════════════════════════════════════════════════════════════════════════╗")
		fmt.Printf("║  Search Results for: %-52s ║\n", search)
		fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
	} else {
		fmt.Println("╔══════════════════════════════════════════════════════════════════════════╗")
		fmt.Println("║                       Component Registry                                ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
	}
	fmt.Printf("Found %d components\n\n", len(components))

	// Group by category
	groups := make(map[string][]registry.Component)
	for _, c := range components {
		groups[c.Category] = append(groups[c.Category], c)
	}

	// Display in order
	categories := []string{"infrastructure", "desktop", "development", "education",
		"science", "medical", "gis", "web", "database", "ai", "system", "other"}

	for _, cat := range categories {
		items := groups[cat]
		if len(items) == 0 {
			continue
		}

		fmt.Printf("─── %s ───\n", strings.ToUpper(cat))
		for _, c := range items {
			vars := ""
			if c.HasVariables {
				vars = " [vars]"
			}
			desc := c.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			fmt.Printf("  %-25s %s%s\n", c.Name, desc, vars)
		}
		fmt.Println()
	}

	// Show summary
	fmt.Printf("Total: %d components\n", len(components))
	if category == "" && search == "" {
		fmt.Println("\nUse --category=NAME to filter by category")
		fmt.Println("Use --search=TERM to search by name/description/tags")
		fmt.Println("Use 'component-info NAME' for detailed information")
	}

	return nil
}

func runComponentInfo(cmd *cobra.Command, args []string) error {
	name := args[0]
	r, err := getRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	component, found := r.GetComponent(name)
	if !found {
		// Try partial match
		results := r.SearchComponents(name)
		if len(results) == 1 {
			component = &results[0]
			found = true
		} else if len(results) > 1 {
			fmt.Printf("Multiple components match '%s':\n", name)
			for _, c := range results {
				fmt.Printf("  - %s: %s\n", c.Name, c.Description)
			}
			fmt.Println("\nUse exact name with 'component-info'")
			return nil
		}
	}

	if !found || component == nil {
		return fmt.Errorf("component not found: %s", name)
	}

	// Display component info
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║  Component: %-59s ║\n", component.Name)
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("Description: %s\n", component.Description)
	fmt.Printf("Category:    %s\n", component.Category)
	fmt.Printf("File:        %s\n", component.File)

	if len(component.Tags) > 0 {
		fmt.Printf("Tags:        %s\n", strings.Join(component.Tags, ", "))
	}

	if component.HasVariables {
		fmt.Printf("Variables:   Yes\n")
		if len(component.Variables) > 0 {
			fmt.Printf("  - %s\n", strings.Join(component.Variables, "\n  - "))
		}
	}

	// Try to read the actual component file
	filePath := filepath.Join(kitchen, "components", component.File)
	fmt.Printf("\nFull path:   %s\n", filePath)

	return nil
}

func runSearch(cmd *cobra.Command, args []string) error {
	term := args[0]
	r, err := getRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	results := r.SearchComponents(term)
	if len(results) == 0 {
		fmt.Printf("No components found matching '%s'\n", term)
		return nil
	}

	fmt.Printf("Found %d component(s) matching '%s':\n\n", len(results), term)
	for _, c := range results {
		fmt.Printf("  %-25s [%s] %s\n", c.Name, c.Category, c.Description)
	}

	return nil
}

func printComponentTree(basePath string) error {
	fmt.Println("kitchen/components/")
	fmt.Println("├── index.yaml (registry)")

	// List YAML files in root
	rootFiles, _ := filepath.Glob(filepath.Join(basePath, "*.yaml"))
	rootFiles = append(rootFiles, glob(filepath.Join(basePath, "*.yml"))...)

	// Filter out index.yaml
	var filtered []string
	for _, f := range rootFiles {
		if filepath.Base(f) != "index.yaml" {
			filtered = append(filtered, f)
		}
	}

	if len(filtered) > 0 {
		for i, file := range filtered {
			name := filepath.Base(file)
			prefix := "├── "
			if i == len(filtered)-1 {
				prefix = "└── "
			}
			fmt.Printf("%s%s\n", prefix, name)
		}
	}

	// List subdirectories
	dirs, _ := filepath.Glob(filepath.Join(basePath, "*"))
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}

		dirName := filepath.Base(dir)
		files, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
		files = append(files, glob(filepath.Join(dir, "*.yml"))...)

		if len(files) == 0 {
			continue
		}

		fmt.Printf("├── %s/ (%d)\n", dirName, len(files))
		for i, file := range files {
			name := filepath.Base(file)
			prefix := "│   ├── "
			if i == len(files)-1 {
				prefix = "│   └── "
			}
			fmt.Printf("%s%s\n", prefix, name)
		}
	}

	return nil
}

func glob(pattern string) []string {
	files, _ := filepath.Glob(pattern)
	return files
}
