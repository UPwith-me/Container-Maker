package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/ai"
	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-powered features",
	Long: `Use AI to analyze your project and generate optimal configurations.

Requires AI to be enabled and an API key to be set:
  cm config set ai.enabled true
  cm config set ai.api_key sk-xxx
  cm config set ai.api_base https://api.openai.com/v1  # Optional

Examples:
  cm ai generate        # Generate devcontainer.json for current project
  cm ai analyze         # Analyze project without generating`,
}

var aiGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate devcontainer.json using AI",
	Long: `Analyze your project and generate an optimal devcontainer.json configuration.

The AI will:
1. Detect programming languages
2. Identify dependencies
3. Choose the best base image
4. Configure appropriate features
5. Set up VS Code extensions

You will be asked to confirm before saving.`,
	RunE: runAIGenerate,
}

var aiAnalyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze project (without generating)",
	RunE:  runAIAnalyze,
}

var aiDryRun bool

func init() {
	aiGenerateCmd.Flags().BoolVar(&aiDryRun, "dry-run", false, "Show generated config without saving")
	aiCmd.AddCommand(aiGenerateCmd)
	aiCmd.AddCommand(aiAnalyzeCmd)
	rootCmd.AddCommand(aiCmd)
}

func runAIGenerate(cmd *cobra.Command, args []string) error {
	fmt.Println("🤖 AI DevContainer Generator")
	fmt.Println()

	// Get project directory
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Create generator
	gen, err := ai.NewGenerator()
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		fmt.Println()
		fmt.Println("💡 To enable AI features:")
		fmt.Println("   cm config set ai.enabled true")
		fmt.Println("   cm config set ai.api_key <your-api-key>")
		return nil
	}

	fmt.Printf("📁 Analyzing project: %s\n", projectDir)
	fmt.Println()

	// Show spinner
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Print("⏳ Generating configuration... ")

	config, err := gen.AnalyzeProject(ctx, projectDir)
	if err != nil {
		fmt.Println("❌")
		return fmt.Errorf("generation failed: %w", err)
	}
	fmt.Println("✅")
	fmt.Println()

	// Show config
	fmt.Println("📄 Generated devcontainer.json:")
	fmt.Println("─────────────────────────────────")
	fmt.Println(config)
	fmt.Println("─────────────────────────────────")
	fmt.Println()

	if aiDryRun {
		fmt.Println("(Dry run - not saving)")
		return nil
	}

	// Confirm
	fmt.Print("💾 Save this configuration? [Y/n] ")
	var response string
	fmt.Scanln(&response)

	if response != "" && response != "y" && response != "Y" {
		fmt.Println("❌ Cancelled")
		return nil
	}

	// Save
	if err := gen.SaveConfig(projectDir, config); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	fmt.Println("✅ Saved to .devcontainer/devcontainer.json")
	fmt.Println()
	fmt.Println("🚀 Run 'cm shell' to start your dev container!")

	return nil
}

func runAIAnalyze(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	fmt.Println("🔍 Project Analysis")
	fmt.Println()

	// Simple manual analysis without AI
	info := analyzeProjectManual(projectDir)

	fmt.Printf("📁 Project: %s\n", info.Name)
	fmt.Println()

	if len(info.Languages) > 0 {
		fmt.Printf("📝 Languages: %v\n", info.Languages)
	}

	if info.HasDockerfile {
		fmt.Println("🐳 Has Dockerfile: Yes")
	}
	if info.HasMakefile {
		fmt.Println("🔧 Has Makefile: Yes")
	}

	if len(info.Dependencies) > 0 {
		fmt.Printf("📦 Dependencies: %d found\n", len(info.Dependencies))
	}

	fmt.Println()
	fmt.Println("💡 Run 'cm ai generate' to create a devcontainer.json")

	return nil
}

type projectInfo struct {
	Name          string
	Languages     []string
	Dependencies  []string
	HasDockerfile bool
	HasMakefile   bool
}

func analyzeProjectManual(dir string) *projectInfo {
	info := &projectInfo{
		Name: dir,
	}

	// Check for files
	if _, err := os.Stat(dir + "/Dockerfile"); err == nil {
		info.HasDockerfile = true
	}
	if _, err := os.Stat(dir + "/Makefile"); err == nil {
		info.HasMakefile = true
	}
	if _, err := os.Stat(dir + "/go.mod"); err == nil {
		info.Languages = append(info.Languages, "Go")
	}
	if _, err := os.Stat(dir + "/package.json"); err == nil {
		info.Languages = append(info.Languages, "JavaScript/Node.js")
	}
	if _, err := os.Stat(dir + "/requirements.txt"); err == nil {
		info.Languages = append(info.Languages, "Python")
	}
	if _, err := os.Stat(dir + "/Cargo.toml"); err == nil {
		info.Languages = append(info.Languages, "Rust")
	}

	return info
}
