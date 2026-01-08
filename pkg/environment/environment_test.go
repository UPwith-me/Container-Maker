package environment

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	// Should have env- prefix
	if len(id1) < 5 || id1[:4] != "env-" {
		t.Errorf("ID should start with 'env-', got %s", id1)
	}

	// Should be unique
	if id1 == id2 {
		t.Error("IDs should be unique")
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"frontend", true},
		{"my-project", true},
		{"My_Project.v2", true},
		{"", false},                        // Empty
		{"123invalid", false},              // Starts with number
		{"has space", false},               // Space
		{"has@special", false},             // Special char
		{string(make([]byte, 100)), false}, // Too long
	}

	for _, tt := range tests {
		err := validateName(tt.name)
		if tt.valid && err != nil {
			t.Errorf("validateName(%q) should be valid, got error: %v", tt.name, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("validateName(%q) should be invalid", tt.name)
		}
	}
}

func TestEnvironmentStatus(t *testing.T) {
	statuses := []EnvironmentStatus{
		StatusCreating,
		StatusRunning,
		StatusStopped,
		StatusPaused,
		StatusError,
		StatusOrphaned,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("Status should not be empty")
		}
	}
}

func TestEnvironmentStruct(t *testing.T) {
	env := &Environment{
		ID:         "env-test123",
		Name:       "test-env",
		ProjectDir: "/tmp/test",
		Status:     StatusRunning,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Backend:    "docker",
	}

	if env.ID == "" {
		t.Error("Environment ID should not be empty")
	}
	if env.Name == "" {
		t.Error("Environment Name should not be empty")
	}
	if env.Status != StatusRunning {
		t.Errorf("Expected status running, got %s", env.Status)
	}
}

func TestFileStateStore(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "cm-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home directory for test
	origHome := os.Getenv("USERPROFILE")
	if origHome == "" {
		origHome = os.Getenv("HOME")
	}
	os.Setenv("USERPROFILE", tmpDir)
	os.Setenv("HOME", tmpDir)
	defer func() {
		os.Setenv("USERPROFILE", origHome)
		os.Setenv("HOME", origHome)
	}()

	store, err := NewFileStateStore()
	if err != nil {
		t.Fatalf("Failed to create state store: %v", err)
	}

	// Test Save and Load
	env := &Environment{
		ID:         "env-test123",
		Name:       "test-env",
		ProjectDir: tmpDir,
		Status:     StatusRunning,
		Backend:    "docker",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := store.Save(env); err != nil {
		t.Fatalf("Failed to save environment: %v", err)
	}

	loaded, err := store.Load(env.ID)
	if err != nil {
		t.Fatalf("Failed to load environment: %v", err)
	}

	if loaded.Name != env.Name {
		t.Errorf("Loaded name %s != saved name %s", loaded.Name, env.Name)
	}

	// Test List
	envs, err := store.List()
	if err != nil {
		t.Fatalf("Failed to list environments: %v", err)
	}
	if len(envs) != 1 {
		t.Errorf("Expected 1 environment, got %d", len(envs))
	}

	// Test SetActive and GetActive
	if err := store.SetActive(env.ID); err != nil {
		t.Fatalf("Failed to set active: %v", err)
	}
	activeID, err := store.GetActive()
	if err != nil {
		t.Fatalf("Failed to get active: %v", err)
	}
	if activeID != env.ID {
		t.Errorf("Active ID %s != expected %s", activeID, env.ID)
	}

	// Test Delete
	if err := store.Delete(env.ID); err != nil {
		t.Fatalf("Failed to delete environment: %v", err)
	}
	_, err = store.Load(env.ID)
	if err == nil {
		t.Error("Should not find deleted environment")
	}
}

func TestEnvironmentCreateOptions(t *testing.T) {
	opts := EnvironmentCreateOptions{
		Name:     "test",
		Template: "python",
		GPUs:     []int{0, 1},
		Memory:   "8g",
		CPU:      4.0,
	}

	if opts.Name != "test" {
		t.Error("Name not set correctly")
	}
	if len(opts.GPUs) != 2 {
		t.Error("GPUs not set correctly")
	}
}

func TestNetworkInfo(t *testing.T) {
	info := &NetworkInfo{
		ID:     "net-123",
		Name:   "cm-test-network",
		Driver: "bridge",
		Labels: map[string]string{
			LabelManagedBy: "container-maker",
		},
	}

	if !IsManagedNetwork(info) {
		t.Error("Should be recognized as managed network")
	}

	info.Labels[LabelManagedBy] = "other"
	if IsManagedNetwork(info) {
		t.Error("Should not be recognized as managed network")
	}
}

func TestStateFilePath(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "cm-test-*")
	defer os.RemoveAll(tmpDir)

	os.Setenv("USERPROFILE", tmpDir)
	os.Setenv("HOME", tmpDir)

	store, _ := NewFileStateStore()

	statePath := store.getStatePath()
	if !filepath.IsAbs(statePath) {
		t.Errorf("State path should be absolute: %s", statePath)
	}
}
