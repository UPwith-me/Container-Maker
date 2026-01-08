package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/UPwith-me/Container-Maker/pkg/sbom"
	"github.com/spf13/cobra"
)

var sbomCmd = &cobra.Command{
	Use:   "sbom [path]",
	Short: "Generate Software Bill of Materials",
	Long: `Generate a CycloneDX JSON SBOM for the workspace.

Scans the project for dependency files (go.mod, package.json, requirements.txt)
and outputs a standard SBOM including all detected components.

EXAMPLES
  cm sbom                   # Generate SBOM for current directory
  cm sbom ./backend         # Generate SBOM for specific directory
  cm sbom -o sbom.json      # Save to file`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		output, _ := cmd.Flags().GetString("output")

		fmt.Printf("Generating SBOM for %s...\n", path)

		s, err := sbom.GenerateSBOM(path, "workspace")
		if err != nil {
			return fmt.Errorf("failed to generate SBOM: %w", err)
		}

		data, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		if output != "" {
			if err := os.WriteFile(output, data, 0644); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
			fmt.Printf("✅ SBOM saved to %s (%d components)\n", output, len(s.Components))
		} else {
			fmt.Println(string(data))
		}

		return nil
	},
}

func init() {
	sbomCmd.Flags().StringP("output", "o", "", "Output valid path")
	rootCmd.AddCommand(sbomCmd)
}
