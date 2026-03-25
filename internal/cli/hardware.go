package cli

import (
	"fmt"

	"github.com/debian-composer/debian-composer-go/internal/hardware"
	"github.com/debian-composer/debian-composer-go/internal/ux"
	"github.com/spf13/cobra"
)

func init() {
	// Probe hardware command
	probeCmd := &cobra.Command{
		Use:   "probe-hardware",
		Short: "Detect and display hardware information",
		RunE:  runProbeHardware,
	}
	probeCmd.Flags().Bool("recommend", false, "Show recommended packages based on hardware")
	rootCmd.AddCommand(probeCmd)
}

func runProbeHardware(cmd *cobra.Command, args []string) error {
	recommend, _ := cmd.Flags().GetBool("recommend")

	logger := ux.NewLogger(verbose)
	logger.Header("Hardware Detection")

	spinner := ux.NewSpinner("Detecting hardware...")
	spinner.Start()

	info, err := hardware.Detect()
	if err != nil {
		spinner.StopWithError("Hardware detection failed")
		if verbose {
			logger.Error("%v", err)
		}
		// Continue with partial results
	} else {
		spinner.StopWithSuccess("Hardware detected")
	}

	// Print hardware info
	fmt.Println()
	fmt.Println(info.String())

	// Show recommendations if requested
	if recommend {
		pkgs := info.RecommendPackages()
		if len(pkgs) > 0 {
			logger.Header("Recommended Packages")
			for _, pkg := range pkgs {
				fmt.Printf("  - %s\n", pkg)
			}
		}
	}

	return nil
}
