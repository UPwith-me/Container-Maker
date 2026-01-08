// Package environment provides multi-environment management for Container-Maker.
// It enables creating, managing, and switching between isolated development environments.
package environment

import (
	"context"
	"time"
)

// EnvironmentStatus represents the current state of an environment
type EnvironmentStatus string

const (
	StatusCreating EnvironmentStatus = "creating"
	StatusRunning  EnvironmentStatus = "running"
	StatusStopped  EnvironmentStatus = "stopped"
	StatusPaused   EnvironmentStatus = "paused"
	StatusError    EnvironmentStatus = "error"
	StatusOrphaned EnvironmentStatus = "orphaned"
)

// Environment represents a single isolated development environment
type Environment struct {
	// Core identification
	ID          string `json:"id"`           // Unique identifier (e.g., "env-abc123")
	Name        string `json:"name"`         // Human-readable name (e.g., "frontend-dev")
	DisplayName string `json:"display_name"` // Optional friendly name

	// Configuration
	Template   string `json:"template,omitempty"`    // Template used (e.g., "pytorch")
	ProjectDir string `json:"project_dir"`           // Absolute path to project
	ConfigFile string `json:"config_file,omitempty"` // devcontainer.json path

	// Container state
	ContainerID   string `json:"container_id,omitempty"`
	ContainerName string `json:"container_name,omitempty"`
	ImageTag      string `json:"image_tag,omitempty"`

	// Networking
	NetworkID   string         `json:"network_id,omitempty"`   // Docker network ID
	NetworkName string         `json:"network_name,omitempty"` // Docker network name
	Ports       map[string]int `json:"ports,omitempty"`        // Service -> Host port

	// Environment linking
	LinkedEnvs []string `json:"linked_envs,omitempty"` // IDs of linked environments

	// Resources
	GPUs        []int   `json:"gpus,omitempty"`         // Allocated GPU IDs
	MemoryLimit string  `json:"memory_limit,omitempty"` // e.g., "8g"
	CPULimit    float64 `json:"cpu_limit,omitempty"`    // e.g., 4.0

	// Status
	Status    EnvironmentStatus `json:"status"`
	StatusMsg string            `json:"status_msg,omitempty"`

	// Metadata
	Labels map[string]string `json:"labels,omitempty"`
	Tags   []string          `json:"tags,omitempty"`

	// Timestamps
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`

	// Backend info
	Backend string `json:"backend"` // "docker", "podman"
}

// EnvironmentCreateOptions contains options for creating a new environment
type EnvironmentCreateOptions struct {
	Name       string // Required: environment name
	ProjectDir string // Optional: project directory (defaults to cwd)
	Template   string // Optional: template to use
	ConfigFile string // Optional: explicit config file path

	// Networking
	ExposePorts []int    // Ports to expose
	Network     string   // Custom network name
	LinkTo      []string // Environment names to link to

	// Resources
	GPUs     []int   // Specific GPU IDs (empty = auto)
	GPUCount int     // Number of GPUs needed
	Memory   string  // Memory limit
	CPU      float64 // CPU limit

	// Options
	NoStart bool              // Create but don't start
	Force   bool              // Force recreate if exists
	Labels  map[string]string // Custom labels
	Tags    []string          // Tags for organization
}

// EnvironmentListOptions contains options for listing environments
type EnvironmentListOptions struct {
	All      bool              // Include stopped environments
	Filter   EnvironmentFilter // Filter criteria
	SortBy   string            // Sort field
	SortDesc bool              // Descending order
	Limit    int               // Max results
}

// EnvironmentFilter defines filtering criteria
type EnvironmentFilter struct {
	Status      EnvironmentStatus // Filter by status
	Project     string            // Filter by project path
	Template    string            // Filter by template
	Tag         string            // Filter by tag
	Network     string            // Filter by network
	NamePattern string            // Glob pattern for name
}

// EnvironmentLinkOptions contains options for linking environments
type EnvironmentLinkOptions struct {
	Bidirectional bool   // Link both ways
	ShareVolumes  bool   // Share named volumes
	DNSAlias      string // Custom DNS alias
}

// EnvironmentMetrics contains real-time metrics for an environment
type EnvironmentMetrics struct {
	ContainerID string

	// CPU
	CPUPercent float64
	CPULimit   float64

	// Memory
	MemoryUsed    int64
	MemoryLimit   int64
	MemoryPercent float64

	// Network I/O
	NetRxBytes int64
	NetTxBytes int64
	NetRxRate  float64 // bytes/sec
	NetTxRate  float64

	// Block I/O
	BlockRead      int64
	BlockWrite     int64
	BlockReadRate  float64
	BlockWriteRate float64

	// GPU (if available)
	GPUPercent     float64
	GPUMemoryUsed  int64
	GPUMemoryTotal int64

	// Process info
	PIDs int

	Timestamp time.Time
}

// EnvironmentManager defines the interface for environment management
type EnvironmentManager interface {
	// Lifecycle
	Create(ctx context.Context, opts EnvironmentCreateOptions) (*Environment, error)
	Start(ctx context.Context, nameOrID string) error
	Stop(ctx context.Context, nameOrID string, timeout int) error
	Restart(ctx context.Context, nameOrID string) error
	Delete(ctx context.Context, nameOrID string, force bool) error

	// Query
	Get(ctx context.Context, nameOrID string) (*Environment, error)
	List(ctx context.Context, opts EnvironmentListOptions) ([]*Environment, error)
	Exists(ctx context.Context, nameOrID string) bool

	// State
	Switch(ctx context.Context, nameOrID string) error
	GetActive(ctx context.Context) (*Environment, error)

	// Linking
	Link(ctx context.Context, env1, env2 string, opts EnvironmentLinkOptions) error
	Unlink(ctx context.Context, env1, env2 string) error

	// Execution
	Shell(ctx context.Context, nameOrID string, shell string) error
	Exec(ctx context.Context, nameOrID string, cmd []string) error

	// Monitoring
	Metrics(ctx context.Context, nameOrID string) (*EnvironmentMetrics, error)
	StreamMetrics(ctx context.Context, nameOrID string) (<-chan *EnvironmentMetrics, error)
	Logs(ctx context.Context, nameOrID string, follow bool, tail int) (chan string, error)

	// Maintenance
	Prune(ctx context.Context, all bool) (int, error) // Remove orphaned
	Sync(ctx context.Context, nameOrID string) error  // Sync state with Docker
}

// NetworkManager defines the interface for Docker network management
type NetworkManager interface {
	CreateNetwork(ctx context.Context, name string, labels map[string]string) (string, error)
	DeleteNetwork(ctx context.Context, nameOrID string) error
	ConnectToNetwork(ctx context.Context, networkID, containerID string, aliases []string) error
	DisconnectFromNetwork(ctx context.Context, networkID, containerID string) error
	GetNetwork(ctx context.Context, nameOrID string) (*NetworkInfo, error)
	ListNetworks(ctx context.Context, labels map[string]string) ([]*NetworkInfo, error)
}

// NetworkInfo contains information about a Docker network
type NetworkInfo struct {
	ID         string
	Name       string
	Driver     string
	Scope      string
	Internal   bool
	Containers map[string]string // ContainerID -> Name
	Labels     map[string]string
	CreatedAt  time.Time
}

// StateStore defines the interface for environment state persistence
type StateStore interface {
	Save(env *Environment) error
	Load(id string) (*Environment, error)
	Delete(id string) error
	List() ([]*Environment, error)
	SetActive(id string) error
	GetActive() (string, error)
	Sync() error // Sync with actual Docker state
}
