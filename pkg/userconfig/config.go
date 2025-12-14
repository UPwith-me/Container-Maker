package userconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// UserConfig holds persistent user preferences
type UserConfig struct {
	SkipWelcome    bool              `json:"skip_welcome"`
	DefaultBackend string            `json:"default_backend,omitempty"`
	AI             AIConfig          `json:"ai,omitempty"`
	RemoteHosts    map[string]string `json:"remote_hosts,omitempty"`
	ActiveRemote   string            `json:"active_remote,omitempty"`
	Team           TeamConfig        `json:"team,omitempty"`
	Analytics      AnalyticsConfig   `json:"analytics,omitempty"`

	// Cloud Control Plane
	CloudAPIKey string `json:"cloud_api_key,omitempty"`
	CloudToken  string `json:"cloud_token,omitempty"`
	CloudAPIURL string `json:"cloud_api_url,omitempty"`
}

// AIConfig holds AI-related settings
type AIConfig struct {
	Enabled bool   `json:"enabled"`
	APIKey  string `json:"api_key,omitempty"`
	APIBase string `json:"api_base,omitempty"`
}

// TeamConfig holds team/org settings
type TeamConfig struct {
	OrgName      string `json:"org_name,omitempty"`
	TemplatesURL string `json:"templates_url,omitempty"`
}

// AnalyticsConfig holds anonymous usage statistics settings
type AnalyticsConfig struct {
	Enabled   bool   `json:"enabled"`
	SessionID string `json:"session_id,omitempty"`
}

// configPath returns the path to the user config file
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cm", "config.json"), nil
}

// Load loads the user config from disk
func Load() (*UserConfig, error) {
	path, err := configPath()
	if err != nil {
		return &UserConfig{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &UserConfig{}, nil
		}
		return nil, err
	}

	var cfg UserConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &UserConfig{}, nil
	}

	return &cfg, nil
}

// Save saves the user config to disk
func Save(cfg *UserConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Get gets a config value
func Get(key string) (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}

	switch key {
	case "skip_welcome":
		if cfg.SkipWelcome {
			return "true", nil
		}
		return "false", nil
	case "default_backend":
		return cfg.DefaultBackend, nil
	case "ai.enabled":
		if cfg.AI.Enabled {
			return "true", nil
		}
		return "false", nil
	case "ai.api_key":
		if cfg.AI.APIKey != "" {
			return "***hidden***", nil
		}
		return "", nil
	case "ai.api_base":
		return cfg.AI.APIBase, nil
	default:
		return "", nil
	}
}

// Set sets a config value
func Set(key, value string) error {
	cfg, err := Load()
	if err != nil {
		cfg = &UserConfig{}
	}

	switch key {
	case "skip_welcome":
		cfg.SkipWelcome = value == "true" || value == "1"
	case "default_backend":
		cfg.DefaultBackend = value
	case "ai.enabled":
		cfg.AI.Enabled = value == "true" || value == "1"
	case "ai.api_key":
		cfg.AI.APIKey = value
	case "ai.api_base":
		cfg.AI.APIBase = value
	}

	return Save(cfg)
}
