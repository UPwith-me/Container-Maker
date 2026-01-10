package main

import (
	"context"
	"fmt"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/monitor"
	"github.com/UPwith-me/Container-Maker/pkg/profile"
	"github.com/UPwith-me/Container-Maker/pkg/runner"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Profile container resources",
	Long:  `Analyze CPU and Memory usage to recommend optimal resource limits.`,
}

var profileStartCmd = &cobra.Command{
	Use:   "start [duration]",
	Short: "Start a profiling session (e.g., 30s, 5m)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		durationStr := "30s"
		if len(args) > 0 {
			durationStr = args[0]
		}
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}

		// 1. Identify Container
		cfg, projectDir, err := loadConfig()
		if err != nil {
			return err
		}
		pr, err := runner.NewPersistentRunner(cfg, projectDir)
		if err != nil {
			return err
		}
		running, containerID, err := pr.IsContainerRunning(context.Background())
		if err != nil {
			return err
		}
		if !running {
			return fmt.Errorf("container is not running")
		}

		fmt.Printf("🕵️  Starting profiling session for %s...\n", duration)
		fmt.Printf("🎯 Target Container: %s\n", containerID[:12])
		fmt.Println("📊 Collecting samples (P95 Weighted Algorithm)...")

		// 2. Setup Collector
		collector, err := monitor.NewDockerCollector()
		if err != nil {
			return fmt.Errorf("failed to init collector: %w", err)
		}
		defer collector.Close()

		profiler := profile.NewProfiler()

		// 3. Stream Metrics
		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		metricsChan, err := collector.Stream(ctx, containerID, 1*time.Second)
		if err != nil {
			return fmt.Errorf("failed to stream metrics: %w", err)
		}

		// 4. Collection Loop
		sampleCount := 0
		spinner := []string{"|", "/", "-", "\\"}

		for {
			select {
			case <-ctx.Done():
				goto Analyze
			case m, ok := <-metricsChan:
				if !ok {
					goto Analyze
				}
				profiler.AddSample(m)
				sampleCount++
				fmt.Printf("\r%s Samples: %d | CPU: %.1f%% | Mem: %dMB",
					spinner[sampleCount%4], sampleCount, m.CPUPercent, m.MemoryUsed/1024/1024)
			}
		}

	Analyze:
		fmt.Println("\n\n✅ Profiling Complete.")

		rec := profiler.Analyze()
		if rec == nil {
			return fmt.Errorf("no samples collected")
		}

		// 5. Print Report
		fmt.Println("==========================================")
		fmt.Println("       RESOURCE OPTIMIZATION REPORT       ")
		fmt.Println("==========================================")
		fmt.Printf("Samples Collected: %d\n", sampleCount)
		fmt.Printf("Algorithm:         Weighted P95 Sliding Window\n")
		fmt.Println("------------------------------------------")
		fmt.Printf("Measured P95 CPU:  %.2f%%\n", rec.P95CPU)
		fmt.Printf("Measured P95 Mem:  %d MB\n", rec.P95MemoryMB)
		fmt.Println("------------------------------------------")
		fmt.Println("💡 AI RECOMMENDATION:")
		fmt.Println("")
		fmt.Printf("   \"devcontainer.json\": {\n")
		fmt.Printf("       \"hostRequirements\": {\n")
		fmt.Printf("           \"cpus\": %.1f,\n", rec.CPULimit)
		fmt.Printf("           \"memory\": \"%dmb\"\n", rec.MemoryLimitMB)
		fmt.Printf("       }\n")
		fmt.Printf("   }\n")
		fmt.Println("==========================================")

		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileStartCmd)
	rootCmd.AddCommand(profileCmd)
}
