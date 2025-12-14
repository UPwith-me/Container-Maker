// Package providers defines the interface for cloud provider integrations
package providers

import (
	"context"
	"time"
)

// InstanceType defines the compute tier
type InstanceType string

const (
	InstanceTypeCPUSmall  InstanceType = "cpu-small"  // 2 vCPU, 4GB RAM
	InstanceTypeCPUMedium InstanceType = "cpu-medium" // 4 vCPU, 8GB RAM
	InstanceTypeCPULarge  InstanceType = "cpu-large"  // 8 vCPU, 16GB RAM
	InstanceTypeGPUT4     InstanceType = "gpu-t4"     // 4 vCPU, 16GB RAM, NVIDIA T4
	InstanceTypeGPUA10    InstanceType = "gpu-a10"    // 8 vCPU, 32GB RAM, NVIDIA A10
	InstanceTypeGPUA100   InstanceType = "gpu-a100"   // 8 vCPU, 80GB RAM, NVIDIA A100
)

// InstanceStatus represents the current state of an instance
type InstanceStatus string

const (
	StatusPending      InstanceStatus = "pending"
	StatusProvisioning InstanceStatus = "provisioning"
	StatusRunning      InstanceStatus = "running"
	StatusStopping     InstanceStatus = "stopping"
	StatusStopped      InstanceStatus = "stopped"
	StatusTerminating  InstanceStatus = "terminating"
	StatusTerminated   InstanceStatus = "terminated"
	StatusError        InstanceStatus = "error"
)

// ProviderType identifies the cloud provider
type ProviderType string

const (
	ProviderDocker       ProviderType = "docker"       // Local Docker (dev/testing)
	ProviderAWS          ProviderType = "aws"          // AWS ECS/Fargate
	ProviderGCP          ProviderType = "gcp"          // Google Cloud Run
	ProviderAzure        ProviderType = "azure"        // Azure Container Instances
	ProviderDigitalOcean ProviderType = "digitalocean" // DigitalOcean Droplets
	ProviderLinode       ProviderType = "linode"       // Linode/Akamai
	ProviderVultr        ProviderType = "vultr"        // Vultr
	ProviderHetzner      ProviderType = "hetzner"      // Hetzner Cloud
	ProviderOCI          ProviderType = "oci"          // Oracle Cloud
	ProviderAlibaba      ProviderType = "alibaba"      // Alibaba Cloud
	ProviderTencent      ProviderType = "tencent"      // Tencent Cloud
	ProviderLambdaLabs   ProviderType = "lambdalabs"   // Lambda Labs (GPU)
	ProviderRunpod       ProviderType = "runpod"       // RunPod (GPU)
	ProviderVast         ProviderType = "vast"         // Vast.ai (GPU)
)

// InstanceConfig defines the configuration for creating an instance
type InstanceConfig struct {
	Name         string            `json:"name"`
	Type         InstanceType      `json:"type"`
	Image        string            `json:"image"`  // Docker image
	Region       string            `json:"region"` // Cloud region
	SSHPublicKey string            `json:"ssh_public_key"`
	Env          map[string]string `json:"env"`          // Environment variables
	Ports        []int             `json:"ports"`        // Exposed ports
	Volumes      []VolumeMount     `json:"volumes"`      // Persistent volumes
	DevContainer *DevContainerSpec `json:"devcontainer"` // Optional devcontainer.json
}

// VolumeMount defines a persistent storage mount
type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	SizeGB    int    `json:"size_gb"`
}

// DevContainerSpec holds devcontainer.json configuration
type DevContainerSpec struct {
	Image             string                 `json:"image,omitempty"`
	Features          map[string]interface{} `json:"features,omitempty"`
	PostCreateCommand string                 `json:"postCreateCommand,omitempty"`
	ForwardPorts      []int                  `json:"forwardPorts,omitempty"`
}

// Instance represents a running cloud development environment
type Instance struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         InstanceType      `json:"type"`
	Status       InstanceStatus    `json:"status"`
	Provider     ProviderType      `json:"provider"`
	Region       string            `json:"region"`
	PublicIP     string            `json:"public_ip,omitempty"`
	PrivateIP    string            `json:"private_ip,omitempty"`
	SSHPort      int               `json:"ssh_port"`
	ExposedPorts map[int]int       `json:"exposed_ports"` // Container:Host port mapping
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	OwnerID      string            `json:"owner_id"`
	TeamID       string            `json:"team_id,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`

	// Cost tracking
	HourlyRate float64 `json:"hourly_rate"` // USD per hour
	TotalCost  float64 `json:"total_cost"`  // Accumulated cost
}

// InstancePricing defines pricing for instance types
type InstancePricing struct {
	Type        InstanceType `json:"type"`
	HourlyRate  float64      `json:"hourly_rate"` // USD
	VCPU        int          `json:"vcpu"`
	MemoryGB    int          `json:"memory_gb"`
	GPUType     string       `json:"gpu_type,omitempty"`
	GPUMemoryGB int          `json:"gpu_memory_gb,omitempty"`
}

// Provider is the interface that all cloud providers must implement
type Provider interface {
	// Metadata
	Name() ProviderType
	DisplayName() string
	Regions() []Region
	InstanceTypes() []InstancePricing

	// Instance lifecycle
	CreateInstance(ctx context.Context, config InstanceConfig) (*Instance, error)
	GetInstance(ctx context.Context, id string) (*Instance, error)
	ListInstances(ctx context.Context, ownerID string) ([]*Instance, error)
	StartInstance(ctx context.Context, id string) error
	StopInstance(ctx context.Context, id string) error
	DeleteInstance(ctx context.Context, id string) error

	// SSH access
	GetSSHEndpoint(ctx context.Context, id string) (host string, port int, err error)

	// Exec
	ExecCommand(ctx context.Context, id string, command []string) (stdout, stderr string, exitCode int, err error)

	// Logs
	GetLogs(ctx context.Context, id string, tail int) (string, error)
	StreamLogs(ctx context.Context, id string) (<-chan string, error)
}

// Region represents a cloud region
type Region struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Country      string `json:"country"`
	Available    bool   `json:"available"`
	GPUAvailable bool   `json:"gpu_available"`
}

// ProviderCredentials holds authentication for a cloud provider
type ProviderCredentials struct {
	Provider    ProviderType      `json:"provider"`
	Credentials map[string]string `json:"credentials"` // Provider-specific keys
}

// Registry holds all registered providers
type Registry struct {
	providers map[ProviderType]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[ProviderType]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(p Provider) {
	r.providers[p.Name()] = p
}

// Get retrieves a provider by type
func (r *Registry) Get(t ProviderType) (Provider, bool) {
	p, ok := r.providers[t]
	return p, ok
}

// List returns all registered providers
func (r *Registry) List() []Provider {
	result := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// AvailableProviders returns the list of all supported providers
func AvailableProviders() []ProviderType {
	return []ProviderType{
		ProviderDocker,
		ProviderAWS,
		ProviderGCP,
		ProviderAzure,
		ProviderDigitalOcean,
		ProviderLinode,
		ProviderVultr,
		ProviderHetzner,
		ProviderOCI,
		ProviderAlibaba,
		ProviderTencent,
		ProviderLambdaLabs,
		ProviderRunpod,
		ProviderVast,
	}
}
