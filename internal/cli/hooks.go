package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/debian-composer/debian-composer-go/internal/hooks"
	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage git hooks for the project",
	Long:  `Manage git hooks (pre-commit, pre-push) for code quality and testing.`,
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install git hooks",
	Long:  `Install pre-commit and pre-push git hooks for the project.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := getRepoPath()
		if err != nil {
			return err
		}

		manager := hooks.NewManager(repoPath, dryRun)

		if !manager.IsGitRepo() {
			return fmt.Errorf("not a git repository: %s", repoPath)
		}

		if err := manager.Install(); err != nil {
			return err
		}

		if !dryRun {
			fmt.Println("✓ Git hooks installed successfully")
		}
		return nil
	},
}

var hooksUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall git hooks",
	Long:  `Remove installed git hooks from the project.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := getRepoPath()
		if err != nil {
			return err
		}

		manager := hooks.NewManager(repoPath, dryRun)

		if err := manager.Uninstall(); err != nil {
			return err
		}

		if !dryRun {
			fmt.Println("✓ Git hooks uninstalled successfully")
		}
		return nil
	},
}

var hooksStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show git hooks status",
	Long:  `Display the status of installed git hooks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := getRepoPath()
		if err != nil {
			return err
		}

		manager := hooks.NewManager(repoPath, dryRun)

		hookList, err := manager.Status()
		if err != nil {
			return err
		}

		fmt.Print(hooks.FormatHookStatus(hookList))
		return nil
	},
}

func init() {
	hooksCmd.AddCommand(hooksInstallCmd)
	hooksCmd.AddCommand(hooksUninstallCmd)
	hooksCmd.AddCommand(hooksStatusCmd)
	rootCmd.AddCommand(hooksCmd)
}

// getRepoPath finds the git repository root
func getRepoPath() (string, error) {
	// Start from current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up to find .git directory
	dir := cwd
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding .git
			return cwd, nil
		}
		dir = parent
	}
}
