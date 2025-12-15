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
	Template    string // Suggested template name
}

// DetectedProject contains detection results
type DetectedProject struct {
	Types       []ProjectType
	Primary     *ProjectType
	HasMultiple bool
}

// Detection rules - ordered by priority (lower = higher priority)
// More specific rules should have lower priority numbers
var detectionRules = []struct {
	Files       []string
	Language    string
	Image       string
	Priority    int
	Description string
	Template    string // Suggested template name
}{
	// === Python Complex Environments (highest priority for specific tools) ===
	{[]string{"environment.yml", "environment.yaml"}, "Python (Conda)", "mcr.microsoft.com/devcontainers/miniconda:3", 1, "Conda environment", "miniconda"},
	{[]string{"poetry.lock"}, "Python (Poetry)", "mcr.microsoft.com/devcontainers/python:3.11", 2, "Poetry project", "python-poetry"},
	{[]string{"Pipfile.lock"}, "Python (Pipenv)", "mcr.microsoft.com/devcontainers/python:3.11", 3, "Pipenv project", "python-pipenv"},

	// === C/C++ Complex Build Systems ===
	{[]string{"conanfile.txt", "conanfile.py"}, "C++ (Conan)", "mcr.microsoft.com/devcontainers/cpp:ubuntu", 4, "C++ Conan project", "cpp-conan"},
	{[]string{"vcpkg.json"}, "C++ (Vcpkg)", "mcr.microsoft.com/devcontainers/cpp:ubuntu", 5, "C++ Vcpkg project", "cpp-vcpkg"},
	{[]string{"CMakeLists.txt"}, "C++ (CMake)", "mcr.microsoft.com/devcontainers/cpp:ubuntu", 6, "CMake project", "cpp-cmake"},

	// === Java Build Systems ===
	{[]string{"pom.xml"}, "Java (Maven)", "mcr.microsoft.com/devcontainers/java:17", 7, "Maven project", "java-maven"},
	{[]string{"build.gradle", "build.gradle.kts"}, "Java (Gradle)", "mcr.microsoft.com/devcontainers/java:17", 8, "Gradle project", "java-gradle"},

	// === Standard Language Detection ===
	{[]string{"go.mod", "go.sum"}, "Go", "golang:1.21-alpine", 10, "Go project", "go-basic"},
	{[]string{"package.json"}, "Node.js", "node:20-alpine", 11, "Node.js project", "node-basic"},
	{[]string{"requirements.txt", "pyproject.toml", "setup.py", "Pipfile"}, "Python", "python:3.11-slim", 12, "Python project", "python-basic"},
	{[]string{"Cargo.toml"}, "Rust", "rust:alpine", 13, "Rust project", "rust-basic"},
	{[]string{"*.csproj", "*.sln"}, ".NET", "mcr.microsoft.com/dotnet/sdk:8.0", 14, ".NET project", "dotnet"},
	{[]string{"composer.json"}, "PHP", "php:8.2-cli", 15, "PHP project", "php-composer"},
	{[]string{"Gemfile"}, "Ruby", "ruby:3.2-slim", 16, "Ruby project", "ruby-basic"},
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
					Template:    rule.Template,
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
			Priority:    20, // Lower priority than CMake
			Description: "C/C++ project with Makefile",
			Template:    "cpp-makefile",
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
	_, _ = fmt.Scanln(&choice)

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
