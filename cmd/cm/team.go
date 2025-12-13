package main

import (
	"fmt"

	"github.com/container-make/cm/pkg/userconfig"
	"github.com/spf13/cobra"
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage team/organization settings",
	Long: `Configure Container-Maker for team usage.

Teams can share DevContainer templates from a centralized repository,
ensuring everyone uses consistent development environments.

Examples:
  cm team set mycompany
  cm team templates https://github.com/mycompany/devcontainers
  cm team info`,
}

var teamSetCmd = &cobra.Command{
	Use:   "set <org-name>",
	Short: "Set organization name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			cfg = &userconfig.UserConfig{}
		}

		cfg.Team.OrgName = args[0]
		if err := userconfig.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("✅ Organization set to '%s'\n", args[0])
		return nil
	},
}

var teamTemplatesCmd = &cobra.Command{
	Use:   "templates <url>",
	Short: "Set team templates repository URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			cfg = &userconfig.UserConfig{}
		}

		cfg.Team.TemplatesURL = args[0]
		if err := userconfig.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("✅ Team templates URL set to: %s\n", args[0])
		fmt.Println("💡 Team templates will be available in 'cm template list'")
		return nil
	},
}

var teamInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show team configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		fmt.Println("🏢 Team Configuration")
		fmt.Println()
		if cfg.Team.OrgName != "" {
			fmt.Printf("  Organization: %s\n", cfg.Team.OrgName)
		} else {
			fmt.Println("  Organization: (not set)")
		}
		if cfg.Team.TemplatesURL != "" {
			fmt.Printf("  Templates:    %s\n", cfg.Team.TemplatesURL)
		} else {
			fmt.Println("  Templates:    (not set)")
		}

		return nil
	},
}

func init() {
	teamCmd.AddCommand(teamSetCmd)
	teamCmd.AddCommand(teamTemplatesCmd)
	teamCmd.AddCommand(teamInfoCmd)
	rootCmd.AddCommand(teamCmd)
}
