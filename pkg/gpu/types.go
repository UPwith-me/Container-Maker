// Package gpu provides GPU detection and intelligent scheduling for Container-Maker.
// It supports NVIDIA and AMD GPUs with automatic detection and allocation.
package gpu

import (
	"time"
)

// GPUVendor identifies the GPU manufacturer
type GPUVendor string

const (
	VendorNVIDIA  GPUVendor = "nvidia"
	VendorAMD     GPUVendor = "amd"
	VendorIntel   GPUVendor = "intel"
	VendorUnknown GPUVendor = "unknown"
)

// GPU represents a single GPU device
type GPU struct {
	ID           string    `json:"id"`
	Index        int       `json:"index"`
	Name         string    `json:"name"`
	Vendor       GPUVendor `json:"vendor"`
	Driver       string    `json:"driver,omitempty"`
	VRAM         int64     `json:"vram"`                  // Total VRAM in bytes
	VRAMUsed     int64     `json:"vram_used,omitempty"`   // Used VRAM in bytes
	ComputeCap   string    `json:"compute_cap,omitempty"` // e.g., "8.6" for NVIDIA
	Architecture string    `json:"architecture,omitempty"`

	// Current state
	Temperature    int `json:"temperature,omitempty"`     // Celsius
	PowerUsage     int `json:"power_usage,omitempty"`     // Watts
	PowerLimit     int `json:"power_limit,omitempty"`     // Watts
	Utilization    int `json:"utilization,omitempty"`     // 0-100%
	MemUtilization int `json:"mem_utilization,omitempty"` // 0-100%

	// Allocation
	Allocated   bool      `json:"allocated"`
	AllocatedTo string    `json:"allocated_to,omitempty"` // Environment/Service name
	AllocatedAt time.Time `json:"allocated_at,omitempty"`
}

// GPURequirements defines what a service needs from GPUs
type GPURequirements struct {
	Count         int       `yaml:"count,omitempty" json:"count,omitempty"`
	MinVRAM       int64     `yaml:"min_vram,omitempty" json:"min_vram,omitempty"` // Minimum VRAM in bytes
	MinComputeCap string    `yaml:"min_compute_cap,omitempty" json:"min_compute_cap,omitempty"`
	Vendor        GPUVendor `yaml:"vendor,omitempty" json:"vendor,omitempty"`
	DeviceIDs     []string  `yaml:"device_ids,omitempty" json:"device_ids,omitempty"`     // Specific GPU IDs
	Exclusive     bool      `yaml:"exclusive,omitempty" json:"exclusive,omitempty"`       // Don't share
	Capabilities  []string  `yaml:"capabilities,omitempty" json:"capabilities,omitempty"` // Required capabilities
}

// GPUAllocation represents an allocated GPU set
type GPUAllocation struct {
	ID           string          `json:"id"`
	GPUs         []GPU           `json:"gpus"`
	Owner        string          `json:"owner"` // Environment/Service name
	Requirements GPURequirements `json:"requirements"`
	AllocatedAt  time.Time       `json:"allocated_at"`
	ExpiresAt    time.Time       `json:"expires_at,omitempty"`
}

// GPUPool represents available GPUs
type GPUPool struct {
	Total       int             `json:"total"`
	Available   int             `json:"available"`
	Allocated   int             `json:"allocated"`
	GPUs        []GPU           `json:"gpus"`
	Allocations []GPUAllocation `json:"allocations,omitempty"`
	LastScan    time.Time       `json:"last_scan"`
}

// GPUSchedulerConfig configures the GPU scheduler
type GPUSchedulerConfig struct {
	// Scheduling strategy
	Strategy ScheduleStrategy `yaml:"strategy,omitempty" json:"strategy,omitempty"`

	// Timeout settings
	AllocationTimeout time.Duration `yaml:"allocation_timeout,omitempty" json:"allocation_timeout,omitempty"`

	// Sharing settings
	AllowSharing  bool `yaml:"allow_sharing,omitempty" json:"allow_sharing,omitempty"`
	MaxShareCount int  `yaml:"max_share_count,omitempty" json:"max_share_count,omitempty"`

	// Auto-release settings
	AutoRelease bool          `yaml:"auto_release,omitempty" json:"auto_release,omitempty"`
	IdleTimeout time.Duration `yaml:"idle_timeout,omitempty" json:"idle_timeout,omitempty"`

	// Priority settings
	PriorityRules []PriorityRule `yaml:"priority_rules,omitempty" json:"priority_rules,omitempty"`
}

