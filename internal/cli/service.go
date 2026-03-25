package cli

import (
	"fmt"
	"strings"

	"github.com/debian-composer/debian-composer-go/internal/service"
	"github.com/debian-composer/debian-composer-go/internal/ux"
	"github.com/spf13/cobra"
)

func init() {
	// Service command
	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Manage systemd services",
		Long:  `Manage systemd services (start, stop, enable, disable, restart).`,
	}

	// Service status command
	statusCmd := &cobra.Command{
		Use:   "status [service]",
		Short: "Show service status",
		Args:  cobra.ExactArgs(1),
		RunE:  runServiceStatus,
	}
	serviceCmd.AddCommand(statusCmd)

	// Service enable command
	enableCmd := &cobra.Command{
		Use:   "enable [service...]",
		Short: "Enable services",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runServiceEnable,
	}
	serviceCmd.AddCommand(enableCmd)

	// Service disable command
	disableCmd := &cobra.Command{
		Use:   "disable [service...]",
		Short: "Disable services",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runServiceDisable,
	}
	serviceCmd.AddCommand(disableCmd)

	// Service start command
	startCmd := &cobra.Command{
		Use:   "start [service...]",
		Short: "Start services",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runServiceStart,
	}
	serviceCmd.AddCommand(startCmd)

	// Service stop command
	stopCmd := &cobra.Command{
		Use:   "stop [service...]",
		Short: "Stop services",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runServiceStop,
	}
	serviceCmd.AddCommand(stopCmd)

	// Service restart command
	restartCmd := &cobra.Command{
		Use:   "restart [service...]",
		Short: "Restart services",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runServiceRestart,
	}
	serviceCmd.AddCommand(restartCmd)

	// Service list command
	listCmd := &cobra.Command{
		Use:   "list [pattern]",
		Short: "List services",
		RunE:  runServiceList,
	}
	serviceCmd.AddCommand(listCmd)

	rootCmd.AddCommand(serviceCmd)
}

func runServiceStatus(cmd *cobra.Command, args []string) error {
	serviceName := args[0]
	mgr := service.New(dryRun)

	svc, err := mgr.GetStatus(serviceName)
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	fmt.Printf("Service: %s\n", svc.Name)
	fmt.Printf("Active: %v\n", svc.Active)
	fmt.Printf("Enabled: %v\n", svc.Enabled)
	fmt.Printf("Failed: %v\n", svc.Failed)

	return nil
}

func runServiceEnable(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	logger := ux.NewLogger(verbose)
	mgr := service.New(dryRun)

	for _, name := range args {
		if dryRun {
			logger.DryRun("Would enable: %s", name)
			continue
		}

		spinner := ux.NewSpinner(fmt.Sprintf("Enabling %s...", name))
		spinner.Start()

		if err := mgr.Enable(name); err != nil {
			spinner.StopWithError(fmt.Sprintf("Failed to enable %s", name))
			return err
		}
		spinner.StopWithSuccess(fmt.Sprintf("Enabled %s", name))
	}

	return nil
}

func runServiceDisable(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	logger := ux.NewLogger(verbose)
	mgr := service.New(dryRun)

	for _, name := range args {
		if dryRun {
			logger.DryRun("Would disable: %s", name)
			continue
		}

		spinner := ux.NewSpinner(fmt.Sprintf("Disabling %s...", name))
		spinner.Start()

		if err := mgr.Disable(name); err != nil {
			spinner.StopWithError(fmt.Sprintf("Failed to disable %s", name))
			return err
		}
		spinner.StopWithSuccess(fmt.Sprintf("Disabled %s", name))
	}

	return nil
}

func runServiceStart(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	logger := ux.NewLogger(verbose)
	mgr := service.New(dryRun)

	for _, name := range args {
		if dryRun {
			logger.DryRun("Would start: %s", name)
			continue
		}

		spinner := ux.NewSpinner(fmt.Sprintf("Starting %s...", name))
		spinner.Start()

		if err := mgr.Start(name); err != nil {
			spinner.StopWithError(fmt.Sprintf("Failed to start %s", name))
			return err
		}
		spinner.StopWithSuccess(fmt.Sprintf("Started %s", name))
	}

	return nil
}

func runServiceStop(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	logger := ux.NewLogger(verbose)
	mgr := service.New(dryRun)

	for _, name := range args {
		if dryRun {
			logger.DryRun("Would stop: %s", name)
			continue
		}

		spinner := ux.NewSpinner(fmt.Sprintf("Stopping %s...", name))
		spinner.Start()

		if err := mgr.Stop(name); err != nil {
			spinner.StopWithError(fmt.Sprintf("Failed to stop %s", name))
			return err
		}
		spinner.StopWithSuccess(fmt.Sprintf("Stopped %s", name))
	}

	return nil
}

func runServiceRestart(cmd *cobra.Command, args []string) error {
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}

	logger := ux.NewLogger(verbose)
	mgr := service.New(dryRun)

	for _, name := range args {
		if dryRun {
			logger.DryRun("Would restart: %s", name)
			continue
		}

		spinner := ux.NewSpinner(fmt.Sprintf("Restarting %s...", name))
		spinner.Start()

		if err := mgr.Restart(name); err != nil {
			spinner.StopWithError(fmt.Sprintf("Failed to restart %s", name))
			return err
		}
		spinner.StopWithSuccess(fmt.Sprintf("Restarted %s", name))
	}

	return nil
}

func runServiceList(cmd *cobra.Command, args []string) error {
	pattern := ""
	if len(args) > 0 {
		pattern = args[0]
	}

	services, err := service.ListServices(pattern)
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	if len(services) == 0 {
		fmt.Println("No services found")
		return nil
	}

	// Print table
	headers := []string{"Name", "Active", "Enabled"}
	rows := make([][]string, len(services))
	for i, svc := range services {
		rows[i] = []string{
			strings.TrimSuffix(svc.Name, ".service"),
			statusText(svc.Active),
			statusText(svc.Enabled),
		}
	}

	ux.PrintTable(headers, rows)
	return nil
}

func statusText(active bool) string {
	if active {
		return "\x1b[32m●\x1b[0m active"
	}
	return "\x1b[31m○\x1b[0m inactive"
}
