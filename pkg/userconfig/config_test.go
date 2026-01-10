package userconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSave(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".cm", "config.json")

	// Override config path for testing
	originalHome := os.Getenv("USERPROFILE")
	os.Setenv("USERPROFILE", tmpDir)
	defer os.Setenv("USERPROFILE", originalHome)

	// Test Save
	cfg := &UserConfig{
		SkipWelcome:    true,
		DefaultBackend: "docker",
		AI: AIConfig{
			Enabled: true,
			APIKey:  "test-key",
		},
		Team: TeamConfig{
			OrgName: "TestOrg",
			Variables: map[string]string{
				"KEY1": "value1",
			},
		},
	}

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(configPath), 0755)

	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Test Load
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.SkipWelcome != true {
		t.Errorf("SkipWelcome = %v, want true", loaded.SkipWelcome)
	}

	if loaded.DefaultBackend != "docker" {
		t.Errorf("DefaultBackend = %q, want 'docker'", loaded.DefaultBackend)
	}

	if loaded.AI.Enabled != true {
		t.Errorf("AI.Enabled = %v, want true", loaded.AI.Enabled)
	}

	if loaded.Team.OrgName != "TestOrg" {
		t.Errorf("Team.OrgName = %q, want 'TestOrg'", loaded.Team.OrgName)
	}

	if loaded.Team.Variables["KEY1"] != "value1" {
		t.Errorf("Team.Variables[KEY1] = %q, want 'value1'", loaded.Team.Variables["KEY1"])
	}
}

func TestGet(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	os.Setenv("USERPROFILE", tmpDir)
	defer os.Setenv("USERPROFILE", "")

	// Save test config
	cfg := &UserConfig{
		SkipWelcome:    true,
		DefaultBackend: "podman",
		AI: AIConfig{
			Enabled: true,
			APIKey:  "secret",
		},
	}
	os.MkdirAll(filepath.Join(tmpDir, ".cm"), 0755)
	_ = Save(cfg)

	tests := []struct {
		key      string
		expected string
	}{
		{"skip_welcome", "true"},
		{"default_backend", "podman"},
		{"ai.enabled", "true"},
		{"ai.api_key", "***hidden***"}, // API key should be hidden
		{"nonexistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result, _ := Get(tt.key)
			if result != tt.expected {
				t.Errorf("Get(%q) = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestSet(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("USERPROFILE", tmpDir)
	defer os.Setenv("USERPROFILE", "")
	os.MkdirAll(filepath.Join(tmpDir, ".cm"), 0755)

	// Test setting values
	if err := Set("skip_welcome", "true"); err != nil {
		t.Errorf("Set() error = %v", err)
	}

	if err := Set("default_backend", "docker"); err != nil {
		t.Errorf("Set() error = %v", err)
	}

	// Verify
	cfg, _ := Load()
	if !cfg.SkipWelcome {
		t.Error("SkipWelcome not set")
	}
	if cfg.DefaultBackend != "docker" {
		t.Errorf("DefaultBackend = %q, want 'docker'", cfg.DefaultBackend)
	}
}

func TestTeamRepository(t *testing.T) {
	repo := TeamRepository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Branch:   "main",
		Priority: 100,
		AuthType: "token",
	}

	if repo.Name != "test-repo" {
		t.Errorf("Name = %q, want 'test-repo'", repo.Name)
	}

	if repo.Priority != 100 {
		t.Errorf("Priority = %d, want 100", repo.Priority)
	}

	if repo.URL != "https://github.com/test/repo" {
		t.Errorf("URL mismatch")
	}
	if repo.Branch != "main" {
		t.Errorf("Branch mismatch")
	}
	if repo.AuthType != "token" {
		t.Errorf("AuthType mismatch")
	}
}
