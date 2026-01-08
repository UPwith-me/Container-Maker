// Package workspace provides multi-service workspace management for Container-Maker.
// It enables defining, managing, and orchestrating multiple services in a single workspace.
package workspace

import (
	"time"
)

// Workspace represents a multi-service development workspace
type Workspace struct {
	// Metadata
	Name        string `yaml:"name" json:"name"`
	Version     string `yaml:"version,omitempty" json:"version,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Services
	Services map[string]*Service `yaml:"services" json:"services"`

	// Networks
	Networks map[string]*NetworkConfig `yaml:"networks,omitempty" json:"networks,omitempty"`

	// Volumes
	Volumes map[string]*VolumeConfig `yaml:"volumes,omitempty" json:"volumes,omitempty"`

	// Global settings
	Defaults *ServiceDefaults `yaml:"defaults,omitempty" json:"defaults,omitempty"`

	// Runtime state (not persisted)
	ConfigFile string    `yaml:"-" json:"-"`
	LoadedAt   time.Time `yaml:"-" json:"-"`
}

// Service represents a single service in the workspace
type Service struct {
	// Basic configuration
	Name     string       `yaml:"-" json:"name"` // Set from map key
	Template string       `yaml:"template,omitempty" json:"template,omitempty"`
	Image    string       `yaml:"image,omitempty" json:"image,omitempty"`
	Build    *BuildConfig `yaml:"build,omitempty" json:"build,omitempty"`

	// Path configuration
	Path       string `yaml:"path,omitempty" json:"path,omitempty"`          // Relative path to service
	ConfigFile string `yaml:"config,omitempty" json:"config_file,omitempty"` // devcontainer.json path

	// Dependencies
	DependsOn []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`

	// Networking
	Ports    []PortConfig `yaml:"ports,omitempty" json:"ports,omitempty"`
	Networks []string     `yaml:"networks,omitempty" json:"networks,omitempty"`
	Expose   []int        `yaml:"expose,omitempty" json:"expose,omitempty"`

	// Environment
	Environment map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	EnvFile     []string          `yaml:"env_file,omitempty" json:"env_file,omitempty"`

	// Resources
	Resources *ResourceConfig `yaml:"resources,omitempty" json:"resources,omitempty"`

	// GPU
	GPU *GPUConfig `yaml:"gpu,omitempty" json:"gpu,omitempty"`

	// Volumes
	Volumes []string `yaml:"volumes,omitempty" json:"volumes,omitempty"`

	// Lifecycle
	Command       []string           `yaml:"command,omitempty" json:"command,omitempty"`
	Entrypoint    []string           `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	WorkingDir    string             `yaml:"working_dir,omitempty" json:"working_dir,omitempty"`
	HealthCheck   *HealthCheckConfig `yaml:"healthcheck,omitempty" json:"healthcheck,omitempty"`
	RestartPolicy string             `yaml:"restart,omitempty" json:"restart,omitempty"`

	// Labels and metadata
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Tags   []string          `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Profile (for selective startup)
	Profiles []string `yaml:"profiles,omitempty" json:"profiles,omitempty"`

	// Runtime state
	Status      ServiceStatus `yaml:"-" json:"status,omitempty"`
	ContainerID string        `yaml:"-" json:"container_id,omitempty"`
	NetworkID   string        `yaml:"-" json:"network_id,omitempty"`
}

// BuildConfig defines how to build a service image
type BuildConfig struct {
	Context    string            `yaml:"context,omitempty" json:"context,omitempty"`
	Dockerfile string            `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`
	Args       map[string]string `yaml:"args,omitempty" json:"args,omitempty"`
	Target     string            `yaml:"target,omitempty" json:"target,omitempty"`
	CacheFrom  []string          `yaml:"cache_from,omitempty" json:"cache_from,omitempty"`
}

// PortConfig defines port mapping
type PortConfig struct {
	Target    int    `yaml:"target" json:"target"`
	Published int    `yaml:"published,omitempty" json:"published,omitempty"`
	Protocol  string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
	HostIP    string `yaml:"host_ip,omitempty" json:"host_ip,omitempty"`
}

// ResourceConfig defines resource limits
type ResourceConfig struct {
	Memory  string  `yaml:"memory,omitempty" json:"memory,omitempty"`     // e.g., "8g"
	CPUs    float64 `yaml:"cpus,omitempty" json:"cpus,omitempty"`         // e.g., 4.0
	ShmSize string  `yaml:"shm_size,omitempty" json:"shm_size,omitempty"` // e.g., "2g"
	Pids    int     `yaml:"pids,omitempty" json:"pids,omitempty"`
}

// GPUConfig defines GPU allocation
type GPUConfig struct {
	Count        int      `yaml:"count,omitempty" json:"count,omitempty"`               // Number of GPUs
	DeviceIDs    []string `yaml:"device_ids,omitempty" json:"device_ids,omitempty"`     // Specific GPU IDs
	Capabilities []string `yaml:"capabilities,omitempty" json:"capabilities,omitempty"` // e.g., ["gpu", "compute"]
	Driver       string   `yaml:"driver,omitempty" json:"driver,omitempty"`             // e.g., "nvidia"
}

