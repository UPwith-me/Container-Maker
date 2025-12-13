package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/container-make/cm/pkg/userconfig"
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

		fmt.Printf("🚀 Connecting to %s...\n", host)

		// SSH to remote and run docker exec
		sshCmd := exec.CommandContext(context.Background(), "ssh", "-t", host, "docker", "exec", "-it", "cm-dev", "/bin/bash")
		sshCmd.Stdin = os.Stdin
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		return sshCmd.Run()
	},
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

func init() {
	remoteCmd.AddCommand(remoteAddCmd)
	remoteCmd.AddCommand(remoteListCmd)
	remoteCmd.AddCommand(remoteUseCmd)
	remoteCmd.AddCommand(remoteTestCmd)
	remoteCmd.AddCommand(remoteShellCmd)
	remoteCmd.AddCommand(remoteRemoveCmd)
	rootCmd.AddCommand(remoteCmd)
}
