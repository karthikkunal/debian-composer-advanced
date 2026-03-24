package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/debian-composer/debian-composer-go/internal/apt"
	"github.com/debian-composer/debian-composer-go/internal/recipe"
	"github.com/debian-composer/debian-composer-go/internal/snapper"
	"github.com/debian-composer/debian-composer-go/internal/state"
	"github.com/spf13/cobra"
)

var (
	dryRun       bool
	verbose      bool
	kitchen      string
	stateDir     string
	withSnapshot bool
	skipSteps    []string
)

// rootCmd is the base command
var rootCmd = &cobra.Command{
	Use:   "debian-composer",
	Short: "Compose your perfect Debian system",
	Long: `Debian Composer - Post-Install Configuration and Recipe Manager

Compose your ideal Debian setup with sensible defaults and modular recipes.`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Default kitchen path is relative to binary
	defaultKitchen := "../kitchen"
	if exe, err := os.Executable(); err == nil {
		defaultKitchen = filepath.Join(filepath.Dir(exe), "..", "kitchen")
	}

	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&kitchen, "kitchen", defaultKitchen, "Path to kitchen directory")
	rootCmd.PersistentFlags().StringVar(&stateDir, "state", "/var/lib/debian-composer", "State directory")
	rootCmd.PersistentFlags().BoolVar(&withSnapshot, "with-snapshot", false, "Create Snapper snapshot before installation")
	rootCmd.PersistentFlags().StringSliceVar(&skipSteps, "skip", nil, "Skip steps (comma-separated): ssh,firewall,fail2ban,bootloader,kernel")
}

// getParser returns a recipe parser
func getParser() *recipe.Parser {
	return recipe.NewParser(kitchen)
}

// getResolver returns a recipe resolver
func getResolver() *recipe.Resolver {
	return recipe.NewResolver(getParser())
}

// getManager returns a recipe manager
func getManager() *recipe.Manager {
	return recipe.NewManager(getParser())
}

// getAPT returns an APT handler
func getAPT() *apt.APT {
	return apt.New(dryRun)
}

// getState returns the state manager
func getState() (*state.Manager, error) {
	return state.New(stateDir)
}

// requireRoot checks for root privileges
func requireRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this command requires root privileges (use sudo)")
	}
	return nil
}

// ShouldSkip returns true if the given step should be skipped
func ShouldSkip(step string) bool {
	for _, s := range skipSteps {
		if strings.EqualFold(s, step) {
			return true
		}
	}
	return false
}

// SkipSteps returns the list of steps to skip
func SkipSteps() []string {
	return skipSteps
}

// createSnapshotIfRequested creates a Snapper snapshot if --with-snapshot is set
func createSnapshotIfRequested(recipeName string) error {
	if !withSnapshot {
		return nil
	}

	// Check if dry-run
	if dryRun {
		fmt.Println("[DRY-RUN] Would create Snapper snapshot before installation")
		return nil
	}

	// Check if snapper is available
	if !snapper.IsAvailable() {
		fmt.Println("Warning: Snapper is not installed, skipping snapshot")
		fmt.Println("Install with: sudo apt install snapper")
		return nil
	}

	mgr := snapper.New(false)
	snap, err := mgr.CreateSnapshotForRecipe(recipeName)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	fmt.Printf("✓ Created snapshot: %s (ID: %d)\n", snap.Name, snap.ID)
	return nil
}
