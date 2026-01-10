package features

import (
	"testing"
)

func TestParseFeatureRef(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		options     interface{}
		wantID      string
		wantVersion string
		wantErr     bool
	}{
		{
			name:        "simple feature reference",
			source:      "ghcr.io/devcontainers/features/go:1",
			options:     nil,
			wantID:      "go",
			wantVersion: "1",
			wantErr:     false,
		},
		{
			name:        "feature with version tag",
			source:      "ghcr.io/devcontainers/features/docker-in-docker:2",
			options:     true,
			wantID:      "docker-in-docker",
			wantVersion: "2",
			wantErr:     false,
		},
		{
			name:   "feature with options",
			source: "ghcr.io/devcontainers/features/node:1",
			options: map[string]interface{}{
				"version": "18",
			},
			wantID:      "node",
			wantVersion: "1",
			wantErr:     false,
		},
		{
			name:        "feature without version (latest)",
			source:      "ghcr.io/owner/features/custom",
			options:     nil,
			wantID:      "custom",
			wantVersion: "latest",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := ParseFeatureRef(tt.source, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("ParseFeatureRef() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseFeatureRef() error: %v", err)
			}

			if ref.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", ref.ID, tt.wantID)
			}
			if ref.Version != tt.wantVersion {
				t.Errorf("Version = %q, want %q", ref.Version, tt.wantVersion)
			}
			if ref.Source != tt.source {
				t.Errorf("Source = %q, want %q", ref.Source, tt.source)
			}
		})
	}
}

func TestParseFeaturesFromConfig(t *testing.T) {
	features := map[string]interface{}{
		"ghcr.io/devcontainers/features/go:1": map[string]interface{}{
			"version": "1.21",
		},
		"ghcr.io/devcontainers/features/node:18": true,
	}

	refs, err := ParseFeaturesFromConfig(features)
	if err != nil {
		t.Fatalf("ParseFeaturesFromConfig() error: %v", err)
	}

	if len(refs) != 2 {
		t.Errorf("got %d refs, want 2", len(refs))
	}
}

func TestGenerateFeatureEnv(t *testing.T) {
	feature := &Feature{
		ID:      "test",
		Version: "1.0",
		Options: map[string]interface{}{
			"version":     "18",
			"installYarn": true,
		},
	}

	env := GenerateFeatureEnv(feature)

	if len(env) != 2 {
		t.Errorf("got %d env vars, want 2", len(env))
	}

	// Check that keys are uppercase
	hasVersion := false
	hasInstallYarn := false
	for _, e := range env {
		if e == "VERSION=18" {
			hasVersion = true
		}
		if e == "INSTALLYARN=true" {
			hasInstallYarn = true
		}
	}

	if !hasVersion {
		t.Error("missing VERSION env var")
	}
	if !hasInstallYarn {
		t.Error("missing INSTALLYARN env var")
	}
}

func TestFeatureInstaller_GenerateInstallScript(t *testing.T) {
	fi := NewFeatureInstaller("/tmp/features")

	fi.AddFeature(&Feature{
		ID:        "go",
		Version:   "1",
		InstallSh: "#!/bin/bash\necho 'Installing Go'",
		Options: map[string]interface{}{
			"version": "1.21",
		},
	})

	fi.AddFeature(&Feature{
		ID:        "node",
		Version:   "18",
		InstallSh: "#!/bin/bash\necho 'Installing Node'",
	})

	script := fi.GenerateInstallScript()

	// Check script contains expected content
	if len(script) == 0 {
		t.Error("generated script is empty")
	}

	expectedContents := []string{
		"#!/bin/bash",
		"set -e",
		"Installing feature: go",
		"Installing feature: node",
		"export VERSION=1.21",
		"./install.sh",
	}

	for _, expected := range expectedContents {
		if !containsString(script, expected) {
			t.Errorf("script missing expected content: %q", expected)
		}
	}
}

func TestFeatureInstaller_SortByDependencies(t *testing.T) {
	fi := NewFeatureInstaller("/tmp/features")

	fi.AddFeature(&Feature{ID: "zsh"})
	fi.AddFeature(&Feature{ID: "git"})
	fi.AddFeature(&Feature{ID: "docker"})

	fi.SortByDependencies()

	// No dependencies means order is preserved (insertion order)
	expected := []string{"zsh", "git", "docker"}
	for i, f := range fi.Features {
		if f.ID != expected[i] {
			t.Errorf("Features[%d].ID = %q, want %q", i, f.ID, expected[i])
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || containsString(s[1:], substr)))
}
