package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/debian-composer/debian-composer-go/internal/registry"
	"github.com/spf13/cobra"
)

var (
	registryConfigPath string
	registryCachePath  string
)

func init() {
	home, _ := os.UserHomeDir()
	registryConfigPath = filepath.Join(home, ".config", "debian-composer", "registries.yaml")
	registryCachePath = filepath.Join(home, ".cache", "debian-composer", "registries")

	cmd := &cobra.Command{Use: "registry", Short: "Manage remote recipe registries"}
	cmd.PersistentFlags().StringVar(&registryConfigPath, "config", registryConfigPath, "Registry source configuration")
	cmd.PersistentFlags().StringVar(&registryCachePath, "cache", registryCachePath, "Registry cache directory")
	cmd.AddCommand(registryListCmd(), registryAddCmd(), registryRemoveCmd(), registryToggleCmd(true), registryToggleCmd(false), registryUpdateCmd(), registrySearchCmd(), registryInstallCmd(), registryCacheCmd())
	rootCmd.AddCommand(cmd)
}

func registryListCmd() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List configured registries", RunE: func(cmd *cobra.Command, args []string) error {
		sources, err := registry.LoadSources(registryConfigPath)
		if err != nil {
			return err
		}
		if len(sources) == 0 {
			fmt.Println("No remote registries configured.")
			return nil
		}
		fmt.Printf("%-20s %-8s %-8s %s\n", "NAME", "ENABLED", "PRIORITY", "URL")
		for _, source := range sources {
			fmt.Printf("%-20s %-8t %-8d %s\n", source.Name, source.Enabled, source.Priority, source.URL)
		}
		return nil
	}}
}

func registryAddCmd() *cobra.Command {
	var priority int
	var disabled, allowHTTP bool
	cmd := &cobra.Command{Use: "add NAME URL", Args: cobra.ExactArgs(2), Short: "Add a remote registry", RunE: func(cmd *cobra.Command, args []string) error {
		return registry.AddSource(registryConfigPath, registry.Source{Name: args[0], URL: args[1], Enabled: !disabled, Priority: priority}, allowHTTP)
	}}
	cmd.Flags().IntVar(&priority, "priority", 50, "Registry priority")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Add in disabled state")
	cmd.Flags().BoolVar(&allowHTTP, "allow-http", false, "Allow insecure HTTP (development only)")
	return cmd
}

func registryRemoveCmd() *cobra.Command {
	return &cobra.Command{Use: "remove NAME", Args: cobra.ExactArgs(1), Short: "Remove a registry", RunE: func(cmd *cobra.Command, args []string) error {
		return registry.RemoveSource(registryConfigPath, args[0])
	}}
}

func registryToggleCmd(enabled bool) *cobra.Command {
	verb := "disable"
	if enabled {
		verb = "enable"
	}
	return &cobra.Command{Use: verb + " NAME", Args: cobra.ExactArgs(1), Short: strings.Title(verb) + " a registry", RunE: func(cmd *cobra.Command, args []string) error {
		return registry.SetSourceEnabled(registryConfigPath, args[0], enabled)
	}}
}

func registryUpdateCmd() *cobra.Command {
	var timeout time.Duration
	var allowHTTP bool
	cmd := &cobra.Command{Use: "update", Short: "Refresh registry indexes", RunE: func(cmd *cobra.Command, args []string) error {
		sources, err := registry.LoadSources(registryConfigPath)
		if err != nil {
			return err
		}
		client := registry.NewClient(timeout)
		client.AllowHTTP = allowHTTP
		cache := registry.Cache{Dir: registryCachePath}
		for _, source := range sources {
			if !source.Enabled {
				continue
			}
			index, metadata, err := client.FetchIndex(cmd.Context(), source, cache.Metadata(source.Name))
			if errors.Is(err, registry.ErrNotModified) {
				fmt.Printf("✓ %s is current\n", source.Name)
				continue
			}
			if err != nil {
				fmt.Printf("✗ %s: %v\n", source.Name, err)
				continue
			}
			if err := cache.Write(source.Name, index, metadata); err != nil {
				return err
			}
			fmt.Printf("✓ Updated %s (%d recipes)\n", source.Name, len(index.Recipes))
		}
		return nil
	}}
	cmd.Flags().DurationVar(&timeout, "timeout", 15*time.Second, "HTTP timeout")
	cmd.Flags().BoolVar(&allowHTTP, "allow-http", false, "Allow insecure HTTP")
	return cmd
}

