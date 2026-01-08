package main

import (
	"fmt"
	"strings"

	"github.com/UPwith-me/Container-Maker/pkg/gpu"
	"github.com/spf13/cobra"
)

var gpuCmd = &cobra.Command{
	Use:   "gpu",
	Short: "GPU detection and management",
	Long: `Detect and manage GPU resources for Container-Maker environments.

This command provides GPU information, allocation status, and management
capabilities for development environments that require GPU acceleration.

COMMANDS
  cm gpu list          List all available GPUs
  cm gpu status        Show GPU status and allocations
  cm gpu allocate      Allocate GPUs for a service
  cm gpu release       Release GPU allocations`,
	Aliases: []string{"gpus"},
}

var gpuListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List available GPUs",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		detector := gpu.NewNVIDIADetector()

		if !detector.IsAvailable() {
			fmt.Println("No NVIDIA GPU drivers detected.")
			fmt.Println("GPU features require nvidia-smi to be available.")
			return nil
		}

		gpus, err := detector.Detect()
		if err != nil {
			return fmt.Errorf("failed to detect GPUs: %w", err)
		}

		if len(gpus) == 0 {
			fmt.Println("No GPUs detected.")
			return nil
		}

		fmt.Printf("Detected %d GPU(s):\n\n", len(gpus))
		fmt.Printf("%-4s %-30s %-10s %-8s %-8s %-10s\n",
			"IDX", "NAME", "VRAM", "TEMP", "UTIL", "STATUS")
		fmt.Println(strings.Repeat("-", 75))

		for _, g := range gpus {
			status := "available"
			if g.Allocated {
				status = g.AllocatedTo
			}

			fmt.Printf("%-4d %-30s %-10s %-8s %-8s %-10s\n",
				g.Index,
				truncateGPU(g.Name, 30),
				gpu.FormatVRAM(g.VRAM),
				fmt.Sprintf("%dC", g.Temperature),
				fmt.Sprintf("%d%%", g.Utilization),
				status)
		}

		return nil
	},
}

var gpuStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show GPU status and allocations",
	RunE: func(cmd *cobra.Command, args []string) error {
		detector := gpu.NewNVIDIADetector()

		if !detector.IsAvailable() {
			fmt.Println("No NVIDIA GPU drivers detected.")
			return nil
		}

		gpus, err := detector.Detect()
		if err != nil {
			return fmt.Errorf("failed to detect GPUs: %w", err)
		}

		fmt.Println("GPU STATUS")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()

		for _, g := range gpus {
			fmt.Printf("GPU %d: %s\n", g.Index, g.Name)
			fmt.Printf("  Vendor:      %s\n", g.Vendor)
			fmt.Printf("  Driver:      %s\n", g.Driver)
			fmt.Printf("  Compute:     %s\n", g.ComputeCap)
			fmt.Printf("  VRAM:        %s / %s (%.1f%%)\n",
				gpu.FormatVRAM(g.VRAMUsed),
				gpu.FormatVRAM(g.VRAM),
				float64(g.VRAMUsed)/float64(g.VRAM)*100)
			fmt.Printf("  Temperature: %dC\n", g.Temperature)
			fmt.Printf("  Power:       %dW / %dW\n", g.PowerUsage, g.PowerLimit)
			fmt.Printf("  Utilization: GPU %d%%, Memory %d%%\n", g.Utilization, g.MemUtilization)

			if g.Allocated {
				fmt.Printf("  Allocated:   %s\n", g.AllocatedTo)
			} else {
				fmt.Printf("  Status:      Available\n")
			}
			fmt.Println()
		}

		return nil
	},
}

var gpuAllocateCmd = &cobra.Command{
	Use:   "allocate <service> [--count N] [--vram MIN]",
	Short: "Allocate GPUs for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]
		count, _ := cmd.Flags().GetInt("count")
		vramStr, _ := cmd.Flags().GetString("vram")

		detector := gpu.NewNVIDIADetector()
		scheduler, err := gpu.NewSimpleScheduler(detector, gpu.GPUSchedulerConfig{
			Strategy:     gpu.StrategyFirstFit,
			AllowSharing: false,
		})
		if err != nil {
			return fmt.Errorf("failed to create scheduler: %w", err)
		}

		req := gpu.GPURequirements{
			Count:     count,
			Exclusive: true,
		}

		if vramStr != "" {
			// Parse VRAM requirement
			req.MinVRAM = parseVRAMRequirement(vramStr)
		}

		alloc, err := scheduler.Allocate(service, req)
		if err != nil {
			return fmt.Errorf("failed to allocate GPUs: %w", err)
		}

		fmt.Printf("Allocated %d GPU(s) for %s:\n", len(alloc.GPUs), service)
		for _, g := range alloc.GPUs {
			fmt.Printf("  - GPU %d: %s (%s)\n", g.Index, g.Name, gpu.FormatVRAM(g.VRAM))
		}

		return nil
	},
}

var gpuReleaseCmd = &cobra.Command{
	Use:   "release <service>",
	Short: "Release GPU allocations for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]

		detector := gpu.NewNVIDIADetector()
		scheduler, err := gpu.NewSimpleScheduler(detector, gpu.GPUSchedulerConfig{})
		if err != nil {
			return fmt.Errorf("failed to create scheduler: %w", err)
		}

		if err := scheduler.ReleaseByOwner(service); err != nil {
			return fmt.Errorf("failed to release GPUs: %w", err)
		}

		fmt.Printf("Released GPU allocations for %s\n", service)
		return nil
	},
}

func truncateGPU(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func parseVRAMRequirement(s string) int64 {
	s = strings.ToLower(strings.TrimSpace(s))
	var multiplier int64 = 1

	if strings.HasSuffix(s, "g") || strings.HasSuffix(s, "gb") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "gb"), "g")
	} else if strings.HasSuffix(s, "m") || strings.HasSuffix(s, "mb") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "mb"), "m")
	}

	var value int64
	_, _ = fmt.Sscanf(s, "%d", &value)
	return value * multiplier
}

func init() {
	gpuAllocateCmd.Flags().IntP("count", "n", 1, "Number of GPUs to allocate")
	gpuAllocateCmd.Flags().String("vram", "", "Minimum VRAM required (e.g., 8G)")

	gpuCmd.AddCommand(gpuListCmd)
	gpuCmd.AddCommand(gpuStatusCmd)
	gpuCmd.AddCommand(gpuAllocateCmd)
	gpuCmd.AddCommand(gpuReleaseCmd)

	rootCmd.AddCommand(gpuCmd)
}
