package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultConfigFile is the default workspace config file name
	DefaultConfigFile = "cm-workspace.yaml"
	// AlternateConfigFile is an alternate config file name
	AlternateConfigFile = "cm-workspace.yml"
)

// Load loads a workspace configuration from file
func Load(configPath string) (*Workspace, error) {
	// If no path specified, search for default files
	if configPath == "" {
		var err error
		configPath, err = FindWorkspaceConfig(".")
		if err != nil {
			return nil, err
		}
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace config: %w", err)
	}

	// Parse YAML
	var ws Workspace
	if err := yaml.Unmarshal(data, &ws); err != nil {
		return nil, fmt.Errorf("failed to parse workspace config: %w", err)
	}

	// Set metadata
	ws.ConfigFile, _ = filepath.Abs(configPath)

	// Post-process services
	for name, svc := range ws.Services {
		svc.Name = name

		// Apply defaults
		if ws.Defaults != nil {
			applyServiceDefaults(svc, ws.Defaults)
		}

		// Resolve relative paths
		if svc.Path == "" {
			svc.Path = name
		}
		if !filepath.IsAbs(svc.Path) {
			svc.Path = filepath.Join(filepath.Dir(ws.ConfigFile), svc.Path)
		}
	}

	// Set default workspace name from directory
	if ws.Name == "" {
		ws.Name = filepath.Base(filepath.Dir(ws.ConfigFile))
	}

	return &ws, nil
}

// FindWorkspaceConfig searches for a workspace config file
func FindWorkspaceConfig(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	// Search up the directory tree
	for {
		// Check for default config files
		for _, name := range []string{DefaultConfigFile, AlternateConfigFile} {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}

	return "", fmt.Errorf("no %s found in current or parent directories", DefaultConfigFile)
}

// Save saves a workspace configuration to file
func Save(ws *Workspace) error {
	if ws.ConfigFile == "" {
		ws.ConfigFile = filepath.Join(".", DefaultConfigFile)
	}

	data, err := yaml.Marshal(ws)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace config: %w", err)
	}

	if err := os.WriteFile(ws.ConfigFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write workspace config: %w", err)
	}

	return nil
}

// Validate validates a workspace configuration
func Validate(ws *Workspace) error {
	if ws == nil {
		return fmt.Errorf("workspace is nil")
	}

	if len(ws.Services) == 0 {
		return fmt.Errorf("workspace must have at least one service")
	}

	// Validate each service
	for name, svc := range ws.Services {
		if err := validateService(name, svc); err != nil {
			return err
		}
	}

	// Check for circular dependencies
	if err := checkCircularDependencies(ws); err != nil {
		return err
	}

	return nil
}

// validateService validates a single service configuration
func validateService(name string, svc *Service) error {
	if svc == nil {
		return fmt.Errorf("service %s is nil", name)
	}

	// Must have either image or template or build
	if svc.Image == "" && svc.Template == "" && svc.Build == nil {
		return fmt.Errorf("service %s must have image, template, or build", name)
	}

	// Check dependencies exist (checked later in full context)
	return nil
}

// checkCircularDependencies checks for circular dependencies
func checkCircularDependencies(ws *Workspace) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var check func(name string) error
	check = func(name string) error {
		visited[name] = true
		recStack[name] = true

		svc := ws.Services[name]
		if svc == nil {
			return nil
		}

		for _, dep := range svc.DependsOn {
			if _, exists := ws.Services[dep]; !exists {
				return fmt.Errorf("service %s depends on unknown service %s", name, dep)
			}
			if !visited[dep] {
				if err := check(dep); err != nil {
					return err
				}
			} else if recStack[dep] {
				return fmt.Errorf("circular dependency detected: %s -> %s", name, dep)
			}
		}

		recStack[name] = false
		return nil
	}

	for name := range ws.Services {
		if !visited[name] {
			if err := check(name); err != nil {
				return err
			}
		}
	}

	return nil
}

