package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/UPwith-me/Container-Maker/pkg/remote"
	"github.com/UPwith-me/Container-Maker/pkg/userconfig"
	"github.com/spf13/cobra"
)

var remoteForwardCmd = &cobra.Command{
	Use:   "forward <local-port>[:<remote-port>] [more ports...]",
	Short: "Forward local ports to remote container",
	Long: `Forward one or more local ports to a remote container.

If only one port is specified, it's used for both local and remote.
Use local:remote syntax to specify different ports.

EXAMPLES:
  cm remote forward 8080              # Forward localhost:8080 to remote:8080
  cm remote forward 3000:8000         # Forward localhost:3000 to remote:8000
  cm remote forward 8080 3000 5432    # Forward multiple ports
  cm remote forward --auto            # Auto-detect and forward all exposed ports`,
	RunE: runRemoteForward,
}

var (
	forwardAuto bool
)

func init() {
	remoteForwardCmd.Flags().BoolVar(&forwardAuto, "auto", false, "Auto-detect and forward all exposed container ports")
}

func runRemoteForward(cmd *cobra.Command, args []string) error {
	host, err := getActiveRemoteHost()
	if err != nil {
		return err
	}

	// Get container name
	containerName := remoteContainerName
	if containerName == "" {
		containerName, err = detectRemoteContainer(host)
		if err != nil {
			return err
		}
	}

	pf := remote.NewPortForwarder(host)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle auto-detect
	if forwardAuto {
		fmt.Printf("🔍 Auto-detecting ports from container '%s'...\n", containerName)
		ports, err := pf.AutoDetectPorts(ctx, containerName)
		if err != nil {
			return fmt.Errorf("auto-detect failed: %w", err)
		}
		if len(ports) == 0 {
			fmt.Println("No exposed ports found in container")
			return nil
		}
		fmt.Printf("✅ Forwarding %d port(s)\n", len(ports))
	} else if len(args) == 0 {
		return fmt.Errorf("specify ports to forward, or use --auto")
	} else {
		// Parse port arguments
		for _, arg := range args {
			local, remote, err := parsePortSpec(arg)
			if err != nil {
				return fmt.Errorf("invalid port spec '%s': %w", arg, err)
			}
			if err := pf.ForwardPort(ctx, local, remote); err != nil {
				fmt.Printf("Warning: %v\n", err)
			}
		}
	}

	fmt.Println()
	pf.PrintStatus()
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop forwarding...")

	// Wait for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\n🔌 Stopping port forwards...")
	pf.StopAll()

	return nil
}

func parsePortSpec(spec string) (local, remote int, err error) {
	parts := strings.Split(spec, ":")
	if len(parts) == 1 {
		port, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, err
		}
		return port, port, nil
	}
	if len(parts) == 2 {
		local, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, err
		}
		remote, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}
		return local, remote, nil
	}
	return 0, 0, fmt.Errorf("invalid format, use port or local:remote")
}

// Container management commands
var remoteContainerCmd = &cobra.Command{
	Use:   "container",
	Short: "Manage remote containers",
	Long:  `Create, start, stop, and manage containers on remote hosts.`,
}

var remoteContainerCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new container on remote host",
	Long: `Create a new development container on the remote host.

Uses the current project's devcontainer.json if available,
or creates a basic container with the specified image.

EXAMPLES:
  cm remote container create                   # From devcontainer.json
  cm remote container create --image python:3.11
  cm remote container create --name my-dev --image golang:1.22`,
	RunE: runRemoteContainerCreate,
}

var (
	containerImage string
	containerGPU   bool
)

func init() {
	remoteContainerCreateCmd.Flags().StringVar(&containerImage, "image", "", "Container image to use")
	remoteContainerCreateCmd.Flags().BoolVar(&containerGPU, "gpu", false, "Enable GPU support")
}

