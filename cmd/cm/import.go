package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/UPwith-me/Container-Maker/pkg/imports"
	"github.com/spf13/cobra"
)

var (
	importDryRun  bool
	importStrict  bool
	importOutput  string
	importName    string
	importAnalyze bool
)

var importCmd = &cobra.Command{
	Use:   "import <source-file>",
	Short: "Import from existing configurations",
	Long: `Import services from docker-compose.yml or Helm charts.

This command converts existing container orchestration configurations
to Container-Maker workspace format.

SUPPORTED SOURCES
  - docker-compose.yml / docker-compose.yaml
  - compose.yml / compose.yaml
  - Helm charts (coming soon)

EXAMPLES
  cm import docker-compose.yml
  cm import docker-compose.yml --output cm-workspace.yaml
  cm import docker-compose.yml --analyze
  cm import docker-compose.yml --dry-run

The importer will:
  1. Parse the source configuration
  2. Analyze compatibility with Container-Maker
  3. Convert services, networks, and volumes
  4. Generate warnings for unsupported features`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourcePath := args[0]

		// Check file exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", sourcePath)
		}

		// Determine importer
		importer := selectImporter(sourcePath)
		if importer == nil {
			return fmt.Errorf("unsupported file format: %s", sourcePath)
		}

		// Analyze only mode
		if importAnalyze {
			return runAnalysis(importer, sourcePath)
		}

		// Run import
		opts := imports.ImportOptions{
			SourcePath:    sourcePath,
			OutputPath:    importOutput,
			ProjectName:   importName,
			DryRun:        importDryRun,
			Strict:        importStrict,
			PreservePorts: true,
			AddLabels:     true,
		}

		result, err := importer.Import(opts)
		if err != nil {
			return err
		}

		// Print result
		printImportResult(result, importDryRun)

		return nil
	},
}

func selectImporter(path string) imports.Importer {
	composeImporter := imports.NewComposeImporter()
	if composeImporter.CanHandle(path) {
		return composeImporter
	}
	return nil
}

func runAnalysis(importer imports.Importer, path string) error {
	fmt.Printf("Analyzing %s...\n\n", filepath.Base(path))

	result, err := importer.Analyze(path)
	if err != nil {
		return err
	}

	// Print analysis
	fmt.Printf("Source: %s\n", result.Source)
	fmt.Printf("Valid: %v\n\n", result.Valid)

	fmt.Printf("Services: %d\n", len(result.Services))
	fmt.Printf("Networks: %d\n", len(result.Networks))
	fmt.Printf("Volumes: %d\n\n", len(result.Volumes))

	// Service details
	fmt.Println("SERVICE ANALYSIS")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-20s %-15s %-10s %-15s\n", "NAME", "IMAGE", "GPU", "WARNINGS")

	for _, svc := range result.Services {
		img := svc.Image
		if img == "" && svc.Build {
			img = "[build]"
		}
		if len(img) > 13 {
			img = img[:10] + "..."
		}

		gpu := "no"
		if svc.HasGPU {
			gpu = "yes"
		}

		warnings := "-"
		if len(svc.Warnings) > 0 {
			warnings = fmt.Sprintf("%d issues", len(svc.Warnings))
		}

		fmt.Printf("%-20s %-15s %-10s %-15s\n", svc.Name, img, gpu, warnings)
	}

	// Compatibility
	fmt.Println()
	fmt.Println("COMPATIBILITY REPORT")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Score: %d/100\n", result.Compatibility.Score)
	fmt.Printf("Fully Supported: %d services\n", len(result.Compatibility.FullySupported))
	fmt.Printf("Partial Support: %d services\n", len(result.Compatibility.PartialSupport))
	fmt.Printf("Not Supported: %d services\n", len(result.Compatibility.NotSupported))

	fmt.Println()
	fmt.Println("Run 'cm import " + filepath.Base(path) + "' to perform the import.")

	return nil
}

func printImportResult(result *imports.ImportResult, dryRun bool) {
	fmt.Println()
	fmt.Println("IMPORT RESULT")
	fmt.Println(strings.Repeat("-", 60))

	if dryRun {
		fmt.Println("Mode: DRY RUN (no files written)")
	}

	fmt.Printf("Source: %s\n", result.SourceFile)
	fmt.Printf("Workspace: %s\n", result.Workspace.Name)
	fmt.Println()

	// Statistics
	fmt.Println("STATISTICS")
	fmt.Printf("  Services imported: %d\n", result.Statistics.ServicesImported)
	fmt.Printf("  Networks imported: %d\n", result.Statistics.NetworksImported)
	fmt.Printf("  Volumes imported: %d\n", result.Statistics.VolumesImported)
	if result.Statistics.SecretsFound > 0 {
		fmt.Printf("  Secrets found: %d (need manual migration)\n", result.Statistics.SecretsFound)
	}

	// Warnings
	if len(result.Warnings) > 0 {
		fmt.Println()
		fmt.Println("WARNINGS")
		for _, w := range result.Warnings {
			fmt.Printf("  [%s] %s\n", w.Code, w.Message)
			if w.Suggestion != "" {
				fmt.Printf("      Suggestion: %s\n", w.Suggestion)
			}
		}
	}

	// Errors
	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Println("ERRORS")
		for _, e := range result.Errors {
			fmt.Printf("  [%s] %s\n", e.Code, e.Message)
		}
	}

	if !dryRun {
		fmt.Println()
		fmt.Printf("Created: %s\n", result.Workspace.ConfigFile)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Review the generated cm-workspace.yaml")
		fmt.Println("  2. Run 'cm up' to start services")
	}
}

var importAnalyzeCmd = &cobra.Command{
	Use:   "analyze <source-file>",
	Short: "Analyze a configuration file",
	Long:  "Analyze a docker-compose.yml or Helm chart for CM compatibility.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		importer := selectImporter(args[0])
		if importer == nil {
			return fmt.Errorf("unsupported file format")
		}
		return runAnalysis(importer, args[0])
	},
}

func init() {
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Don't write output files")
	importCmd.Flags().BoolVar(&importStrict, "strict", false, "Fail on warnings")
	importCmd.Flags().StringVarP(&importOutput, "output", "o", "", "Output file path")
	importCmd.Flags().StringVar(&importName, "name", "", "Project name")
	importCmd.Flags().BoolVar(&importAnalyze, "analyze", false, "Analyze only, don't import")

	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(importAnalyzeCmd)
}
