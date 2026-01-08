package main

import (
	"context"
	"fmt"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/workspace"
	"github.com/spf13/cobra"
)

var (
	upBuild   bool
	upNoDeps  bool
	upForce   bool
	upProfile string
	upDetach  bool
)

var upCmd = &cobra.Command{
	Use:   "up [services...]",
	Short: "Start workspace services",
	Long: `Start all or specified services in the workspace.

This command reads cm-workspace.yaml and starts services in dependency order.
Dependencies are automatically started before their dependents.

EXAMPLES
  cm up                     # Start all services
  cm up frontend backend    # Start specific services (+ dependencies)
  cm up --no-deps frontend  # Start without dependencies
  cm up --profile dev       # Start services with 'dev' profile
  cm up --build             # Build images before starting

WORKSPACE FILE
  Create a cm-workspace.yaml to define your services:

  name: my-project
  services:
    frontend:
      template: node
      ports:
        - 3000:3000
    backend:
      template: python
      depends_on:
        - database
    database:
      image: postgres:15`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Find and load workspace config
		ws, err := workspace.Load("")
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			fmt.Println()
			fmt.Println("Create a cm-workspace.yaml to get started:")
			fmt.Println("  cm workspace init")
			return nil
		}

		// Validate
		if err := workspace.Validate(ws); err != nil {
			fmt.Printf("❌ Invalid workspace config: %v\n", err)
			return nil
		}

		// Create orchestrator
		orch, err := workspace.NewOrchestrator(ws)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return nil
		}
		defer orch.Close()

		// Build start options
		opts := workspace.StartOptions{
			Services: args,
			Build:    upBuild,
			NoDeps:   upNoDeps,
			Force:    upForce,
			Profile:  upProfile,
			Detach:   upDetach,
			Timeout:  120,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		return orch.Up(ctx, opts)
	},
}

var (
	downTimeout int
	downRemove  bool
	downVolumes bool
)

var downCmd = &cobra.Command{
	Use:   "down [services...]",
	Short: "Stop workspace services",
	Long: `Stop all or specified services in the workspace.

Services are stopped in reverse dependency order - dependents are stopped
before their dependencies.

EXAMPLES
  cm down                 # Stop all services
  cm down frontend        # Stop specific services (+ dependents)
  cm down --remove        # Stop and remove containers
  cm down --volumes       # Also remove volumes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspace.Load("")
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return nil
		}

		orch, err := workspace.NewOrchestrator(ws)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return nil
		}
		defer orch.Close()

		opts := workspace.StopOptions{
			Services: args,
			Timeout:  downTimeout,
			Remove:   downRemove,
			Volumes:  downVolumes,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		return orch.Down(ctx, opts)
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart [services...]",
	Short: "Restart workspace services",
	Long:  `Restart all or specified services in the workspace.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspace.Load("")
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return nil
		}

		orch, err := workspace.NewOrchestrator(ws)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return nil
		}
		defer orch.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		return orch.Restart(ctx, args)
	},
}

var (
	logsFollow bool
	logsTail   int
)

var logsCmd = &cobra.Command{
	Use:   "logs <service>",
	Short: "View service logs",
	Long: `View logs from a running service.

EXAMPLES
  cm logs backend           # View recent logs
  cm logs backend -f        # Follow logs
  cm logs backend -n 200    # Last 200 lines`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspace.Load("")
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return nil
		}

		orch, err := workspace.NewOrchestrator(ws)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return nil
		}
		defer orch.Close()

		ctx := context.Background()
		if !logsFollow {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
		}

		return orch.Logs(ctx, args[0], logsFollow, logsTail)
	},
}

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running services",
	Long:  `List all services in the workspace and their status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspace.Load("")
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return nil
		}

		orch, err := workspace.NewOrchestrator(ws)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return nil
		}
		defer orch.Close()

		state := orch.Status()

		fmt.Printf("Workspace: %s\n\n", ws.Name)
		fmt.Printf("%-20s %-12s %-20s\n", "SERVICE", "STATUS", "CONTAINER")
		fmt.Printf("%-20s %-12s %-20s\n", "-------", "------", "---------")

		for name := range ws.Services {
			svcState := state.Services[name]
			status := "stopped"
			containerID := "-"
			if svcState != nil {
				status = string(svcState.Status)
				if svcState.ContainerID != "" {
					containerID = svcState.ContainerID[:12]
				}
			}
			fmt.Printf("%-20s %-12s %-20s\n", name, status, containerID)
		}

		return nil
	},
}

func init() {
	// up flags
	upCmd.Flags().BoolVar(&upBuild, "build", false, "Build images before starting")
	upCmd.Flags().BoolVar(&upNoDeps, "no-deps", false, "Don't start dependencies")
	upCmd.Flags().BoolVarP(&upForce, "force", "f", false, "Force recreate containers")
	upCmd.Flags().StringVar(&upProfile, "profile", "", "Activate specific profile")
	upCmd.Flags().BoolVarP(&upDetach, "detach", "d", true, "Run in background")

	// down flags
	downCmd.Flags().IntVar(&downTimeout, "timeout", 10, "Stop timeout in seconds")
	downCmd.Flags().BoolVar(&downRemove, "remove", false, "Remove containers after stopping")
	downCmd.Flags().BoolVar(&downVolumes, "volumes", false, "Remove volumes too")

	// logs flags
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&logsTail, "tail", "n", 100, "Number of lines to show")

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(psCmd)
}
