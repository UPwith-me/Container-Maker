package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/UPwith-me/Container-Maker/pkg/userconfig"
)

// Generator generates devcontainer.json using AI
type Generator struct {
	apiKey  string
	apiBase string
	model   string
}

// NewGenerator creates a new AI generator
func NewGenerator() (*Generator, error) {
	cfg, err := userconfig.Load()
	if err != nil {
		return nil, err
	}

	if !cfg.AI.Enabled {
		return nil, fmt.Errorf("AI is not enabled. Run 'cm config set ai.enabled true' first")
	}

	if cfg.AI.APIKey == "" {
		return nil, fmt.Errorf("AI API key not set. Run 'cm config set ai.api_key <key>'")
	}

	apiBase := cfg.AI.APIBase
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}

	return &Generator{
		apiKey:  cfg.AI.APIKey,
		apiBase: apiBase,
		model:   "gpt-4o-mini", // Default to cheaper model
	}, nil
}

// AnalyzeProject analyzes a project and generates devcontainer.json
func (g *Generator) AnalyzeProject(ctx context.Context, projectDir string) (string, error) {
	// Collect project info
	projectInfo := g.collectProjectInfo(projectDir)

	// Generate prompt
	prompt := g.buildPrompt(projectInfo)

	// Call AI API
	response, err := g.callAPI(ctx, prompt)
	if err != nil {
		return "", err
	}

	return response, nil
}

// ProjectInfo holds information about a project
type ProjectInfo struct {
	Name            string
	Languages       []string
	Dependencies    []string
	HasDockerfile   bool
	HasMakefile     bool
	HasPackageJSON  bool
	HasGoMod        bool
	HasPyProject    bool
	HasCondaEnv     bool   // environment.yml
	HasPoetry       bool   // poetry.lock
	HasPipenv       bool   // Pipfile.lock
	HasCMake        bool   // CMakeLists.txt
	HasConan        bool   // conanfile.txt/py
	HasVcpkg        bool   // vcpkg.json
	HasMaven        bool   // pom.xml
	HasGradle       bool   // build.gradle
	HasCargo        bool   // Cargo.toml
	HasDotnet       bool   // *.csproj
	HasComposer     bool   // composer.json
	CondaEnvContent string // Contents of environment.yml
	Files           []string
}

// collectProjectInfo gathers project information
func (g *Generator) collectProjectInfo(projectDir string) *ProjectInfo {
	info := &ProjectInfo{
		Name: filepath.Base(projectDir),
	}

	// Check for common files
	checks := map[string]*bool{
		"Dockerfile":      &info.HasDockerfile,
		"Makefile":        &info.HasMakefile,
		"package.json":    &info.HasPackageJSON,
		"go.mod":          &info.HasGoMod,
		"pyproject.toml":  &info.HasPyProject,
		"environment.yml": &info.HasCondaEnv,
		"poetry.lock":     &info.HasPoetry,
		"Pipfile.lock":    &info.HasPipenv,
		"CMakeLists.txt":  &info.HasCMake,
		"conanfile.txt":   &info.HasConan,
		"vcpkg.json":      &info.HasVcpkg,
		"pom.xml":         &info.HasMaven,
		"build.gradle":    &info.HasGradle,
		"Cargo.toml":      &info.HasCargo,
		"composer.json":   &info.HasComposer,
	}

	for file, flag := range checks {
		if _, err := os.Stat(filepath.Join(projectDir, file)); err == nil {
			*flag = true
		}
	}

	// Check for .csproj files
	if matches, _ := filepath.Glob(filepath.Join(projectDir, "*.csproj")); len(matches) > 0 {
		info.HasDotnet = true
	}

	// Detect languages based on files
	if info.HasGoMod {
		info.Languages = append(info.Languages, "Go")
	}
	if info.HasPackageJSON {
		info.Languages = append(info.Languages, "JavaScript/TypeScript")
	}
	if info.HasPyProject || info.HasCondaEnv || info.HasPoetry || info.HasPipenv {
		info.Languages = append(info.Languages, "Python")
	}
	if info.HasCMake || info.HasConan || info.HasVcpkg {
		info.Languages = append(info.Languages, "C/C++")
	}
	if info.HasMaven || info.HasGradle {
		info.Languages = append(info.Languages, "Java")
	}
	if info.HasCargo {
		info.Languages = append(info.Languages, "Rust")
	}
	if info.HasDotnet {
		info.Languages = append(info.Languages, ".NET/C#")
	}
	if info.HasComposer {
		info.Languages = append(info.Languages, "PHP")
	}

	// Read environment.yml for Conda
	if data, err := os.ReadFile(filepath.Join(projectDir, "environment.yml")); err == nil {
		info.CondaEnvContent = string(data)
		// Extract dependencies from YAML (simple parsing)
		lines := strings.Split(string(data), "\n")
		inDeps := false
		for _, line := range lines {
			if strings.Contains(line, "dependencies:") {
				inDeps = true
				continue
			}
			if inDeps && strings.HasPrefix(strings.TrimSpace(line), "-") {
				dep := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
				if dep != "" && !strings.HasPrefix(dep, "pip:") {
					info.Dependencies = append(info.Dependencies, dep)
				}
			}
		}
	}

	// Read requirements.txt if exists
	if data, err := os.ReadFile(filepath.Join(projectDir, "requirements.txt")); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines[:min(10, len(lines))] {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				info.Dependencies = append(info.Dependencies, line)
			}
		}
		if !info.HasCondaEnv && !info.HasPoetry && !info.HasPipenv {
			info.Languages = append(info.Languages, "Python")
		}
	}

	// Read package.json dependencies
	if data, err := os.ReadFile(filepath.Join(projectDir, "package.json")); err == nil {
		var pkg map[string]interface{}
		if json.Unmarshal(data, &pkg) == nil {
			if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
				for dep := range deps {
					info.Dependencies = append(info.Dependencies, dep)
				}
			}
		}
	}

	// Read Cargo.toml dependencies
	if data, err := os.ReadFile(filepath.Join(projectDir, "Cargo.toml")); err == nil {
		lines := strings.Split(string(data), "\n")
		inDeps := false
		for _, line := range lines {
			if strings.Contains(line, "[dependencies]") {
				inDeps = true
				continue
			}
			if inDeps && strings.HasPrefix(line, "[") {
				break
			}
			if inDeps && strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) > 0 {
					info.Dependencies = append(info.Dependencies, strings.TrimSpace(parts[0]))
				}
			}
		}
	}

	// List top-level files
	entries, _ := os.ReadDir(projectDir)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			info.Files = append(info.Files, e.Name())
		}
	}

	return info
}

