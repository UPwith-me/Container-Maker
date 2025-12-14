package runtime

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// BackendConfig stores user preferences and custom backends
type BackendConfig struct {
	Preferred  string          `json:"preferred,omitempty"`
	LastUsed   string          `json:"lastUsed,omitempty"`
	DetectedAt string          `json:"detectedAt,omitempty"`
	Custom     []CustomBackend `json:"custom,omitempty"`
}

// CustomBackend represents a user-defined backend
type CustomBackend struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // docker, podman, nerdctl
}

// DetectionResult holds the result of backend detection
type DetectionResult struct {
	Backends  []BackendInfo
	Preferred string
	Active    *BackendInfo
}

// Detector handles backend detection and management
type Detector struct {
	configPath string
	config     *BackendConfig
	mu         sync.RWMutex
}

// NewDetector creates a new backend detector
func NewDetector() *Detector {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".cm", "config.json")

	d := &Detector{
		configPath: configPath,
		config:     &BackendConfig{},
	}
	d.loadConfig()
	return d
}

// loadConfig loads the backend configuration from disk
func (d *Detector) loadConfig() {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := os.ReadFile(d.configPath)
	if err != nil {
		return // No config file yet
	}

	// Parse outer config to get backend section
	var outerConfig struct {
		Backend BackendConfig `json:"backend"`
	}
	if err := json.Unmarshal(data, &outerConfig); err != nil {
		return
	}
	d.config = &outerConfig.Backend
}

// saveConfig saves the backend configuration to disk
func (d *Detector) saveConfig() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(d.configPath), 0755); err != nil {
		return err
	}

	// Read existing config to preserve other fields
	var outerConfig map[string]interface{}
	if data, err := os.ReadFile(d.configPath); err == nil {
		_ = json.Unmarshal(data, &outerConfig)
	}
	if outerConfig == nil {
		outerConfig = make(map[string]interface{})
	}

	outerConfig["backend"] = d.config

	data, err := json.MarshalIndent(outerConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(d.configPath, data, 0644)
}

// Detect discovers all available container runtimes
func (d *Detector) Detect() *DetectionResult {
	result := &DetectionResult{
		Backends: []BackendInfo{},
	}

	// Check environment variable first
	if envBackend := os.Getenv("CM_BACKEND"); envBackend != "" {
		result.Preferred = envBackend
	} else if d.config.Preferred != "" {
		result.Preferred = d.config.Preferred
	}

	// Detect built-in backends in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex

	builtIns := []struct {
		name     string
		typ      string
		binaries []string
	}{
		{"docker", "docker", []string{"docker"}},
		{"podman", "podman", []string{"podman"}},
		{"nerdctl", "nerdctl", []string{"nerdctl"}},
	}

	for _, b := range builtIns {
		wg.Add(1)
		go func(name, typ string, binaries []string) {
			defer wg.Done()
			info := d.checkBackend(name, typ, binaries)
			if info != nil {
				mu.Lock()
				result.Backends = append(result.Backends, *info)
				mu.Unlock()
			}
		}(b.name, b.typ, b.binaries)
	}

	// Check custom backends
	for _, custom := range d.config.Custom {
		wg.Add(1)
		go func(c CustomBackend) {
			defer wg.Done()
			info := d.checkCustomBackend(c)
			if info != nil {
				mu.Lock()
				result.Backends = append(result.Backends, *info)
				mu.Unlock()
			}
		}(custom)
	}

	wg.Wait()

	// Update detection time
	d.config.DetectedAt = time.Now().Format(time.RFC3339)
	_ = d.saveConfig()

	// Set active backend
	for i := range result.Backends {
		if result.Backends[i].Name == result.Preferred && result.Backends[i].Running {
			result.Backends[i].IsActive = true
			result.Active = &result.Backends[i]
			break
		}
	}

	// If no preferred or preferred not running, pick first running
	if result.Active == nil {
		for i := range result.Backends {
			if result.Backends[i].Running {
				result.Backends[i].IsActive = true
				result.Active = &result.Backends[i]
				break
			}
		}
	}

	return result
}

