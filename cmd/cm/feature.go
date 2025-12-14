package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/UPwith-me/Container-Maker/pkg/runner"
	"github.com/spf13/cobra"
)

var featureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Manage DevContainer features",
	Long: `List, search, download, and get information about DevContainer features.

Features are modular additions to dev containers that provide
additional tools, runtimes, or configurations.

Supports downloading from OCI registries (ghcr.io, etc.)

Examples:
  cm feature list                    # List available features
  cm feature info go                 # Show feature details and options
  cm feature download node           # Download feature to cache
  cm feature cache                   # Show cached features
  cm feature cache clear             # Clear feature cache`,
}

var featureListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available features",
	RunE:  runFeatureList,
}

var featureInfoCmd = &cobra.Command{
	Use:   "info <feature-name>",
	Short: "Show feature details and options",
	Args:  cobra.ExactArgs(1),
	RunE:  runFeatureInfo,
}

var featureDownloadCmd = &cobra.Command{
	Use:   "download <feature-ref>",
	Short: "Download a feature from OCI registry",
	Long: `Download a DevContainer feature from an OCI registry.

Supports full OCI references:
  cm feature download go
  cm feature download ghcr.io/devcontainers/features/node:1
  cm feature download ghcr.io/some-org/some-feature:latest`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureDownload,
}

var featureCacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage feature cache",
	RunE:  runFeatureCache,
}

var featureCacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear feature cache",
	RunE:  runFeatureCacheClear,
}

func init() {
	featureCacheCmd.AddCommand(featureCacheClearCmd)
	featureCmd.AddCommand(featureListCmd)
	featureCmd.AddCommand(featureInfoCmd)
	featureCmd.AddCommand(featureDownloadCmd)
	featureCmd.AddCommand(featureCacheCmd)
	rootCmd.AddCommand(featureCmd)
}

func runFeatureList(cmd *cobra.Command, args []string) error {
	fmt.Println("📦 Available DevContainer Features")
	fmt.Println()

	features, err := runner.ListOfficialFeatures()
	if err != nil {
		fmt.Println("⚠️  Could not fetch feature list, showing cached features")
	}

	// Group by category
	builtIn := []string{"git", "common-utils", "node", "python", "go", "rust", "java", "dotnet"}
	devops := []string{"docker-in-docker", "docker-from-docker", "kubectl-helm-minikube", "aws-cli", "azure-cli", "terraform"}
	tools := []string{"github-cli", "powershell", "sshd"}

	fmt.Println("📝 Languages & Runtimes:")
	for _, f := range builtIn {
		fmt.Printf("   • ghcr.io/devcontainers/features/%s:1\n", f)
	}

	fmt.Println("\n🛠️  DevOps & Cloud:")
	for _, f := range devops {
		fmt.Printf("   • ghcr.io/devcontainers/features/%s:1\n", f)
	}

	fmt.Println("\n🔧 Tools:")
	for _, f := range tools {
		fmt.Printf("   • ghcr.io/devcontainers/features/%s:1\n", f)
	}

	if len(features) > len(builtIn)+len(devops)+len(tools) {
		fmt.Printf("\n   ... and %d more at ghcr.io/devcontainers/features/\n", len(features)-len(builtIn)-len(devops)-len(tools))
	}

	fmt.Println()
	fmt.Println("💡 Use 'cm feature info <name>' to see options")
	fmt.Println("💡 Use 'cm feature download <name>' to download")
	fmt.Println("💡 Add features to devcontainer.json:")
	fmt.Println(`   "features": { "ghcr.io/devcontainers/features/go:1": {} }`)

	return nil
}

func runFeatureInfo(cmd *cobra.Command, args []string) error {
	featureName := args[0]

	// Normalize feature ID
	featureID := normalizeFeatureID(featureName)

	fmt.Printf("🔍 Fetching info for: %s\n\n", featureID)

	// Try to download and get metadata
	downloader := runner.NewOCIFeatureDownloader("docker")
	featurePath, err := downloader.DownloadFeature(context.Background(), featureID)

	if err == nil {
		meta, metaErr := downloader.GetFeatureMetadata(featurePath)
		if metaErr == nil {
			fmt.Printf("📦 Feature: %s\n", meta.Name)
			fmt.Printf("   ID: %s\n", meta.ID)
			fmt.Printf("   Version: %s\n", meta.Version)
			fmt.Printf("   Description: %s\n", meta.Description)

			if len(meta.Options) > 0 {
				fmt.Println("\n📋 Options:")
				for name, opt := range meta.Options {
					fmt.Printf("   • %s (%s)\n", name, opt.Type)
					fmt.Printf("     %s\n", opt.Description)
					if opt.Default != nil {
						fmt.Printf("     Default: %v\n", opt.Default)
					}
					if len(opt.Enum) > 0 {
						fmt.Printf("     Values: %s\n", strings.Join(opt.Enum, ", "))
					}
				}
			}
			return nil
		}
	}

	// Fallback to simple info
	info, err := runner.GetFeatureInfo(featureID)
	if err != nil {
		fmt.Println("⚠️  Could not fetch metadata, showing basic info")
		fmt.Println()
		fmt.Printf("📦 Feature: %s\n", featureName)
		fmt.Printf("   Full ID: %s\n", featureID)
		fmt.Println()
		fmt.Println("📋 Usage in devcontainer.json:")
		fmt.Printf(`   "features": {
     "%s": {
       // options here
     }
   }
`, featureID)
		return nil
	}

	fmt.Println(info)
	return nil
}

func runFeatureDownload(cmd *cobra.Command, args []string) error {
	featureRef := normalizeFeatureID(args[0])

	fmt.Printf("📥 Downloading feature: %s\n\n", featureRef)

	downloader := runner.NewOCIFeatureDownloader("docker")
	featurePath, err := downloader.DownloadFeature(context.Background(), featureRef)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Printf("✅ Downloaded to: %s\n\n", featurePath)

	// Show metadata if available
	meta, err := downloader.GetFeatureMetadata(featurePath)
	if err == nil {
		fmt.Printf("📦 %s v%s\n", meta.Name, meta.Version)
		fmt.Printf("   %s\n", meta.Description)
		if len(meta.Options) > 0 {
			fmt.Printf("   Options: %d available\n", len(meta.Options))
		}
	}

	return nil
}

func runFeatureCache(cmd *cobra.Command, args []string) error {
	fmt.Println("📦 Cached Features")
	fmt.Println()

	downloader := runner.NewOCIFeatureDownloader("docker")
	cached, err := downloader.ListCachedFeatures()
	if err != nil {
		return err
	}

	if len(cached) == 0 {
		fmt.Println("   No features cached")
		fmt.Println()
		fmt.Println("💡 Use 'cm feature download <name>' to cache a feature")
		return nil
	}

	for _, f := range cached {
		fmt.Printf("   • %s\n", f)
	}

	fmt.Printf("\n   Total: %d cached features\n", len(cached))
	fmt.Println()
	fmt.Println("💡 Use 'cm feature cache clear' to remove all cached features")

	return nil
}

func runFeatureCacheClear(cmd *cobra.Command, args []string) error {
	fmt.Println("🧹 Clearing feature cache...")

	downloader := runner.NewOCIFeatureDownloader("docker")
	if err := downloader.ClearCache(); err != nil {
		return err
	}

	fmt.Println("✅ Feature cache cleared")
	return nil
}

func normalizeFeatureID(name string) string {
	if strings.HasPrefix(name, "ghcr.io/") || strings.Contains(name, "/") {
		return name
	}
	return fmt.Sprintf("ghcr.io/devcontainers/features/%s:1", name)
}
