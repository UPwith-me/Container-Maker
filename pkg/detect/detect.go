package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectType represents a detected project type
type ProjectType struct {
	Name        string
	Language    string
	Image       string
	DetectedBy  string
	Priority    int
	Description string
}

// DetectedProject contains detection results
type DetectedProject struct {
	Types       []ProjectType
	Primary     *ProjectType
	HasMultiple bool
}

// Detection rules - ordered by priority (lower = higher priority)
var detectionRules = []struct {
	Files       []string
	Language    string
	Image       string
	Priority    int
	Description string
}{
	{[]string{"go.mod", "go.sum"}, "Go", "golang:1.21-alpine", 1, "Go project"},
	{[]string{"package.json"}, "Node.js", "node:20-alpine", 2, "Node.js project"},
	{[]string{"requirements.txt", "pyproject.toml", "setup.py", "Pipfile"}, "Python", "python:3.11-slim", 3, "Python project"},
	{[]string{"Cargo.toml"}, "Rust", "rust:alpine", 4, "Rust project"},
	{[]string{"pom.xml", "build.gradle", "build.gradle.kts"}, "Java", "eclipse-temurin:17", 5, "Java project"},
	{[]string{"*.csproj", "*.sln"}, ".NET", "mcr.microsoft.com/dotnet/sdk:8.0", 6, ".NET project"},
	{[]string{"composer.json"}, "PHP", "php:8.2-cli", 7, "PHP project"},
	{[]string{"Gemfile"}, "Ruby", "ruby:3.2-slim", 8, "Ruby project"},
}

// DetectProjectType scans the current directory for project indicators
func DetectProjectType(dir string) *DetectedProject {
	result := &DetectedProject{
		Types: []ProjectType{},
	}

	// Check each detection rule
	for _, rule := range detectionRules {
		for _, pattern := range rule.Files {
			matches, _ := filepath.Glob(filepath.Join(dir, pattern))
			if len(matches) > 0 {
				pt := ProjectType{
					Name:        rule.Language,
					Language:    rule.Language,
					Image:       rule.Image,
					DetectedBy:  filepath.Base(matches[0]),
					Priority:    rule.Priority,
					Description: rule.Description,
				}
				result.Types = append(result.Types, pt)
				break // Only count once per rule
			}
		}
	}

	// Check for Makefile with C/C++ files
	if hasMakefile(dir) && hasCFiles(dir) {
		result.Types = append(result.Types, ProjectType{
			Name:        "C/C++",
			Language:    "C/C++",
			Image:       "gcc:latest",
			DetectedBy:  "Makefile + *.c/*.cpp",
			Priority:    5,
			Description: "C/C++ project with Makefile",
		})
	}

	// Sort by priority and set primary
	if len(result.Types) > 0 {
		result.HasMultiple = len(result.Types) > 1
		result.Primary = &result.Types[0]
		for i := range result.Types {
			if result.Types[i].Priority < result.Primary.Priority {
				result.Primary = &result.Types[i]
			}
		}
	}

	return result
}

func hasMakefile(dir string) bool {
	names := []string{"Makefile", "makefile", "GNUmakefile"}
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func hasCFiles(dir string) bool {
	patterns := []string{"*.c", "*.cpp", "*.cc", "*.h", "*.hpp"}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

// HasDockerfile checks if there's a Dockerfile in the directory
func HasDockerfile(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "Dockerfile"))
	return err == nil
}

// FormatDetectionResult returns a formatted string for display
func FormatDetectionResult(result *DetectedProject) string {
	if result.Primary == nil {
		return "📂 No project type detected"
	}

	var sb strings.Builder

	if result.HasMultiple {
		sb.WriteString("📂 Multiple project types detected:\n")
		for _, t := range result.Types {
			sb.WriteString(fmt.Sprintf("   • %s found → %s\n", t.DetectedBy, t.Language))
		}
		sb.WriteString(fmt.Sprintf("\n📦 Primary suggestion: %s (%s)\n", result.Primary.Image, result.Primary.Language))
	} else {
		sb.WriteString(fmt.Sprintf("📂 Detected: %s (%s found)\n", result.Primary.Description, result.Primary.DetectedBy))
		sb.WriteString(fmt.Sprintf("📦 Suggested image: %s\n", result.Primary.Image))
	}

	return sb.String()
}

// PromptAutoDetect prompts user to confirm auto-detected image
func PromptAutoDetect(result *DetectedProject) (string, bool, error) {
	if result.Primary == nil {
		return "", false, fmt.Errorf("no project type detected")
	}

	fmt.Println()
	fmt.Println("🔍 No devcontainer.json found")
	fmt.Println()
	fmt.Print(FormatDetectionResult(result))
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  [1] Use this image (one-time)")
	fmt.Println("  [2] Use and save to devcontainer.json")
	fmt.Println("  [3] Choose different image (cm images)")
	fmt.Println("  [q] Cancel")
	fmt.Println()
	fmt.Print("Choice [1]: ")

	var choice string
	fmt.Scanln(&choice)

	if choice == "" {
		choice = "1"
	}

	switch strings.ToLower(choice) {
	case "1":
		return result.Primary.Image, false, nil
	case "2":
		return result.Primary.Image, true, nil
	case "3":
		fmt.Println("\nRun 'cm images' to see available images,")
		fmt.Println("then use 'cm images use <name>' to set up.")
		return "", false, fmt.Errorf("user chose to select different image")
	case "q", "quit", "exit":
		return "", false, fmt.Errorf("cancelled by user")
	default:
		// Treat as "1"
		return result.Primary.Image, false, nil
	}
}

// CreateDevcontainerConfig creates a minimal devcontainer.json
func CreateDevcontainerConfig(dir, image, name string) error {
	devcontainerDir := filepath.Join(dir, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return err
	}

	content := fmt.Sprintf(`{
  "name": "%s",
  "image": "%s"
}
`, name, image)

	configPath := filepath.Join(devcontainerDir, "devcontainer.json")
	return os.WriteFile(configPath, []byte(content), 0644)
}
