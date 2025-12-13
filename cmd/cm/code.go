package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Open project in VS Code with Dev Container",
	Long: `Open the current project in VS Code and connect to the dev container.

Requires VS Code and the "Dev Containers" extension to be installed.

Examples:
  cm code           # Open current directory
  cm code ./myapp   # Open specific directory`,
	RunE: runCode,
}

func init() {
	rootCmd.AddCommand(codeCmd)
}

func runCode(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	// Get absolute path
	absPath, err := os.Getwd()
	if err != nil {
		return err
	}
	if dir != "." {
		absPath = dir
	}

	// Check if devcontainer.json exists
	hasConfig := false
	if _, err := os.Stat(".devcontainer/devcontainer.json"); err == nil {
		hasConfig = true
	} else if _, err := os.Stat("devcontainer.json"); err == nil {
		hasConfig = true
	}

	if !hasConfig {
		fmt.Println("⚠️  No devcontainer.json found. Run 'cm init' first.")
		return nil
	}

	fmt.Printf("🚀 Opening %s in VS Code...\n", absPath)

	// Find VS Code command
	codeExe := findVSCode()
	if codeExe == "" {
		fmt.Println("❌ VS Code not found. Please install it from https://code.visualstudio.com")
		return nil
	}

	// Open in VS Code with Dev Containers
	// The --folder-uri approach opens directly in container
	devContainerURI := fmt.Sprintf("vscode-remote://dev-container+%s/workspaces/%s",
		hexEncode(absPath), getBaseName(absPath))

	// First, just open the folder - VS Code will prompt to reopen in container
	execCmd := exec.Command(codeExe, absPath)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Start(); err != nil {
		return fmt.Errorf("failed to open VS Code: %w", err)
	}

	fmt.Println("✅ VS Code opened!")
	fmt.Println("💡 Tip: Click 'Reopen in Container' when prompted.")
	fmt.Printf("   Dev Container URI: %s\n", devContainerURI)

	return nil
}

func findVSCode() string {
	// Try common VS Code commands
	candidates := []string{"code", "code-insiders"}

	if runtime.GOOS == "windows" {
		// Add Windows-specific paths
		localAppData := os.Getenv("LOCALAPPDATA")
		programFiles := os.Getenv("ProgramFiles")

		if localAppData != "" {
			candidates = append(candidates,
				localAppData+"\\Programs\\Microsoft VS Code\\bin\\code.cmd",
				localAppData+"\\Programs\\Microsoft VS Code Insiders\\bin\\code-insiders.cmd",
			)
		}
		if programFiles != "" {
			candidates = append(candidates,
				programFiles+"\\Microsoft VS Code\\bin\\code.cmd",
			)
		}
	}

	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			return path
		}
		// Also try the full path directly
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return ""
}

func hexEncode(s string) string {
	var result strings.Builder
	for _, b := range []byte(s) {
		result.WriteString(fmt.Sprintf("%02x", b))
	}
	return result.String()
}

func getBaseName(path string) string {
	// Get the last component of the path
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return "workspace"
}
