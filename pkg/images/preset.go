package images

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
)

// PresetImage represents a preset development image
type PresetImage struct {
	Name        string `json:"name"`
	Image       string `json:"image"`
	Description string `json:"description"`
	Size        string `json:"size"`
	Tools       string `json:"tools"`
	Downloaded  bool   `json:"downloaded"`
}

// ImagesConfig stores user's image preferences
type ImagesConfig struct {
	Presets map[string]*PresetImage `json:"presets"`
	Custom  map[string]*PresetImage `json:"custom"`
	Default string                  `json:"default"`
}

// DefaultPresets returns the built-in preset images
func DefaultPresets() map[string]*PresetImage {
	return map[string]*PresetImage{
		"go": {
			Name:        "go",
			Image:       "golang:1.21-alpine",
			Description: "Go development",
			Size:        "~300MB",
			Tools:       "go, git, make",
		},
		"python": {
			Name:        "python",
			Image:       "python:3.11-slim",
			Description: "Python development",
			Size:        "~150MB",
			Tools:       "python, pip",
		},
		"node": {
			Name:        "node",
			Image:       "node:20-alpine",
			Description: "Node.js development",
			Size:        "~170MB",
			Tools:       "node, npm, npx",
		},
		"rust": {
			Name:        "rust",
			Image:       "rust:alpine",
			Description: "Rust development",
			Size:        "~800MB",
			Tools:       "rustc, cargo",
		},
		"cpp": {
			Name:        "cpp",
			Image:       "gcc:latest",
			Description: "C/C++ development",
			Size:        "~1.2GB",
			Tools:       "gcc, g++, make",
		},
		"base": {
			Name:        "base",
			Image:       "debian:bookworm-slim",
			Description: "Minimal base",
			Size:        "~80MB",
			Tools:       "apt",
		},
		"devcontainer": {
			Name:        "devcontainer",
			Image:       "mcr.microsoft.com/devcontainers/base:debian",
			Description: "Full DevContainer",
			Size:        "~500MB",
			Tools:       "git, curl, zsh, ssh",
		},
	}
}

// GetConfigPath returns the path to the images config file
func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cm", "images.json")
}

// LoadConfig loads the images configuration
func LoadConfig() (*ImagesConfig, error) {
	configPath := GetConfigPath()

	// If config doesn't exist, create with defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := &ImagesConfig{
			Presets: DefaultPresets(),
			Custom:  make(map[string]*PresetImage),
			Default: "",
		}
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config ImagesConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Merge with defaults (in case new presets were added)
	defaults := DefaultPresets()
	for name, preset := range defaults {
		if _, exists := config.Presets[name]; !exists {
			config.Presets[name] = preset
		}
	}

	return &config, nil
}

// SaveConfig saves the images configuration
func SaveConfig(config *ImagesConfig) error {
	configPath := GetConfigPath()

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// CheckImageExists checks if a Docker image exists locally
func CheckImageExists(imageName string) bool {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false
	}
	defer cli.Close()

	_, _, err = cli.ImageInspectWithRaw(nil, imageName)
	return err == nil
}

// UpdateDownloadedStatus updates the downloaded status of all images
func UpdateDownloadedStatus(config *ImagesConfig) {
	for _, preset := range config.Presets {
		preset.Downloaded = CheckImageExists(preset.Image)
	}
	for _, custom := range config.Custom {
		custom.Downloaded = CheckImageExists(custom.Image)
	}
}

// ListImages returns a formatted list of all images
func ListImages(config *ImagesConfig) string {
	var sb strings.Builder

	sb.WriteString("📦 Container-Make Images\n\n")
	sb.WriteString("  NAME           IMAGE                                    STATUS\n")
	sb.WriteString("  ──────────────────────────────────────────────────────────────\n")

	// Presets
	for name, preset := range config.Presets {
		marker := "  "
		if name == config.Default {
			marker = "* "
		}

		status := "⬇️  Not downloaded"
		if preset.Downloaded {
			status = "✅ Ready"
		}

		imageName := preset.Image
		if len(imageName) > 35 {
			imageName = imageName[:32] + "..."
		}

		sb.WriteString(fmt.Sprintf("  %s%-12s %-38s %s\n", marker, name, imageName, status))
	}

	// Custom images
	if len(config.Custom) > 0 {
		sb.WriteString("\n  Custom:\n")
		for name, custom := range config.Custom {
			status := "⬇️  Not downloaded"
			if custom.Downloaded {
				status = "✅ Ready"
			}
			sb.WriteString(fmt.Sprintf("    %-12s %-38s %s\n", name, custom.Image, status))
		}
	}

	sb.WriteString("\n  * = default for new projects\n")
	sb.WriteString("\nCommands:\n")
	sb.WriteString("  cm images use <name>    Switch current project's image\n")
	sb.WriteString("  cm images pull <name>   Download an image\n")
	sb.WriteString("  cm images add <image>   Add custom image\n")
	sb.WriteString("  cm images setup         Run setup wizard\n")

	return sb.String()
}

// GetImage returns an image by name (preset or custom)
func GetImage(config *ImagesConfig, name string) (*PresetImage, bool) {
	if preset, ok := config.Presets[name]; ok {
		return preset, true
	}
	if custom, ok := config.Custom[name]; ok {
		return custom, true
	}
	return nil, false
}

// AddCustomImage adds a custom image to the config
func AddCustomImage(config *ImagesConfig, name, image string) error {
	if _, exists := config.Presets[name]; exists {
		return fmt.Errorf("'%s' is a preset name, choose a different name", name)
	}

	config.Custom[name] = &PresetImage{
		Name:        name,
		Image:       image,
		Description: "Custom image",
		Downloaded:  CheckImageExists(image),
	}

	return SaveConfig(config)
}

// RemoveCustomImage removes a custom image from the config
func RemoveCustomImage(config *ImagesConfig, name string) error {
	if _, exists := config.Presets[name]; exists {
		return fmt.Errorf("'%s' is a preset and cannot be removed", name)
	}

	if _, exists := config.Custom[name]; !exists {
		return fmt.Errorf("custom image '%s' not found", name)
	}

	delete(config.Custom, name)
	return SaveConfig(config)
}
