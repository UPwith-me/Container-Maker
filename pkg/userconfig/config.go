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

	// System state
	LastUpdateCheck int64 `json:"last_update_check,omitempty"` // Unix timestamp
}

// AIConfig holds AI-related settings
type AIConfig struct {
	Enabled bool   `json:"enabled"`
	APIKey  string `json:"api_key,omitempty"`
	APIBase string `json:"api_base,omitempty"`
	Model   string `json:"model,omitempty"`
}

// TeamConfig holds team/org settings for enterprise template management
type TeamConfig struct {
	OrgName      string            `json:"org_name,omitempty"`
	OrgLogo      string            `json:"org_logo,omitempty"`     // Brand logo URL
	Repositories []TeamRepository  `json:"repositories,omitempty"` // Multi-repo support
	Variables    map[string]string `json:"variables,omitempty"`    // Global template variables
	AuditLog     bool              `json:"audit_log"`              // Enable usage logging

	// Legacy field for backward compatibility
	TemplatesURL string `json:"templates_url,omitempty"`
}

// TeamRepository represents a single team template repository
type TeamRepository struct {
	Name     string `json:"name"`             // "hq", "ml-team", "devops"
	URL      string `json:"url"`              // Git repo URL
	Branch   string `json:"branch,omitempty"` // Default: main
	Tag      string `json:"tag,omitempty"`    // Lock to specific version
	Priority int    `json:"priority"`         // Display order (higher = first)

	// Authentication
	AuthType    string `json:"auth_type,omitempty"`     // "ssh", "token", "none"
	TokenEnvVar string `json:"token_env_var,omitempty"` // e.g. "GITHUB_TOKEN"

	// Caching
	LastSyncTime int64  `json:"last_sync_time,omitempty"`  // Unix timestamp
	LastCommit   string `json:"last_commit,omitempty"`     // Git commit hash
	AutoUpdate   bool   `json:"auto_update"`               // Auto-sync on cm init
	CacheTTL     int    `json:"cache_ttl_hours,omitempty"` // Cache validity (hours)
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

// Load loads the user config from disk and applies Environment Variable overrides
func Load() (*UserConfig, error) {
	// 1. Load from file
	cfg, err := loadFile()
	if err != nil {
		return nil, err
	}

	// 2. Override with environment variables (Zero Hardcoding Principle)
	applyEnvOverrides(cfg)

	return cfg, nil
}

// loadFile loads the config solely from the JSON file
func loadFile() (*UserConfig, error) {
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

func applyEnvOverrides(cfg *UserConfig) {
	// CM_AI_API_KEY
	if v := os.Getenv("CM_AI_API_KEY"); v != "" {
		cfg.AI.APIKey = v
	}
	// CM_AI_MODEL
	if v := os.Getenv("CM_AI_MODEL"); v != "" {
		cfg.AI.Model = v
	}
	// CM_AI_API_BASE
	if v := os.Getenv("CM_AI_API_BASE"); v != "" {
		cfg.AI.APIBase = v
	}
	// CM_DEFAULT_BACKEND
	if v := os.Getenv("CM_DEFAULT_BACKEND"); v != "" {
		cfg.DefaultBackend = v
	}
}

// Save saves the user config to disk
// Save saves the user config to disk explicitly using atomic write pattern
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

	// Atomic write: create temp -> write -> rename
	tmpFile, err := os.CreateTemp(dir, "config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name()) // Clean up in case of failure before rename

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close() // Close before attempting removal (though defer handles it, good practice)
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	return os.Rename(tmpFile.Name(), path)
}

// Get gets a config value (Merged)
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
	case "ai.model":
		return cfg.AI.Model, nil
	default:
		return "", nil
	}
}

// Set sets a config value (File Only)
func Set(key, value string) error {
	// Load CLEAN file config, ignoring Env vars
	cfg, err := loadFile()
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
	case "ai.model":
		cfg.AI.Model = value
	}

	return Save(cfg)
}

// UpdateLastCheck updates the LastUpdateCheck timestamp in config file
func UpdateLastCheck(timestamp int64) error {
	cfg, err := loadFile()
	if err != nil {
		cfg = &UserConfig{}
	}
	cfg.LastUpdateCheck = timestamp
	return Save(cfg)
}
