package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/debian-composer/debian-composer-go/internal/deb"
	"github.com/spf13/cobra"
)

var (
	debOutput     string
	debVersion    string
	debArch       string
	debMaintainer string
	debEmail      string
)

var debCmd = &cobra.Command{
	Use:   "deb",
	Short: "Build .deb packages",
	Long:  `Build Debian packages (.deb) for distributing configuration or tools.`,
}

var debBuildCmd = &cobra.Command{
	Use:   "build [directory...]",
	Short: "Build a .deb package from files",
	Long:  `Build a .deb package by specifying directories or files to include.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return buildPackage(args)
	},
}

var debInfoCmd = &cobra.Command{
	Use:   "info <package.deb>",
	Short: "Show package information",
	Long:  `Show information about an existing .deb package file.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return showPackageInfo(args[0])
	},
}

func init() {
	debBuildCmd.Flags().StringVar(&debOutput, "output", ".", "Output directory for .deb file")
	debBuildCmd.Flags().StringVar(&debVersion, "version", "1.0.0", "Package version")
	debBuildCmd.Flags().StringVar(&debArch, "arch", "all", "Package architecture (amd64, i386, arm64, all)")
	debBuildCmd.Flags().StringVar(&debMaintainer, "maintainer", "Debian Composer Team", "Package maintainer name")
	debBuildCmd.Flags().StringVar(&debEmail, "email", "debian-composer@example.com", "Maintainer email")

	debCmd.AddCommand(debBuildCmd)
	debCmd.AddCommand(debInfoCmd)
	rootCmd.AddCommand(debCmd)
}

func buildPackage(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("at least one directory or file must be specified")
	}

	// Get package info from flags or defaults
	info := deb.GetDefaultPackageInfo()
	info.Version = debVersion
	info.Architecture = debArch
	info.Maintainer = debMaintainer
	info.MaintainerEmail = debEmail

	// If dry-run, show what would be built
	if dryRun {
		fmt.Println("[DRY-RUN] Would build package:")
		fmt.Printf("  Name: %s\n", info.Name)
		fmt.Printf("  Version: %s\n", info.Version)
		fmt.Printf("  Architecture: %s\n", info.Architecture)
		fmt.Printf("  Maintainer: %s <%s>\n", info.Maintainer, info.MaintainerEmail)
		return nil
	}

	// Create builder
	builder := deb.NewBuilder(info, debOutput, dryRun, verbose)

	// Add files from arguments
	for _, arg := range args {
		path, err := filepath.Abs(arg)
		if err != nil {
			return fmt.Errorf("invalid path: %s", arg)
		}

		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("path not found: %s", arg)
		}

		if info.IsDir() {
			// Add all files from directory
			if verbose {
				fmt.Printf("Adding directory: %s\n", path)
			}
			if err := builder.AddDirectory(path, "/usr/local/share/debian-composer"); err != nil {
				return fmt.Errorf("failed to add directory %s: %w", arg, err)
			}
		} else {
			// Add single file
			if verbose {
				fmt.Printf("Adding file: %s\n", path)
			}
			destPath := fmt.Sprintf("/usr/local/bin/%s", filepath.Base(path))
			builder.AddFile(path, destPath, info.Mode())
		}
	}

	// Build the package
	outputPath, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build package: %w", err)
	}

	fmt.Println(deb.FormatBuildSummary(info, outputPath, 0))
	fmt.Printf("✓ Package built: %s\n", outputPath)

	return nil
}

func showPackageInfo(debPath string) error {
	if err := deb.ValidatePackage(debPath); err != nil {
		return err
	}

	if dryRun {
		fmt.Printf("[DRY-RUN] Would show info for: %s\n", debPath)
		return nil
	}

	// Basic info from filename
	filename := filepath.Base(debPath)
	fmt.Printf("Package: %s\n", filename)
	fmt.Printf("Path: %s\n", debPath)

	// Get file size
	info, err := os.Stat(debPath)
	if err != nil {
		return err
	}
	fmt.Printf("Size: %d bytes\n", info.Size())
	fmt.Printf("Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

	return nil
}
