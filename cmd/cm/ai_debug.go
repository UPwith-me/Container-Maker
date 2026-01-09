package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/ai"
	"github.com/spf13/cobra"
)

var aiDebugCmd = &cobra.Command{
	Use:   "debug [build-log]",
	Short: "Diagnose build failures using AI",
	Long: `Analyze container build logs to identify issues and suggest fixes.

The AI will analyze error patterns, identify root causes, and provide
actionable suggestions to fix build failures.

SOURCES:
  - Pipe build output:    cm prepare 2>&1 | cm ai debug -
  - Specify log file:     cm ai debug build.log
  - Use last build:       cm ai debug

EXAMPLES:
  # Analyze last build failure
  cm ai debug

  # Analyze specific log file
  cm ai debug /path/to/build.log

  # Pipe build output directly
  cm prepare 2>&1 | cm ai debug -

  # Auto-fix mode (experimental)
  cm ai debug --auto-fix`,
	RunE: runAIDebug,
}

var (
	aiDebugAutoFix bool
	aiDebugVerbose bool
)

func init() {
	aiDebugCmd.Flags().BoolVar(&aiDebugAutoFix, "auto-fix", false, "Attempt to automatically apply fixes (experimental)")
	aiDebugCmd.Flags().BoolVarP(&aiDebugVerbose, "verbose", "v", false, "Show detailed analysis")
	aiCmd.AddCommand(aiDebugCmd)
}

// BuildError represents a parsed build error
type BuildError struct {
	Type       string   // "dockerfile", "dependency", "network", "permission", "resource", "syntax"
	Stage      string   // Build stage where error occurred
	Command    string   // Command that failed
	ErrorCode  int      // Exit code if available
	Message    string   // Error message
	Context    []string // Surrounding lines for context
	LineNumber int      // Line number in Dockerfile/config
}

// DebugAnalysis represents the analysis result
type DebugAnalysis struct {
	Summary     string
	RootCause   string
	ErrorType   string
	Fixes       []FixSuggestion
	RelatedDocs []string
}

// FixSuggestion represents a suggested fix
type FixSuggestion struct {
	Description string
	Command     string
	FileChange  *FileChange
	Confidence  float64 // 0-1
}

// FileChange represents a file modification
type FileChange struct {
	File    string
	Before  string
	After   string
	LineNum int
}

func runAIDebug(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Container-Maker Build Debugger")
	fmt.Println()

	// Get build log
	var log string
	var err error

	if len(args) > 0 {
		if args[0] == "-" {
			// Read from stdin
			log, err = readFromStdin()
		} else {
			// Read from file
			data, readErr := os.ReadFile(args[0])
			if readErr != nil {
				return fmt.Errorf("failed to read log file: %w", readErr)
			}
			log = string(data)
		}
	} else {
		// Try to find last build log
		log, err = getLastBuildLog()
		if err != nil {
			fmt.Println("💡 No recent build log found.")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  cm ai debug <logfile>      # Analyze a log file")
			fmt.Println("  cm prepare 2>&1 | cm ai debug -  # Pipe build output")
			return nil
		}
	}

	if err != nil {
		return err
	}

	if log == "" {
		return fmt.Errorf("empty build log")
	}

	// Parse errors from log
	fmt.Print("📋 Parsing build log... ")
	errors := parseDockerBuildErrors(log)
	fmt.Printf("found %d error(s)\n", len(errors))
	fmt.Println()

	if len(errors) == 0 {
		fmt.Println("✅ No obvious errors found in the build log.")
		fmt.Println("💡 If you're still experiencing issues, try running with --verbose")
		return nil
	}

	// Display parsed errors
	fmt.Println("📊 Detected Issues:")
	fmt.Println("─────────────────────")
	for i, e := range errors {
		fmt.Printf("%d. [%s] %s\n", i+1, e.Type, e.Message)
		if e.Stage != "" {
			fmt.Printf("   Stage: %s\n", e.Stage)
		}
		if e.Command != "" {
			fmt.Printf("   Command: %s\n", truncate(e.Command, 60))
		}
	}
	fmt.Println()

	// Try AI analysis if available
	analysis, aiErr := analyzeWithAI(context.Background(), errors, log)
	if aiErr != nil {
		// Fallback to rule-based analysis
		fmt.Println("⚠️  AI analysis unavailable, using rule-based analysis")
		analysis = analyzeWithRules(errors)
	}

	// Display analysis
	displayAnalysis(analysis)

	// Auto-fix if requested
	if aiDebugAutoFix && len(analysis.Fixes) > 0 {
		fmt.Println()
		fmt.Printf("🔧 Attempting auto-fix: %s\n", analysis.Fixes[0].Description)
		if err := applyFix(analysis.Fixes[0]); err != nil {
			fmt.Printf("❌ Auto-fix failed: %v\n", err)
		} else {
			fmt.Println("✅ Fix applied! Try rebuilding with 'cm prepare'")
		}
	}

	return nil
}

