package main

import (
	"fmt"
	"strings"

	"github.com/UPwith-me/Container-Maker/pkg/plugin"
	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Long: `Manage Container-Maker plugins.

Plugins extend the functionality of Container-Maker, adding new commands,
lifecycle hooks, and integrations.

COMMANDS
  cm plugin list     List installed and active plugins`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active plugins",
	Run: func(cmd *cobra.Command, args []string) {
		manager := plugin.GetManager()
		plugins := manager.GetPlugins()

		fmt.Printf("Active Plugins: %d\n\n", len(plugins))
		fmt.Printf("%-20s %-20s %-10s %-30s\n", "ID", "NAME", "VERSION", "DESCRIPTION")
		fmt.Println(strings.Repeat("-", 85))

		for _, p := range plugins {
			meta := p.Metadata()
			fmt.Printf("%-20s %-20s %-10s %-30s\n",
				meta.ID,
				truncate(meta.Name, 18),
				meta.Version,
				truncate(meta.Description, 28))
		}
	},
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)
	rootCmd.AddCommand(pluginCmd)
}
