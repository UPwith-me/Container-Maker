package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Generate a shareable link for this project",
	Long: `Generate a one-click launch link that others can use to instantly
set up the same development environment.

The link encodes the repository URL and configuration, allowing anyone
with Container-Maker installed to clone and run with a single command.

Examples:
  cm share                    # Generate link for current project
  cm share --format markdown  # Output as markdown link`,
	RunE: runShare,
}

var shareFormat string

func init() {
	shareCmd.Flags().StringVar(&shareFormat, "format", "plain", "Output format: plain, markdown, html")
	rootCmd.AddCommand(shareCmd)
}

func runShare(cmd *cobra.Command, args []string) error {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Try to detect git remote
	repoURL := detectGitRemote(cwd)
	if repoURL == "" {
		fmt.Println("⚠️  No git remote found. Please specify the repository URL.")
		return nil
	}

	// Check for devcontainer.json
	configContent := ""
	if data, err := os.ReadFile(".devcontainer/devcontainer.json"); err == nil {
		configContent = base64.StdEncoding.EncodeToString(data)
	} else if data, err := os.ReadFile("devcontainer.json"); err == nil {
		configContent = base64.StdEncoding.EncodeToString(data)
	}

	// Generate the share command
	projectName := filepath.Base(cwd)
	shareCommand := fmt.Sprintf("cm clone %s", repoURL)

	// Generate output based on format
	switch shareFormat {
	case "markdown":
		fmt.Printf("# %s\n\n", projectName)
		fmt.Printf("[![Open in Container-Maker](https://img.shields.io/badge/Open_in-Container--Maker-blue)](%s)\n\n", shareCommand)
		fmt.Printf("```bash\n%s\n```\n", shareCommand)
	case "html":
		fmt.Printf("<a href=\"%s\">Open in Container-Maker</a>\n", shareCommand)
	default:
		fmt.Println("📤 Share Link for", projectName)
		fmt.Println()
		fmt.Println("Run this command to clone and enter the dev container:")
		fmt.Println()
		fmt.Printf("  %s\n", shareCommand)
		fmt.Println()
		if configContent != "" {
			fmt.Println("✅ devcontainer.json will be used")
		} else {
			fmt.Println("⚠️  No devcontainer.json - will auto-detect project type")
		}
	}

	return nil
}

func detectGitRemote(dir string) string {
	// Try to read git config
	gitConfigPath := filepath.Join(dir, ".git", "config")
	data, err := os.ReadFile(gitConfigPath)
	if err != nil {
		return ""
	}

	// Parse for remote origin URL
	lines := strings.Split(string(data), "\n")
	inRemoteOrigin := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "[remote \"origin\"]" {
			inRemoteOrigin = true
			continue
		}
		if inRemoteOrigin && strings.HasPrefix(line, "url = ") {
			return strings.TrimPrefix(line, "url = ")
		}
		if strings.HasPrefix(line, "[") && inRemoteOrigin {
			break
		}
	}

	return ""
}