// applyServiceDefaults applies default values to a service
func applyServiceDefaults(svc *Service, defaults *ServiceDefaults) {
	if svc.RestartPolicy == "" && defaults.Restart != "" {
		svc.RestartPolicy = defaults.Restart
	}

	if svc.Resources == nil && defaults.Resources != nil {
		svc.Resources = defaults.Resources
	}

	// Merge environment (defaults first, service overrides)
	if defaults.Environment != nil {
		if svc.Environment == nil {
			svc.Environment = make(map[string]string)
		}
		for k, v := range defaults.Environment {
			if _, exists := svc.Environment[k]; !exists {
				svc.Environment[k] = v
			}
		}
	}

	// Merge labels
	if defaults.Labels != nil {
		if svc.Labels == nil {
			svc.Labels = make(map[string]string)
		}
		for k, v := range defaults.Labels {
			if _, exists := svc.Labels[k]; !exists {
				svc.Labels[k] = v
			}
		}
	}
}

// GetService returns a service by name
func (ws *Workspace) GetService(name string) (*Service, error) {
	svc, ok := ws.Services[name]
	if !ok {
		return nil, fmt.Errorf("service %s not found", name)
	}
	return svc, nil
}

// ServiceNames returns all service names
func (ws *Workspace) ServiceNames() []string {
	names := make([]string, 0, len(ws.Services))
	for name := range ws.Services {
		names = append(names, name)
	}
	return names
}

// GetServicesByProfile returns services matching a profile
func (ws *Workspace) GetServicesByProfile(profile string) []*Service {
	var result []*Service
	for _, svc := range ws.Services {
		// If service has no profiles, it's always included
		if len(svc.Profiles) == 0 {
			result = append(result, svc)
			continue
		}
		// Check if profile matches
		for _, p := range svc.Profiles {
			if p == profile {
				result = append(result, svc)
				break
			}
		}
	}
	return result
}

// GenerateNetworkName generates a network name for the workspace
func (ws *Workspace) GenerateNetworkName() string {
	return fmt.Sprintf("cm-%s-network", sanitizeName(ws.Name))
}

// sanitizeName sanitizes a name for Docker
func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

// ParsePortConfig parses a port string or int into PortConfig
func ParsePortConfig(port interface{}) (*PortConfig, error) {
	switch v := port.(type) {
	case int:
		return &PortConfig{Target: v, Published: v, Protocol: "tcp"}, nil
	case float64:
		return &PortConfig{Target: int(v), Published: int(v), Protocol: "tcp"}, nil
	case string:
		// Parse "host:container" or "container"
		parts := strings.Split(v, ":")
		if len(parts) == 1 {
			var p int
			_, _ = fmt.Sscanf(parts[0], "%d", &p)
			return &PortConfig{Target: p, Published: p, Protocol: "tcp"}, nil
		}
		var host, container int
		_, _ = fmt.Sscanf(parts[0], "%d", &host)
		_, _ = fmt.Sscanf(parts[1], "%d", &container)
		return &PortConfig{Target: container, Published: host, Protocol: "tcp"}, nil
	default:
		return nil, fmt.Errorf("invalid port format: %v", port)
	}
}

// CreateDefaultWorkspace creates a default workspace with common structure
func CreateDefaultWorkspace(name string) *Workspace {
	return &Workspace{
		Name:     name,
		Version:  "1.0",
		Services: make(map[string]*Service),
		Networks: map[string]*NetworkConfig{
			"default": {Driver: "bridge"},
		},
		Defaults: &ServiceDefaults{
			Restart: "unless-stopped",
		},
	}
}

// AddService adds a service to the workspace
func (ws *Workspace) AddService(name string, svc *Service) error {
	if ws.Services == nil {
		ws.Services = make(map[string]*Service)
	}
	if _, exists := ws.Services[name]; exists {
		return fmt.Errorf("service %s already exists", name)
	}
	svc.Name = name
	ws.Services[name] = svc
	return nil
}

// RemoveService removes a service from the workspace
func (ws *Workspace) RemoveService(name string) error {
	if _, exists := ws.Services[name]; !exists {
		return fmt.Errorf("service %s not found", name)
	}
	delete(ws.Services, name)
	return nil
}
