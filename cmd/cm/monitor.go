package main

import (
	"fmt"

	"github.com/UPwith-me/Container-Maker/pkg/monitor"
	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Real-time container monitoring dashboard",
	Long: `Open a real-time monitoring dashboard for all containers.

The dashboard shows:
  • Container list with status indicators
  • CPU and memory usage with visual bars
  • Network I/O statistics
  • Block I/O statistics

KEYBOARD SHORTCUTS
  ↑/↓     Navigate containers
  r       Refresh data
  s       Start selected container
  x       Stop selected container
  l       Toggle logs view
  Enter   Open shell in selected container
  q       Quit dashboard

EXAMPLES
  cm monitor                  # Open dashboard for all containers
  cm status                   # Alias for cm monitor`,
	Aliases: []string{"status", "dashboard"},
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("📊 Starting Container-Maker Monitor...")
		fmt.Println("   Press 'q' to quit, '?' for help")
		fmt.Println()

		if err := monitor.RunDashboard(); err != nil {
			return fmt.Errorf("dashboard error: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)
}
