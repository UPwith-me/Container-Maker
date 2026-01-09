package detect

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MultiLangConfig generates a devcontainer.json that supports multiple languages
type MultiLangConfig struct {
	Name              string                 `json:"name"`
	Image             string                 `json:"image"`
	Features          map[string]interface{} `json:"features,omitempty"`
	RunArgs           []string               `json:"runArgs,omitempty"`
	ForwardPorts      []int                  `json:"forwardPorts,omitempty"`
	PostCreateCommand string                 `json:"postCreateCommand,omitempty"`
	Customizations    map[string]interface{} `json:"customizations,omitempty"`
	RemoteUser        string                 `json:"remoteUser,omitempty"`
}

// LanguageFeature maps languages to devcontainer features
var LanguageFeatures = map[string]struct {
	Feature    string
	Config     map[string]interface{}
	Extensions []string
	PostCmd    string
}{
	"Go": {
		Feature:    "ghcr.io/devcontainers/features/go:1",
		Config:     map[string]interface{}{"version": "latest"},
		Extensions: []string{"golang.go"},
		PostCmd:    "go mod download 2>/dev/null || true",
	},
	"Python": {
		Feature:    "ghcr.io/devcontainers/features/python:1",
		Config:     map[string]interface{}{"version": "3.11"},
		Extensions: []string{"ms-python.python", "ms-python.vscode-pylance"},
		PostCmd:    "pip install -r requirements.txt 2>/dev/null || true",
	},
	"JavaScript": {
		Feature:    "ghcr.io/devcontainers/features/node:1",
		Config:     map[string]interface{}{"version": "20"},
		Extensions: []string{"dbaeumer.vscode-eslint"},
		PostCmd:    "npm install 2>/dev/null || true",
	},
	"TypeScript": {
		Feature:    "ghcr.io/devcontainers/features/node:1",
		Config:     map[string]interface{}{"version": "20"},
		Extensions: []string{"dbaeumer.vscode-eslint", "ms-vscode.vscode-typescript-next"},
		PostCmd:    "npm install 2>/dev/null || true",
	},
	"Rust": {
		Feature:    "ghcr.io/devcontainers/features/rust:1",
		Config:     map[string]interface{}{"version": "latest"},
		Extensions: []string{"rust-lang.rust-analyzer"},
		PostCmd:    "cargo fetch 2>/dev/null || true",
	},
	"Java": {
		Feature:    "ghcr.io/devcontainers/features/java:1",
		Config:     map[string]interface{}{"version": "21"},
		Extensions: []string{"vscjava.vscode-java-pack"},
		PostCmd:    "",
	},
	"C++": {
		Feature:    "ghcr.io/devcontainers/features/common-utils:2",
		Config:     map[string]interface{}{},
		Extensions: []string{"ms-vscode.cpptools", "ms-vscode.cmake-tools"},
		PostCmd:    "",
	},
	".NET/C#": {
		Feature:    "ghcr.io/devcontainers/features/dotnet:2",
		Config:     map[string]interface{}{"version": "8.0"},
		Extensions: []string{"ms-dotnettools.csdevkit"},
		PostCmd:    "dotnet restore 2>/dev/null || true",
	},
	"Ruby": {
		Feature:    "ghcr.io/devcontainers/features/ruby:1",
		Config:     map[string]interface{}{"version": "latest"},
		Extensions: []string{"rebornix.ruby"},
		PostCmd:    "bundle install 2>/dev/null || true",
	},
	"PHP": {
		Feature:    "ghcr.io/devcontainers/features/php:1",
		Config:     map[string]interface{}{"version": "8.3"},
		Extensions: []string{"bmewburn.vscode-intelephense-client"},
		PostCmd:    "composer install 2>/dev/null || true",
	},
}

// GPU features
var GPUFeatures = map[string]struct {
	Image   string
	RunArgs []string
	Feature string
}{
	"PyTorch": {
		Image:   "pytorch/pytorch:2.2.0-cuda12.1-cudnn8-runtime",
		RunArgs: []string{"--gpus", "all", "--shm-size=4g"},
	},
	"TensorFlow": {
		Image:   "tensorflow/tensorflow:2.15.0-gpu",
		RunArgs: []string{"--gpus", "all"},
	},
	"JAX": {
		Image:   "ghcr.io/nvidia/jax:jax",
		RunArgs: []string{"--gpus", "all"},
	},
}

