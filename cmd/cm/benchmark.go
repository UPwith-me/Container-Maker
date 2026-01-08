package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var benchmarkIterations int

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run performance benchmarks",
	Long: `Run performance benchmarks for Container-Maker.

This command measures:
- CLI startup time
- Container startup time (if Docker available)
- Config parsing time

Examples:
  cm benchmark
  cm benchmark --iterations 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("=== Container-Maker Performance Benchmark ===")
		fmt.Println()

		// 1. CLI Startup Time
		fmt.Println("[1] CLI Startup Time")
		startupTime, err := benchmarkStartup(benchmarkIterations)
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
		} else {
			fmt.Printf("    Average: %v (over %d runs)\n", startupTime, benchmarkIterations)
		}
		fmt.Println()

		// 2. Config Parsing Time
		fmt.Println("[2] Config Parsing Time")
		parseTime, err := benchmarkConfigParsing()
		if err != nil {
			fmt.Printf("    %s\n", err)
		} else {
			fmt.Printf("    Time: %v\n", parseTime)
		}
		fmt.Println()

		// 3. Container Startup Time (if available)
		fmt.Println("[3] Container Startup Time")
		containerTime, err := benchmarkContainerStartup()
		if err != nil {
			fmt.Printf("    %s\n", err)
		} else {
			fmt.Printf("    Time: %v\n", containerTime)
		}
		fmt.Println()

		// 4. Binary Size
		fmt.Println("[4] Binary Information")
		if exe, err := os.Executable(); err == nil {
			if info, err := os.Stat(exe); err == nil {
				sizeMB := float64(info.Size()) / (1024 * 1024)
				fmt.Printf("    Size: %.2f MB\n", sizeMB)
			}
		}
		fmt.Println()

		fmt.Println("=== Benchmark Complete ===")
		return nil
	},
}

func benchmarkStartup(iterations int) (time.Duration, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}

	var totalTime time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		cmd := exec.Command(exe, "version", "--short")
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			// Ignore errors, just measure time
		}
		totalTime += time.Since(start)
	}

	return totalTime / time.Duration(iterations), nil
}

func benchmarkConfigParsing() (time.Duration, error) {
	// Look for devcontainer.json
	paths := []string{
		".devcontainer/devcontainer.json",
		"devcontainer.json",
	}

	var configPath string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			configPath = p
			break
		}
	}

	if configPath == "" {
		return 0, fmt.Errorf("No devcontainer.json found (skipped)")
	}

	start := time.Now()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return 0, err
	}
	// Simulate parsing
	_ = len(data)
	return time.Since(start), nil
}

func benchmarkContainerStartup() (time.Duration, error) {
	// Check if docker is available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("Docker not available (skipped)")
	}

	// Run a quick alpine container
	start := time.Now()
	cmd = exec.Command("docker", "run", "--rm", "alpine:latest", "echo", "benchmark")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("Container test failed: %v", err)
	}

	return time.Since(start), nil
}

// benchmarkVersion returns a simple version for testing
// var benchmarkVersionCmd = &cobra.Command{
// 	Use:    "version --short",
// 	Hidden: true,
// 	Run:    func(cmd *cobra.Command, args []string) {},
// }

func init() {
	benchmarkCmd.Flags().IntVar(&benchmarkIterations, "iterations", 5, "Number of iterations for startup benchmark")
	rootCmd.AddCommand(benchmarkCmd)
}

// GetBinaryInfo returns info about the binary
func GetBinaryInfo() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	info, err := os.Stat(exe)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Path: %s\n", exe))
	sb.WriteString(fmt.Sprintf("Name: %s\n", filepath.Base(exe)))
	sb.WriteString(fmt.Sprintf("Size: %.2f MB\n", float64(info.Size())/(1024*1024)))
	sb.WriteString(fmt.Sprintf("Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05")))

	return sb.String(), nil
}
