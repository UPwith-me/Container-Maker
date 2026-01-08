// Package imports provides migration tools for converting existing configurations
// to Container-Maker workspace format. Supports docker-compose.yml and Helm charts.
package imports

import (
	"time"

	"github.com/UPwith-me/Container-Maker/pkg/workspace"
)

// ImportSource represents the source format for import
type ImportSource string

const (
	SourceDockerCompose ImportSource = "docker-compose"
	SourceHelm          ImportSource = "helm"
	SourceKubernetes    ImportSource = "kubernetes"
	SourceDevContainer  ImportSource = "devcontainer"
)

// ImportResult contains the result of an import operation
type ImportResult struct {
	Source     ImportSource         `json:"source"`
	SourceFile string               `json:"source_file"`
	Workspace  *workspace.Workspace `json:"workspace"`
	Warnings   []ImportWarning      `json:"warnings,omitempty"`
	Errors     []ImportError        `json:"errors,omitempty"`
	Statistics ImportStats          `json:"statistics"`
	CreatedAt  time.Time            `json:"created_at"`
}

// ImportWarning represents a non-fatal issue during import
type ImportWarning struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Service    string `json:"service,omitempty"`
	Field      string `json:"field,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// ImportError represents a fatal issue during import
type ImportError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
	Service string `json:"service,omitempty"`
}

// ImportStats contains statistics about the import
type ImportStats struct {
	ServicesImported  int `json:"services_imported"`
	ServicesSkipped   int `json:"services_skipped"`
	NetworksImported  int `json:"networks_imported"`
	VolumesImported   int `json:"volumes_imported"`
	SecretsFound      int `json:"secrets_found"`
	UnsupportedFields int `json:"unsupported_fields"`
}

// ImportOptions configures the import behavior
type ImportOptions struct {
	Source      ImportSource `json:"source"`
	SourcePath  string       `json:"source_path"`
	OutputPath  string       `json:"output_path"`
	ProjectName string       `json:"project_name,omitempty"`

	// Behavior options
	Strict          bool `json:"strict"`           // Fail on warnings
	PreservePorts   bool `json:"preserve_ports"`   // Keep original port mappings
	AddLabels       bool `json:"add_labels"`       // Add CM labels
	ConvertVolumes  bool `json:"convert_volumes"`  // Convert named volumes
	IncludeComments bool `json:"include_comments"` // Add helpful comments

	// Override options
	ImagePrefix string `json:"image_prefix,omitempty"` // Prefix for images
	NetworkName string `json:"network_name,omitempty"` // Override network name

	// Analysis options
	DryRun  bool `json:"dry_run"` // Don't write output
	Analyze bool `json:"analyze"` // Show analysis only
}

// Importer is the interface for importing configurations
type Importer interface {
	// Import imports a configuration file
	Import(opts ImportOptions) (*ImportResult, error)

	// Analyze analyzes a configuration without importing
	Analyze(path string) (*AnalysisResult, error)

	// Validate checks if the source file is valid
	Validate(path string) error

	// CanHandle checks if this importer can handle the given file
	CanHandle(path string) bool
}

// AnalysisResult contains the analysis of a source file
type AnalysisResult struct {
	Source        ImportSource        `json:"source"`
	SourceFile    string              `json:"source_file"`
	Valid         bool                `json:"valid"`
	Services      []ServiceAnalysis   `json:"services"`
	Networks      []string            `json:"networks"`
	Volumes       []string            `json:"volumes"`
	Compatibility CompatibilityReport `json:"compatibility"`
}

// ServiceAnalysis contains analysis for a single service
type ServiceAnalysis struct {
	Name           string   `json:"name"`
	Image          string   `json:"image,omitempty"`
	Build          bool     `json:"build"`
	Ports          []string `json:"ports,omitempty"`
	Dependencies   []string `json:"dependencies,omitempty"`
	Volumes        []string `json:"volumes,omitempty"`
	Environment    int      `json:"environment_count"`
	HasHealthCheck bool     `json:"has_health_check"`
	HasGPU         bool     `json:"has_gpu"`
	Warnings       []string `json:"warnings,omitempty"`
}

// CompatibilityReport shows CM compatibility
type CompatibilityReport struct {
	Score           int      `json:"score"` // 0-100
	FullySupported  []string `json:"fully_supported"`
	PartialSupport  []string `json:"partial_support"`
	NotSupported    []string `json:"not_supported"`
	Recommendations []string `json:"recommendations"`
}

// ComposeFile represents a docker-compose.yml structure
type ComposeFile struct {
	Version  string                     `yaml:"version,omitempty"`
	Services map[string]*ComposeService `yaml:"services"`
	Networks map[string]*ComposeNetwork `yaml:"networks,omitempty"`
	Volumes  map[string]*ComposeVolume  `yaml:"volumes,omitempty"`
	Secrets  map[string]*ComposeSecret  `yaml:"secrets,omitempty"`
	Configs  map[string]*ComposeConfig  `yaml:"configs,omitempty"`
}

// ComposeService represents a service in docker-compose
type ComposeService struct {
	Image           string                 `yaml:"image,omitempty"`
	Build           interface{}            `yaml:"build,omitempty"` // string or object
	Command         interface{}            `yaml:"command,omitempty"`
	Entrypoint      interface{}            `yaml:"entrypoint,omitempty"`
	Environment     interface{}            `yaml:"environment,omitempty"`
	EnvFile         interface{}            `yaml:"env_file,omitempty"`
	Ports           []interface{}          `yaml:"ports,omitempty"`
	Expose          []interface{}          `yaml:"expose,omitempty"`
	Volumes         []interface{}          `yaml:"volumes,omitempty"`
	DependsOn       interface{}            `yaml:"depends_on,omitempty"`
	Networks        interface{}            `yaml:"networks,omitempty"`
	Restart         string                 `yaml:"restart,omitempty"`
	HealthCheck     *ComposeHealthCheck    `yaml:"healthcheck,omitempty"`
	Deploy          *ComposeDeploy         `yaml:"deploy,omitempty"`
	Labels          interface{}            `yaml:"labels,omitempty"`
	WorkingDir      string                 `yaml:"working_dir,omitempty"`
	User            string                 `yaml:"user,omitempty"`
	Privileged      bool                   `yaml:"privileged,omitempty"`
	CapAdd          []string               `yaml:"cap_add,omitempty"`
	CapDrop         []string               `yaml:"cap_drop,omitempty"`
	Devices         []string               `yaml:"devices,omitempty"`
	Sysctls         map[string]string      `yaml:"sysctls,omitempty"`
	Ulimits         map[string]interface{} `yaml:"ulimits,omitempty"`
	ExtraHosts      []string               `yaml:"extra_hosts,omitempty"`
	Hostname        string                 `yaml:"hostname,omitempty"`
	DomainName      string                 `yaml:"domainname,omitempty"`
	StdinOpen       bool                   `yaml:"stdin_open,omitempty"`
	Tty             bool                   `yaml:"tty,omitempty"`
	ShmSize         string                 `yaml:"shm_size,omitempty"`
	StopSignal      string                 `yaml:"stop_signal,omitempty"`
	StopGracePeriod string                 `yaml:"stop_grace_period,omitempty"`
	Runtime         string                 `yaml:"runtime,omitempty"`
}

// ComposeHealthCheck represents healthcheck configuration
type ComposeHealthCheck struct {
	Test        interface{} `yaml:"test,omitempty"`
	Interval    string      `yaml:"interval,omitempty"`
	Timeout     string      `yaml:"timeout,omitempty"`
	Retries     int         `yaml:"retries,omitempty"`
	StartPeriod string      `yaml:"start_period,omitempty"`
	Disable     bool        `yaml:"disable,omitempty"`
}

// ComposeDeploy represents deploy configuration
type ComposeDeploy struct {
	Mode          string                `yaml:"mode,omitempty"`
	Replicas      int                   `yaml:"replicas,omitempty"`
	Resources     *ComposeResources     `yaml:"resources,omitempty"`
	RestartPolicy *ComposeRestartPolicy `yaml:"restart_policy,omitempty"`
	Placement     *ComposePlacement     `yaml:"placement,omitempty"`
}

// ComposeResources represents resource limits
type ComposeResources struct {
	Limits       *ComposeResourceSpec `yaml:"limits,omitempty"`
	Reservations *ComposeResourceSpec `yaml:"reservations,omitempty"`
}

// ComposeResourceSpec represents resource specification
type ComposeResourceSpec struct {
	CPUs    string                 `yaml:"cpus,omitempty"`
	Memory  string                 `yaml:"memory,omitempty"`
	Devices []ComposeDeviceRequest `yaml:"devices,omitempty"`
}

// ComposeDeviceRequest represents a device request (GPU)
type ComposeDeviceRequest struct {
	Driver       string      `yaml:"driver,omitempty"`
	Count        interface{} `yaml:"count,omitempty"` // int or "all"
	DeviceIDs    []string    `yaml:"device_ids,omitempty"`
	Capabilities []string    `yaml:"capabilities,omitempty"`
}

// ComposeRestartPolicy represents restart policy
type ComposeRestartPolicy struct {
	Condition   string `yaml:"condition,omitempty"`
	Delay       string `yaml:"delay,omitempty"`
	MaxAttempts int    `yaml:"max_attempts,omitempty"`
	Window      string `yaml:"window,omitempty"`
}

// ComposePlacement represents placement constraints
type ComposePlacement struct {
	Constraints []string            `yaml:"constraints,omitempty"`
	Preferences []map[string]string `yaml:"preferences,omitempty"`
}

// ComposeNetwork represents a network definition
type ComposeNetwork struct {
	Driver     string            `yaml:"driver,omitempty"`
	External   interface{}       `yaml:"external,omitempty"`
	Name       string            `yaml:"name,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
	Attachable bool              `yaml:"attachable,omitempty"`
	Internal   bool              `yaml:"internal,omitempty"`
}

// ComposeVolume represents a volume definition
type ComposeVolume struct {
	Driver     string            `yaml:"driver,omitempty"`
	External   interface{}       `yaml:"external,omitempty"`
	Name       string            `yaml:"name,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
}

// ComposeSecret represents a secret definition
type ComposeSecret struct {
	File     string `yaml:"file,omitempty"`
	External bool   `yaml:"external,omitempty"`
	Name     string `yaml:"name,omitempty"`
}

// ComposeConfig represents a config definition
type ComposeConfig struct {
	File     string `yaml:"file,omitempty"`
	External bool   `yaml:"external,omitempty"`
	Name     string `yaml:"name,omitempty"`
}
