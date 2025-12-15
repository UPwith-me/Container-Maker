package runtime

import (
	"context"
	"io"
)

// ContainerRuntime defines the interface for container runtime backends
type ContainerRuntime interface {
	// Metadata
	Name() string
	Type() string // "docker", "podman", "nerdctl"
	Path() string
	Version() (string, error)

	// Status checks
	IsAvailable() bool
	IsRunning() error

	// Container operations
	CreateContainer(ctx context.Context, config *ContainerConfig) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, timeout int) error
	RemoveContainer(ctx context.Context, id string, force bool) error
	ExecInContainer(ctx context.Context, id string, cmd []string, opts ExecOptions) error
	AttachContainer(ctx context.Context, id string, opts AttachOptions) (*AttachResponse, error)
	WaitContainer(ctx context.Context, id string) (<-chan int64, <-chan error)
	InspectContainer(ctx context.Context, id string) (*ContainerInfo, error)

	// Image operations
	PullImage(ctx context.Context, image string) error
	BuildImage(ctx context.Context, opts BuildOptions) (string, error)
	ImageExists(ctx context.Context, image string) bool

	// File operations
	CopyToContainer(ctx context.Context, id, destPath string, content io.Reader) error

	// Resize terminal
	ResizeContainerTTY(ctx context.Context, id string, height, width uint) error
}

// ContainerConfig holds container creation parameters
type ContainerConfig struct {
	Image        string
	Cmd          []string
	Env          []string
	WorkingDir   string
	User         string
	Hostname     string
	Entrypoint   []string
	ExposedPorts map[string]struct{}

	// Host config
	Binds          []string
	PortBindings   map[string][]PortBinding
	AutoRemove     bool
	Init           bool
	Privileged     bool
	NetworkMode    string
	CapAdd         []string
	CapDrop        []string
	Devices        []DeviceMapping
	DeviceRequests []DeviceRequest // GPU access
	SecurityOpt    []string
	ShmSize        int64

	// TTY
	Tty       bool
	OpenStdin bool
}

// DeviceRequest represents a GPU device request
type DeviceRequest struct {
	Count        int      // -1 means all GPUs
	DeviceIDs    []string // Specific device IDs
	Capabilities [][]string
}

// PortBinding represents a port binding
type PortBinding struct {
	HostIP   string
	HostPort string
}

// DeviceMapping represents a device mapping
type DeviceMapping struct {
	PathOnHost        string
	PathInContainer   string
	CgroupPermissions string
}

// ExecOptions holds exec configuration
type ExecOptions struct {
	Cmd          []string
	AttachStdout bool
	AttachStderr bool
	AttachStdin  bool
	Tty          bool
	User         string
	WorkingDir   string
}

// AttachOptions holds attach configuration
type AttachOptions struct {
	Stream bool
	Stdin  bool
	Stdout bool
	Stderr bool
	Logs   bool
}

// AttachResponse wraps an attach connection
type AttachResponse struct {
	Conn   io.ReadWriteCloser
	Reader io.Reader
}

// BuildOptions holds image build parameters
type BuildOptions struct {
	ContextDir string
	Dockerfile string
	Tags       []string
	BuildArgs  map[string]string
	CacheFrom  string
	CacheTo    string
}

// ContainerInfo holds container inspection data
type ContainerInfo struct {
	ID      string
	Name    string
	Image   string
	State   string
	Running bool
}

// BackendInfo holds backend metadata for display
type BackendInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // docker, podman, nerdctl
	Path      string `json:"path"`
	Version   string `json:"version,omitempty"`
	Available bool   `json:"available"`
	Running   bool   `json:"running"`
	IsCustom  bool   `json:"isCustom,omitempty"`
	IsActive  bool   `json:"isActive,omitempty"`
}
