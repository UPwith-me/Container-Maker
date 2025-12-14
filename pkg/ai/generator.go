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
	Name           string
	Languages      []string
	Dependencies   []string
	HasDockerfile  bool
	HasMakefile    bool
	HasPackageJSON bool
	HasGoMod       bool
	HasPyProject   bool
	Files          []string
}

// collectProjectInfo gathers project information
func (g *Generator) collectProjectInfo(projectDir string) *ProjectInfo {
	info := &ProjectInfo{
		Name: filepath.Base(projectDir),
	}

	// Check for common files
	checks := map[string]*bool{
		"Dockerfile":     &info.HasDockerfile,
		"Makefile":       &info.HasMakefile,
		"package.json":   &info.HasPackageJSON,
		"go.mod":         &info.HasGoMod,
		"pyproject.toml": &info.HasPyProject,
	}

	for file, flag := range checks {
		if _, err := os.Stat(filepath.Join(projectDir, file)); err == nil {
			*flag = true
		}
	}

	// Detect languages
	if info.HasGoMod {
		info.Languages = append(info.Languages, "Go")
	}
	if info.HasPackageJSON {
		info.Languages = append(info.Languages, "JavaScript/TypeScript")
	}
	if info.HasPyProject {
		info.Languages = append(info.Languages, "Python")
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
		info.Languages = append(info.Languages, "Python")
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
		sb.WriteString(fmt.Sprintf("Dependencies: %s\n", strings.Join(info.Dependencies[:min(10, len(info.Dependencies))], ", ")))
	}

	sb.WriteString(fmt.Sprintf("Has Dockerfile: %v\n", info.HasDockerfile))
	sb.WriteString(fmt.Sprintf("Has Makefile: %v\n", info.HasMakefile))
	sb.WriteString(fmt.Sprintf("Files: %s\n", strings.Join(info.Files[:min(20, len(info.Files))], ", ")))

	sb.WriteString(`
Generate a complete devcontainer.json with:
1. Appropriate base image
2. Necessary features for the detected languages
3. postCreateCommand to install dependencies
4. Useful VS Code extensions

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
	json.Indent(&buf, []byte(config), "", "  ")

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
