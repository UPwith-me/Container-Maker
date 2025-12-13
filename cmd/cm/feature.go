package main

import (
	"fmt"

	"github.com/container-make/cm/pkg/runner"
	"github.com/spf13/cobra"
)

var featureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Manage DevContainer features",
	Long: `List, search, and get information about DevContainer features.

Features are modular additions to dev containers that provide
additional tools, runtimes, or configurations.

Examples:
  cm feature list             # List available features
  cm feature info go          # Show feature details and options
  cm feature info node        # Show Node.js feature options`,
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

func init() {
	featureCmd.AddCommand(featureListCmd)
	featureCmd.AddCommand(featureInfoCmd)
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
	fmt.Println("💡 Add features to devcontainer.json:")
	fmt.Println(`   "features": { "ghcr.io/devcontainers/features/go:1": {} }`)

	return nil
}

func runFeatureInfo(cmd *cobra.Command, args []string) error {
	featureName := args[0]

	// Normalize feature ID
	featureID := featureName
	if !hasPrefix(featureName, "ghcr.io/") {
		featureID = fmt.Sprintf("ghcr.io/devcontainers/features/%s", featureName)
	}

	fmt.Printf("🔍 Fetching info for: %s\n\n", featureID)

	info, err := runner.GetFeatureInfo(featureID)
	if err != nil {
		// Fall back to built-in info
		fmt.Println("⚠️  Could not fetch metadata, showing built-in info")
		fmt.Println()
		fmt.Printf("📦 Feature: %s\n", featureName)
		fmt.Printf("   Full ID: ghcr.io/devcontainers/features/%s:1\n", featureName)
		fmt.Println()
		fmt.Println("📋 Usage in devcontainer.json:")
		fmt.Printf(`   "features": {
     "ghcr.io/devcontainers/features/%s:1": {
       // options here
     }
   }
`, featureName)
		return nil
	}

	fmt.Println(info)
	return nil
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
