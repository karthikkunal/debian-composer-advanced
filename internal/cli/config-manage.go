package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Config represents a configuration file structure
type Config struct {
	Name        string                 `yaml:"name"`
	Version     string                 `yaml:"version"`
	Description string                 `yaml:"description,omitempty"`
	Environment string                 `yaml:"environment,omitempty"`
	Author      string                 `yaml:"author,omitempty"`
	License     string                 `yaml:"license,omitempty"`
	Tags        []string               `yaml:"tags,omitempty"`
	Extends     []string               `yaml:"extends,omitempty"`
	Variables   map[string]interface{} `yaml:"variables,omitempty"`
	Packages    []interface{}          `yaml:"packages,omitempty"`
	Recipes     []string               `yaml:"recipes,omitempty"`
	Blends      []string               `yaml:"blends,omitempty"`
	Settings    map[string]interface{} `yaml:"settings,omitempty"`
}

// ValidationResult holds validation results
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// LintResult holds lint results
type LintResult struct {
	Passed bool
	Issues []LintIssue
}

// LintIssue represents a single lint issue
type LintIssue struct {
	Severity string // "error", "warning", "info"
	Message  string
	Line     int
}

// ConfigInfo holds config metadata for listing
type ConfigInfo struct {
	Name        string
	Version     string
	Description string
	Path        string
}

