package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/UPwith-me/Container-Maker/pkg/sync"
	"github.com/UPwith-me/Container-Maker/pkg/userconfig"
	"github.com/spf13/cobra"
)

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote container hosts",
	Long: `Connect to remote Docker/Podman hosts via SSH.

This allows you to run containers on remote servers while
developing locally.

Examples:
  cm remote add myserver user@192.168.1.100
  cm remote list
  cm remote use myserver
  cm remote shell myserver`,
}

var remoteAddCmd = &cobra.Command{
	Use:   "add <name> <ssh-host>",
	Short: "Add a remote host",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		sshHost := args[1]

		cfg, err := userconfig.Load()
		if err != nil {
			cfg = &userconfig.UserConfig{}
		}

		if cfg.RemoteHosts == nil {
			cfg.RemoteHosts = make(map[string]string)
		}

		cfg.RemoteHosts[name] = sshHost

		if err := userconfig.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("✅ Added remote host '%s' -> %s\n", name, sshHost)
		fmt.Println("\n💡 Test connection with: cm remote test", name)
		return nil
	},
}

var remoteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List remote hosts",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		fmt.Println("📡 Remote Hosts")
		fmt.Println()

		if len(cfg.RemoteHosts) == 0 {
			fmt.Println("  No remote hosts configured.")
			fmt.Println("  Add one with: cm remote add <name> <ssh-host>")
			return nil
		}

		for name, host := range cfg.RemoteHosts {
			active := ""
			if name == cfg.ActiveRemote {
				active = " (active)"
			}
			fmt.Printf("  %s -> %s%s\n", name, host, active)
		}

		return nil
	},
}

var remoteUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set active remote host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		if _, ok := cfg.RemoteHosts[name]; !ok {
			return fmt.Errorf("remote host '%s' not found", name)
		}

		cfg.ActiveRemote = name
		if err := userconfig.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("✅ Now using remote host '%s'\n", name)
		fmt.Println("💡 Run 'cm shell' to connect to remote container")
		return nil
	},
}

var remoteTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Test connection to remote host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		host, ok := cfg.RemoteHosts[name]
		if !ok {
			return fmt.Errorf("remote host '%s' not found", name)
		}

		fmt.Printf("🔍 Testing connection to %s...\n", host)

		// Test SSH connection
		sshCmd := exec.Command("ssh", "-o", "ConnectTimeout=5", host, "docker", "version", "--format", "{{.Server.Version}}")
		output, err := sshCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("❌ Connection failed: %v\n", err)
			fmt.Println("💡 Make sure SSH key authentication is set up")
			return nil
		}

		version := strings.TrimSpace(string(output))
		fmt.Printf("✅ Connected! Docker version: %s\n", version)
		return nil
	},
}

var remoteShellCmd = &cobra.Command{
	Use:   "shell [name]",
	Short: "Open shell on remote container",
	Long: `Open an interactive shell in a remote container.

If no container is running, this command will list available containers
and prompt you to select one, or optionally create a new one.

Examples:
  cm remote shell                    # Use active remote, auto-detect container
  cm remote shell myserver           # Specify remote host
  cm remote shell --container myapp  # Specify container name`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		name := cfg.ActiveRemote
		if len(args) > 0 {
			name = args[0]
		}

		if name == "" {
			return fmt.Errorf("no remote host specified. Use 'cm remote use <name>' or provide name as argument")
		}

		host, ok := cfg.RemoteHosts[name]
		if !ok {
			return fmt.Errorf("remote host '%s' not found", name)
		}

		// Get container name from flag or auto-detect
		containerName := remoteContainerName
		if containerName == "" {
			// Auto-detect: list CM-managed containers on remote
			containerName, err = detectRemoteContainer(host)
			if err != nil {
				return err
			}
		}

		fmt.Printf("🚀 Connecting to %s (container: %s)...\n", host, containerName)

		// Determine shell
		shell := "/bin/bash"
		// Try to detect if bash is available, fallback to sh
		testCmd := exec.Command("ssh", "-o", "ConnectTimeout=5", host,
			"docker", "exec", containerName, "which", "bash")
		if err := testCmd.Run(); err != nil {
			shell = "/bin/sh"
		}

		// SSH to remote and run docker exec
		sshCmd := exec.CommandContext(context.Background(), "ssh", "-t", host,
			"docker", "exec", "-it", containerName, shell)
		sshCmd.Stdin = os.Stdin
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		return sshCmd.Run()
	},
}