// buildPrompt creates the AI prompt
func (g *Generator) buildPrompt(info *ProjectInfo) string {
	var sb strings.Builder

	sb.WriteString("Generate a devcontainer.json for the following project:\n\n")
	sb.WriteString(fmt.Sprintf("Project Name: %s\n", info.Name))
	sb.WriteString(fmt.Sprintf("Languages: %s\n", strings.Join(info.Languages, ", ")))

	if len(info.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("Dependencies: %s\n", strings.Join(info.Dependencies[:min(15, len(info.Dependencies))], ", ")))
	}

	// Add environment-specific context
	sb.WriteString("\nEnvironment Details:\n")
	if info.HasCondaEnv {
		sb.WriteString("- Has Conda environment.yml (use miniconda image)\n")
	}
	if info.HasPoetry {
		sb.WriteString("- Has Poetry project (install poetry in postCreateCommand)\n")
	}
	if info.HasPipenv {
		sb.WriteString("- Has Pipenv project (install pipenv in postCreateCommand)\n")
	}
	if info.HasCMake {
		sb.WriteString("- Has CMake project (use C++ devcontainer with cmake)\n")
	}
	if info.HasConan {
		sb.WriteString("- Has Conan package manager (install conan)\n")
	}
	if info.HasVcpkg {
		sb.WriteString("- Has Vcpkg manifest (add vcpkg feature)\n")
	}
	if info.HasMaven {
		sb.WriteString("- Has Maven project (use Java devcontainer with Maven)\n")
	}
	if info.HasGradle {
		sb.WriteString("- Has Gradle project (use Java devcontainer with Gradle)\n")
	}
	if info.HasDotnet {
		sb.WriteString("- Has .NET project (use dotnet devcontainer)\n")
	}
	if info.HasCargo {
		sb.WriteString("- Has Rust/Cargo project\n")
	}
	if info.HasDockerfile {
		sb.WriteString("- Has Dockerfile (consider using build context)\n")
	}
	if info.HasMakefile {
		sb.WriteString("- Has Makefile\n")
	}

	sb.WriteString(fmt.Sprintf("\nFiles: %s\n", strings.Join(info.Files[:min(20, len(info.Files))], ", ")))

	sb.WriteString(`
Generate a complete devcontainer.json with:
1. Appropriate base image for the detected environment
2. Necessary devcontainer features for the detected languages/tools
3. postCreateCommand to install all dependencies
4. Useful VS Code extensions for the languages

Return ONLY the JSON, no explanation.`)

	return sb.String()
}

// callAPI calls the OpenAI-compatible API
func (g *Generator) callAPI(ctx context.Context, prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model": g.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are an expert DevOps engineer. Generate valid devcontainer.json configurations."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", g.apiBase+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	// Extract JSON from response
	content := result.Choices[0].Message.Content
	content = strings.TrimSpace(content)

	// Remove markdown code blocks if present
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	return content, nil
}

// SaveConfig saves the generated config to disk
func (g *Generator) SaveConfig(projectDir, config string) error {
	// Validate JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(config), &js); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Pretty print
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(config), "", "  "); err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	// Create .devcontainer directory
	devcontainerDir := filepath.Join(projectDir, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return err
	}

	// Write file
	configPath := filepath.Join(devcontainerDir, "devcontainer.json")
	return os.WriteFile(configPath, buf.Bytes(), 0644)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
