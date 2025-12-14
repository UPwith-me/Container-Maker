package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfig_Simple(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "devcontainer.json")

	configContent := `{
		"image": "ubuntu:22.04",
		"forwardPorts": [8080, 3000],
		"containerEnv": {
			"NODE_ENV": "development"
		}
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if cfg.Image != "ubuntu:22.04" {
		t.Errorf("Expected image 'ubuntu:22.04', got '%s'", cfg.Image)
	}

	if len(cfg.ForwardPorts) != 2 {
		t.Errorf("Expected 2 forward ports, got %d", len(cfg.ForwardPorts))
	}

	if cfg.ContainerEnv["NODE_ENV"] != "development" {
		t.Errorf("Expected NODE_ENV='development', got '%s'", cfg.ContainerEnv["NODE_ENV"])
	}
}

func TestParseConfig_WithComments(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "devcontainer.json")

	// JSONC with comments and trailing commas
	configContent := `{
		// This is a comment
		"image": "node:20",
		"features": {
			"ghcr.io/devcontainers/features/git:1": {},
		}, // trailing comma
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig failed with JSONC: %v", err)
	}

	if cfg.Image != "node:20" {
		t.Errorf("Expected image 'node:20', got '%s'", cfg.Image)
	}
}

func TestParseConfig_WithBuild(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "devcontainer.json")

	configContent := `{
		"build": {
			"dockerfile": "Dockerfile",
			"context": ".",
			"args": {
				"VARIANT": "3.11"
			}
		}
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig failed: %v", err)
	}

	if cfg.Build == nil {
		t.Fatal("Expected Build config to be present")
	}

	if cfg.Build.Dockerfile != "Dockerfile" {
		t.Errorf("Expected dockerfile 'Dockerfile', got '%s'", cfg.Build.Dockerfile)
	}

	if cfg.Build.Args["VARIANT"] != "3.11" {
		t.Errorf("Expected VARIANT='3.11', got '%s'", cfg.Build.Args["VARIANT"])
	}
}

func TestParseConfig_NotFound(t *testing.T) {
	_, err := ParseConfig("/nonexistent/path/devcontainer.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}