var remoteContainerName string

// detectRemoteContainer finds a CM-managed container on the remote host
func detectRemoteContainer(host string) (string, error) {
	// List containers with CM labels
	listCmd := exec.Command("ssh", "-o", "ConnectTimeout=10", host,
		"docker", "ps", "--filter", "label=cm.managed_by=container-maker",
		"--format", "{{.Names}}")
	output, err := listCmd.Output()
	if err != nil {
		// Fallback: list all running containers
		listCmd = exec.Command("ssh", "-o", "ConnectTimeout=10", host,
			"docker", "ps", "--format", "{{.Names}}")
		output, err = listCmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to list containers: %w", err)
		}
	}

	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	containers = filterEmpty(containers)

	if len(containers) == 0 {
		return "", fmt.Errorf("no running containers found on remote host.\n" +
			"💡 Start a container first, or use 'cm remote shell --container <name>'")
	}

	if len(containers) == 1 {
		return containers[0], nil
	}

	// Multiple containers: prefer cm-prefixed ones
	for _, c := range containers {
		if strings.HasPrefix(c, "cm-") {
			fmt.Printf("📦 Found CM container: %s\n", c)
			return c, nil
		}
	}

	// Multiple non-CM containers: prompt user
	fmt.Println("📦 Multiple containers found:")
	for i, c := range containers {
		fmt.Printf("   %d. %s\n", i+1, c)
	}
	fmt.Print("Select container [1]: ")

	var choice string
	_, _ = fmt.Scanln(&choice)

	if choice == "" {
		return containers[0], nil
	}

	var idx int
	if _, err := fmt.Sscanf(choice, "%d", &idx); err == nil && idx >= 1 && idx <= len(containers) {
		return containers[idx-1], nil
	}

	// Treat as container name
	return choice, nil
}

func filterEmpty(s []string) []string {
	var result []string
	for _, str := range s {
		if strings.TrimSpace(str) != "" {
			result = append(result, str)
		}
	}
	return result
}

var remoteRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a remote host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		if _, ok := cfg.RemoteHosts[name]; !ok {
			return fmt.Errorf("remote host '%s' not found", name)
		}

		delete(cfg.RemoteHosts, name)
		if cfg.ActiveRemote == name {
			cfg.ActiveRemote = ""
		}

		if err := userconfig.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("✅ Removed remote host '%s'\n", name)
		return nil
	},
}

// Sync subcommand group
var remoteSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "File synchronization with remote host",
	Long: `Manage file synchronization between local and remote host.

File sync uses rsync over SSH to efficiently synchronize your local
project files with the remote development container.

Examples:
  cm remote sync start myserver     # Start syncing to remote
  cm remote sync stop               # Stop sync daemon
  cm remote sync push               # One-time push to remote
  cm remote sync pull               # One-time pull from remote`,
}

var syncRemotePath string

var remoteSyncStartCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start file sync daemon",
	Long: `Start continuous file synchronization with remote host.

This will:
1. Perform an initial full sync to remote
2. Watch for local file changes
3. Automatically sync changes to remote

The sync is one-way (local -> remote) by default.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		name := cfg.ActiveRemote
		if len(args) > 0 {
			name = args[0]
		}

		if name == "" {
			return fmt.Errorf("no remote host specified")
		}

		host, ok := cfg.RemoteHosts[name]
		if !ok {
			return fmt.Errorf("remote host '%s' not found", name)
		}

		// Get current directory
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		// Determine remote path
		remotePath := syncRemotePath
		if remotePath == "" {
			projectName := filepath.Base(cwd)
			remotePath = fmt.Sprintf("/workspace/%s", projectName)
		}

		fmt.Printf("📂 Local:  %s\n", cwd)
		fmt.Printf("📡 Remote: %s:%s\n", host, remotePath)
		fmt.Println()

		// Create syncer
		syncer, err := sync.New(sync.SyncConfig{
			LocalPath:  cwd,
			RemoteHost: host,
			RemotePath: remotePath,
		})
		if err != nil {
			return err
		}

		// Handle Ctrl+C
		ctx, cancel := context.WithCancel(context.Background())
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\n👋 Stopping sync...")
			cancel()
		}()

		return syncer.Start(ctx)
	},
}

var remoteSyncPushCmd = &cobra.Command{
	Use:   "push [name]",
	Short: "One-time sync local to remote",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		name := cfg.ActiveRemote
		if len(args) > 0 {
			name = args[0]
		}

		if name == "" {
			return fmt.Errorf("no remote host specified")
		}

		host, ok := cfg.RemoteHosts[name]
		if !ok {
			return fmt.Errorf("remote host '%s' not found", name)
		}

		cwd, _ := os.Getwd()
		remotePath := syncRemotePath
		if remotePath == "" {
			projectName := filepath.Base(cwd)
			remotePath = fmt.Sprintf("/workspace/%s", projectName)
		}

		fmt.Printf("⬆️  Pushing to %s:%s...\n", host, remotePath)

		syncer, err := sync.New(sync.SyncConfig{
			LocalPath:  cwd,
			RemoteHost: host,
			RemotePath: remotePath,
		})
		if err != nil {
			return err
		}

		if err := syncer.SyncToRemote(); err != nil {
			return err
		}

		fmt.Println("✅ Push complete!")
		return nil
	},
}

var remoteSyncPullCmd = &cobra.Command{
	Use:   "pull [name]",
	Short: "One-time sync remote to local",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := userconfig.Load()
		if err != nil {
			return err
		}

		name := cfg.ActiveRemote
		if len(args) > 0 {
			name = args[0]
		}

		if name == "" {
			return fmt.Errorf("no remote host specified")
		}

		host, ok := cfg.RemoteHosts[name]
		if !ok {
			return fmt.Errorf("remote host '%s' not found", name)
		}

		cwd, _ := os.Getwd()
		remotePath := syncRemotePath
		if remotePath == "" {
			projectName := filepath.Base(cwd)
			remotePath = fmt.Sprintf("/workspace/%s", projectName)
		}

		fmt.Printf("⬇️  Pulling from %s:%s...\n", host, remotePath)

		syncer, err := sync.New(sync.SyncConfig{
			LocalPath:  cwd,
			RemoteHost: host,
			RemotePath: remotePath,
		})
		if err != nil {
			return err
		}

		if err := syncer.SyncFromRemote(); err != nil {
			return err
		}

		fmt.Println("✅ Pull complete!")
		return nil
	},
}

func init() {
	remoteSyncStartCmd.Flags().StringVar(&syncRemotePath, "remote-path", "", "Remote directory path (default: /workspace/<project>)")
	remoteSyncPushCmd.Flags().StringVar(&syncRemotePath, "remote-path", "", "Remote directory path")
	remoteSyncPullCmd.Flags().StringVar(&syncRemotePath, "remote-path", "", "Remote directory path")

	// Add --container flag to shell command
	remoteShellCmd.Flags().StringVarP(&remoteContainerName, "container", "c", "", "Container name to connect to")

	remoteSyncCmd.AddCommand(remoteSyncStartCmd)
	remoteSyncCmd.AddCommand(remoteSyncPushCmd)
	remoteSyncCmd.AddCommand(remoteSyncPullCmd)

	remoteCmd.AddCommand(remoteAddCmd)
	remoteCmd.AddCommand(remoteListCmd)
	remoteCmd.AddCommand(remoteUseCmd)
	remoteCmd.AddCommand(remoteTestCmd)
	remoteCmd.AddCommand(remoteShellCmd)
	remoteCmd.AddCommand(remoteRemoveCmd)
	remoteCmd.AddCommand(remoteSyncCmd)
	remoteCmd.AddCommand(remoteForwardCmd)
	remoteCmd.AddCommand(remoteContainerCmd)
	remoteCmd.AddCommand(remoteContextCmd)
	rootCmd.AddCommand(remoteCmd)
}