func registrySearchCmd() *cobra.Command {
	return &cobra.Command{Use: "search [TERM]", Args: cobra.MaximumNArgs(1), Short: "Search cached remote recipes", RunE: func(cmd *cobra.Command, args []string) error {
		term := ""
		if len(args) == 1 {
			term = args[0]
		}
		sources, err := registry.LoadSources(registryConfigPath)
		if err != nil {
			return err
		}
		cache := registry.Cache{Dir: registryCachePath}
		type result struct {
			Registry string
			Recipe   registry.RemoteRecipe
		}
		var results []result
		for _, source := range sources {
			if !source.Enabled {
				continue
			}
			index, _, err := cache.Read(source.Name)
			if err != nil {
				continue
			}
			for _, recipe := range registry.SearchRemote(map[string]*registry.RemoteIndex{source.Name: index}, term) {
				results = append(results, result{source.Name, recipe})
			}
		}
		sort.Slice(results, func(i, j int) bool { return results[i].Recipe.Name < results[j].Recipe.Name })
		for _, item := range results {
			fmt.Printf("%-18s %-28s %-10s %s\n", item.Registry, item.Recipe.Name, item.Recipe.Version, item.Recipe.Description)
		}
		if len(results) == 0 {
			fmt.Println("No matching remote recipes found. Run 'registry update' first.")
		}
		return nil
	}}
}

func registryInstallCmd() *cobra.Command {
	var destination string
	var force, allowHTTP bool
	cmd := &cobra.Command{Use: "install REGISTRY/RECIPE", Args: cobra.ExactArgs(1), Short: "Download a verified recipe into the kitchen", RunE: func(cmd *cobra.Command, args []string) error {
		parts := strings.SplitN(args[0], "/", 2)
		if len(parts) != 2 {
			return errors.New("use REGISTRY/RECIPE")
		}
		cache := registry.Cache{Dir: registryCachePath}
		index, _, err := cache.Read(parts[0])
		if err != nil {
			return fmt.Errorf("read cache: %w", err)
		}
		var selected *registry.RemoteRecipe
		for i := range index.Recipes {
			if index.Recipes[i].Name == parts[1] {
				selected = &index.Recipes[i]
				break
			}
		}
		if selected == nil {
			return registry.ErrRecipeNotFound
		}
		if destination == "" {
			destination = filepath.Join(kitchen, "recipes", selected.Name+".yaml")
		}
		client := registry.NewClient(30 * time.Second)
		client.AllowHTTP = allowHTTP
		if err := client.DownloadRecipe(context.Background(), *selected, destination, force); err != nil {
			return err
		}
		fmt.Printf("✓ Installed %s to %s\n", selected.Name, destination)
		return nil
	}}
	cmd.Flags().StringVarP(&destination, "destination", "d", "", "Destination recipe path")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing recipe")
	cmd.Flags().BoolVar(&allowHTTP, "allow-http", false, "Allow insecure HTTP")
	return cmd
}

func registryCacheCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "cache", Short: "Manage registry cache"}
	cmd.AddCommand(&cobra.Command{Use: "clear [NAME]", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cache := registry.Cache{Dir: registryCachePath}
		if len(args) == 0 {
			return cache.ClearAll()
		}
		return cache.Clear(args[0])
	}})
	return cmd
}
