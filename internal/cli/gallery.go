package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/debian-composer/debian-composer-go/internal/recipe"
	"github.com/debian-composer/debian-composer-go/internal/ux"
	"github.com/spf13/cobra"
)

// recipeInfo holds recipe metadata for display
type recipeInfo struct {
	Name         string
	Description  string
	Kind         string
	PackageCount int
}

func init() {
	// Gallery command - shows all recipes grouped by kind
	galleryCmd := &cobra.Command{
		Use:   "gallery",
		Short: "Browse the recipe gallery",
		Long: `Browse available recipes organized by category.

The gallery shows all recipes in the kitchen grouped by their kind:
  - Blends     - Debian Pure Blends (education, science, etc.)
  - Distros    - Opinionated system configurations
  - Personas   - User type-based setups
  - Mixins     - Specialization overlays
  - Components - Reusable building blocks
  - Recipes    - General purpose recipes

Examples:
  debian-composer gallery
  debian-composer gallery --kind=blend
  debian-composer gallery --search=python`,
		RunE: runGallery,
	}
	galleryCmd.Flags().String("kind", "", "Filter by kind (blend, distro, persona, mixin, component)")
	galleryCmd.Flags().String("search", "", "Search recipes by name or description")
	galleryCmd.Flags().String("sort", "name", "Sort recipes (name, kind, packages, desc)")
	galleryCmd.Flags().Bool("all", false, "Show all recipes without grouping")
	rootCmd.AddCommand(galleryCmd)
}

func runGallery(cmd *cobra.Command, args []string) error {
	parser := getParser()
	kind, _ := cmd.Flags().GetString("kind")
	search, _ := cmd.Flags().GetString("search")
	sortField, _ := cmd.Flags().GetString("sort")
	showAll, _ := cmd.Flags().GetBool("all")

	names, err := parser.List()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		fmt.Println("No recipes found in kitchen.")
		return nil
	}

	// Collect recipe info - be resilient to parse errors
	var recipes []recipeInfo
	for _, name := range names {
		r, err := parser.Parse(name)
		if err != nil {
			// Try to get basic info from YAML without full parsing
			info := quickParseRecipe(parser, name)

			// Apply kind filter
			if kind != "" && info.Kind != kind {
				continue
			}

			// Apply search filter
			if search != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(search)) {
				continue
			}

			recipes = append(recipes, info)
			continue
		}

		// Apply kind filter
		if kind != "" && r.Kind != kind {
			continue
		}

		// Apply search filter
		if search != "" {
			if !strings.Contains(strings.ToLower(r.Name), strings.ToLower(search)) &&
				!strings.Contains(strings.ToLower(r.Description), strings.ToLower(search)) {
				continue
			}
		}

		recipes = append(recipes, recipeInfo{
			Name:         r.Name,
			Description:  r.Description,
			Kind:         r.Kind,
			PackageCount: len(r.Packages),
		})
	}

	if len(recipes) == 0 {
		fmt.Println("No recipes match the criteria.")
		return nil
	}

	// Apply sorting
	sortRecipes(recipes, sortField)

	// Show all without grouping
	if showAll || kind != "" || search != "" {
		return printRecipeList(recipes)
	}

	// Group by kind
	groups := make(map[string][]recipeInfo)
	for _, r := range recipes {
		kind := r.Kind
		if kind == "" {
			kind = "recipe"
		}
		groups[kind] = append(groups[kind], r)
	}

	// Define display order and labels
	kindOrder := []struct {
		kind  string
		label string
		icon  string
	}{
		{"pure blend", "Debian Pure Blends", ""},
		{"blend", "Blends", ""},
		{"distro", "Opinionated Distros", ""},
		{"persona", "Personas", ""},
		{"mixin", "Mixins", ""},
		{"component", "Components", ""},
		{"", "Recipes", ""},
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Recipe Gallery                               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Found %d recipes in kitchen\n\n", len(recipes))

	for _, ko := range kindOrder {
		items := groups[ko.kind]
		if len(items) == 0 {
			continue
		}

		fmt.Printf("┌─ %s (%d)\n", ko.label, len(items))
		for _, r := range items {
			desc := truncate(r.Description, 50)
			fmt.Printf("│  %-25s %s\n", r.Name, desc)
		}
		fmt.Println("└─")
		fmt.Println()
	}

	// Show any remaining kinds not in the predefined list
	for kind, items := range groups {
		found := false
		for _, ko := range kindOrder {
			if ko.kind == kind {
				found = true
				break
			}
		}
		if !found && len(items) > 0 {
			fmt.Printf("┌─ %s (%d)\n", strings.Title(kind), len(items))
			for _, r := range items {
				desc := truncate(r.Description, 50)
				fmt.Printf("│  %-25s %s\n", r.Name, desc)
			}
			fmt.Println("└─")
			fmt.Println()
		}
	}

	fmt.Println("Use 'debian-composer info <name>' to see recipe details")
	fmt.Println("Use 'debian-composer install <name>' to install a recipe")

	return nil
}

func printRecipeList(recipes []recipeInfo) error {
	fmt.Printf("Recipes (%d):\n\n", len(recipes))

	headers := []string{"Name", "Kind", "Packages", "Description"}
	rows := make([][]string, len(recipes))
	for i, r := range recipes {
		kind := r.Kind
		if kind == "" {
			kind = "recipe"
		}
		rows[i] = []string{
			r.Name,
			kind,
			fmt.Sprintf("%d", r.PackageCount),
			truncate(r.Description, 45),
		}
	}

	ux.PrintTable(headers, rows)
	return nil
}

// quickParseRecipe extracts basic info from a recipe file without full parsing
func quickParseRecipe(parser *recipe.Parser, name string) recipeInfo {
	info := recipeInfo{
		Name:        name,
		Kind:        "recipe",
		Description: "",
	}

	// Try to find the file
	path := filepath.Join(kitchen, "recipes", name+".yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Join(kitchen, "recipes", name+".yml")
	}

	file, err := os.Open(path)
	if err != nil {
		return info
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "description:") {
			info.Description = strings.TrimPrefix(line, "description:")
			info.Description = strings.TrimSpace(info.Description)
			// Remove quotes if present
			info.Description = strings.Trim(info.Description, "\"'")
		} else if strings.HasPrefix(line, "kind:") {
			info.Kind = strings.TrimPrefix(line, "kind:")
			info.Kind = strings.TrimSpace(info.Kind)
			info.Kind = strings.Trim(info.Kind, "\"'")
		}

		// Stop early if we have what we need
		if info.Description != "" && info.Kind != "recipe" {
			break
		}
	}

	return info
}

func sortRecipes(recipes []recipeInfo, sortField string) {
	sort.Slice(recipes, func(i, j int) bool {
		switch strings.ToLower(sortField) {
		case "kind":
			if recipes[i].Kind != recipes[j].Kind {
				return recipes[i].Kind < recipes[j].Kind
			}
			return recipes[i].Name < recipes[j].Name
		case "packages", "pkg":
			if recipes[i].PackageCount != recipes[j].PackageCount {
				return recipes[i].PackageCount > recipes[j].PackageCount // Descending
			}
			return recipes[i].Name < recipes[j].Name
		case "description", "desc":
			if recipes[i].Description != recipes[j].Description {
				return recipes[i].Description < recipes[j].Description
			}
			return recipes[i].Name < recipes[j].Name
		case "name":
			fallthrough
		default:
			return recipes[i].Name < recipes[j].Name
		}
	})
}