// GenerateMultiLangConfig generates a config supporting all detected languages
func GenerateMultiLangConfig(info *ProjectInfo) (*MultiLangConfig, error) {
	config := &MultiLangConfig{
		Name:       info.Name,
		Image:      "mcr.microsoft.com/devcontainers/base:ubuntu",
		Features:   make(map[string]interface{}),
		RemoteUser: "vscode",
	}

	var postCmds []string
	var extensions []string
	seenFeatures := make(map[string]bool)

	// Handle GPU requirements first (affects base image)
	if info.NeedsGPU {
		for _, gpuFw := range info.GPUFrameworks {
			if gpuCfg, ok := GPUFeatures[gpuFw]; ok {
				config.Image = gpuCfg.Image
				config.RunArgs = append(config.RunArgs, gpuCfg.RunArgs...)
				break
			}
		}
		// Default GPU image if no specific framework
		if config.Image == "mcr.microsoft.com/devcontainers/base:ubuntu" {
			config.Image = "nvidia/cuda:12.1.0-cudnn8-devel-ubuntu22.04"
			config.RunArgs = []string{"--gpus", "all"}
		}

		// Add Python for GPU projects
		if feat, ok := LanguageFeatures["Python"]; ok {
			config.Features[feat.Feature] = feat.Config
			extensions = append(extensions, feat.Extensions...)
			postCmds = append(postCmds, "pip install -r requirements.txt 2>/dev/null || true")
		}
	}

	// Add features for each detected language
	for _, lang := range info.Languages {
		langName := normalizeLangName(lang.Name)

		if feat, ok := LanguageFeatures[langName]; ok {
			// Avoid duplicate features
			if !seenFeatures[feat.Feature] {
				config.Features[feat.Feature] = feat.Config
				seenFeatures[feat.Feature] = true
			}

			// Add extensions
			for _, ext := range feat.Extensions {
				if !contains(extensions, ext) {
					extensions = append(extensions, ext)
				}
			}

			// Add post-create command
			if feat.PostCmd != "" && !contains(postCmds, feat.PostCmd) {
				postCmds = append(postCmds, feat.PostCmd)
			}
		}
	}

	// Add common development features
	config.Features["ghcr.io/devcontainers/features/common-utils:2"] = map[string]interface{}{
		"installZsh":                 true,
		"configureZshAsDefaultShell": true,
	}

	config.Features["ghcr.io/devcontainers/features/git:1"] = map[string]interface{}{}

	// Docker-in-Docker if Dockerfile exists
	if info.HasDockerfile {
		config.Features["ghcr.io/devcontainers/features/docker-in-docker:2"] = map[string]interface{}{}
	}

	// Build post-create command
	if len(postCmds) > 0 {
		config.PostCreateCommand = strings.Join(postCmds, " && ")
	}

	// Add VS Code extensions
	if len(extensions) > 0 {
		config.Customizations = map[string]interface{}{
			"vscode": map[string]interface{}{
				"extensions": extensions,
			},
		}
	}

	// Add common ports based on detected frameworks
	config.ForwardPorts = detectPorts(info)

	return config, nil
}

// ToJSON converts the config to formatted JSON
func (c *MultiLangConfig) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Summary returns a human-readable summary of the config
func (c *MultiLangConfig) Summary() string {
	var sb strings.Builder

	sb.WriteString("📦 Multi-Language Configuration:\n")
	sb.WriteString("─────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("  Base Image: %s\n", c.Image))
	sb.WriteString(fmt.Sprintf("  Languages/Features: %d\n", len(c.Features)))

	for feat := range c.Features {
		// Extract feature name from URL
		parts := strings.Split(feat, "/")
		name := parts[len(parts)-1]
		if idx := strings.Index(name, ":"); idx > 0 {
			name = name[:idx]
		}
		sb.WriteString(fmt.Sprintf("    • %s\n", name))
	}

	if c.Customizations != nil {
		if vscode, ok := c.Customizations["vscode"].(map[string]interface{}); ok {
			if exts, ok := vscode["extensions"].([]string); ok {
				sb.WriteString(fmt.Sprintf("  VS Code Extensions: %d\n", len(exts)))
			}
		}
	}

	if len(c.RunArgs) > 0 {
		if contains(c.RunArgs, "--gpus") {
			sb.WriteString("  🎮 GPU Support: Enabled\n")
		}
	}

	return sb.String()
}

// Helper functions
func normalizeLangName(name string) string {
	// Normalize language names to match our feature map
	switch {
	case strings.Contains(name, "JavaScript") || name == "JavaScript/TypeScript":
		return "JavaScript"
	case strings.Contains(name, "TypeScript"):
		return "TypeScript"
	case strings.Contains(name, ".NET") || name == ".NET/C#":
		return ".NET/C#"
	case strings.Contains(name, "C++") || name == "C/C++":
		return "C++"
	default:
		return name
	}
}

func detectPorts(info *ProjectInfo) []int {
	var ports []int
	portSet := make(map[int]bool)

	// Framework-specific ports
	frameworkPorts := map[string][]int{
		"Next.js": {3000},
		"React":   {3000},
		"Vue":     {5173, 8080},
		"Angular": {4200},
		"Django":  {8000},
		"FastAPI": {8000},
		"Flask":   {5000},
		"Express": {3000},
		"NestJS":  {3000},
		"Gin":     {8080},
		"Echo":    {8080},
		"Fiber":   {3000},
		"Spring":  {8080},
	}

	for _, fw := range info.Frameworks {
		if fwPorts, ok := frameworkPorts[fw]; ok {
			for _, p := range fwPorts {
				if !portSet[p] {
					ports = append(ports, p)
					portSet[p] = true
				}
			}
		}
	}

	return ports
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
