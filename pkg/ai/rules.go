package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RuleEngine provides rule-based config generation without AI
type RuleEngine struct {
	rules []ConfigRule
}

// ConfigRule represents a single configuration rule
type ConfigRule struct {
	Name      string
	Condition func(*ProjectInfo) bool
	Apply     func(*DevcontainerConfig)
	Priority  int
}

// DevcontainerConfig represents generated devcontainer configuration
type DevcontainerConfig struct {
	Name              string                 `json:"name,omitempty"`
	Image             string                 `json:"image"`
	Features          map[string]interface{} `json:"features,omitempty"`
	RunArgs           []string               `json:"runArgs,omitempty"`
	PostCreateCommand string                 `json:"postCreateCommand,omitempty"`
	ForwardPorts      []int                  `json:"forwardPorts,omitempty"`
	Customizations    map[string]interface{} `json:"customizations,omitempty"`
	RemoteUser        string                 `json:"remoteUser,omitempty"`
	ContainerEnv      map[string]string      `json:"containerEnv,omitempty"`
}

// NewRuleEngine creates a new rule engine with default rules
func NewRuleEngine() *RuleEngine {
	engine := &RuleEngine{}
	engine.registerDefaultRules()
	return engine
}

// registerDefaultRules sets up the built-in rules
func (e *RuleEngine) registerDefaultRules() {
	// Go projects
	e.rules = append(e.rules, ConfigRule{
		Name: "go-basic",
		Condition: func(p *ProjectInfo) bool {
			return p.HasGoMod
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "golang:1.22-alpine"
			c.PostCreateCommand = "go mod download"
			addExtension(c, "golang.go")
		},
		Priority: 100,
	})

	// Python with PyTorch (GPU)
	e.rules = append(e.rules, ConfigRule{
		Name: "pytorch-gpu",
		Condition: func(p *ProjectInfo) bool {
			return containsDep(p.Dependencies, "torch") || containsDep(p.Dependencies, "pytorch")
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "pytorch/pytorch:2.2.0-cuda12.1-cudnn8-runtime"
			c.RunArgs = []string{"--gpus", "all", "--shm-size=4g"}
			c.PostCreateCommand = "pip install -r requirements.txt"
			addExtension(c, "ms-python.python")
			addExtension(c, "ms-toolsai.jupyter")
		},
		Priority: 200,
	})

	// Python with TensorFlow (GPU)
	e.rules = append(e.rules, ConfigRule{
		Name: "tensorflow-gpu",
		Condition: func(p *ProjectInfo) bool {
			return containsDep(p.Dependencies, "tensorflow")
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "tensorflow/tensorflow:2.15.0-gpu"
			c.RunArgs = []string{"--gpus", "all"}
			c.PostCreateCommand = "pip install -r requirements.txt"
			addExtension(c, "ms-python.python")
		},
		Priority: 200,
	})

	// Python with Poetry
	e.rules = append(e.rules, ConfigRule{
		Name: "python-poetry",
		Condition: func(p *ProjectInfo) bool {
			return p.HasPoetry
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "python:3.11-slim"
			c.PostCreateCommand = "pip install poetry && poetry install"
			addExtension(c, "ms-python.python")
		},
		Priority: 90,
	})

	// Python with Conda
	e.rules = append(e.rules, ConfigRule{
		Name: "python-conda",
		Condition: func(p *ProjectInfo) bool {
			return p.HasCondaEnv
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "continuumio/miniconda3:latest"
			c.PostCreateCommand = "conda env update -f environment.yml"
			addExtension(c, "ms-python.python")
		},
		Priority: 90,
	})

	// Python basic
	e.rules = append(e.rules, ConfigRule{
		Name: "python-basic",
		Condition: func(p *ProjectInfo) bool {
			return p.HasPyProject || containsLanguage(p, "Python")
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "python:3.11-slim"
			c.PostCreateCommand = "pip install -r requirements.txt 2>/dev/null || true"
			addExtension(c, "ms-python.python")
		},
		Priority: 50,
	})

	// Node.js with Next.js
	e.rules = append(e.rules, ConfigRule{
		Name: "nextjs",
		Condition: func(p *ProjectInfo) bool {
			return containsDep(p.Dependencies, "next")
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "node:20-slim"
			c.PostCreateCommand = "npm install"
			c.ForwardPorts = []int{3000}
			addExtension(c, "dbaeumer.vscode-eslint")
		},
		Priority: 150,
	})

	// Node.js basic
	e.rules = append(e.rules, ConfigRule{
		Name: "node-basic",
		Condition: func(p *ProjectInfo) bool {
			return p.HasPackageJSON
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "node:20-slim"
			c.PostCreateCommand = "npm install"
			addExtension(c, "dbaeumer.vscode-eslint")
		},
		Priority: 80,
	})

	// Rust
	e.rules = append(e.rules, ConfigRule{
		Name: "rust-basic",
		Condition: func(p *ProjectInfo) bool {
			return p.HasCargo
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "rust:1.75-slim"
			c.PostCreateCommand = "cargo fetch"
			addExtension(c, "rust-lang.rust-analyzer")
		},
		Priority: 100,
	})

	// Java with Maven
	e.rules = append(e.rules, ConfigRule{
		Name: "java-maven",
		Condition: func(p *ProjectInfo) bool {
			return p.HasMaven
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "maven:3.9-eclipse-temurin-21"
			c.PostCreateCommand = "mvn dependency:resolve"
			addExtension(c, "vscjava.vscode-java-pack")
		},
		Priority: 100,
	})

	// Java with Gradle
	e.rules = append(e.rules, ConfigRule{
		Name: "java-gradle",
		Condition: func(p *ProjectInfo) bool {
			return p.HasGradle
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "gradle:8.5-jdk21"
			c.PostCreateCommand = "gradle dependencies"
			addExtension(c, "vscjava.vscode-java-pack")
		},
		Priority: 100,
	})

	// C++ with CMake
	e.rules = append(e.rules, ConfigRule{
		Name: "cpp-cmake",
		Condition: func(p *ProjectInfo) bool {
			return p.HasCMake
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "gcc:13"
			c.PostCreateCommand = "apt-get update && apt-get install -y cmake ninja-build"
			addExtension(c, "ms-vscode.cpptools")
			addExtension(c, "ms-vscode.cmake-tools")
		},
		Priority: 100,
	})

	// .NET
	e.rules = append(e.rules, ConfigRule{
		Name: "dotnet",
		Condition: func(p *ProjectInfo) bool {
			return p.HasDotnet
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "mcr.microsoft.com/dotnet/sdk:8.0"
			c.PostCreateCommand = "dotnet restore"
			addExtension(c, "ms-dotnettools.csdevkit")
		},
		Priority: 100,
	})

	// PHP with Composer
	e.rules = append(e.rules, ConfigRule{
		Name: "php-composer",
		Condition: func(p *ProjectInfo) bool {
			return p.HasComposer
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "php:8.3-cli"
			c.PostCreateCommand = "composer install"
			addExtension(c, "bmewburn.vscode-intelephense-client")
		},
		Priority: 100,
	})

	// Default fallback
	e.rules = append(e.rules, ConfigRule{
		Name: "default",
		Condition: func(p *ProjectInfo) bool {
			return true
		},
		Apply: func(c *DevcontainerConfig) {
			c.Image = "mcr.microsoft.com/devcontainers/base:ubuntu"
		},
		Priority: 1,
	})
}