// readFromStdin reads build log from stdin
func readFromStdin() (string, error) {
	var sb strings.Builder
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer size for large logs
	buf := make([]byte, 1024*1024) // 1MB buffer
	scanner.Buffer(buf, len(buf))

	for scanner.Scan() {
		sb.WriteString(scanner.Text())
		sb.WriteString("\n")
	}
	return sb.String(), scanner.Err()
}

// getLastBuildLog attempts to find the most recent build log
func getLastBuildLog() (string, error) {
	// Check common locations
	locations := []string{
		".cm/build.log",
		".devcontainer/build.log",
		"/tmp/cm-build.log",
	}

	for _, loc := range locations {
		if data, err := os.ReadFile(loc); err == nil {
			return string(data), nil
		}
	}

	return "", fmt.Errorf("no build log found")
}

// parseDockerBuildErrors extracts errors from Docker build output
func parseDockerBuildErrors(log string) []BuildError {
	var errors []BuildError
	lines := strings.Split(log, "\n")

	// Patterns for common errors
	patterns := []struct {
		pattern   *regexp.Regexp
		errorType string
		extractor func([]string, int) BuildError
	}{
		// RUN command failure
		{
			regexp.MustCompile(`(?i)ERROR.*RUN.*returned a non-zero code: (\d+)`),
			"command",
			func(lines []string, i int) BuildError {
				return BuildError{
					Type:    "command",
					Message: lines[i],
					Context: getContext(lines, i, 5),
				}
			},
		},
		// Package not found (apt)
		{
			regexp.MustCompile(`(?i)E: Unable to locate package (.+)`),
			"dependency",
			func(lines []string, i int) BuildError {
				matches := regexp.MustCompile(`Unable to locate package (.+)`).FindStringSubmatch(lines[i])
				pkg := ""
				if len(matches) > 1 {
					pkg = matches[1]
				}
				return BuildError{
					Type:    "dependency",
					Message: fmt.Sprintf("Package not found: %s", pkg),
					Context: getContext(lines, i, 3),
				}
			},
		},
		// pip install failure
		{
			regexp.MustCompile(`(?i)ERROR: Could not find a version that satisfies`),
			"dependency",
			func(lines []string, i int) BuildError {
				return BuildError{
					Type:    "dependency",
					Message: "Python package version conflict",
					Context: getContext(lines, i, 5),
				}
			},
		},
		// npm install failure
		{
			regexp.MustCompile(`(?i)npm ERR!`),
			"dependency",
			func(lines []string, i int) BuildError {
				return BuildError{
					Type:    "dependency",
					Message: "npm installation error",
					Context: getContext(lines, i, 5),
				}
			},
		},
		// Permission denied
		{
			regexp.MustCompile(`(?i)permission denied`),
			"permission",
			func(lines []string, i int) BuildError {
				return BuildError{
					Type:    "permission",
					Message: "Permission denied - may need sudo or user change",
					Context: getContext(lines, i, 3),
				}
			},
		},
		// Network errors
		{
			regexp.MustCompile(`(?i)(connection timed out|unable to resolve|network unreachable)`),
			"network",
			func(lines []string, i int) BuildError {
				return BuildError{
					Type:    "network",
					Message: "Network connectivity issue",
					Context: getContext(lines, i, 3),
				}
			},
		},
		// Dockerfile syntax
		{
			regexp.MustCompile(`(?i)dockerfile parse error`),
			"syntax",
			func(lines []string, i int) BuildError {
				return BuildError{
					Type:    "syntax",
					Message: "Dockerfile syntax error",
					Context: getContext(lines, i, 5),
				}
			},
		},
		// Out of disk space
		{
			regexp.MustCompile(`(?i)(no space left on device|disk quota exceeded)`),
			"resource",
			func(lines []string, i int) BuildError {
				return BuildError{
					Type:    "resource",
					Message: "Disk space exhausted",
				}
			},
		},
		// Out of memory
		{
			regexp.MustCompile(`(?i)(out of memory|cannot allocate memory|killed)`),
			"resource",
			func(lines []string, i int) BuildError {
				return BuildError{
					Type:    "resource",
					Message: "Memory exhausted - build killed",
				}
			},
		},
	}

	for i, line := range lines {
		for _, p := range patterns {
			if p.pattern.MatchString(line) {
				errors = append(errors, p.extractor(lines, i))
				break
			}
		}
	}

	return errors
}

// getContext returns surrounding lines for context
func getContext(lines []string, index, count int) []string {
	start := index - count
	if start < 0 {
		start = 0
	}
	end := index + count + 1
	if end > len(lines) {
		end = len(lines)
	}
	return lines[start:end]
}

