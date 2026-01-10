package main

import (
	"context"
	"fmt"

	"github.com/UPwith-me/Container-Maker/pkg/scan"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan [image]",
	Short: "Scan an image for vulnerabilities securely",
	Long:  `Scan a container image for vulnerabilities using Trivy.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var image string
		if len(args) > 0 {
			image = args[0]
		} else {
			// Try to resolve current project image from config
			// "Industrial Grade": Use PersistentRunner to get image
			cfg, _, err := loadConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w (provide image manually or run in project root)", err)
			}
			image = cfg.Image
			if image == "" {
				// Try to construct from build?
				// For simple MVP+, prompt user
				return fmt.Errorf("no image specified and could not detect from config")
			}
			fmt.Printf("🔍 Auto-detected image from config: %s\n", image)
		}

		scanner := scan.NewTrivyScanner()
		if !scanner.IsAvailable() {
			fmt.Println("❌ Security Scanner (Trivy) not found.")
			fmt.Println("   Please install it: https://aquasecurity.github.io/trivy/v0.18.3/installation/")
			// Attempt auto-download could happen here in strict Industrial Enterprise version
			return fmt.Errorf("trivy binary not found in PATH")
		}

		fmt.Printf("🛡️  Scanning image %s...\n", image)
		report, err := scanner.Scan(context.Background(), image)
		if err != nil {
			return err
		}

		// Print Report
		fmt.Println("\nScanning Result:")
		fmt.Printf("Image: %s\n", report.Image)
		fmt.Printf("Time:  %s\n", report.ScannedAt)
		fmt.Println("Summary:")
		fmt.Printf("  CRITICAL: %d\n", report.Summary[scan.SeverityCritical])
		fmt.Printf("  HIGH:     %d\n", report.Summary[scan.SeverityHigh])
		fmt.Printf("  MEDIUM:   %d\n", report.Summary[scan.SeverityMedium])
		fmt.Printf("  LOW:      %d\n", report.Summary[scan.SeverityLow])

		if len(report.Vulns) > 0 {
			fmt.Println("\nTop Vulnerabilities (High/Critical):")
			count := 0
			for _, v := range report.Vulns {
				if v.Severity == scan.SeverityCritical || v.Severity == scan.SeverityHigh {
					fmt.Printf("- [%s] %s (%s) - Fixed in: %s\n", v.Severity, v.PkgName, v.VulnerabilityID, v.FixedVersion)
					count++
					if count >= 10 {
						fmt.Println("  ... and more")
						break
					}
				}
			}
		} else {
			fmt.Println("\n✅ No vulnerabilities found!")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
