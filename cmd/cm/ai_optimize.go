package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/UPwith-me/Container-Maker/pkg/ai"
	"github.com/spf13/cobra"
)

var aiOptimizeCmd = &cobra.Command{
	Use:   "optimize [config-path]",
	Short: "Optimize devcontainer.json configuration",
	Long: `Analyze and suggest optimizations for your devcontainer.json.

This command will:
  - Analyze your current configuration
  - Identify performance improvements
  - Suggest security enhancements
  - Recommend productivity boosts

EXAMPLES:
  # Optimize current project config
  cm ai optimize

  # Optimize specific config file
  cm ai optimize .devcontainer/devcontainer.json

  # Apply suggestions interactively
  cm ai optimize --apply`,
	RunE: runAIOptimize,
}

var (
	optimizeApply   bool
	optimizeVerbose bool
)

func init() {
	aiOptimizeCmd.Flags().BoolVar(&optimizeApply, "apply", false, "Apply selected optimizations")
	aiOptimizeCmd.Flags().BoolVarP(&optimizeVerbose, "verbose", "v", false, "Show verbose analysis")
	aiCmd.AddCommand(aiOptimizeCmd)
}

func runAIOptimize(cmd *cobra.Command, args []string) error {
	fmt.Println("🔧 Container-Maker Config Optimizer")
	fmt.Println()

	// Find config file
	configPath := ".devcontainer/devcontainer.json"
	if len(args) > 0 {
		configPath = args[0]
	}

	// Try fallback paths
	if _, err := os.Stat(configPath); err != nil {
		fallbacks := []string{
			"devcontainer.json",
			".devcontainer.json",
		}
		found := false
		for _, fb := range fallbacks {
			if _, err := os.Stat(fb); err == nil {
				configPath = fb
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no devcontainer.json found. Create one with 'cm ai generate'")
		}
	}

	fmt.Printf("📄 Analyzing: %s\n", configPath)
	fmt.Println()

	// Read config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Validate JSON
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Run validation first
	fmt.Println("🔍 Running validation...")
	validator := ai.NewValidator(false)
	validationResult := validator.Validate(string(data))

	if !validationResult.Valid {
		fmt.Println(ai.FormatValidationResult(validationResult))
		fmt.Println()
	} else {
		fmt.Println("✅ Configuration is valid")
		fmt.Println()
	}

	// Run optimizer
	fmt.Println("📊 Analyzing for optimizations...")
	optimizer := ai.NewOptimizer()
	suggestions := optimizer.Analyze(string(data))

	if len(suggestions) == 0 {
		fmt.Println("✅ No optimizations suggested. Your config looks great!")
		return nil
	}

	// Display suggestions
	fmt.Println()
	fmt.Println(ai.FormatSuggestions(suggestions))

	// Show detailed analysis if verbose
	if optimizeVerbose {
		showDetailedAnalysis(config)
	}

	// Apply if requested
	if optimizeApply {
		return applyOptimizations(configPath, config, suggestions)
	}

	fmt.Println("💡 Run with --apply to apply selected optimizations")
	return nil
}

func showDetailedAnalysis(config map[string]interface{}) {
	fmt.Println("📋 Detailed Analysis:")
	fmt.Println("─────────────────────")

	// Image analysis
	if image, ok := config["image"].(string); ok {
		fmt.Printf("Image: %s\n", image)

		// Check if it's a devcontainer image
		if strings.Contains(image, "devcontainers") {
			fmt.Println("  ✅ Using official devcontainer image")
		} else if strings.Contains(image, ":latest") {
			fmt.Println("  ⚠️  Using :latest tag (not recommended)")
		}
	}

	// Features analysis
	if features, ok := config["features"].(map[string]interface{}); ok {
		fmt.Printf("Features: %d configured\n", len(features))
		for feature := range features {
			fmt.Printf("  • %s\n", feature)
		}
	}

	// RunArgs analysis
	if runArgs, ok := config["runArgs"].([]interface{}); ok {
		fmt.Printf("RunArgs: %d configured\n", len(runArgs))
		for _, arg := range runArgs {
			if argStr, ok := arg.(string); ok {
				if argStr == "--privileged" {
					fmt.Println("  ⚠️  --privileged flag detected")
				} else if strings.Contains(argStr, "--gpus") {
					fmt.Println("  🎮 GPU support enabled")
				}
			}
		}
	}

	// Extensions analysis
	if customizations, ok := config["customizations"].(map[string]interface{}); ok {
		if vscode, ok := customizations["vscode"].(map[string]interface{}); ok {
			if extensions, ok := vscode["extensions"].([]interface{}); ok {
				fmt.Printf("VS Code Extensions: %d configured\n", len(extensions))
			}
		}
	}

	fmt.Println()
}

func applyOptimizations(configPath string, config map[string]interface{}, suggestions []ai.OptimizationSuggestion) error {
	fmt.Println()
	fmt.Println("🔧 Applying optimizations...")

	modified := false

	for _, s := range suggestions {
		if s.Apply == nil {
			continue
		}

		fmt.Printf("  Applying: %s... ", s.Title)
		s.Apply(config)
		fmt.Println("✓")
		modified = true
	}

	if !modified {
		fmt.Println("  No automatic fixes available for these suggestions.")
		fmt.Println("  Please apply them manually based on the recommendations above.")
		return nil
	}

	// Write back
	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Backup original
	backupPath := configPath + ".backup"
	if err := os.Rename(configPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup config: %w", err)
	}

	// Write new config
	if err := os.WriteFile(configPath, newData, 0644); err != nil {
		// Restore backup
		_ = os.Rename(backupPath, configPath)
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Println()
	fmt.Printf("✅ Optimizations applied! Backup saved to: %s\n", backupPath)
	return nil
}

// Add cm ai local command for local model usage
var aiLocalCmd = &cobra.Command{
	Use:   "local",
	Short: "Use local AI model (Ollama)",
	Long: `Use Ollama local model for AI operations.

This allows you to generate configs without an API key using
locally installed models like codellama, deepseek-coder, etc.

PREREQUISITES:
  1. Install Ollama: https://ollama.ai
  2. Pull a code model: ollama pull codellama

EXAMPLES:
  # Generate config using local model
  cm ai local generate

  # List available local models
  cm ai local models

  # Pull a recommended model
  cm ai local pull codellama`,
}

var aiLocalGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate devcontainer.json using local model",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🤖 Using local Ollama model")
		fmt.Println()

		// Check Ollama availability
		ollama := ai.NewOllamaProvider()
		if !ollama.IsAvailable() {
			fmt.Println("❌ Ollama is not running")
			fmt.Println()
			fmt.Println("To use local models:")
			fmt.Println("  1. Install Ollama: https://ollama.ai")
			fmt.Println("  2. Start Ollama: ollama serve")
			fmt.Println("  3. Pull a model: ollama pull codellama")
			return nil
		}

		model := ollama.GetModel()
		fmt.Printf("📦 Using model: %s\n", model)
		fmt.Println()

		// Generate config
		cwd, _ := os.Getwd()
		sg := &ai.SmartGenerator{}

		fmt.Print("🔄 Generating configuration... ")
		config, err := sg.GenerateConfig(cmd.Context(), cwd)
		if err != nil {
			return err
		}
		fmt.Println("✓")
		fmt.Println()

		// Show preview
		fmt.Println("📄 Generated Configuration:")
		fmt.Println("─────────────────────────────")
		fmt.Println(config)
		fmt.Println()

		fmt.Print("Save to .devcontainer/devcontainer.json? [Y/n]: ")
		var confirm string
		_, _ = fmt.Scanln(&confirm)

		if confirm == "" || strings.ToLower(confirm) == "y" {
			configDir := ".devcontainer"
			_ = os.MkdirAll(configDir, 0755)
			configPath := filepath.Join(configDir, "devcontainer.json")

			if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
				return err
			}
			fmt.Printf("✅ Saved to: %s\n", configPath)
		}

		return nil
	},
}

var aiLocalModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available local models",
	RunE: func(cmd *cobra.Command, args []string) error {
		ollama := ai.NewOllamaProvider()
		if !ollama.IsAvailable() {
			fmt.Println("❌ Ollama is not running")
			return nil
		}

		models, err := ollama.ListModels()
		if err != nil {
			return err
		}

		if len(models) == 0 {
			fmt.Println("No models installed")
			fmt.Println()
			fmt.Println("Pull a model with:")
			fmt.Println("  ollama pull codellama")
			return nil
		}

		fmt.Println("📦 Available Models:")
		for _, m := range models {
			fmt.Printf("  • %s\n", m)
		}

		return nil
	},
}

var aiLocalPullCmd = &cobra.Command{
	Use:   "pull <model>",
	Short: "Pull a model from Ollama",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ollama := ai.NewOllamaProvider()
		if !ollama.IsAvailable() {
			fmt.Println("❌ Ollama is not running. Start it with: ollama serve")
			return nil
		}

		return ollama.PullModel(cmd.Context(), args[0])
	},
}

func init() {
	aiLocalCmd.AddCommand(aiLocalGenerateCmd)
	aiLocalCmd.AddCommand(aiLocalModelsCmd)
	aiLocalCmd.AddCommand(aiLocalPullCmd)
	aiCmd.AddCommand(aiLocalCmd)
}