// checkBackend checks if a built-in backend is available
func (d *Detector) checkBackend(name, typ string, binaries []string) *BackendInfo {
	for _, bin := range binaries {
		path, err := exec.LookPath(bin)
		if err != nil {
			continue
		}

		info := &BackendInfo{
			Name:      name,
			Type:      typ,
			Path:      path,
			Available: true,
		}

		// Get version
		if version, err := d.getVersion(path, typ); err == nil {
			info.Version = version
		}

		// Check if running
		info.Running = d.isRunning(path, typ)

		return info
	}
	return nil
}

// checkCustomBackend checks if a custom backend is available
func (d *Detector) checkCustomBackend(c CustomBackend) *BackendInfo {
	if _, err := os.Stat(c.Path); os.IsNotExist(err) {
		return &BackendInfo{
			Name:      c.Name,
			Type:      c.Type,
			Path:      c.Path,
			Available: false,
			IsCustom:  true,
		}
	}

	info := &BackendInfo{
		Name:      c.Name,
		Type:      c.Type,
		Path:      c.Path,
		Available: true,
		IsCustom:  true,
	}

	if version, err := d.getVersion(c.Path, c.Type); err == nil {
		info.Version = version
	}

	info.Running = d.isRunning(c.Path, c.Type)
	return info
}

// getVersion gets the version of a container runtime
func (d *Detector) getVersion(path, typ string) (string, error) {
	var cmd *exec.Cmd
	switch typ {
	case "docker", "podman", "nerdctl":
		cmd = exec.Command(path, "version", "--format", "{{.Client.Version}}")
	default:
		cmd = exec.Command(path, "--version")
	}

	output, err := cmd.Output()
	if err != nil {
		// Fallback to simple --version
		cmd = exec.Command(path, "--version")
		output, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}

	version := strings.TrimSpace(string(output))
	// Extract version number if needed
	if strings.Contains(version, "version") {
		parts := strings.Fields(version)
		for i, p := range parts {
			if p == "version" && i+1 < len(parts) {
				version = strings.TrimSuffix(parts[i+1], ",")
				break
			}
		}
	}
	return version, nil
}

// isRunning checks if the container runtime daemon is running
func (d *Detector) isRunning(path, typ string) bool {
	var cmd *exec.Cmd
	switch typ {
	case "docker":
		cmd = exec.Command(path, "info")
	case "podman":
		// Podman is daemonless, just check if it works
		cmd = exec.Command(path, "info")
	case "nerdctl":
		cmd = exec.Command(path, "info")
	default:
		cmd = exec.Command(path, "info")
	}

	err := cmd.Run()
	return err == nil
}

// GetPreferred returns the preferred backend name
func (d *Detector) GetPreferred() string {
	if env := os.Getenv("CM_BACKEND"); env != "" {
		return env
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config.Preferred
}

// SetPreferred sets the preferred backend
func (d *Detector) SetPreferred(name string) error {
	d.mu.Lock()
	d.config.Preferred = name
	d.config.LastUsed = name
	d.mu.Unlock()
	return d.saveConfig()
}

// AddCustomBackend adds a custom backend
func (d *Detector) AddCustomBackend(name, path, typ string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if already exists
	for i := range d.config.Custom {
		if d.config.Custom[i].Name == name {
			// Update existing
			d.config.Custom[i].Path = path
			d.config.Custom[i].Type = typ
			return d.saveConfig()
		}
	}

	d.config.Custom = append(d.config.Custom, CustomBackend{
		Name: name,
		Path: path,
		Type: typ,
	})
	return d.saveConfig()
}

// RemoveCustomBackend removes a custom backend
func (d *Detector) RemoveCustomBackend(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i, c := range d.config.Custom {
		if c.Name == name {
			d.config.Custom = append(d.config.Custom[:i], d.config.Custom[i+1:]...)
			return d.saveConfig()
		}
	}
	return nil
}

// GetCustomBackends returns all custom backends
func (d *Detector) GetCustomBackends() []CustomBackend {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.config.Custom
}
