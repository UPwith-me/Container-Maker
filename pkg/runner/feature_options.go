package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// FeatureMetadata represents devcontainer-feature.json
type FeatureMetadata struct {
	ID           string                   `json:"id"`
	Version      string                   `json:"version"`
	Name         string                   `json:"name"`
	Description  string                   `json:"description"`
	Options      map[string]FeatureOption `json:"options"`
	InstallAfter []string                 `json:"installsAfter,omitempty"`
	EntryPoints  []string                 `json:"entrypoints,omitempty"`
	Privileged   bool                     `json:"privileged,omitempty"`
	Init         bool                     `json:"init,omitempty"`
	CapAdd       []string                 `json:"capAdd,omitempty"`
	SecurityOpt  []string                 `json:"securityOpt,omitempty"`
	Mounts       []FeatureMount           `json:"mounts,omitempty"`
}

// FeatureOption represents a feature option definition
type FeatureOption struct {
	Type        string      `json:"type"` // boolean, string, enum
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
	Enum        []string    `json:"enum,omitempty"`
	Proposals   []string    `json:"proposals,omitempty"`
}

// FeatureMount represents a mount configuration
type FeatureMount struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// ResolvedFeature contains parsed feature info with resolved options
type ResolvedFeature struct {
	ID       string
	Version  string
	Metadata *FeatureMetadata
	Options  map[string]interface{}
	EnvVars  map[string]string
}

// ParseFeatureRef parses a feature reference like "ghcr.io/devcontainers/features/go:1"
func ParseFeatureRef(ref string) (registry, path, version string) {
	// Default version
	version = "latest"

	// Extract version
	if idx := strings.LastIndex(ref, ":"); idx != -1 {
		// Check if this is a port number (ghcr.io:443/...) or version
		afterColon := ref[idx+1:]
		if !strings.Contains(afterColon, "/") {
			version = afterColon
			ref = ref[:idx]
		}
	}

	// Extract registry
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) == 2 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		registry = parts[0]
		path = parts[1]
	} else {
		registry = "ghcr.io"
		path = ref
	}

	return
}

// FetchFeatureMetadata downloads and parses devcontainer-feature.json
func FetchFeatureMetadata(featureID string) (*FeatureMetadata, error) {
	registry, path, version := ParseFeatureRef(featureID)
	_ = version // version would be used for specific version downloads

	// For ghcr.io/devcontainers/features/*, try GitHub raw
	if registry == "ghcr.io" && strings.HasPrefix(path, "devcontainers/features/") {
		featureName := strings.TrimPrefix(path, "devcontainers/features/")
		url := fmt.Sprintf("https://raw.githubusercontent.com/devcontainers/features/main/src/%s/devcontainer-feature.json", featureName)

		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("feature metadata not found")
		}

		var metadata FeatureMetadata
		if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
			return nil, err
		}

		return &metadata, nil
	}

	return nil, fmt.Errorf("unsupported registry: %s", registry)
}

// ResolveFeatureOptions resolves user options against feature defaults
func ResolveFeatureOptions(metadata *FeatureMetadata, userOptions interface{}) (map[string]string, error) {
	envVars := make(map[string]string)

	// Start with defaults
	for name, opt := range metadata.Options {
		envName := strings.ToUpper(name)
		if opt.Default != nil {
			envVars[envName] = fmt.Sprintf("%v", opt.Default)
		}
	}

	// Override with user options
	if opts, ok := userOptions.(map[string]interface{}); ok {
		for name, value := range opts {
			envName := strings.ToUpper(name)

			// Validate against option definition
			if optDef, exists := metadata.Options[name]; exists {
				switch optDef.Type {
				case "boolean":
					// Convert to "true" or "false"
					switch v := value.(type) {
					case bool:
						envVars[envName] = fmt.Sprintf("%v", v)
					case string:
						envVars[envName] = v
					}
				case "string":
					envVars[envName] = fmt.Sprintf("%v", value)
				case "enum":
					// Validate enum value
					strVal := fmt.Sprintf("%v", value)
					valid := false
					for _, allowed := range optDef.Enum {
						if allowed == strVal {
							valid = true
							break
						}
					}
					if valid {
						envVars[envName] = strVal
					}
				}
			} else {
				// Unknown option, pass through
				envVars[envName] = fmt.Sprintf("%v", value)
			}
		}
	}

	return envVars, nil
}

// GetFeatureInfo fetches and displays feature information
func GetFeatureInfo(featureID string) (string, error) {
	metadata, err := FetchFeatureMetadata(featureID)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📦 Feature: %s\n", metadata.Name))
	sb.WriteString(fmt.Sprintf("   ID: %s\n", metadata.ID))
	sb.WriteString(fmt.Sprintf("   Version: %s\n", metadata.Version))
	sb.WriteString(fmt.Sprintf("   Description: %s\n", metadata.Description))

	if len(metadata.Options) > 0 {
		sb.WriteString("\n📋 Options:\n")
		for name, opt := range metadata.Options {
			sb.WriteString(fmt.Sprintf("   • %s (%s)\n", name, opt.Type))
			sb.WriteString(fmt.Sprintf("     %s\n", opt.Description))
			if opt.Default != nil {
				sb.WriteString(fmt.Sprintf("     Default: %v\n", opt.Default))
			}
			if len(opt.Enum) > 0 {
				sb.WriteString(fmt.Sprintf("     Values: %s\n", strings.Join(opt.Enum, ", ")))
			}
		}
	}

	return sb.String(), nil
}

// ListOfficialFeatures lists official devcontainer features
func ListOfficialFeatures() ([]string, error) {
	// Fetch the list from devcontainers/features repo
	url := "https://api.github.com/repos/devcontainers/features/contents/src"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Return cached list
		return []string{
			"common-utils", "git", "github-cli", "node", "python", "go", "rust",
			"docker-in-docker", "docker-from-docker", "kubectl-helm-minikube",
			"java", "dotnet", "aws-cli", "azure-cli", "terraform", "powershell",
		}, nil
	}

	data, _ := io.ReadAll(resp.Body)
	var contents []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	json.Unmarshal(data, &contents)

	var features []string
	for _, c := range contents {
		if c.Type == "dir" {
			features = append(features, c.Name)
		}
	}

	return features, nil
}
