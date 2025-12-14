package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/UPwith-me/Container-Maker/pkg/userconfig"
	"github.com/spf13/cobra"
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Manage anonymous usage statistics",
	Long: `Manage anonymous usage statistics collection.

Container-Maker can collect anonymous usage data to help improve the product.
This is DISABLED by default and requires explicit opt-in.

No personal information is collected. Only:
- Command usage counts
- Template popularity
- Error frequencies

Examples:
  cm analytics enable   # Opt-in to analytics
  cm analytics disable  # Opt-out
  cm analytics status   # Check current status`,
}

var analyticsEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable anonymous usage statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			cfg = &userconfig.UserConfig{}
		}

		cfg.Analytics.Enabled = true

		// Generate session ID if not exists
		if cfg.Analytics.SessionID == "" {
			cfg.Analytics.SessionID = generateSessionID()
		}

		if err := userconfig.Save(cfg); err != nil {
			return err
		}

		fmt.Println("✅ Anonymous usage statistics enabled")
		fmt.Println("   Thank you for helping improve Container-Maker!")
		fmt.Println()
		fmt.Println("📊 What we collect:")
		fmt.Println("   - Command usage counts")
		fmt.Println("   - Template popularity")
		fmt.Println("   - Error frequencies")
		fmt.Println()
		fmt.Println("🔒 What we DON'T collect:")
		fmt.Println("   - Personal information")
		fmt.Println("   - Project names or contents")
		fmt.Println("   - IP addresses")
		return nil
	},
}

var analyticsDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable anonymous usage statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			cfg = &userconfig.UserConfig{}
		}

		cfg.Analytics.Enabled = false

		if err := userconfig.Save(cfg); err != nil {
			return err
		}

		fmt.Println("✅ Anonymous usage statistics disabled")
		return nil
	},
}

var analyticsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show analytics status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		fmt.Println("📊 Analytics Status")
		fmt.Println()
		if cfg.Analytics.Enabled {
			fmt.Println("  Status: ✅ Enabled")
			fmt.Printf("  Session: %s\n", cfg.Analytics.SessionID)
		} else {
			fmt.Println("  Status: ❌ Disabled (opt-in required)")
		}
		fmt.Println()
		fmt.Println("  Enable with:  cm analytics enable")
		fmt.Println("  Disable with: cm analytics disable")

		return nil
	},
}

func generateSessionID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func init() {
	analyticsCmd.AddCommand(analyticsEnableCmd)
	analyticsCmd.AddCommand(analyticsDisableCmd)
	analyticsCmd.AddCommand(analyticsStatusCmd)
	rootCmd.AddCommand(analyticsCmd)
}