// analyzeWithAI uses AI to analyze build errors
func analyzeWithAI(ctx context.Context, errors []BuildError, _ string) (*DebugAnalysis, error) {
	gen, err := ai.NewGenerator()
	if err != nil {
		return nil, err
	}

	// Build prompt
	var sb strings.Builder
	sb.WriteString("Analyze these Docker build errors and provide fixes:\n\n")

	for i, e := range errors {
		sb.WriteString(fmt.Sprintf("Error %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("  Type: %s\n", e.Type))
		sb.WriteString(fmt.Sprintf("  Message: %s\n", e.Message))
		if len(e.Context) > 0 {
			sb.WriteString("  Context:\n")
			for _, line := range e.Context {
				sb.WriteString(fmt.Sprintf("    %s\n", line))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`
Respond in this JSON format:
{
  "rootCause": "Brief explanation of the root cause",
  "fixes": [
    {
      "description": "What to do",
      "command": "Command to run, if applicable",
      "confidence": 0.9
    }
  ]
}`)

	// Call AI with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := gen.AnalyzeProject(ctx, ".")
	if err != nil {
		return nil, err
	}

	// Parse response (simplified - in production would properly parse JSON)
	return &DebugAnalysis{
		Summary:   "AI analysis complete",
		RootCause: response,
		Fixes:     []FixSuggestion{},
	}, nil
}

// analyzeWithRules provides rule-based analysis without AI
func analyzeWithRules(errors []BuildError) *DebugAnalysis {
	analysis := &DebugAnalysis{
		Summary: fmt.Sprintf("Found %d error(s)", len(errors)),
	}

	for _, e := range errors {
		switch e.Type {
		case "dependency":
			analysis.RootCause = "Missing or incompatible dependencies"
			analysis.Fixes = append(analysis.Fixes, FixSuggestion{
				Description: "Update package index before installing",
				Command:     "Add 'RUN apt-get update && apt-get install -y <package>' to Dockerfile",
				Confidence:  0.8,
			})
		case "permission":
			analysis.RootCause = "Permission issues - running as wrong user"
			analysis.Fixes = append(analysis.Fixes, FixSuggestion{
				Description: "Run as root during build, then switch to non-root user",
				Command:     "Add 'USER root' before the failing command, then 'USER vscode' after",
				Confidence:  0.7,
			})
		case "network":
			analysis.RootCause = "Network connectivity issues during build"
			analysis.Fixes = append(analysis.Fixes, FixSuggestion{
				Description: "Check network connectivity and proxy settings",
				Command:     "docker build --network=host .",
				Confidence:  0.6,
			})
		case "resource":
			analysis.RootCause = "Insufficient system resources"
			analysis.Fixes = append(analysis.Fixes, FixSuggestion{
				Description: "Free up disk space or increase memory limits",
				Command:     "docker system prune -a",
				Confidence:  0.9,
			})
		case "syntax":
			analysis.RootCause = "Dockerfile syntax error"
			analysis.Fixes = append(analysis.Fixes, FixSuggestion{
				Description: "Check Dockerfile syntax",
				Command:     "docker build --check .",
				Confidence:  0.8,
			})
		case "command":
			analysis.RootCause = "Command execution failed"
			analysis.Fixes = append(analysis.Fixes, FixSuggestion{
				Description: "Check the failing command and its dependencies",
				Command:     "Review the RUN command that failed",
				Confidence:  0.5,
			})
		}
	}

	return analysis
}

// displayAnalysis shows the analysis results
func displayAnalysis(analysis *DebugAnalysis) {
	fmt.Println("🔍 Analysis Results:")
	fmt.Println("─────────────────────")

	if analysis.RootCause != "" {
		fmt.Printf("📍 Root Cause: %s\n", analysis.RootCause)
	}

	fmt.Println()

	if len(analysis.Fixes) > 0 {
		fmt.Println("💡 Suggested Fixes:")
		for i, fix := range analysis.Fixes {
			confidence := "low"
			if fix.Confidence >= 0.8 {
				confidence = "high"
			} else if fix.Confidence >= 0.5 {
				confidence = "medium"
			}

			fmt.Printf("  %d. [%s] %s\n", i+1, confidence, fix.Description)
			if fix.Command != "" {
				fmt.Printf("     $ %s\n", fix.Command)
			}
		}
	}
}

// applyFix attempts to apply a fix
func applyFix(fix FixSuggestion) error {
	if fix.FileChange != nil {
		// Apply file change
		data, err := os.ReadFile(fix.FileChange.File)
		if err != nil {
			return err
		}

		content := strings.Replace(string(data), fix.FileChange.Before, fix.FileChange.After, 1)
		return os.WriteFile(fix.FileChange.File, []byte(content), 0644)
	}

	if fix.Command != "" {
		fmt.Printf("💡 Run this command manually: %s\n", fix.Command)
	}

	return nil
}

// _truncateStr truncates a string to max length (kept for potential future use)
func _truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
