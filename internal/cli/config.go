package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/debian-composer/debian-composer-go/internal/config"
	"github.com/spf13/cobra"
)

var (
	configSource string
	configDest   string
	configBackup bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage dotfiles configuration",
	Long:  `Manage dotfiles configuration using a source directory with templates and apply to target locations.`,
}

var configApplyCmd = &cobra.Command{
	Use:   "apply [file...]",
	Short: "Apply dotfiles to target locations",
	Long:  `Apply dotfiles from the source directory to their target locations. Optionally specify specific files to apply.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := newConfigManager()
		if err != nil {
			return err
		}
		cfg.SetBackup(configBackup)

		// Get template data from recipe if available
		data := getTemplateData()

		if len(args) > 0 {
			// Apply specific files
			for _, file := range args {
				dotfile := config.Dotfile{
					Source: file,
					Target: file,
				}
				if err := cfg.Apply(dotfile, data); err != nil {
					return fmt.Errorf("failed to apply %s: %w", file, err)
				}
			}
		} else {
			// Apply all files
			if err := cfg.ApplyAll(data); err != nil {
				return err
			}
		}

		if !dryRun {
			fmt.Println("✓ Dotfiles applied successfully")
		}
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List dotfiles status",
	Long:  `List all dotfiles in the source directory and their status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := newConfigManager()
		if err != nil {
			return err
		}

		statuses, err := cfg.List()
		if err != nil {
			return err
		}

		fmt.Print(config.FormatDotfileStatus(statuses))
		return nil
	},
}

var configDiffCmd = &cobra.Command{
	Use:   "diff <file>",
	Short: "Show differences between source and target",
	Long:  `Show differences between a dotfile in the source directory and its target location.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := newConfigManager()
		if err != nil {
			return err
		}

		dotfile := config.Dotfile{
			Source: args[0],
			Target: args[0],
		}

		diff, err := cfg.Diff(dotfile)
		if err != nil {
			return err
		}

		fmt.Println(diff)
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize config source directory",
	Long:  `Initialize a config source directory with example dotfiles.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := initConfigDir(); err != nil {
			return err
		}

		sourceDir := configSource
		if sourceDir == "" {
			sourceDir = filepath.Join(kitchen, "dotfiles")
		}
		fmt.Printf("✓ Config source directory initialized at %s\n", sourceDir)
		return nil
	},
}

func init() {
	configCmd.Flags().StringVar(&configSource, "source", "", "Source directory for dotfiles")
	configCmd.Flags().StringVar(&configDest, "dest", "", "Destination directory for dotfiles")
	configCmd.Flags().BoolVar(&configBackup, "backup", true, "Backup existing files before overwriting")

	configCmd.AddCommand(configApplyCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configDiffCmd)
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}

func newConfigManager() (*config.Config, error) {
	sourceDir := configSource
	if sourceDir == "" {
		// Default source directory in kitchen
		sourceDir = filepath.Join(kitchen, "dotfiles")
	}

	destDir := configDest
	if destDir == "" {
		// Default to home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		destDir = home
	}

	// Create source directory if it doesn't exist
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create source directory: %w", err)
	}

	return config.NewConfig(sourceDir, destDir, dryRun, verbose), nil
}

func getTemplateData() interface{} {
	// Return environment variables and basic system info as template data
	home, _ := os.UserHomeDir()
	hostname, _ := os.Hostname()

	return map[string]interface{}{
		"Home":     home,
		"Hostname": hostname,
		"User":     os.Getenv("USER"),
		"Env":      os.Environ(),
	}
}

func initConfigDir() error {
	// Get source directory from config
	sourceDir := configSource
	if sourceDir == "" {
		sourceDir = filepath.Join(kitchen, "dotfiles")
	}

	// Create example dotfiles
	examples := map[string]string{
		".gitconfig": `[user]
    name = {{.User}}
    email = {{.User}}@{{.Hostname}}

[core]
    editor = vim
    excludesfile = ~/.gitignore_global

[alias]
    st = status
    co = checkout
    br = branch
    ci = commit
`,
		".vimrc": `" Example .vimrc managed by debian-composer
set number
set relativenumber
set tabstop=4
set shiftwidth=4
set expandtab
set autoindent
set smartindent
syntax on
`,
		".bashrc.d/aliases.sh": `# Custom aliases
alias ll='ls -la'
alias la='ls -A'
alias l='ls -CF'
alias ..='cd ..'
`,
	}

	for path, content := range examples {
		fullPath := filepath.Join(sourceDir, path)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if dryRun {
			fmt.Printf("[DRY-RUN] Would create %s\n", fullPath)
			continue
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", fullPath, err)
		}
	}

	return nil
}
