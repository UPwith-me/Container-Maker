// Package monitor provides real-time container monitoring capabilities.
// It enables tracking CPU, memory, network, and GPU metrics for containers.
package monitor

import (
	"context"
	"time"
)

// ContainerMetrics contains real-time metrics for a container
type ContainerMetrics struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`

	// CPU metrics
	CPUPercent float64 `json:"cpu_percent"`
	CPUCount   int     `json:"cpu_count"`
	SystemCPU  uint64  `json:"system_cpu"`

	// Memory metrics
	MemoryUsed    int64   `json:"memory_used"`
	MemoryLimit   int64   `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryCache   int64   `json:"memory_cache"`

	// Network I/O
	NetworkRx     int64   `json:"network_rx"`      // Total bytes received
	NetworkTx     int64   `json:"network_tx"`      // Total bytes transmitted
	NetworkRxRate float64 `json:"network_rx_rate"` // bytes/sec
	NetworkTxRate float64 `json:"network_tx_rate"` // bytes/sec

	// Block I/O
	BlockRead      int64   `json:"block_read"`
	BlockWrite     int64   `json:"block_write"`
	BlockReadRate  float64 `json:"block_read_rate"`
	BlockWriteRate float64 `json:"block_write_rate"`

	// Process info
	PIDs int `json:"pids"`

	// GPU metrics (if available)
	HasGPU         bool    `json:"has_gpu"`
	GPUPercent     float64 `json:"gpu_percent"`
	GPUMemoryUsed  int64   `json:"gpu_memory_used"`
	GPUMemoryTotal int64   `json:"gpu_memory_total"`
	GPUTemp        int     `json:"gpu_temp,omitempty"`

	// Status
	State  string        `json:"state"` // running, stopped, paused
	Status string        `json:"status"`
	Health string        `json:"health,omitempty"` // healthy, unhealthy, starting
	Uptime time.Duration `json:"uptime"`

	// Timestamp
	Timestamp time.Time `json:"timestamp"`
}

// LogEntry represents a single log line from a container
type LogEntry struct {
	ContainerID string    `json:"container_id"`
	Timestamp   time.Time `json:"timestamp"`
	Stream      string    `json:"stream"` // "stdout" or "stderr"
	Message     string    `json:"message"`
}

// ContainerEvent represents a Docker container event
type ContainerEvent struct {
	ContainerID   string            `json:"container_id"`
	ContainerName string            `json:"container_name"`
	Type          string            `json:"type"`   // container, image, network, etc.
	Action        string            `json:"action"` // start, stop, die, create, etc.
	Timestamp     time.Time         `json:"timestamp"`
	Attributes    map[string]string `json:"attributes,omitempty"`
}

// ContainerInfo contains basic container information
type ContainerInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	State    string            `json:"state"`
	Status   string            `json:"status"`
	Created  time.Time         `json:"created"`
	Ports    []PortMapping     `json:"ports,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
	Networks []string          `json:"networks,omitempty"`
	Mounts   []MountInfo       `json:"mounts,omitempty"`
}

// PortMapping represents a container port mapping
type PortMapping struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	Protocol      string `json:"protocol"` // tcp, udp
	HostIP        string `json:"host_ip,omitempty"`
}

// MountInfo represents a container mount
type MountInfo struct {
	Type        string `json:"type"` // bind, volume
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"` // ro, rw
}

// SystemMetrics contains overall system metrics
type SystemMetrics struct {
	TotalCPU       float64   `json:"total_cpu"`
	TotalMemory    int64     `json:"total_memory"`
	UsedMemory     int64     `json:"used_memory"`
	ContainerCount int       `json:"container_count"`
	RunningCount   int       `json:"running_count"`
	StoppedCount   int       `json:"stopped_count"`
	TotalGPUs      int       `json:"total_gpus"`
	AvailableGPUs  int       `json:"available_gpus"`
	Timestamp      time.Time `json:"timestamp"`
}

// DashboardState represents the current state of the monitoring dashboard
type DashboardState struct {
	Containers    []*ContainerInfo             `json:"containers"`
	Metrics       map[string]*ContainerMetrics `json:"metrics"` // ContainerID -> Metrics
	SystemMetrics *SystemMetrics               `json:"system_metrics"`
	Events        []*ContainerEvent            `json:"events"`                // Recent events
	ActiveLogs    string                       `json:"active_logs,omitempty"` // ContainerID showing logs
}

// MetricsCollector is the interface for collecting container metrics
type MetricsCollector interface {
	// Collect returns current metrics for a container
	Collect(ctx context.Context, containerID string) (*ContainerMetrics, error)

	// CollectAll returns metrics for all running containers
	CollectAll(ctx context.Context) ([]*ContainerMetrics, error)

	// Stream returns a channel that receives metrics updates
	Stream(ctx context.Context, containerID string, interval time.Duration) (<-chan *ContainerMetrics, error)

	// StreamAll streams metrics for all containers
	StreamAll(ctx context.Context, interval time.Duration) (<-chan *ContainerMetrics, error)
}

// LogCollector is the interface for collecting container logs
type LogCollector interface {
	// Tail returns the last n lines of logs
	Tail(ctx context.Context, containerID string, n int) ([]*LogEntry, error)

	// Stream returns a channel that receives log entries
	Stream(ctx context.Context, containerID string) (<-chan *LogEntry, error)
}

// EventCollector is the interface for collecting container events
type EventCollector interface {
	// Recent returns recent events
	Recent(ctx context.Context, since time.Duration) ([]*ContainerEvent, error)

	// Stream returns a channel that receives events
	Stream(ctx context.Context) (<-chan *ContainerEvent, error)
}

// ContainerLister is the interface for listing containers
type ContainerLister interface {
	// List returns all containers
	List(ctx context.Context, all bool) ([]*ContainerInfo, error)

	// Get returns a specific container
	Get(ctx context.Context, containerID string) (*ContainerInfo, error)
}

// Dashboard is the interface for the monitoring dashboard
type Dashboard interface {
	// Run starts the dashboard TUI
	Run() error

	// Refresh refreshes the dashboard data
	Refresh() error

	// SetActiveContainer sets the container to show logs for
	SetActiveContainer(containerID string)

	// GetState returns the current dashboard state
	GetState() *DashboardState
}

// FilterOptions defines filtering options for containers
type FilterOptions struct {
	All      bool              // Include stopped containers
	Labels   map[string]string // Filter by labels
	Name     string            // Filter by name pattern
	Network  string            // Filter by network
	Ancestor string            // Filter by image
	Status   []string          // Filter by status (running, stopped, paused)
}
