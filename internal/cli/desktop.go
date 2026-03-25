package cli

import (
	"fmt"
	"strings"

	"github.com/debian-composer/debian-composer-go/internal/desktop"
	"github.com/debian-composer/debian-composer-go/internal/hardware"
	"github.com/spf13/cobra"
)

func init() {
	desktopCmd := &cobra.Command{
		Use:   "desktop [command]",
		Short: "Manage desktop environments",
	}
	rootCmd.AddCommand(desktopCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available desktop environments",
		RunE:  runDesktopList,
	}
	desktopCmd.AddCommand(listCmd)

	recommendCmd := &cobra.Command{
		Use:   "recommend",
		Short: "Recommend desktop environments based on hardware",
		RunE:  runDesktopRecommend,
	}
	desktopCmd.AddCommand(recommendCmd)

	infoCmd := &cobra.Command{
		Use:   "info [name]",
		Short: "Show desktop environment details",
		Args:  cobra.ExactArgs(1),
		RunE:  runDesktopInfo,
	}
	desktopCmd.AddCommand(infoCmd)
}

func runDesktopList(cmd *cobra.Command, args []string) error {
	envs := desktop.ListEnvironments()

	fmt.Println("Available Desktop Environments:")
	fmt.Println("(sorted by minimum RAM requirement)")
	fmt.Println()

	for _, env := range envs {
		fmt.Printf("  %-10s %s\n", env.Name, env.Description)
		fmt.Printf("             RAM: %d MB, Disk: %d GB", env.MinRAMMB, env.MinDiskGB)
		if env.RequiresGPU {
			fmt.Printf(", GPU required")
		}
		fmt.Println()
	}

	return nil
}

func runDesktopRecommend(cmd *cobra.Command, args []string) error {
	hwInfo, err := hardware.Detect()
	if err != nil {
		return fmt.Errorf("hardware detection failed: %w", err)
	}

	fmt.Printf("Detected Hardware:\n")
	fmt.Printf("  RAM: %d MB\n", hwInfo.Memory.TotalMB)
	fmt.Printf("  CPUs: %d cores\n", hwInfo.CPU.Cores)
	fmt.Printf("  GPUs: %d\n", len(hwInfo.GPUs))
	for _, gpu := range hwInfo.GPUs {
		fmt.Printf("    - %s %s\n", gpu.Vendor, gpu.Model)
	}
	fmt.Println()

	recommended := desktop.GetRecommendations(hwInfo)
	if len(recommended) == 0 {
		fmt.Println("No desktop environments meet your hardware requirements.")
		return nil
	}

	fmt.Println("Recommended Desktop Environments:")
	fmt.Println("(based on your hardware)")
	fmt.Println()

	for _, env := range recommended {
		fmt.Printf("  %-10s %s\n", env.Name, env.Description)
		fmt.Printf("             RAM: %d MB, Disk: %d GB", env.MinRAMMB, env.MinDiskGB)
		if env.RequiresGPU {
			fmt.Printf(", GPU required")
		}
		fmt.Println()
	}

	return nil
}

func runDesktopInfo(cmd *cobra.Command, args []string) error {
	name := strings.ToLower(args[0])
	env, ok := desktop.GetEnvironment(name)
	if !ok {
		return fmt.Errorf("desktop environment not found: %s", name)
	}

	fmt.Printf("Name: %s\n", env.Name)
	fmt.Printf("Description: %s\n", env.Description)
	fmt.Printf("Minimum RAM: %d MB\n", env.MinRAMMB)
	fmt.Printf("Minimum Disk: %d GB\n", env.MinDiskGB)
	fmt.Printf("Requires GPU: %v\n", env.RequiresGPU)
	fmt.Printf("\nPackages: %v\n", strings.Join(env.Packages, ", "))

	return nil
}
