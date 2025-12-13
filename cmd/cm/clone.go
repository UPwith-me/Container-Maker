package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/container-make/cm/pkg/config"
	"github.com/container-make/cm/pkg/runner"
	"github.com/container-make/cm/pkg/template"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <repository>",
	Short: "Clone a repository and enter its dev container",
	Long: `Clone a Git repository and automatically set up and enter its development container.

If the repository has a devcontainer.json, it will be used directly.
If not, Container-Maker will detect the project type and create one automatically.

Examples:
  cm clone https://github.com/user/repo
  cm clone git@github.com:user/repo.git
  cm clone https://github.com/user/repo --template pytorch`,
	Args: cobra.ExactArgs(1),
	RunE: runClone,
}

var cloneTemplate string
var cloneNoShell bool

func init() {
	cloneCmd.Flags().StringVar(&cloneTemplate, "template", "", "Force use a specific template")
	cloneCmd.Flags().BoolVar(&cloneNoShell, "no-shell", false, "Don't enter shell after clone")
	rootCmd.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) error {
	repoURL := args[0]

	// Extract repo name from URL
	repoName := extractRepoName(repoURL)
	if repoName == "" {
		return fmt.Errorf("could not determine repository name from URL")
	}

	fmt.Printf("🚀 Cloning %s...\n", repoURL)

	// Step 1: Git clone
	if err := gitClone(repoURL, repoName); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// Step 2: Change to repo directory
	if err := os.Chdir(repoName); err != nil {
		return fmt.Errorf("failed to enter directory: %w", err)
	}

	cwd, _ := os.Getwd()
	fmt.Printf("📁 Entered %s\n", cwd)

	// Step 3: Check for devcontainer.json
	hasConfig := false
	configPath := ""
	if _, err := os.Stat(".devcontainer/devcontainer.json"); err == nil {
		hasConfig = true
		configPath = ".devcontainer/devcontainer.json"
		fmt.Println("✅ Found .devcontainer/devcontainer.json")
	} else if _, err := os.Stat("devcontainer.json"); err == nil {
		hasConfig = true
		configPath = "devcontainer.json"
		fmt.Println("✅ Found devcontainer.json")
	}

	// Step 4: If no config, create one
	if !hasConfig {
		fmt.Println("📦 No devcontainer.json found, creating one...")

		if cloneTemplate != "" {
			// Use specified template
			if err := template.ApplyTemplate(cloneTemplate, cwd); err != nil {
				return fmt.Errorf("failed to apply template: %w", err)
			}
			fmt.Printf("✅ Applied template: %s\n", cloneTemplate)
		} else {
			// Auto-detect project type
			if err := autoCreateConfig(cwd); err != nil {
				return fmt.Errorf("failed to create config: %w", err)
			}
		}
		configPath = ".devcontainer/devcontainer.json"
	}

	// Step 5: Enter shell (unless --no-shell)
	if cloneNoShell {
		fmt.Println("\n✅ Clone complete! Run 'cm shell' to enter the container.")
		return nil
	}

	fmt.Println("\n🐳 Starting dev container...")

	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	pr, err := runner.NewPersistentRunner(cfg, cwd)
	if err != nil {
		return err
	}

	return pr.Shell(context.Background())
}

// extractRepoName extracts the repository name from a Git URL
func extractRepoName(url string) string {
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Handle different URL formats
	// https://github.com/user/repo
	// git@github.com:user/repo
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	// Try colon format (git@github.com:user/repo)
	if idx := strings.LastIndex(url, ":"); idx != -1 {
		remaining := url[idx+1:]
		parts = strings.Split(remaining, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	return ""
}

// gitClone runs git clone
func gitClone(url, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// autoCreateConfig detects project type and creates a devcontainer.json
func autoCreateConfig(projectDir string) error {
	// Detect project type by looking for common files
	templateName := "python-basic" // default

	if fileExists(filepath.Join(projectDir, "go.mod")) {
		templateName = "go-basic"
	} else if fileExists(filepath.Join(projectDir, "package.json")) {
		templateName = "node-basic"
	} else if fileExists(filepath.Join(projectDir, "Cargo.toml")) {
		templateName = "rust-basic"
	} else if fileExists(filepath.Join(projectDir, "requirements.txt")) ||
		fileExists(filepath.Join(projectDir, "pyproject.toml")) ||
		fileExists(filepath.Join(projectDir, "setup.py")) {
		templateName = "python-basic"
	} else if fileExists(filepath.Join(projectDir, "CMakeLists.txt")) {
		templateName = "cpp-cmake"
	}

	fmt.Printf("🔍 Detected project type: %s\n", templateName)

	return template.ApplyTemplate(templateName, projectDir)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
