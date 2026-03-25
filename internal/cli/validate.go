package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/debian-composer/debian-composer-go/internal/ux"
	"github.com/debian-composer/debian-composer-go/internal/validation"
	"github.com/spf13/cobra"
)

func init() {
	// Validate command
	validateCmd := &cobra.Command{
		Use:   "validate [recipe]",
		Short: "Validate a recipe file",
		Long: `Validate a recipe file against the schema.

Checks for:
- Valid YAML syntax
- Required fields (name, version)
- Correct field types
- Variable constraints`,
		Args: cobra.ExactArgs(1),
		RunE: runValidate,
	}
	validateCmd.Flags().Bool("strict", false, "Treat warnings as errors")
	rootCmd.AddCommand(validateCmd)

	// Validate all command
	validateAllCmd := &cobra.Command{
		Use:   "validate-all",
		Short: "Validate all recipes in the kitchen",
		RunE:  runValidateAll,
	}
	rootCmd.AddCommand(validateAllCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	logger := ux.NewLogger(verbose)
	recipeName := args[0]
	strict, _ := cmd.Flags().GetBool("strict")

	logger.Info("Validating recipe: %s", recipeName)

	// Find recipe file
	path, err := findRecipeFile(kitchen, recipeName)
	if err != nil {
		return fmt.Errorf("recipe not found: %w", err)
	}

	// Read file
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read recipe: %w", err)
	}

	// Validate against schema
	validator, err := validation.NewValidator()
	if err != nil {
		return fmt.Errorf("failed to create validator: %w", err)
	}

	result, err := validator.ValidateYAML(content)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Println(validation.FormatErrors(result))

	if strict && len(result.Warnings) > 0 {
		return fmt.Errorf("validation failed with %d warnings (strict mode)", len(result.Warnings))
	}

	if result.Valid {
		logger.Success("Recipe '%s' is valid", recipeName)
	} else {
		return fmt.Errorf("recipe validation failed")
	}

	return nil
}

func runValidateAll(cmd *cobra.Command, args []string) error {
	logger := ux.NewLogger(verbose)

	// Find all recipe files
	recipeDir := filepath.Join(kitchen, "recipes")
	files, err := filepath.Glob(filepath.Join(recipeDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to list recipes: %w", err)
	}

	ymlFiles, _ := filepath.Glob(filepath.Join(recipeDir, "*.yml"))
	files = append(files, ymlFiles...)

	logger.Info("Validating %d recipes...", len(files))

	validator, err := validation.NewValidator()
	if err != nil {
		return fmt.Errorf("failed to create validator: %w", err)
	}

	passed := 0
	failed := 0
	warnings := 0

	for _, path := range files {
		name := filepath.Base(path)
		name = name[:len(name)-len(filepath.Ext(name))]

		content, err := ioutil.ReadFile(path)
		if err != nil {
			logger.Error("%s: failed to read", name)
			failed++
			continue
		}

		result, err := validator.ValidateYAML(content)
		if err != nil {
			logger.Error("%s: %v", name, err)
			failed++
			continue
		}

		if result.Valid {
			if len(result.Warnings) > 0 {
				logger.Warning("%s: valid with %d warnings", name, len(result.Warnings))
				warnings++
			} else {
				logger.Success("%s: valid", name)
			}
			passed++
		} else {
			logger.Error("%s: invalid (%d errors)", name, len(result.Errors))
			failed++
		}
	}

	fmt.Println()
	fmt.Printf("Results: %d passed, %d failed, %d with warnings\n", passed, failed, warnings)

	if failed > 0 {
		return fmt.Errorf("%d recipes failed validation", failed)
	}

	return nil
}

// findRecipeFile finds a recipe file in the kitchen
func findRecipeFile(kitchenDir, name string) (string, error) {
	extensions := []string{".yaml", ".yml"}
	dirs := []string{"recipes", "components", ""}

	for _, dir := range dirs {
		for _, ext := range extensions {
			var path string
			if dir == "" {
				path = filepath.Join(kitchenDir, name+ext)
			} else {
				path = filepath.Join(kitchenDir, dir, name+ext)
			}
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("recipe file not found: %s", name)
}