// ScheduleStrategy defines how GPUs are allocated
type ScheduleStrategy string

const (
	StrategyFirstFit    ScheduleStrategy = "first_fit"    // First available GPUs
	StrategyBestFit     ScheduleStrategy = "best_fit"     // Best matching GPUs
	StrategyRoundRobin  ScheduleStrategy = "round_robin"  // Distribute evenly
	StrategyPackedFirst ScheduleStrategy = "packed_first" // Fill GPUs before spreading
	StrategySpread      ScheduleStrategy = "spread"       // Spread across GPUs
)

// PriorityRule defines allocation priority
type PriorityRule struct {
	Match    string `yaml:"match" json:"match"`                           // Service name pattern
	Priority int    `yaml:"priority" json:"priority"`                     // Higher = more priority
	Reserved int    `yaml:"reserved,omitempty" json:"reserved,omitempty"` // Reserved GPU count
}

// GPUDetector is the interface for detecting GPUs
type GPUDetector interface {
	// Detect detects all available GPUs
	Detect() ([]GPU, error)

	// DetectVendor detects GPUs of a specific vendor
	DetectVendor(vendor GPUVendor) ([]GPU, error)

	// GetGPU returns a specific GPU by ID
	GetGPU(id string) (*GPU, error)

	// Refresh refreshes GPU state (utilization, temperature, etc.)
	Refresh(gpu *GPU) error

	// IsAvailable checks if GPU subsystem is available
	IsAvailable() bool
}

// GPUScheduler is the interface for GPU scheduling
type GPUScheduler interface {
	// Allocate allocates GPUs based on requirements
	Allocate(owner string, req GPURequirements) (*GPUAllocation, error)

	// Release releases a GPU allocation
	Release(allocationID string) error

	// ReleaseByOwner releases all allocations for an owner
	ReleaseByOwner(owner string) error

	// GetAllocation returns an allocation by ID
	GetAllocation(id string) (*GPUAllocation, error)

	// GetPool returns the current GPU pool state
	GetPool() *GPUPool

	// CanSatisfy checks if requirements can be satisfied
	CanSatisfy(req GPURequirements) bool

	// WaitForGPUs waits for GPUs to become available
	WaitForGPUs(req GPURequirements, timeout time.Duration) (*GPUAllocation, error)
}

// GPUStats contains GPU usage statistics
type GPUStats struct {
	GPU GPU `json:"gpu"`

	// Time-series data (last hour, 5-minute intervals)
	Timestamps  []time.Time `json:"timestamps"`
	Utilization []int       `json:"utilization"`
	Memory      []int       `json:"memory"`
	Temperature []int       `json:"temperature"`
	Power       []int       `json:"power"`
}

// GPUReport contains a comprehensive GPU report
type GPUReport struct {
	GeneratedAt     time.Time           `json:"generated_at"`
	Pool            GPUPool             `json:"pool"`
	Stats           map[string]GPUStats `json:"stats"` // GPU ID -> Stats
	Allocations     []GPUAllocation     `json:"allocations"`
	Recommendations []string            `json:"recommendations,omitempty"`
}

// ContainerGPUConfig returns Docker container GPU configuration
type ContainerGPUConfig struct {
	DeviceIDs    []string `json:"device_ids,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Driver       string   `json:"driver,omitempty"`
	Count        int      `json:"count,omitempty"`
}

// ToContainerConfig converts an allocation to Docker container config
func (a *GPUAllocation) ToContainerConfig() *ContainerGPUConfig {
	cfg := &ContainerGPUConfig{
		Capabilities: []string{"gpu", "compute", "utility"},
		Driver:       "nvidia",
	}

	for _, gpu := range a.GPUs {
		cfg.DeviceIDs = append(cfg.DeviceIDs, gpu.ID)
	}

	return cfg
}

// ParseVRAM parses VRAM string (e.g., "8G", "16GB") to bytes
func ParseVRAM(s string) int64 {
	// Implementation in detector.go
	return 0
}

// FormatVRAM formats VRAM bytes to human readable
func FormatVRAM(bytes int64) string {
	const gb = 1024 * 1024 * 1024
	if bytes >= gb {
		return formatFloat(float64(bytes)/float64(gb)) + "GB"
	}
	const mb = 1024 * 1024
	if bytes >= mb {
		return formatFloat(float64(bytes)/float64(mb)) + "MB"
	}
	return formatFloat(float64(bytes)/1024) + "KB"
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return string(rune(int64(f) + '0'))
	}
	return ""
}