// Generate creates a devcontainer.json from rules
func (e *RuleEngine) Generate(info *ProjectInfo) (string, error) {
	config := &DevcontainerConfig{
		Name:       info.Name,
		RemoteUser: "vscode",
	}

	// Apply matching rules in priority order
	var appliedRules []string
	for _, rule := range e.rules {
		if rule.Condition(info) {
			rule.Apply(config)
			appliedRules = append(appliedRules, rule.Name)
			break // Apply only the highest priority matching rule
		}
	}

	// Add common settings
	if config.Customizations == nil {
		config.Customizations = make(map[string]interface{})
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Helper functions
func addExtension(c *DevcontainerConfig, ext string) {
	if c.Customizations == nil {
		c.Customizations = make(map[string]interface{})
	}
	if c.Customizations["vscode"] == nil {
		c.Customizations["vscode"] = map[string]interface{}{
			"extensions": []string{},
		}
	}
	vscode := c.Customizations["vscode"].(map[string]interface{})
	extensions := vscode["extensions"].([]string)
	vscode["extensions"] = append(extensions, ext)
}

func containsDep(deps []string, name string) bool {
	for _, d := range deps {
		if strings.Contains(strings.ToLower(d), strings.ToLower(name)) {
			return true
		}
	}
	return false
}

func containsLanguage(p *ProjectInfo, lang string) bool {
	for _, l := range p.Languages {
		if strings.EqualFold(l, lang) {
			return true
		}
	}
	return false
}

// Optimization suggestions
type OptimizationSuggestion struct {
	Title       string
	Description string
	Impact      string // "high", "medium", "low"
	Category    string // "performance", "security", "productivity"
	Apply       func(config map[string]interface{})
}

// Optimizer analyzes and suggests improvements for configs
type Optimizer struct {
	suggestions []OptimizationSuggestion
}

// NewOptimizer creates a new config optimizer
func NewOptimizer() *Optimizer {
	o := &Optimizer{}
	o.registerSuggestions()
	return o
}

func (o *Optimizer) registerSuggestions() {
	// Performance optimizations
	o.suggestions = append(o.suggestions, OptimizationSuggestion{
		Title:       "Add build cache mount",
		Description: "Mount a cache volume to speed up repeated builds",
		Impact:      "high",
		Category:    "performance",
		Apply: func(c map[string]interface{}) {
			// Add cache mount suggestion
		},
	})

	o.suggestions = append(o.suggestions, OptimizationSuggestion{
		Title:       "Use multi-stage build",
		Description: "Reduce final image size by separating build and runtime stages",
		Impact:      "medium",
		Category:    "performance",
	})

	// Security optimizations
	o.suggestions = append(o.suggestions, OptimizationSuggestion{
		Title:       "Run as non-root user",
		Description: "Set remoteUser to avoid running as root",
		Impact:      "high",
		Category:    "security",
		Apply: func(c map[string]interface{}) {
			c["remoteUser"] = "vscode"
		},
	})

	o.suggestions = append(o.suggestions, OptimizationSuggestion{
		Title:       "Pin image versions",
		Description: "Use specific version tags instead of :latest",
		Impact:      "medium",
		Category:    "security",
	})

	// Productivity optimizations
	o.suggestions = append(o.suggestions, OptimizationSuggestion{
		Title:       "Add recommended extensions",
		Description: "Include language-specific VS Code extensions",
		Impact:      "medium",
		Category:    "productivity",
	})

	o.suggestions = append(o.suggestions, OptimizationSuggestion{
		Title:       "Configure port forwarding",
		Description: "Auto-forward common development ports",
		Impact:      "low",
		Category:    "productivity",
	})
}

// Analyze analyzes a config and returns suggestions
func (o *Optimizer) Analyze(configJSON string) []OptimizationSuggestion {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil
	}

	var applicable []OptimizationSuggestion

	// Check for non-root user
	if _, hasRemoteUser := config["remoteUser"]; !hasRemoteUser {
		applicable = append(applicable, o.findSuggestion("Run as non-root user"))
	}

	// Check for image version pinning
	if image, ok := config["image"].(string); ok {
		if strings.Contains(image, ":latest") || !strings.Contains(image, ":") {
			applicable = append(applicable, o.findSuggestion("Pin image versions"))
		}
	}

	// Check for extensions
	if customizations, ok := config["customizations"].(map[string]interface{}); ok {
		if vscode, ok := customizations["vscode"].(map[string]interface{}); ok {
			if extensions, ok := vscode["extensions"].([]interface{}); ok {
				if len(extensions) == 0 {
					applicable = append(applicable, o.findSuggestion("Add recommended extensions"))
				}
			}
		}
	} else {
		applicable = append(applicable, o.findSuggestion("Add recommended extensions"))
	}

	// Check for port forwarding
	if _, hasPorts := config["forwardPorts"]; !hasPorts {
		applicable = append(applicable, o.findSuggestion("Configure port forwarding"))
	}

	return applicable
}

func (o *Optimizer) findSuggestion(title string) OptimizationSuggestion {
	for _, s := range o.suggestions {
		if s.Title == title {
			return s
		}
	}
	return OptimizationSuggestion{}
}

// FormatSuggestions formats suggestions for display
func FormatSuggestions(suggestions []OptimizationSuggestion) string {
	if len(suggestions) == 0 {
		return "✅ No optimizations suggested. Your config looks good!"
	}

	var sb strings.Builder
	sb.WriteString("🚀 Optimization Suggestions:\n\n")

	for i, s := range suggestions {
		impact := "🟡"
		if s.Impact == "high" {
			impact = "🔴"
		} else if s.Impact == "low" {
			impact = "🟢"
		}

		sb.WriteString(fmt.Sprintf("%d. [%s %s] %s\n", i+1, impact, s.Impact, s.Title))
		sb.WriteString(fmt.Sprintf("   %s\n", s.Description))
		sb.WriteString(fmt.Sprintf("   Category: %s\n\n", s.Category))
	}

	return sb.String()
}