// HealthCheckConfig defines health check settings
type HealthCheckConfig struct {
	Test        []string      `yaml:"test,omitempty" json:"test,omitempty"`
	Interval    time.Duration `yaml:"interval,omitempty" json:"interval,omitempty"`
	Timeout     time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retries     int           `yaml:"retries,omitempty" json:"retries,omitempty"`
	StartPeriod time.Duration `yaml:"start_period,omitempty" json:"start_period,omitempty"`
}

// NetworkConfig defines network settings
type NetworkConfig struct {
	Driver   string            `yaml:"driver,omitempty" json:"driver,omitempty"`
	External bool              `yaml:"external,omitempty" json:"external,omitempty"`
	Labels   map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	IPAM     *IPAMConfig       `yaml:"ipam,omitempty" json:"ipam,omitempty"`
}

// IPAMConfig defines IP address management
type IPAMConfig struct {
	Driver string     `yaml:"driver,omitempty" json:"driver,omitempty"`
	Config []IPAMPool `yaml:"config,omitempty" json:"config,omitempty"`
}

// IPAMPool defines an IP pool
type IPAMPool struct {
	Subnet  string `yaml:"subnet,omitempty" json:"subnet,omitempty"`
	Gateway string `yaml:"gateway,omitempty" json:"gateway,omitempty"`
}

// VolumeConfig defines volume settings
type VolumeConfig struct {
	Driver   string            `yaml:"driver,omitempty" json:"driver,omitempty"`
	External bool              `yaml:"external,omitempty" json:"external,omitempty"`
	Labels   map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// ServiceDefaults defines default values for all services
type ServiceDefaults struct {
	Restart     string            `yaml:"restart,omitempty" json:"restart,omitempty"`
	Resources   *ResourceConfig   `yaml:"resources,omitempty" json:"resources,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// ServiceStatus represents the current state of a service
type ServiceStatus string

const (
	ServiceStatusUnknown  ServiceStatus = "unknown"
	ServiceStatusCreating ServiceStatus = "creating"
	ServiceStatusRunning  ServiceStatus = "running"
	ServiceStatusStopped  ServiceStatus = "stopped"
	ServiceStatusError    ServiceStatus = "error"
	ServiceStatusStarting ServiceStatus = "starting"
	ServiceStatusStopping ServiceStatus = "stopping"
)

// WorkspaceState represents the runtime state of a workspace
type WorkspaceState struct {
	Name         string                   `json:"name"`
	Services     map[string]*ServiceState `json:"services"`
	Networks     []string                 `json:"networks"`
	StartedAt    time.Time                `json:"started_at,omitempty"`
	LastUpdateAt time.Time                `json:"last_update_at"`
}

// ServiceState represents the runtime state of a service
type ServiceState struct {
	Name        string        `json:"name"`
	Status      ServiceStatus `json:"status"`
	ContainerID string        `json:"container_id,omitempty"`
	NetworkID   string        `json:"network_id,omitempty"`
	Ports       []PortMapping `json:"ports,omitempty"`
	StartedAt   time.Time     `json:"started_at,omitempty"`
	Error       string        `json:"error,omitempty"`
}

// PortMapping represents a port mapping
type PortMapping struct {
	Container int    `json:"container"`
	Host      int    `json:"host"`
	Protocol  string `json:"protocol"`
}

// StartOptions defines options for starting services
type StartOptions struct {
	Services []string // Specific services to start (empty = all)
	Build    bool     // Build images before starting
	Force    bool     // Force recreate containers
	NoDeps   bool     // Don't start dependencies
	Detach   bool     // Run in background
	Profile  string   // Activate specific profile
	Timeout  int      // Startup timeout in seconds
}

// StopOptions defines options for stopping services
type StopOptions struct {
	Services []string // Specific services to stop (empty = all)
	Timeout  int      // Stop timeout in seconds
	Remove   bool     // Remove containers after stopping
	Volumes  bool     // Remove volumes too
}

// WorkspaceManager is the interface for workspace management
type WorkspaceManager interface {
	// Workspace lifecycle
	Load(configPath string) (*Workspace, error)
	Save(ws *Workspace) error
	Validate(ws *Workspace) error

	// Service operations
	Up(ws *Workspace, opts StartOptions) error
	Down(ws *Workspace, opts StopOptions) error
	Start(ws *Workspace, services []string) error
	Stop(ws *Workspace, services []string) error
	Restart(ws *Workspace, services []string) error

	// Status
	Status(ws *Workspace) (*WorkspaceState, error)
	Logs(ws *Workspace, service string, follow bool, tail int) error

	// Operations
	Build(ws *Workspace, services []string, noCache bool) error
	Pull(ws *Workspace, services []string) error
	Exec(ws *Workspace, service string, command []string) error
	Shell(ws *Workspace, service string) error
}

// DependencyGraph represents service dependencies
type DependencyGraph struct {
	// nodes map[string]*DependencyNode

}

// DependencyNode represents a service in the dependency graph
type DependencyNode struct {
	Service    string
	DependsOn  []string
	Dependents []string
}

// StartOrder returns services in the order they should be started
func (g *DependencyGraph) StartOrder() ([]string, error) {
	// Topological sort for dependency ordering
	return nil, nil // Implemented in graph.go
}

// StopOrder returns services in the order they should be stopped
func (g *DependencyGraph) StopOrder() ([]string, error) {
	// Reverse of start order
	return nil, nil // Implemented in graph.go
}
