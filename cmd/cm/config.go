package main

import (
	"fmt"
	"sort"

	"github.com/UPwith-me/Container-Maker/pkg/userconfig"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage global configuration",
	Long:  `Manage global configuration settings (e.g. AI keys, defaults).`,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	RunE: func(cmd *cobra.Command, args []string) error {
		// We manually construct the list for now based on known keys
		// In a reflection-based system this would be automatic
		keys := []string{
			"skip_welcome",
			"default_backend",
			"ai.enabled",
			"ai.api_base",
			"ai.model",
			"ai.api_key", // We will mask this
			"analytics.enabled",
			"team.org_name",
		}
		sort.Strings(keys)

		fmt.Println("Global Configuration:")
		fmt.Println("---------------------")
		for _, k := range keys {
			val, err := userconfig.Get(k)
			if err != nil {
				continue
			}
			if val == "" {
				val = "(unset)"
			}
			fmt.Printf("%-20s : %s\n", k, val)
		}
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		val, err := userconfig.Get(key)
		if err != nil {
			return err
		}
		if val == "" {
			fmt.Println("(unset)")
		} else {
			fmt.Println(val)
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Example: `  cm config set ai.model gpt-4
  cm config set ai.enabled true`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		val := args[1]
		if err := userconfig.Set(key, val); err != nil {
			return err
		}
		fmt.Printf("✅ Set %s = %s\n", key, val)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