func runRemoteContainerCreate(cmd *cobra.Command, args []string) error {
	host, err := getActiveRemoteHost()
	if err != nil {
		return err
	}

	cm := remote.NewContainerManager(host)
	ctx := context.Background()

	// Try to use devcontainer.json first
	configPath := ".devcontainer/devcontainer.json"
	if _, err := os.Stat(configPath); err == nil && containerImage == "" {
		cwd, _ := os.Getwd()
		fmt.Println("📦 Creating container from devcontainer.json...")
		return cm.CreateFromDevcontainer(ctx, cwd, configPath)
	}

	// Create with specified or default image
	image := containerImage
	if image == "" {
		image = "mcr.microsoft.com/devcontainers/base:ubuntu"
	}

	name := remoteContainerName
	if name == "" {
		cwd, _ := os.Getwd()
		name = "cm-" + strings.ToLower(strings.ReplaceAll(cwd[strings.LastIndex(cwd, string(os.PathSeparator))+1:], " ", "-"))
	}

	cfg := &remote.ContainerConfig{
		Name:  name,
		Image: image,
		GPU:   containerGPU,
	}

	return cm.CreateContainer(ctx, cfg)
}

var remoteContainerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List containers on remote host",
	RunE: func(cmd *cobra.Command, args []string) error {
		host, err := getActiveRemoteHost()
		if err != nil {
			return err
		}

		cm := remote.NewContainerManager(host)
		containers, err := cm.ListContainers(context.Background())
		if err != nil {
			return err
		}

		if len(containers) == 0 {
			fmt.Println("No Container-Maker managed containers found")
			return nil
		}

		fmt.Println("📦 CM-Managed Containers:")
		for _, c := range containers {
			fmt.Printf("  • %s\n", c)
		}
		return nil
	},
}

var remoteContainerStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a remote container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		host, err := getActiveRemoteHost()
		if err != nil {
			return err
		}

		cm := remote.NewContainerManager(host)
		return cm.StopContainer(context.Background(), args[0])
	},
}

var remoteContainerRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a remote container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		host, err := getActiveRemoteHost()
		if err != nil {
			return err
		}

		cm := remote.NewContainerManager(host)
		return cm.RemoveContainer(context.Background(), args[0], true)
	},
}

// Docker context command
var remoteContextCmd = &cobra.Command{
	Use:   "context [name]",
	Short: "Use Docker context for remote operations",
	Long: `Configure Docker to use a remote host via Docker context.

This allows all Docker commands to transparently operate on the remote host.

EXAMPLES:
  cm remote context            # Use active remote's context
  cm remote context myserver   # Use specific remote's context
  cm remote context --reset    # Reset to default context`,
	RunE: runRemoteContext,
}

var contextReset bool

func init() {
	remoteContextCmd.Flags().BoolVar(&contextReset, "reset", false, "Reset to default Docker context")
}

func runRemoteContext(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if contextReset {
		rc := remote.NewContext("default", "")
		return rc.ResetContext(ctx)
	}

	var host string
	var err error

	if len(args) > 0 {
		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}
		h, ok := cfg.RemoteHosts[args[0]]
		if !ok {
			return fmt.Errorf("remote host '%s' not found", args[0])
		}
		host = h
	} else {
		host, err = getActiveRemoteHost()
		if err != nil {
			return err
		}
	}

	rc := remote.NewContext("remote", host)

	// Test connection first
	if err := rc.TestConnection(ctx); err != nil {
		return err
	}

	return rc.UseDockerContext(ctx)
}

func getActiveRemoteHost() (string, error) {
	cfg, err := userconfig.Load()
	if err != nil {
		return "", err
	}

	if cfg.ActiveRemote == "" {
		return "", fmt.Errorf("no active remote host. Use 'cm remote use <name>' to set one")
	}

	host, ok := cfg.RemoteHosts[cfg.ActiveRemote]
	if !ok {
		return "", fmt.Errorf("active remote '%s' not found in config", cfg.ActiveRemote)
	}

	return host, nil
}

func init() {
	// Container subcommands
	remoteContainerCmd.AddCommand(remoteContainerCreateCmd)
	remoteContainerCmd.AddCommand(remoteContainerListCmd)
	remoteContainerCmd.AddCommand(remoteContainerStopCmd)
	remoteContainerCmd.AddCommand(remoteContainerRemoveCmd)

	// Add to remote command (done in remote.go init)
}