func init() {
	// Config command for managing configuration files
	configRecipeCmd := &cobra.Command{
		Use:     "config-manage",
		Aliases: []string{"cfg", "configs"},
		Short:   "Manage configuration files (validate, lint, list, export)",
		Long: `Manage configuration files for debian-composer.

Configuration files define system configurations that can be applied.
They follow a YAML schema with required fields (name, version).

Examples:
  debian-composer config-manage validate my-config.yaml    # Validate schema
  debian-composer config-manage lint my-config.yaml        # Lint for issues
  debian-composer config-manage list                       # List configs
  debian-composer config-manage export dev --output=out.yaml`,
	}

	// Validate subcommand
	validateConfigCmd := &cobra.Command{
		Use:   "validate [file]",
		Short: "Validate configuration file schema",
		Long:  `Validate a configuration file against the schema. Checks required fields and format.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigValidate,
	}
	validateConfigCmd.Flags().Bool("strict", false, "Use strict validation (all fields required)")

	// Lint subcommand
	lintConfigCmd := &cobra.Command{
		Use:   "lint [file]",
		Short: "Lint configuration file for common issues",
		Long:  `Lint a configuration file for common issues like trailing whitespace, deprecated fields, etc.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigLint,
	}

	// List subcommand
	listConfigsCmd := &cobra.Command{
		Use:   "list",
		Short: "List available configuration files",
		Long:  `List all configuration files in the kitchen/recipes/configs directory.`,
		RunE:  runConfigList,
	}

	// Export subcommand
	exportConfigCmd := &cobra.Command{
		Use:   "export [name]",
		Short: "Export a named configuration to a file",
		Long:  `Export a named configuration from the configs directory to a standalone file.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigExport,
	}
	exportConfigCmd.Flags().StringP("output", "o", "", "Output file path")

	configRecipeCmd.AddCommand(validateConfigCmd)
	configRecipeCmd.AddCommand(lintConfigCmd)
	configRecipeCmd.AddCommand(listConfigsCmd)
	configRecipeCmd.AddCommand(exportConfigCmd)
	rootCmd.AddCommand(configRecipeCmd)
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	strict, _ := cmd.Flags().GetBool("strict")

	fmt.Printf("Validating config: %s\n", filePath)

	// Check file exists
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not found: %s", filePath)
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Validate
	result := validateConfig(&config, strict)

	if result.Valid {
		fmt.Printf("%s Config validation passed: %s\n", green+"✓"+nc, filePath)
		return nil
	}

	fmt.Printf("%s Config validation failed: %s\n", red+"✗"+nc, filePath)
	for _, err := range result.Errors {
		fmt.Printf("  - %s\n", err)
	}
	os.Exit(1)
	return nil
}

func validateConfig(config *Config, strict bool) ValidationResult {
	result := ValidationResult{Valid: true}

	// Required: name
	if config.Name == "" {
		result.Errors = append(result.Errors, "missing required field: name")
		result.Valid = false
	}

	// Required: version
	if config.Version == "" {
		result.Errors = append(result.Errors, "missing required field: version")
		result.Valid = false
	} else {
		// Validate semver format
		semverRegex := regexp.MustCompile(`^\d+\.\d+\.\d+(-[\w.]+)?$`)
		if !semverRegex.MatchString(config.Version) {
			result.Errors = append(result.Errors, fmt.Sprintf("invalid version format: %s (expected: X.Y.Z or X.Y.Z-tag)", config.Version))
			result.Valid = false
		}
	}

	// Validate environment if present
	if config.Environment != "" {
		validEnvs := []string{"dev", "development", "prod", "production", "personal", "test", "staging"}
		valid := false
		for _, env := range validEnvs {
			if config.Environment == env {
				valid = true
				break
			}
		}
		if !valid {
			result.Errors = append(result.Errors, fmt.Sprintf("invalid environment: %s (must be one of: %s)", config.Environment, strings.Join(validEnvs, ", ")))
			result.Valid = false
		}
	}

	// Strict mode: check more fields
	if strict {
		if config.Description == "" {
			result.Errors = append(result.Errors, "strict mode: missing recommended field: description")
			result.Valid = false
		}
		if len(config.Tags) == 0 {
			result.Errors = append(result.Errors, "strict mode: missing recommended field: tags")
		}
	}

	return result
}

func runConfigLint(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	fmt.Printf("Linting config: %s\n", filePath)

	// Check file exists
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	result := lintConfig(string(data))

	if result.Passed {
		fmt.Printf("%s Config lint passed: %s\n", green+"✓"+nc, filePath)
		return nil
	}

	fmt.Printf("%s Config lint found issues: %s\n", yellow+"⚠"+nc, filePath)
	for _, issue := range result.Issues {
		severity := issue.Severity
		switch severity {
		case "error":
			severity = red + "error" + nc
		case "warning":
			severity = yellow + "warning" + nc
		default:
			severity = blue + "info" + nc
		}
		if issue.Line > 0 {
			fmt.Printf("  %s line %d: %s\n", severity, issue.Line, issue.Message)
		} else {
			fmt.Printf("  %s: %s\n", severity, issue.Message)
		}
	}

	return nil
}

func lintConfig(content string) LintResult {
	result := LintResult{Passed: true}
	lines := strings.Split(content, "\n")

	// Check trailing whitespace
	for i, line := range lines {
		if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
			result.Issues = append(result.Issues, LintIssue{
				Severity: "warning",
				Message:  "trailing whitespace",
				Line:     i + 1,
			})
		}
	}

	// Check for tabs
	for i, line := range lines {
		if strings.Contains(line, "\t") {
			result.Issues = append(result.Issues, LintIssue{
				Severity: "warning",
				Message:  "use of tabs (YAML prefers spaces)",
				Line:     i + 1,
			})
		}
	}

	// Check for deprecated fields
	deprecatedFields := []string{"package_manager", "custom_commands", "legacy_apt"}
	for _, field := range deprecatedFields {
		if strings.Contains(content, field+":") {
			result.Issues = append(result.Issues, LintIssue{
				Severity: "warning",
				Message:  fmt.Sprintf("deprecated field: %s", field),
			})
		}
	}

	// Check for empty arrays
	emptyArrayRegex := regexp.MustCompile(`\w+:\s*\[\s*\]`)
	if emptyArrayRegex.MatchString(content) {
		result.Issues = append(result.Issues, LintIssue{
			Severity: "info",
			Message:  "empty array found (consider removing if not needed)",
		})
	}

	if len(result.Issues) > 0 {
		// Check if any are errors
		for _, issue := range result.Issues {
			if issue.Severity == "error" {
				result.Passed = false
				return result
			}
		}
	}

	return result
}

func runConfigList(cmd *cobra.Command, args []string) error {
	configsPath := filepath.Join(kitchen, "recipes", "configs")

	// Check if directory exists
	if _, err := os.Stat(configsPath); os.IsNotExist(err) {
		fmt.Println("No configs directory found. Create kitchen/recipes/configs/ to add configurations.")
		return nil
	}

	// Find config files
	files, err := filepath.Glob(filepath.Join(configsPath, "*.yaml"))
	if err != nil {
		return err
	}
	files = append(files, glob(filepath.Join(configsPath, "*.yml"))...)

	if len(files) == 0 {
		fmt.Println("No configuration files found in kitchen/recipes/configs/")
		return nil
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                     Available Configurations                            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	var configs []ConfigInfo
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var config Config
		if err := yaml.Unmarshal(data, &config); err != nil {
			configs = append(configs, ConfigInfo{
				Name: strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)),
				Path: file,
			})
			continue
		}

		configs = append(configs, ConfigInfo{
			Name:        config.Name,
			Version:     config.Version,
			Description: config.Description,
			Path:        file,
		})
	}

	for _, c := range configs {
		name := c.Name
		if name == "" {
			name = "(unnamed)"
		}
		version := c.Version
		if version != "" {
			version = "v" + version
		}

		fmt.Printf("  %-25s %-10s %s\n", name, version, c.Description)
		fmt.Printf("  %sPath: %s\n", "  ", c.Path)
		fmt.Println()
	}

	fmt.Printf("Found %d configuration(s)\n", len(configs))

	// Show config directories
	fmt.Println("\nConfig directories:")
	fmt.Printf("  - %s\n", configsPath)
	home, _ := os.UserHomeDir()
	if home != "" {
		fmt.Printf("  - %s/.config/debian-composer/configs\n", home)
	}

	return nil
}

func runConfigExport(cmd *cobra.Command, args []string) error {
	name := args[0]
	outputPath, _ := cmd.Flags().GetString("output")

	// Search for config
	configsPath := filepath.Join(kitchen, "recipes", "configs")
	files, _ := filepath.Glob(filepath.Join(configsPath, "*.yaml"))
	files = append(files, glob(filepath.Join(configsPath, "*.yml"))...)

	var sourcePath string
	for _, file := range files {
		base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		if base == name {
			sourcePath = file
			break
		}
	}

	if sourcePath == "" {
		return fmt.Errorf("config not found: %s", name)
	}

	// Read source
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Determine output path
	if outputPath == "" {
		outputPath = name + ".yaml"
	}

	// Write to output
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	fmt.Printf("%s Config '%s' exported to %s\n", green+"✓"+nc, name, outputPath)
	return nil
}
