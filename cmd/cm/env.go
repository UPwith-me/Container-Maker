package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/environment"
	"github.com/spf13/cobra"
)

var (
	// Flags for env create
	envCreateTemplate string
	envCreateDir      string
	envCreateNoStart  bool
	envCreateForce    bool
	envCreateGPU      []int
	envCreateMemory   string
	envCreateCPU      float64
	envCreateLink     []string

	// Flags for env list
	envListAll    bool
	envListStatus string

	// Flags for env delete
	envDeleteForce bool

	// Flags for env stop
	envStopTimeout int
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage development environments",
	Long: `Manage isolated development environments.

Container-Maker environments provide complete isolation with dedicated 
Docker networks, allowing multiple environments to run simultaneously 
without conflicts.

QUICK START
  cm env create frontend --template node
  cm env create backend --template python
  cm env link frontend backend
  cm env list

LIFECYCLE
  cm env create <name>    Create a new environment
  cm env start <name>     Start a stopped environment
  cm env stop <name>      Stop a running environment
  cm env restart <name>   Restart an environment
  cm env delete <name>    Delete an environment

SWITCHING
  cm env switch <name>    Set the active environment
  cm env status           Show current environment

NETWORKING
  cm env link <a> <b>     Connect two environments
  cm env unlink <a> <b>   Disconnect two environments`,
}

var envCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new environment",
	Long: `Create a new isolated development environment.

The environment will have its own Docker network and can be linked
to other environments for inter-service communication.

EXAMPLES
  # Create from devcontainer.json in current directory
  cm env create myproject

  # Create with a specific template
  cm env create frontend --template node

  # Create with GPU support
  cm env create ml-training --template pytorch --gpu 0,1

  # Create and link to existing environment
  cm env create backend --template python --link frontend`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		mgr, err := environment.NewManager()
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		opts := environment.EnvironmentCreateOptions{
			Name:       name,
			Template:   envCreateTemplate,
			ProjectDir: envCreateDir,
			NoStart:    envCreateNoStart,
			Force:      envCreateForce,
			GPUs:       envCreateGPU,
			Memory:     envCreateMemory,
			CPU:        envCreateCPU,
			LinkTo:     envCreateLink,
		}

		fmt.Printf("🚀 Creating environment '%s'...\n", name)

		env, err := mgr.Create(ctx, opts)
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		fmt.Println()
		fmt.Printf("✅ Environment '%s' created successfully!\n", env.Name)
		fmt.Printf("   ID:      %s\n", env.ID)
		fmt.Printf("   Status:  %s\n", env.Status)
		fmt.Printf("   Network: %s\n", env.NetworkName)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Printf("  cm env switch %s    # Set as active environment\n", env.Name)
		fmt.Printf("  cm shell             # Enter the environment\n")

		return nil
	},
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all environments",
	Long: `List all development environments.

By default, shows all environments with their status, network, and age.

EXAMPLES
  cm env list
  cm env list --all
  cm env list --status running`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := environment.NewManager()
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		ctx := context.Background()

		opts := environment.EnvironmentListOptions{
			All: envListAll,
		}
		if envListStatus != "" {
			opts.Filter.Status = environment.EnvironmentStatus(envListStatus)
		}

		envs, err := mgr.List(ctx, opts)
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		if len(envs) == 0 {
			fmt.Println("No environments found.")
			fmt.Println()
			fmt.Println("Create one with:")
			fmt.Println("  cm env create myproject --template python")
			return nil
		}

		// Get active environment
		active, _ := mgr.GetActive(ctx)
		activeID := ""
		if active != nil {
			activeID = active.ID
		}

		// Print table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  \tNAME\tSTATUS\tNETWORK\tTEMPLATE\tAGE")
		fmt.Fprintln(w, "  \t----\t------\t-------\t--------\t---")

		for _, env := range envs {
			marker := " "
			if env.ID == activeID {
				marker = "*"
			}

			status := statusIcon(env.Status) + " " + string(env.Status)
			template := env.Template
			if template == "" {
				template = "-"
			}
			network := env.NetworkName
			if network == "" {
				network = "-"
			}
			age := formatAge(env.CreatedAt)

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				marker, env.Name, status, network, template, age)
		}
		w.Flush()

		fmt.Println()
		fmt.Printf("Total: %d environments (* = active)\n", len(envs))

		return nil
	},
}

var envSwitchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Set the active environment",
	Long: `Set the active environment.

The active environment is used by default for commands like 'cm shell'.

EXAMPLES
  cm env switch frontend
  cm env switch backend`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		mgr, err := environment.NewManager()
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		ctx := context.Background()

		if err := mgr.Switch(ctx, name); err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		env, _ := mgr.Get(ctx, name)
		fmt.Printf("✅ Switched to environment '%s'\n", env.Name)
		fmt.Printf("   Status: %s\n", env.Status)

		return nil
	},
}

var envStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start an environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		mgr, err := environment.NewManager()
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		ctx := context.Background()

		fmt.Printf("🚀 Starting environment '%s'...\n", name)
		if err := mgr.Start(ctx, name); err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		fmt.Printf("✅ Environment '%s' started\n", name)
		return nil
	},
}

var envStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop an environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		mgr, err := environment.NewManager()
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		ctx := context.Background()

		fmt.Printf("🛑 Stopping environment '%s'...\n", name)
		if err := mgr.Stop(ctx, name, envStopTimeout); err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		fmt.Printf("✅ Environment '%s' stopped\n", name)
		return nil
	},
}

var envRestartCmd = &cobra.Command{
	Use:   "restart <name>",
	Short: "Restart an environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		mgr, err := environment.NewManager()
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		ctx := context.Background()

		fmt.Printf("🔄 Restarting environment '%s'...\n", name)
		if err := mgr.Restart(ctx, name); err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		fmt.Printf("✅ Environment '%s' restarted\n", name)
		return nil
	},
}

var envDeleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Short:   "Delete an environment",
	Aliases: []string{"rm", "remove"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		mgr, err := environment.NewManager()
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		ctx := context.Background()

		// Confirm if not forced
		if !envDeleteForce {
			fmt.Printf("Delete environment '%s'? This cannot be undone. [y/N] ", name)
			var response string
			_, _ = fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		fmt.Printf("🗑️  Deleting environment '%s'...\n", name)
		if err := mgr.Delete(ctx, name, envDeleteForce); err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		fmt.Printf("✅ Environment '%s' deleted\n", name)
		return nil
	},
}

var envLinkCmd = &cobra.Command{
	Use:   "link <env1> <env2>",
	Short: "Link two environments",
	Long: `Link two environments for network communication.

Linked environments can communicate with each other using their 
environment names as hostnames.

EXAMPLE
  cm env link frontend backend
  
Then from frontend, you can access backend at http://backend:PORT`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		env1, env2 := args[0], args[1]

		mgr, err := environment.NewManager()
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		ctx := context.Background()

		fmt.Printf("🔗 Linking '%s' and '%s'...\n", env1, env2)

		e1, err := mgr.Get(ctx, env1)
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		e2, err := mgr.Get(ctx, env2)
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		if err := mgr.Link(ctx, e1.ID, e2.ID, environment.EnvironmentLinkOptions{
			Bidirectional: true,
		}); err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		fmt.Printf("✅ Environments linked!\n")
		fmt.Printf("   From %s: access %s at http://%s:<port>\n", env1, env2, env2)
		fmt.Printf("   From %s: access %s at http://%s:<port>\n", env2, env1, env1)

		return nil
	},
}

var envUnlinkCmd = &cobra.Command{
	Use:   "unlink <env1> <env2>",
	Short: "Unlink two environments",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		env1, env2 := args[0], args[1]

		mgr, err := environment.NewManager()
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		ctx := context.Background()

		e1, err := mgr.Get(ctx, env1)
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		e2, err := mgr.Get(ctx, env2)
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		if err := mgr.Unlink(ctx, e1.ID, e2.ID); err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		fmt.Printf("✅ Environments '%s' and '%s' unlinked\n", env1, env2)
		return nil
	},
}

var envStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show environment status",
	Long: `Show detailed status of an environment.

If no name is given, shows the active environment.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := environment.NewManager()
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		ctx := context.Background()

		var env *environment.Environment
		if len(args) > 0 {
			env, err = mgr.Get(ctx, args[0])
		} else {
			env, err = mgr.GetActive(ctx)
		}

		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		fmt.Printf("Environment: %s\n", env.Name)
		fmt.Printf("ID:          %s\n", env.ID)
		fmt.Printf("Status:      %s %s\n", statusIcon(env.Status), env.Status)
		fmt.Printf("Template:    %s\n", valueOrDash(env.Template))
		fmt.Printf("Project:     %s\n", env.ProjectDir)
		fmt.Printf("Network:     %s\n", valueOrDash(env.NetworkName))
		fmt.Printf("Container:   %s\n", shortID(env.ContainerID))
		fmt.Printf("Image:       %s\n", valueOrDash(env.ImageTag))
		fmt.Printf("Created:     %s\n", env.CreatedAt.Format(time.RFC3339))

		if len(env.LinkedEnvs) > 0 {
			fmt.Printf("Linked to:   %v\n", env.LinkedEnvs)
		}
		if len(env.GPUs) > 0 {
			fmt.Printf("GPUs:        %v\n", env.GPUs)
		}

		return nil
	},
}

var envShellCmd = &cobra.Command{
	Use:   "shell [name]",
	Short: "Open shell in environment",
	Long: `Open an interactive shell in an environment.

If no name is given, uses the active environment.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := environment.NewManager()
		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		ctx := context.Background()

		var env *environment.Environment
		if len(args) > 0 {
			env, err = mgr.Get(ctx, args[0])
		} else {
			env, err = mgr.GetActive(ctx)
		}

		if err != nil {
			fmt.Println(environment.FormatUserError(err))
			return nil
		}

		if env.Status != environment.StatusRunning {
			fmt.Printf("Starting environment '%s'...\n", env.Name)
			if err := mgr.Start(ctx, env.Name); err != nil {
				fmt.Println(environment.FormatUserError(err))
				return nil
			}
			// Refresh env
			env, _ = mgr.Get(ctx, env.Name)
		}

		fmt.Printf("🚀 Entering shell in '%s'...\n", env.Name)

		// Execute docker exec
		execCmd := exec.Command("docker", "exec", "-it", env.ContainerID, "/bin/sh")
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		return execCmd.Run()
	},
}

// Helper functions

func statusIcon(status environment.EnvironmentStatus) string {
	switch status {
	case environment.StatusRunning:
		return "●"
	case environment.StatusStopped:
		return "○"
	case environment.StatusCreating:
		return "◔"
	case environment.StatusError:
		return "✗"
	case environment.StatusPaused:
		return "◑"
	case environment.StatusOrphaned:
		return "?"
	default:
		return "·"
	}
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func valueOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	if id == "" {
		return "-"
	}
	return id
}

func init() {
	// env create flags
	envCreateCmd.Flags().StringVarP(&envCreateTemplate, "template", "t", "", "Template to use")
	envCreateCmd.Flags().StringVarP(&envCreateDir, "dir", "d", "", "Project directory")
	envCreateCmd.Flags().BoolVar(&envCreateNoStart, "no-start", false, "Create but don't start")
	envCreateCmd.Flags().BoolVarP(&envCreateForce, "force", "f", false, "Force recreate if exists")
	envCreateCmd.Flags().IntSliceVar(&envCreateGPU, "gpu", nil, "GPU IDs to allocate")
	envCreateCmd.Flags().StringVar(&envCreateMemory, "memory", "", "Memory limit (e.g., 8g)")
	envCreateCmd.Flags().Float64Var(&envCreateCPU, "cpu", 0, "CPU limit")
	envCreateCmd.Flags().StringSliceVar(&envCreateLink, "link", nil, "Environments to link to")

	// env list flags
	envListCmd.Flags().BoolVarP(&envListAll, "all", "a", false, "Show all environments")
	envListCmd.Flags().StringVar(&envListStatus, "status", "", "Filter by status")

	// env delete flags
	envDeleteCmd.Flags().BoolVarP(&envDeleteForce, "force", "f", false, "Force delete")

	// env stop flags
	envStopCmd.Flags().IntVar(&envStopTimeout, "timeout", 10, "Stop timeout in seconds")

	// Add subcommands
	envCmd.AddCommand(envCreateCmd)
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envSwitchCmd)
	envCmd.AddCommand(envStartCmd)
	envCmd.AddCommand(envStopCmd)
	envCmd.AddCommand(envRestartCmd)
	envCmd.AddCommand(envDeleteCmd)
	envCmd.AddCommand(envLinkCmd)
	envCmd.AddCommand(envUnlinkCmd)
	envCmd.AddCommand(envStatusCmd)
	envCmd.AddCommand(envShellCmd)

	rootCmd.AddCommand(envCmd)
}
