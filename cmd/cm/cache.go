package main

import (
	"context"
	"fmt"
	"os"

	"github.com/UPwith-me/Container-Maker/pkg/runner"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage build caches",
	Long: `Manage build cache volumes for faster incremental builds.

Container-Maker automatically detects your project's languages and creates
persistent cache volumes for package managers and build tools.

Supported Languages:
  - Go:     /go/pkg, /root/.cache/go-build
  - Rust:   /root/.cargo/registry, /target
  - Node:   /root/.npm, /root/.yarn, /root/.pnpm-store
  - Python: /root/.cache/pip
  - C/C++:  /root/.ccache
  - Java:   /root/.m2, /root/.gradle
  - .NET:   /root/.nuget

Examples:
  cm cache status       # Show cache status
  cm cache clean        # Remove all caches
  cm cache create       # Pre-create cache volumes`,
}

var cacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cache status",
	RunE:  runCacheStatus,
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove all cache volumes",
	RunE:  runCacheClean,
}

var cacheCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Pre-create cache volumes",
	RunE:  runCacheCreate,
}

func init() {
	cacheCmd.AddCommand(cacheStatusCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	cacheCmd.AddCommand(cacheCreateCmd)
	rootCmd.AddCommand(cacheCmd)
}

func runCacheStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	fmt.Println("📦 Build Cache Manager")
	fmt.Println()

	manager := runner.NewCacheManager("docker", cwd)
	fmt.Println(manager.GetCacheStats(context.Background()))

	return nil
}

func runCacheClean(cmd *cobra.Command, args []string) error {
	fmt.Println("🧹 Cleaning all build caches...")
	fmt.Println()

	cwd, _ := os.Getwd()
	manager := runner.NewCacheManager("docker", cwd)

	if err := manager.CleanCaches(context.Background()); err != nil {
		return err
	}

	fmt.Println("✅ All cache volumes removed")
	return nil
}

func runCacheCreate(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	fmt.Println("📦 Creating cache volumes...")
	fmt.Println()

	manager := runner.NewCacheManager("docker", cwd)

	langs := manager.DetectLanguages()
	if len(langs) == 0 {
		fmt.Println("⚠️  No languages detected in current directory")
		return nil
	}

	fmt.Printf("   Detected: %v\n\n", langs)

	if err := manager.EnsureCacheVolumes(context.Background()); err != nil {
		return err
	}

	fmt.Println(manager.GetCacheStats(context.Background()))
	fmt.Println("✅ Cache volumes ready")

	return nil
}
