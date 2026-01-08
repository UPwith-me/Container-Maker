package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/UPwith-me/Container-Maker/pkg/policy"
	"github.com/UPwith-me/Container-Maker/pkg/workspace"
	"github.com/spf13/cobra"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage security and compliance policies",
	Long: `Enforce security, best practices, and resource limits using Policy as Code.

Check your workspace configuration against built-in or custom policies to identify
potential security risks and misconfigurations.

COMMANDS
  cm policy check    Check workspace against policies
  cm policy list     List active policies`,
}

var (
	policyFailOnWarn bool
	policyQuiet      bool
)

var policyCheckCmd = &cobra.Command{
	Use:   "check [workspace-file]",
	Short: "Check workspace against policies",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load workspace
		path := ""
		if len(args) > 0 {
			path = args[0]
		}

		ws, err := workspace.Load(path)
		if err != nil {
			return fmt.Errorf("failed to load workspace: %w", err)
		}

		// Create engine
		engine := policy.NewEngine()

		// Evaluate
		result, err := engine.EvaluateWorkspace(cmd.Context(), ws)
		if err != nil {
			return fmt.Errorf("evaluation failed: %w", err)
		}

		// Print output
		if !policyQuiet {
			printPolicyResult(result)
		}

		// Determine exit code
		if !result.Passed {
			// If strict mode or critical errors, exit non-zero
			hasCritical := false
			for _, v := range result.Violations {
				if v.Severity == policy.SeverityCritical || v.Severity == policy.SeverityError {
					hasCritical = true
					break
				}
			}

			if hasCritical || policyFailOnWarn {
				os.Exit(1)
			}
		}

		return nil
	},
}

var policyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active policies",
	Run: func(cmd *cobra.Command, args []string) {
		engine := policy.NewEngine()
		policies := engine.GetPolicies()

		fmt.Printf("Active Policies: %d\n\n", len(policies))
		fmt.Printf("%-10s %-30s %-10s %-10s\n", "ID", "NAME", "TYPE", "SEVERITY")
		fmt.Println(strings.Repeat("-", 70))

		for _, p := range policies {
			fmt.Printf("%-10s %-30s %-10s %-10s\n",
				p.ID,
				truncate(p.Name, 28),
				p.Type,
				p.Severity)
		}
	},
}

func printPolicyResult(res *policy.EvaluationResult) {
	fmt.Println()
	fmt.Printf("Policy Check Results\n")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Score: %d/100\n", res.Score)
	fmt.Printf("Time:  %s\n", res.Duration)
	fmt.Println()

	if len(res.Violations) == 0 {
		fmt.Println("✅ No violations found. Great job!")
		return
	}

	// Group by Resource
	byResource := make(map[string][]policy.Violation)
	for _, v := range res.Violations {
		byResource[v.Resource] = append(byResource[v.Resource], v)
	}

	for resource, violations := range byResource {
		fmt.Printf("Service: %s\n", resource)
		for _, v := range violations {
			icon := "ℹ️"
			switch v.Severity {
			case policy.SeverityCritical:
				icon = "🚨"
			case policy.SeverityError:
				icon = "❌"
			case policy.SeverityWarning:
				icon = "⚠️"
			}

			fmt.Printf("  %s [%s] %s: %s\n", icon, v.PolicyID, v.Severity, v.Message)
			if v.Suggestion != "" {
				fmt.Printf("     Suggestion: %s\n", v.Suggestion)
			}
		}
		fmt.Println()
	}

	criticals := 0
	warnings := 0
	for _, v := range res.Violations {
		if v.Severity == policy.SeverityCritical || v.Severity == policy.SeverityError {
			criticals++
		} else {
			warnings++
		}
	}

	fmt.Printf("Summary: %d critical/error, %d warning/info\n", criticals, warnings)
}

func truncate(s string, l int) string {
	if len(s) > l {
		return s[:l-3] + "..."
	}
	return s
}

func init() {
	policyCheckCmd.Flags().BoolVar(&policyFailOnWarn, "strict", false, "Fail on warnings")
	policyCheckCmd.Flags().BoolVarP(&policyQuiet, "quiet", "q", false, "Suppress output")

	policyCmd.AddCommand(policyCheckCmd)
	policyCmd.AddCommand(policyListCmd)

	rootCmd.AddCommand(policyCmd)
}
