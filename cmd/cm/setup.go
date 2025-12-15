package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/UPwith-me/Container-Maker/pkg/runtime"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "One-click Docker container environment setup",
	Long: `Intelligently detect your system and recommend the best container runtime installation.

Supported platforms:
  - Windows (Docker Desktop, Rancher Desktop, Podman Desktop)
  - macOS (Docker Desktop, OrbStack, Colima, Podman)
  - Linux (Docker Engine, Podman)
  - WSL (Windows Docker or standalone)

Examples:
  cm setup              # Interactive installation wizard
  cm setup --detect     # Only detect environment
  cm setup --auto       # Auto-install recommended option`,
	RunE: runSetup,
}

var (
	setupDetectOnly bool
	setupAuto       bool
)

func init() {
	setupCmd.Flags().BoolVar(&setupDetectOnly, "detect", false, "Only detect environment, skip installation")
	setupCmd.Flags().BoolVar(&setupAuto, "auto", false, "Auto-install the recommended container runtime")
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Container-Maker Setup Wizard")
	fmt.Println()

	// Detect host
	fmt.Println("🔍 Detecting system environment...")
	host := runtime.DetectHost()
	fmt.Println()
	fmt.Println(host.FormatHostInfo())

	// Check if already installed
	if host.HasDocker || host.HasPodman {
		fmt.Println("✅ Container runtime detected, no installation needed!")
		fmt.Println()

		// Run doctor to verify
		if host.HasDocker {
			fmt.Println("💡 Run 'cm doctor' to check Docker status")
		}
		return nil
	}

	if setupDetectOnly {
		fmt.Println("💡 Use 'cm setup' to install container runtime")
		return nil
	}

	// Get installation options
	options := host.GetInstallOptions()
	if len(options) == 0 {
		fmt.Println("❌ Cannot provide installation recommendations for your system")
		return nil
	}

	// Sort by priority
	sort.Slice(options, func(i, j int) bool {
		return options[i].Priority > options[j].Priority
	})

	// Auto mode: install the first (highest priority) option
	if setupAuto {
		fmt.Printf("🔧 Auto-installing: %s\n", options[0].Name)
		return executeInstall(options[0])
	}

	// Interactive mode
	fmt.Println("📋 Recommended installation options:")
	fmt.Println()

	for i, opt := range options {
		marker := "  "
		if i == 0 {
			marker = "⭐"
		}
		fmt.Printf("%s [%d] %s\n", marker, i+1, opt.Name)
		fmt.Printf("      %s\n", opt.Description)
		fmt.Println()
	}

	fmt.Print("Select option (1-", len(options), ") or 'q' to quit: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "q" || input == "Q" {
		fmt.Println("Cancelled")
		return nil
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(options) {
		fmt.Println("❌ Invalid selection")
		return nil
	}

	return executeInstall(options[choice-1])
}

func executeInstall(opt runtime.InstallOption) error {
	fmt.Println()
	fmt.Printf("🔧 Installing %s...\n", opt.Name)
	fmt.Println()
	fmt.Println("📝 Executing command:")
	fmt.Printf("   %s\n", opt.Command)
	fmt.Println()

	// Confirm
	fmt.Print("Confirm execution? [Y/n] ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "" && input != "y" && input != "yes" {
		fmt.Println("Cancelled")
		return nil
	}

	// Detect shell and execute
	var cmd *exec.Cmd

	switch {
	case isWindows():
		// PowerShell on Windows
		cmd = exec.Command("powershell", "-Command", opt.Command)
	default:
		// Bash on Unix
		cmd = exec.Command("bash", "-c", opt.Command)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Printf("\n❌ Installation failed: %v\n", err)
		fmt.Println()
		fmt.Println("💡 Please try running the command manually or check the error")
		return nil
	}

	fmt.Println()
	fmt.Println("✅ Installation complete!")
	fmt.Println()
	fmt.Println("📋 Next steps:")
	fmt.Println("   1. If Docker Desktop was installed, start the application")
	fmt.Println("   2. Run 'cm doctor' to verify installation")
	fmt.Println("   3. Run 'cm shell' to start using container dev environment")

	if !isWindows() {
		fmt.Println()
		fmt.Println("⚠️  Note: If you added docker user group, re-login or run:")
		fmt.Println("   newgrp docker")
	}

	return nil
}

func isWindows() bool {
	return strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") ||
		strings.Contains(strings.ToLower(os.Getenv("GOOS")), "windows")
}
