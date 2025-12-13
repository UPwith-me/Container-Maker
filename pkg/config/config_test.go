package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "devcontainer.json")

	tests := []struct {
		name      string
		content   string
		wantImage string
		wantPorts int
		wantErr   bool
	}{
		{
			name: "simple image config",
			content: `{
				"image": "mcr.microsoft.com/devcontainers/base:ubuntu"
			}`,
			wantImage: "mcr.microsoft.com/devcontainers/base:ubuntu",
			wantErr:   false,
		},
		{
			name: "config with ports",
			content: `{
				"image": "nginx:latest",
				"forwardPorts": [8080, 443, "3000/tcp"]
			}`,
			wantImage: "nginx:latest",
			wantPorts: 3,
			wantErr:   false,
		},
		{
			name: "config with comments (JSONC)",
			content: `{
				// This is a comment
				"image": "alpine:latest",
				"containerEnv": {
					"KEY": "value"  // trailing comment
				}
			}`,
			wantImage: "alpine:latest",
			wantErr:   false,
		},
		{
			name: "config with trailing comma",
			content: `{
				"image": "ubuntu:22.04",
				"mounts": [
					"source=/host,target=/container,type=bind",
				]
			}`,
			wantImage: "ubuntu:22.04",
			wantErr:   false,
		},
		{
			name: "docker compose config",
			content: `{
				"dockerComposeFile": "docker-compose.yml",
				"service": "app",
				"runServices": ["app", "db"]
			}`,
			wantImage: "",
			wantErr:   false,
		},
		{
			name: "build config",
			content: `{
				"build": {
					"dockerfile": "Dockerfile",
					"context": ".",
					"args": {
						"VARIANT": "18"
					}
				}
			}`,
			wantImage: "",
			wantErr:   false,
		},
		{
			name:    "invalid json",
			content: `{ invalid json }`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write config file
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			// Parse config
			cfg, err := ParseConfig(configPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseConfig() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseConfig() unexpected error: %v", err)
			}

			if cfg.Image != tt.wantImage {
				t.Errorf("Image = %q, want %q", cfg.Image, tt.wantImage)
			}

			if tt.wantPorts > 0 && len(cfg.ForwardPorts) != tt.wantPorts {
				t.Errorf("ForwardPorts count = %d, want %d", len(cfg.ForwardPorts), tt.wantPorts)
			}
		})
	}
}

func TestParseConfig_NonExistentFile(t *testing.T) {
	_, err := ParseConfig("/non/existent/file.json")
	if err == nil {
		t.Error("ParseConfig() expected error for non-existent file, got nil")
	}
}

func TestParseConfig_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "devcontainer.json")

	content := `{
		"image": "ubuntu:latest",
		"runArgs": ["--cap-add=SYS_PTRACE", "--security-opt", "seccomp=unconfined"],
		"mounts": ["source=/host,target=/container,type=bind"],
		"containerEnv": {"ENV1": "value1"},
		"remoteEnv": {"ENV2": "value2"},
		"postCreateCommand": "npm install",
		"postStartCommand": ["echo", "started"],
		"postAttachCommand": "zsh",
		"features": {
			"ghcr.io/devcontainers/features/go:1": {"version": "1.21"}
		},
		"forwardPorts": [8080],
		"user": "vscode",
		"workspaceMount": "source=/local,target=/remote,type=bind",
		"workspaceFolder": "/workspaces/project"
	}`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := ParseConfig(configPath)
	if err != nil {
		t.Fatalf("ParseConfig() error: %v", err)
	}

	// Verify all fields
	if cfg.Image != "ubuntu:latest" {
		t.Errorf("Image = %q, want %q", cfg.Image, "ubuntu:latest")
	}
	if len(cfg.RunArgs) != 3 {
		t.Errorf("RunArgs count = %d, want 3", len(cfg.RunArgs))
	}
	if len(cfg.Mounts) != 1 {
		t.Errorf("Mounts count = %d, want 1", len(cfg.Mounts))
	}
	if cfg.ContainerEnv["ENV1"] != "value1" {
		t.Errorf("ContainerEnv[ENV1] = %q, want %q", cfg.ContainerEnv["ENV1"], "value1")
	}
	if cfg.RemoteEnv["ENV2"] != "value2" {
		t.Errorf("RemoteEnv[ENV2] = %q, want %q", cfg.RemoteEnv["ENV2"], "value2")
	}
	if cfg.User != "vscode" {
		t.Errorf("User = %q, want %q", cfg.User, "vscode")
	}
	if cfg.WorkspaceFolder != "/workspaces/project" {
		t.Errorf("WorkspaceFolder = %q, want %q", cfg.WorkspaceFolder, "/workspaces/project")
	}
	if len(cfg.Features) != 1 {
		t.Errorf("Features count = %d, want 1", len(cfg.Features))
	}
}
