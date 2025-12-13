package main

import (
	"fmt"

	"github.com/container-make/cm/pkg/userconfig"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Container-Maker configuration",
	Long: `Manage persistent user configuration for Container-Maker.

Configuration is stored in ~/.cm/config.json

Examples:
  cm config get skip_welcome
  cm config set skip_welcome true
  cm config set ai.api_key sk-xxx
  cm config list`,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		value, err := userconfig.Get(args[0])
		if err != nil {
			return err
		}
		if value == "" {
			fmt.Printf("%s: (not set)\n", args[0])
		} else {
			fmt.Printf("%s: %s\n", args[0], value)
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := userconfig.Set(args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("✅ Set %s = %s\n", args[0], args[1])
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all config values",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		fmt.Println("📋 Container-Maker Configuration")
		fmt.Println()
		fmt.Printf("  skip_welcome:    %v\n", cfg.SkipWelcome)
		fmt.Printf("  default_backend: %s\n", cfg.DefaultBackend)
		fmt.Println()
		fmt.Println("  AI Settings:")
		fmt.Printf("    ai.enabled:  %v\n", cfg.AI.Enabled)
		if cfg.AI.APIKey != "" {
			fmt.Printf("    ai.api_key:  ***hidden***\n")
		} else {
			fmt.Printf("    ai.api_key:  (not set)\n")
		}
		fmt.Printf("    ai.api_base: %s\n", cfg.AI.APIBase)

		return nil
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	rootCmd.AddCommand(configCmd)
}
